package live

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"futhub/notification/internal/contracts"
	"futhub/notification/internal/messaging"
)

func newTestProcessor(t *testing.T) (*Processor, *RedisStore, *Hub) {
	t.Helper()
	store, _ := newTestStore(t)
	hub := NewHub()
	return NewProcessor(store, hub, quietLogger()), store, hub
}

func TestStartedCreatesStateAndSummary(t *testing.T) {
	processor, store, _ := newTestProcessor(t)
	matchID := uuid.NewString()

	if err := processor.Process(context.Background(), startedMessage(t, matchID)); err != nil {
		t.Fatal(err)
	}

	state, err := store.LoadState(context.Background(), matchID)
	if err != nil || state == nil {
		t.Fatalf("state was not created: %v", err)
	}
	if state.Sequence != 1 || state.HomeTeam.Name != "Home FC" || len(state.Events) != 0 || state.Score != (Score{}) {
		t.Errorf("unexpected initial state: %+v", state)
	}
	summary, _ := store.LoadSummary(context.Background(), matchID)
	if summary == nil {
		t.Error("summary should exist right after the match starts")
	}
}

func TestEventsAppendAndRemoveReplacingTheScoreFromTheMessage(t *testing.T) {
	processor, store, _ := newTestProcessor(t)
	ctx := context.Background()
	matchID, goalID, foulID := uuid.NewString(), uuid.NewString(), uuid.NewString()

	steps := []Envelope{
		startedMessage(t, matchID),
		goalMessage(t, matchID, goalID, 2, 1, 0),
		eventMessage(t, matchID, foulID, "FOUL", 3, 1, 0),
		removedMessage(t, matchID, goalID, 4, 0, 0),
	}
	for _, step := range steps {
		if err := processor.Process(ctx, step); err != nil {
			t.Fatalf("%s: %v", step.Type, err)
		}
	}

	state, _ := store.LoadState(ctx, matchID)
	if state.Sequence != 4 {
		t.Errorf("sequence = %d, want 4", state.Sequence)
	}
	if state.Score != (Score{Home: 0, Away: 0}) {
		t.Errorf("score = %+v, want the one carried by the removal message", state.Score)
	}
	if len(state.Events) != 1 || state.Events[0].ID != foulID {
		t.Errorf("events = %+v, want only the foul", state.Events)
	}
}

func TestDuplicateSequenceIsIgnored(t *testing.T) {
	processor, store, _ := newTestProcessor(t)
	ctx := context.Background()
	matchID := uuid.NewString()
	goal := goalMessage(t, matchID, uuid.NewString(), 2, 1, 0)

	for _, step := range []Envelope{startedMessage(t, matchID), goal} {
		if err := processor.Process(ctx, step); err != nil {
			t.Fatal(err)
		}
	}
	duplicateWithAnotherMessageID := goal
	duplicateWithAnotherMessageID.MessageID = uuid.NewString()

	if err := processor.Process(ctx, duplicateWithAnotherMessageID); err != nil {
		t.Fatalf("duplicate must be ignored without error, got %v", err)
	}

	state, _ := store.LoadState(ctx, matchID)
	if len(state.Events) != 1 || state.Sequence != 2 {
		t.Errorf("duplicate changed the state: %+v", state)
	}
}

func TestSequenceGapIsRejectedAndNothingIsApplied(t *testing.T) {
	processor, store, _ := newTestProcessor(t)
	ctx := context.Background()
	matchID := uuid.NewString()
	if err := processor.Process(ctx, startedMessage(t, matchID)); err != nil {
		t.Fatal(err)
	}

	err := processor.Process(ctx, goalMessage(t, matchID, uuid.NewString(), 3, 1, 0))

	if !errors.Is(err, ErrSequenceGap) {
		t.Fatalf("err = %v, want ErrSequenceGap", err)
	}
	if errors.Is(err, messaging.ErrInvalidMessage) {
		t.Error("a gap is transient and must go to retry, not straight to the dead letter queue")
	}
	state, _ := store.LoadState(ctx, matchID)
	if state.Sequence != 1 || len(state.Events) != 0 {
		t.Errorf("gap message was applied: %+v", state)
	}
}

func TestMessageBeforeMatchStartIsAGap(t *testing.T) {
	processor, _, _ := newTestProcessor(t)

	err := processor.Process(context.Background(), goalMessage(t, uuid.NewString(), uuid.NewString(), 2, 1, 0))

	if !errors.Is(err, ErrSequenceGap) {
		t.Fatalf("err = %v, want ErrSequenceGap", err)
	}
}

func TestFinishedPushesLastUpdateThenDeletesStateAndSnapshot(t *testing.T) {
	processor, store, hub := newTestProcessor(t)
	ctx := context.Background()
	matchID := uuid.NewString()
	if err := processor.Process(ctx, startedMessage(t, matchID)); err != nil {
		t.Fatal(err)
	}
	subscription := hub.Subscribe(matchID)
	defer subscription.Close()

	if err := processor.Process(ctx, finishedMessage(t, matchID, 2, 0, 0)); err != nil {
		t.Fatal(err)
	}

	last, open := <-subscription.Updates
	if !open || last.Type != MessageMatchFinished || !last.Final {
		t.Errorf("viewers did not receive the final update: %+v open=%v", last, open)
	}
	if _, stillOpen := <-subscription.Updates; stillOpen {
		t.Error("the stream of a finished match must be closed")
	}
	if state, _ := store.LoadState(ctx, matchID); state != nil {
		t.Error("state must be deleted when the match finishes")
	}
	if summary, _ := store.LoadSummary(ctx, matchID); summary != nil {
		t.Error("snapshot must be deleted when the match finishes")
	}
}

func TestMessagesAfterFinishAreIgnored(t *testing.T) {
	processor, store, _ := newTestProcessor(t)
	ctx := context.Background()
	matchID := uuid.NewString()
	started := startedMessage(t, matchID)
	for _, step := range []Envelope{started, finishedMessage(t, matchID, 2, 0, 0)} {
		if err := processor.Process(ctx, step); err != nil {
			t.Fatal(err)
		}
	}
	redelivered := started
	redelivered.MessageID = uuid.NewString()

	if err := processor.Process(ctx, redelivered); err != nil {
		t.Fatalf("late duplicate must be ignored, got %v", err)
	}
	if state, _ := store.LoadState(ctx, matchID); state != nil {
		t.Error("a late duplicate must not recreate the cache of a finished match")
	}
}

func TestRedisOutageIsATransientError(t *testing.T) {
	processor, _, _ := newTestProcessor(t)
	store, server := newTestStore(t)
	processor.store = store
	server.Close()

	err := processor.Process(context.Background(), startedMessage(t, uuid.NewString()))

	if !errors.Is(err, ErrCacheUnavailable) {
		t.Fatalf("err = %v, want ErrCacheUnavailable", err)
	}
	if errors.Is(err, messaging.ErrInvalidMessage) {
		t.Error("an outage must be retried, never dead lettered as invalid")
	}
}

func newTestMessageHandler(t *testing.T) (*MessageHandler, *RedisStore) {
	t.Helper()
	validator, err := contracts.NewValidator(contractsDirectory)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := newTestStore(t)
	processor := NewProcessor(store, NewHub(), quietLogger())
	return NewMessageHandler(validator, newMemoryDeduplicator(), processor, quietLogger()), store
}

func TestHandlerAppliesAMessageOnlyOnceWhenItIsDelivered5Times(t *testing.T) {
	handler, store := newTestMessageHandler(t)
	ctx := context.Background()
	matchID := uuid.NewString()
	started := rawEnvelope(t, startedMessage(t, matchID))
	goal := rawEnvelope(t, goalMessage(t, matchID, uuid.NewString(), 2, 1, 0))

	if err := handler.Handle(ctx, started); err != nil {
		t.Fatal(err)
	}
	for range 5 {
		if err := handler.Handle(ctx, goal); err != nil {
			t.Fatal(err)
		}
	}

	state, _ := store.LoadState(ctx, matchID)
	if len(state.Events) != 1 || state.Sequence != 2 || state.Score.Home != 1 {
		t.Errorf("the repeated message had more than one effect: %+v", state)
	}
}

func TestHandlerSendsInvalidMessagesToTheDeadLetterQueue(t *testing.T) {
	handler, _ := newTestMessageHandler(t)
	invalid := envelopeOf(t, MessageMatchFinished, map[string]any{"matchId": "not-a-uuid", "sequence": 0})

	for name, body := range map[string][]byte{
		"payload outside the contract": rawEnvelope(t, invalid),
		"not json":                     []byte("{broken"),
		"unknown type":                 rawEnvelope(t, envelopeOf(t, "SomethingElse", map[string]any{})),
	} {
		t.Run(name, func(t *testing.T) {
			err := handler.Handle(context.Background(), body)
			if !errors.Is(err, messaging.ErrInvalidMessage) {
				t.Errorf("err = %v, want ErrInvalidMessage", err)
			}
		})
	}
}

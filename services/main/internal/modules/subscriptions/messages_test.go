package subscriptions

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/messaging"
)

type fakeValidator struct{ err error }

func (f fakeValidator) Validate([]byte) error { return f.err }

type fakeDeduplicator struct {
	seen map[string]bool
}

func (f *fakeDeduplicator) RunOnce(ctx context.Context, messageID string, effect func(ctx context.Context) error) (bool, error) {
	if f.seen[messageID] {
		return false, nil
	}
	if err := effect(ctx); err != nil {
		return false, err
	}
	f.seen[messageID] = true
	return true, nil
}

func approvedMessage(messageID, subscriptionID, userID uuid.UUID) []byte {
	body, _ := json.Marshal(map[string]any{
		"messageId": messageID, "correlationId": uuid.New(), "type": entities.MessagePaymentApproved, "version": 1,
		"occurredAt": "2026-11-26T12:00:00Z",
		"payload":    map[string]any{"subscriptionId": subscriptionID, "userId": userID, "chargeId": uuid.New()},
	})
	return body
}

func newHandler(env testEnvironment, validator fakeValidator) *MessageHandler {
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewMessageHandler(validator, &fakeDeduplicator{seen: map[string]bool{}}, env.useCase, quiet)
}

func TestHandlerActivatesTheSubscriptionAndIgnoresTheSameMessageDeliveredAgain(t *testing.T) {
	env := newTestEnvironment()
	caller := newCaller()
	subscription := env.subscribe(t, caller)
	handler := newHandler(env, fakeValidator{})
	body := approvedMessage(uuid.New(), subscription.ID, caller.UserID)

	firstErr := handler.Handle(context.Background(), body)
	secondErr := handler.Handle(context.Background(), body)

	if firstErr != nil || secondErr != nil {
		t.Fatalf("unexpected errors: %v / %v", firstErr, secondErr)
	}
	if got := env.outbox.types(); len(got) != 2 {
		t.Errorf("the duplicate must not publish again, got %v", got)
	}
}

func TestHandlerRejectsMessagesThatBreakTheContractWithoutTouchingTheSubscription(t *testing.T) {
	env := newTestEnvironment()
	caller := newCaller()
	subscription := env.subscribe(t, caller)
	handler := newHandler(env, fakeValidator{err: errors.Join(messaging.ErrInvalidMessage, errors.New("missing chargeId"))})

	err := handler.Handle(context.Background(), approvedMessage(uuid.New(), subscription.ID, caller.UserID))

	stored, _ := env.repo.FindByID(context.Background(), subscription.ID)
	if !errors.Is(err, messaging.ErrInvalidMessage) || stored.Status != entities.SubscriptionPending {
		t.Errorf("want ErrInvalidMessage and an untouched subscription, got %v / %s", err, stored.Status)
	}
}

func TestHandlerRejectsMessageTypesItDoesNotHandle(t *testing.T) {
	env := newTestEnvironment()
	handler := newHandler(env, fakeValidator{})
	body, _ := json.Marshal(map[string]any{"messageId": uuid.New(), "correlationId": uuid.New(), "type": "MatchStarted", "payload": map[string]any{}})

	err := handler.Handle(context.Background(), body)

	if !errors.Is(err, messaging.ErrInvalidMessage) {
		t.Errorf("want ErrInvalidMessage, got %v", err)
	}
}

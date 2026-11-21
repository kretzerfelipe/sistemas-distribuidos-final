package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
)

type inMemoryStore struct {
	messages  []Message
	published map[uuid.UUID]bool
}

func newInMemoryStore(messages ...Message) *inMemoryStore {
	return &inMemoryStore{messages: messages, published: map[uuid.UUID]bool{}}
}

func (s *inMemoryStore) FetchUnpublished(_ context.Context, limit int) ([]Message, error) {
	pending := []Message{}
	for _, message := range s.messages {
		if !s.published[message.ID] && len(pending) < limit {
			pending = append(pending, message)
		}
	}
	return pending, nil
}

func (s *inMemoryStore) MarkPublished(_ context.Context, id uuid.UUID, _ time.Time) error {
	s.published[id] = true
	return nil
}

type fakeBroker struct {
	delivered []string
	failFrom  int
	attempts  int
}

func (b *fakeBroker) Publish(_ context.Context, _, routingKey, _ string, _ []byte) error {
	b.attempts++
	brokerIsDown := b.failFrom > 0 && b.attempts >= b.failFrom
	if brokerIsDown {
		return errors.New("broker is down")
	}
	b.delivered = append(b.delivered, routingKey)
	return nil
}

func newMessage(routingKey string) Message {
	return Message{ID: uuid.New(), CorrelationID: uuid.New(), Type: "Test", RoutingKey: routingKey, Body: []byte("{}")}
}

func newTestPublisher(store Store, broker MessagePublisher, batchSize int) *Publisher {
	return NewPublisher(store, broker, "futhub.events", time.Millisecond, batchSize, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestPublisherPublishesEverythingInOrderAndMarksAsPublished(t *testing.T) {
	store := newInMemoryStore(newMessage("a"), newMessage("b"), newMessage("c"))
	broker := &fakeBroker{}

	newTestPublisher(store, broker, 10).drain(context.Background())

	if len(broker.delivered) != 3 || broker.delivered[0] != "a" || broker.delivered[1] != "b" || broker.delivered[2] != "c" {
		t.Errorf("messages must be published in order, got %v", broker.delivered)
	}
	pending, _ := store.FetchUnpublished(context.Background(), 10)
	if len(pending) != 0 {
		t.Errorf("everything must be marked as published, %d are pending", len(pending))
	}
}

func TestPublisherStopsAtTheFirstFailureAndKeepsTheRest(t *testing.T) {
	store := newInMemoryStore(newMessage("a"), newMessage("b"), newMessage("c"))
	broker := &fakeBroker{failFrom: 2}

	newTestPublisher(store, broker, 10).drain(context.Background())

	if len(broker.delivered) != 1 || broker.delivered[0] != "a" {
		t.Errorf("only the first message was delivered, got %v", broker.delivered)
	}
	pending, _ := store.FetchUnpublished(context.Background(), 10)
	if len(pending) != 2 || pending[0].RoutingKey != "b" || pending[1].RoutingKey != "c" {
		t.Errorf("b and c must stay pending in order, got %+v", pending)
	}
}

func TestPublisherDrainsTheBacklogWhenTheBrokerComesBack(t *testing.T) {
	store := newInMemoryStore(newMessage("a"), newMessage("b"), newMessage("c"), newMessage("d"))
	broker := &fakeBroker{failFrom: 1}
	publisher := newTestPublisher(store, broker, 2)

	publisher.drain(context.Background())
	brokerWasDownAndNothingWasPublished := len(broker.delivered) == 0
	broker.failFrom = 0
	publisher.drain(context.Background())

	if !brokerWasDownAndNothingWasPublished {
		t.Fatal("nothing can be delivered while the broker is down")
	}
	if len(broker.delivered) != 4 || broker.delivered[0] != "a" || broker.delivered[3] != "d" {
		t.Errorf("the whole backlog must drain in order, got %v", broker.delivered)
	}
}

func TestPublisherNeverPublishesTheSameMessageTwice(t *testing.T) {
	store := newInMemoryStore(newMessage("a"), newMessage("b"))
	broker := &fakeBroker{}
	publisher := newTestPublisher(store, broker, 10)

	publisher.drain(context.Background())
	publisher.drain(context.Background())

	if len(broker.delivered) != 2 {
		t.Errorf("each message must be published once, got %v", broker.delivered)
	}
}

func TestBuildMessageCreatesTheEnvelopeOfTheContract(t *testing.T) {
	correlationID := uuid.New()
	now := time.Date(2027, 3, 1, 15, 0, 0, 0, time.UTC)

	message, err := buildMessage(Event{Type: "MatchStarted", RoutingKey: "match.started", Payload: map[string]int{"sequence": 1}}, correlationID.String(), now)

	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(message.Body, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"messageId", "correlationId", "type", "version", "occurredAt", "payload"} {
		if _, exists := decoded[field]; !exists {
			t.Errorf("the envelope must have %q: %s", field, message.Body)
		}
	}
	if decoded["messageId"] != message.ID.String() || decoded["correlationId"] != correlationID.String() || decoded["type"] != "MatchStarted" || decoded["version"] != float64(1) {
		t.Errorf("unexpected envelope: %s", message.Body)
	}
	if message.RoutingKey != "match.started" {
		t.Errorf("unexpected routing key %q", message.RoutingKey)
	}
}

func TestBuildMessageGeneratesACorrelationIDWhenTheRequestOneIsNotAUUID(t *testing.T) {
	first, err := buildMessage(Event{Type: "Test", Payload: struct{}{}}, "not-a-uuid", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	second, _ := buildMessage(Event{Type: "Test", Payload: struct{}{}}, "", time.Now())

	if first.CorrelationID == uuid.Nil || second.CorrelationID == uuid.Nil || first.CorrelationID == second.CorrelationID {
		t.Errorf("a valid and distinct correlationId must be generated: %v / %v", first.CorrelationID, second.CorrelationID)
	}
}

func TestBuildMessageGivesEachMessageItsOwnID(t *testing.T) {
	first, _ := buildMessage(Event{Type: "Test", Payload: struct{}{}}, "", time.Now())
	second, _ := buildMessage(Event{Type: "Test", Payload: struct{}{}}, "", time.Now())

	if first.ID == second.ID {
		t.Error("messageId must be unique per message")
	}
}

func TestAddingToTheOutboxOutsideATransactionIsRefused(t *testing.T) {
	writer := &postgresWriter{now: time.Now}

	err := writer.Add(context.Background(), Event{Type: "Test", Payload: struct{}{}})

	if !errors.Is(err, ErrNoTransaction) {
		t.Errorf("want ErrNoTransaction, got %v", err)
	}
}

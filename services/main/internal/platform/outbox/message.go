package outbox

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const envelopeVersion = 1

type Event struct {
	Type       string
	RoutingKey string
	Payload    any
}

type Message struct {
	ID            uuid.UUID
	CorrelationID uuid.UUID
	Type          string
	RoutingKey    string
	Body          []byte
	CreatedAt     time.Time
}

type envelope struct {
	MessageID     uuid.UUID       `json:"messageId"`
	CorrelationID uuid.UUID       `json:"correlationId"`
	Type          string          `json:"type"`
	Version       int             `json:"version"`
	OccurredAt    time.Time       `json:"occurredAt"`
	Payload       json.RawMessage `json:"payload"`
}

func buildMessage(event Event, requestCorrelationID string, now time.Time) (*Message, error) {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload of %s: %w", event.Type, err)
	}

	messageID := uuid.New()
	correlationID := parseCorrelationID(requestCorrelationID)
	occurredAt := now.UTC()

	body, err := json.Marshal(envelope{
		MessageID:     messageID,
		CorrelationID: correlationID,
		Type:          event.Type,
		Version:       envelopeVersion,
		OccurredAt:    occurredAt,
		Payload:       payload,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal envelope of %s: %w", event.Type, err)
	}

	return &Message{
		ID:            messageID,
		CorrelationID: correlationID,
		Type:          event.Type,
		RoutingKey:    event.RoutingKey,
		Body:          body,
		CreatedAt:     occurredAt,
	}, nil
}

func parseCorrelationID(requestCorrelationID string) uuid.UUID {
	correlationID, err := uuid.Parse(requestCorrelationID)
	correlationIDIsNotAUUID := err != nil
	if correlationIDIsNotAUUID {
		return uuid.New()
	}
	return correlationID
}

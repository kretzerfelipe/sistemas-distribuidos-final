package subscriptions

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/httpx"
	"futhub/main/internal/platform/messaging"
)

type SchemaValidator interface {
	Validate(body []byte) error
}

type Deduplicator interface {
	RunOnce(ctx context.Context, messageID string, effect func(ctx context.Context) error) (bool, error)
}

type envelope struct {
	MessageID     string          `json:"messageId"`
	CorrelationID string          `json:"correlationId"`
	Type          string          `json:"type"`
	Payload       json.RawMessage `json:"payload"`
}

type MessageHandler struct {
	validator    SchemaValidator
	deduplicator Deduplicator
	uc           UseCase
	log          *slog.Logger
}

func NewMessageHandler(validator SchemaValidator, deduplicator Deduplicator, uc UseCase, log *slog.Logger) *MessageHandler {
	return &MessageHandler{validator: validator, deduplicator: deduplicator, uc: uc, log: log}
}

func (h *MessageHandler) Handle(ctx context.Context, body []byte) error {
	err := h.validator.Validate(body)
	if err != nil {
		return err
	}

	var message envelope
	err = json.Unmarshal(body, &message)
	if err != nil {
		return fmt.Errorf("%w: envelope: %v", messaging.ErrInvalidMessage, err)
	}

	log := h.log.With("messageId", message.MessageID, "correlationId", message.CorrelationID, "type", message.Type)
	ctx = httpx.WithCorrelationID(ctx, message.CorrelationID)

	processedNow, err := h.deduplicator.RunOnce(ctx, message.MessageID, func(ctx context.Context) error {
		return h.apply(ctx, message)
	})
	if err != nil {
		return err
	}

	if !processedNow {
		log.Info("duplicate message ignored")
		return nil
	}
	log.Info("message processed")
	return nil
}

func (h *MessageHandler) apply(ctx context.Context, message envelope) error {
	switch message.Type {
	case entities.MessagePaymentApproved:
		var payload entities.PaymentApprovedPayload
		err := decode(message, &payload)
		if err != nil {
			return err
		}
		return h.uc.HandlePaymentApproved(ctx, payload)
	case entities.MessagePaymentRejected:
		var payload entities.PaymentRejectedPayload
		err := decode(message, &payload)
		if err != nil {
			return err
		}
		return h.uc.HandlePaymentRejected(ctx, payload)
	default:
		return fmt.Errorf("%w: unsupported message type %q", messaging.ErrInvalidMessage, message.Type)
	}
}

func decode(message envelope, target any) error {
	err := json.Unmarshal(message.Payload, target)
	if err != nil {
		return fmt.Errorf("%w: %s payload: %v", messaging.ErrInvalidMessage, message.Type, err)
	}
	return nil
}

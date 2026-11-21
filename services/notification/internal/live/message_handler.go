package live

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"futhub/notification/internal/messaging"
)

type SchemaValidator interface {
	Validate(body []byte) error
}

type Deduplicator interface {
	RunOnce(ctx context.Context, messageID string, effect func(ctx context.Context) error) (bool, error)
}

type MessageHandler struct {
	validator    SchemaValidator
	deduplicator Deduplicator
	processor    *Processor
	log          *slog.Logger
}

func NewMessageHandler(validator SchemaValidator, deduplicator Deduplicator, processor *Processor, log *slog.Logger) *MessageHandler {
	return &MessageHandler{validator: validator, deduplicator: deduplicator, processor: processor, log: log}
}

func (h *MessageHandler) Handle(ctx context.Context, body []byte) error {
	if err := h.validator.Validate(body); err != nil {
		return err
	}

	var envelope Envelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("%w: envelope: %v", messaging.ErrInvalidMessage, err)
	}

	log := h.log.With("messageId", envelope.MessageID, "correlationId", envelope.CorrelationID, "type", envelope.Type)

	processedNow, err := h.deduplicator.RunOnce(ctx, envelope.MessageID, func(ctx context.Context) error {
		return h.processor.Process(ctx, envelope)
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

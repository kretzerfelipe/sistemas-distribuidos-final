package live

import (
	"context"
	"fmt"

	"futhub/notification/internal/messaging"
)

type EnvelopeProcessor interface {
	Process(ctx context.Context, envelope Envelope) error
}

type TypeRouter struct {
	processors map[string]EnvelopeProcessor
}

func NewTypeRouter() *TypeRouter {
	return &TypeRouter{processors: make(map[string]EnvelopeProcessor)}
}

func (r *TypeRouter) Route(messageTypes []string, processor EnvelopeProcessor) {
	for _, messageType := range messageTypes {
		r.processors[messageType] = processor
	}
}

func (r *TypeRouter) Process(ctx context.Context, envelope Envelope) error {
	processor, known := r.processors[envelope.Type]
	if !known {
		return fmt.Errorf("%w: unsupported message type %q", messaging.ErrInvalidMessage, envelope.Type)
	}
	return processor.Process(ctx, envelope)
}

var MatchMessageTypes = []string{MessageMatchStarted, MessageMatchEventRegistered, MessageMatchEventRemoved, MessageMatchFinished}

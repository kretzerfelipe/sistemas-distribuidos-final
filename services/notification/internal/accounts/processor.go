package accounts

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"futhub/notification/internal/live"
	"futhub/notification/internal/messaging"
)

const (
	MessageSubscriptionActivated = "SubscriptionActivated"
	MessageSubscriptionCanceled  = "SubscriptionCanceled"
	MessageRefundCompleted       = "RefundCompleted"

	cancellationPaymentRejected = "PAYMENT_REJECTED"
	cancellationTimeout         = "TIMEOUT"
)

var HandledMessageTypes = []string{MessageSubscriptionActivated, MessageSubscriptionCanceled, MessageRefundCompleted}

type subscriptionPayload struct {
	SubscriptionID string `json:"subscriptionId"`
	UserID         string `json:"userId"`
	Reason         string `json:"reason"`
}

type Processor struct {
	store Store
	now   func() time.Time
}

func NewProcessor(store Store) *Processor {
	return &Processor{store: store, now: time.Now}
}

func (p *Processor) Process(ctx context.Context, envelope live.Envelope) error {
	var payload subscriptionPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return fmt.Errorf("%w: %s payload: %v", messaging.ErrInvalidMessage, envelope.Type, err)
	}

	switch envelope.Type {
	case MessageSubscriptionActivated:
		return p.activate(ctx, payload)
	case MessageSubscriptionCanceled:
		return p.cancel(ctx, payload)
	case MessageRefundCompleted:
		return p.refund(ctx, payload)
	default:
		return fmt.Errorf("%w: unsupported message type %q", messaging.ErrInvalidMessage, envelope.Type)
	}
}

func (p *Processor) activate(ctx context.Context, payload subscriptionPayload) error {
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		return fmt.Errorf("%w: userId: %v", messaging.ErrInvalidMessage, err)
	}
	if err := p.store.SetPremium(ctx, userID, true); err != nil {
		return err
	}
	return p.notify(ctx, userID, "SUBSCRIPTION_ACTIVATED", "Sua assinatura premium foi ativada. Agora você acompanha as partidas em tempo real.")
}

func (p *Processor) cancel(ctx context.Context, payload subscriptionPayload) error {
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		return fmt.Errorf("%w: userId: %v", messaging.ErrInvalidMessage, err)
	}
	if err := p.store.SetPremium(ctx, userID, false); err != nil {
		return err
	}

	message := "Sua assinatura foi cancelada."
	switch payload.Reason {
	case cancellationPaymentRejected:
		message = "O pagamento da assinatura foi recusado e ela foi cancelada. Você pode tentar de novo quando quiser."
	case cancellationTimeout:
		message = "O pagamento da assinatura demorou mais do que o permitido e ela foi cancelada. Você pode tentar de novo quando quiser."
	}
	return p.notify(ctx, userID, "SUBSCRIPTION_CANCELED", message)
}

func (p *Processor) refund(ctx context.Context, payload subscriptionPayload) error {
	userIDIsUnknown := payload.UserID == ""
	if userIDIsUnknown {
		return nil
	}
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		return fmt.Errorf("%w: userId: %v", messaging.ErrInvalidMessage, err)
	}
	return p.notify(ctx, userID, "REFUND_COMPLETED", "O valor cobrado na assinatura foi estornado.")
}

func (p *Processor) notify(ctx context.Context, userID uuid.UUID, notificationType, message string) error {
	return p.store.AddNotification(ctx, Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      notificationType,
		Message:   message,
		CreatedAt: p.now().UTC(),
	})
}

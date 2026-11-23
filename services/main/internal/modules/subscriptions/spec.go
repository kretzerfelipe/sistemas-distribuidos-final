package subscriptions

import (
	"context"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/outbox"
)

func NewSubscriptionResponse(subscription *entities.Subscription) entities.SubscriptionResponse {
	return entities.SubscriptionResponse{
		ID:          subscription.ID,
		Status:      subscription.Status,
		Premium:     subscription.Status == entities.SubscriptionActive,
		Reason:      subscription.Reason,
		AmountCents: subscription.AmountCents,
		ExpiresAt:   subscription.ExpiresAt,
		CreatedAt:   subscription.CreatedAt,
	}
}

type Settings struct {
	PaymentTimeout time.Duration
	AmountCents    int
}

type Transactions interface {
	WithTx(ctx context.Context, work func(ctx context.Context) error) error
}

type UseCase interface {
	Subscribe(ctx context.Context, caller *entities.Caller) (*entities.Subscription, error)
	GetMine(ctx context.Context, caller *entities.Caller) (*entities.Subscription, error)
	GetByID(ctx context.Context, caller *entities.Caller, id uuid.UUID) (*entities.Subscription, error)

	HandlePaymentApproved(ctx context.Context, in entities.PaymentApprovedPayload) error
	HandlePaymentRejected(ctx context.Context, in entities.PaymentRejectedPayload) error
	ExpirePending(ctx context.Context) (int, error)
}

type Repository interface {
	Insert(ctx context.Context, subscription *entities.Subscription) error
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Subscription, error)
	FindLatestByUser(ctx context.Context, userID uuid.UUID) (*entities.Subscription, error)
	Activate(ctx context.Context, id, chargeID uuid.UUID, updatedAt time.Time) (bool, error)
	Close(ctx context.Context, id uuid.UUID, to entities.SubscriptionStatus, reason string, updatedAt time.Time) (bool, error)
	ExpireOverdue(ctx context.Context, now time.Time) ([]entities.Subscription, error)
}

type Outbox = outbox.Writer

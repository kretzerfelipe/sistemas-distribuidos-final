package entities

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionStatus string

const (
	SubscriptionPending  SubscriptionStatus = "PENDING"
	SubscriptionActive   SubscriptionStatus = "ACTIVE"
	SubscriptionCanceled SubscriptionStatus = "CANCELED"
	SubscriptionExpired  SubscriptionStatus = "EXPIRED"
)

const (
	CancellationPaymentRejected = "PAYMENT_REJECTED"
	CancellationTimeout         = "TIMEOUT"
)

type Subscription struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Status      SubscriptionStatus
	Reason      string
	ChargeID    *uuid.UUID
	AmountCents int
	ExpiresAt   time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SubscriptionResponse struct {
	ID          uuid.UUID          `json:"id"`
	Status      SubscriptionStatus `json:"status"`
	Premium     bool               `json:"premium"`
	Reason      string             `json:"reason,omitempty"`
	AmountCents int                `json:"amountCents"`
	ExpiresAt   time.Time          `json:"expiresAt"`
	CreatedAt   time.Time          `json:"createdAt"`
}

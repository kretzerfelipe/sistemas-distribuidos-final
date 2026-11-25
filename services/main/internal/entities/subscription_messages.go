package entities

import "github.com/google/uuid"

const (
	MessagePaymentRequested     = "PaymentRequested"
	MessageSubscriptionActive   = "SubscriptionActivated"
	MessageSubscriptionCanceled = "SubscriptionCanceled"
	MessageRefundRequested      = "RefundRequested"

	MessagePaymentApproved = "PaymentApproved"
	MessagePaymentRejected = "PaymentRejected"

	RoutingKeyPaymentRequested     = "payment.requested"
	RoutingKeySubscriptionActive   = "subscription.activated"
	RoutingKeySubscriptionCanceled = "subscription.canceled"
	RoutingKeyRefundRequested      = "refund.requested"
)

type PaymentRequestedPayload struct {
	SubscriptionID uuid.UUID `json:"subscriptionId"`
	UserID         uuid.UUID `json:"userId"`
	AmountCents    int       `json:"amountCents"`
}

type SubscriptionActivatedPayload struct {
	SubscriptionID uuid.UUID `json:"subscriptionId"`
	UserID         uuid.UUID `json:"userId"`
}

type SubscriptionCanceledPayload struct {
	SubscriptionID uuid.UUID `json:"subscriptionId"`
	UserID         uuid.UUID `json:"userId"`
	Reason         string    `json:"reason"`
}

type RefundRequestedPayload struct {
	SubscriptionID uuid.UUID `json:"subscriptionId"`
	ChargeID       uuid.UUID `json:"chargeId"`
}

type PaymentApprovedPayload struct {
	SubscriptionID uuid.UUID `json:"subscriptionId"`
	UserID         uuid.UUID `json:"userId"`
	ChargeID       uuid.UUID `json:"chargeId"`
}

type PaymentRejectedPayload struct {
	SubscriptionID uuid.UUID `json:"subscriptionId"`
	UserID         uuid.UUID `json:"userId"`
	Reason         string    `json:"reason"`
}

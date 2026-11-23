package entities

import (
	"testing"

	"github.com/google/uuid"
)

func TestSubscriptionMessagesPublishedByMainFollowTheirContracts(t *testing.T) {
	scenarios := map[string]struct {
		schema      string
		messageType string
		payload     any
	}{
		"PaymentRequested": {"PaymentRequested.schema.json", MessagePaymentRequested,
			PaymentRequestedPayload{SubscriptionID: uuid.New(), UserID: uuid.New(), AmountCents: 1990}},
		"SubscriptionActivated": {"SubscriptionActivated.schema.json", MessageSubscriptionActive,
			SubscriptionActivatedPayload{SubscriptionID: uuid.New(), UserID: uuid.New()}},
		"SubscriptionCanceled by rejection": {"SubscriptionCanceled.schema.json", MessageSubscriptionCanceled,
			SubscriptionCanceledPayload{SubscriptionID: uuid.New(), UserID: uuid.New(), Reason: CancellationPaymentRejected}},
		"SubscriptionCanceled by timeout": {"SubscriptionCanceled.schema.json", MessageSubscriptionCanceled,
			SubscriptionCanceledPayload{SubscriptionID: uuid.New(), UserID: uuid.New(), Reason: CancellationTimeout}},
		"RefundRequested": {"RefundRequested.schema.json", MessageRefundRequested,
			RefundRequestedPayload{SubscriptionID: uuid.New(), ChargeID: uuid.New()}},
	}

	for name, scenario := range scenarios {
		t.Run(name, func(t *testing.T) {
			err := validateAgainstContract(t, scenario.schema, envelopeAround(scenario.messageType, scenario.payload))

			if err != nil {
				t.Errorf("the contract does not accept what main publishes: %v", err)
			}
		})
	}
}

func TestSubscriptionMessagesConsumedByMainAreWhatPaymentPublishes(t *testing.T) {
	approved := map[string]any{
		"subscriptionId": uuid.NewString(), "userId": uuid.NewString(), "chargeId": uuid.NewString(),
		"amountCents": 1990, "attempts": 1,
	}
	rejected := map[string]any{
		"subscriptionId": uuid.NewString(), "userId": uuid.NewString(), "reason": "GATEWAY_ATTEMPTS_EXHAUSTED", "attempts": 3,
	}

	approvedErr := validateAgainstContract(t, "PaymentApproved.schema.json", envelopeAround(MessagePaymentApproved, approved))
	rejectedErr := validateAgainstContract(t, "PaymentRejected.schema.json", envelopeAround(MessagePaymentRejected, rejected))

	if approvedErr != nil || rejectedErr != nil {
		t.Errorf("main cannot read what payment publishes: %v / %v", approvedErr, rejectedErr)
	}
}

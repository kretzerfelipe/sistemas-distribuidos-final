package subscriptions

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/outbox"
	"futhub/main/internal/platform/util"
)

type useCase struct {
	repo         Repository
	transactions Transactions
	outbox       Outbox
	settings     Settings
	now          func() time.Time
}

func newUseCase(repo Repository, transactions Transactions, outbox Outbox, settings Settings) UseCase {
	return &useCase{repo: repo, transactions: transactions, outbox: outbox, settings: settings, now: time.Now}
}

func (uc *useCase) Subscribe(ctx context.Context, caller *entities.Caller) (*entities.Subscription, error) {
	now := uc.now().UTC()
	subscription := &entities.Subscription{
		ID:          uuid.New(),
		UserID:      caller.UserID,
		Status:      entities.SubscriptionPending,
		AmountCents: uc.settings.AmountCents,
		ExpiresAt:   now.Add(uc.settings.PaymentTimeout),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := uc.transactions.WithTx(ctx, func(ctx context.Context) error {
		err := uc.repo.Insert(ctx, subscription)
		if err != nil {
			return err
		}
		return uc.outbox.Add(ctx, outbox.Event{
			Type:       entities.MessagePaymentRequested,
			RoutingKey: entities.RoutingKeyPaymentRequested,
			Payload: entities.PaymentRequestedPayload{
				SubscriptionID: subscription.ID,
				UserID:         subscription.UserID,
				AmountCents:    subscription.AmountCents,
			},
		})
	})
	if err != nil {
		return nil, util.SendMessageWithError("use case error creating subscription", err)
	}
	return subscription, nil
}

func (uc *useCase) GetMine(ctx context.Context, caller *entities.Caller) (*entities.Subscription, error) {
	subscription, err := uc.repo.FindLatestByUser(ctx, caller.UserID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding the subscription of the caller", err)
	}
	return subscription, nil
}

func (uc *useCase) GetByID(ctx context.Context, caller *entities.Caller, id uuid.UUID) (*entities.Subscription, error) {
	subscription, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding subscription", err)
	}

	callerOwnsIt := subscription.UserID == caller.UserID
	callerIsAdmin := caller.Role == entities.RoleAdmin
	if !callerOwnsIt && !callerIsAdmin {
		return nil, entities.ErrForbidden
	}
	return subscription, nil
}

func (uc *useCase) HandlePaymentApproved(ctx context.Context, in entities.PaymentApprovedPayload) error {
	activated, err := uc.repo.Activate(ctx, in.SubscriptionID, in.ChargeID, uc.now().UTC())
	if err != nil {
		return util.SendMessageWithError("use case error activating subscription", err)
	}
	if activated {
		return uc.outbox.Add(ctx, outbox.Event{
			Type:       entities.MessageSubscriptionActive,
			RoutingKey: entities.RoutingKeySubscriptionActive,
			Payload:    entities.SubscriptionActivatedPayload{SubscriptionID: in.SubscriptionID, UserID: in.UserID},
		})
	}

	current, err := uc.repo.FindByID(ctx, in.SubscriptionID)
	subscriptionIsUnknown := errors.Is(err, entities.ErrNotFound)
	if err != nil && !subscriptionIsUnknown {
		return util.SendMessageWithError("use case error finding subscription of a late payment", err)
	}

	alreadyActiveWithThisCharge := current != nil && current.Status == entities.SubscriptionActive
	if alreadyActiveWithThisCharge {
		return nil
	}

	slog.Warn("payment approved for a subscription that is not pending, requesting a refund",
		"subscriptionId", in.SubscriptionID, "chargeId", in.ChargeID)
	return uc.outbox.Add(ctx, outbox.Event{
		Type:       entities.MessageRefundRequested,
		RoutingKey: entities.RoutingKeyRefundRequested,
		Payload:    entities.RefundRequestedPayload{SubscriptionID: in.SubscriptionID, ChargeID: in.ChargeID},
	})
}

func (uc *useCase) HandlePaymentRejected(ctx context.Context, in entities.PaymentRejectedPayload) error {
	closed, err := uc.repo.Close(ctx, in.SubscriptionID, entities.SubscriptionCanceled, entities.CancellationPaymentRejected, uc.now().UTC())
	if err != nil {
		return util.SendMessageWithError("use case error canceling subscription", err)
	}

	subscriptionWasNotPending := !closed
	if subscriptionWasNotPending {
		return nil
	}
	return uc.addCanceled(ctx, in.SubscriptionID, in.UserID, entities.CancellationPaymentRejected)
}

func (uc *useCase) ExpirePending(ctx context.Context) (int, error) {
	expiredCount := 0

	err := uc.transactions.WithTx(ctx, func(ctx context.Context) error {
		expired, err := uc.repo.ExpireOverdue(ctx, uc.now().UTC())
		if err != nil {
			return err
		}
		for _, subscription := range expired {
			err = uc.addCanceled(ctx, subscription.ID, subscription.UserID, entities.CancellationTimeout)
			if err != nil {
				return err
			}
		}
		expiredCount = len(expired)
		return nil
	})
	if err != nil {
		return 0, util.SendMessageWithError("use case error expiring pending subscriptions", err)
	}
	return expiredCount, nil
}

func (uc *useCase) addCanceled(ctx context.Context, subscriptionID, userID uuid.UUID, reason string) error {
	return uc.outbox.Add(ctx, outbox.Event{
		Type:       entities.MessageSubscriptionCanceled,
		RoutingKey: entities.RoutingKeySubscriptionCanceled,
		Payload:    entities.SubscriptionCanceledPayload{SubscriptionID: subscriptionID, UserID: userID, Reason: reason},
	})
}

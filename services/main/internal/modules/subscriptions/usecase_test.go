package subscriptions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/outbox"
)

type inMemoryRepository struct {
	byID map[uuid.UUID]entities.Subscription
}

func (r *inMemoryRepository) Insert(_ context.Context, subscription *entities.Subscription) error {
	for _, existing := range r.byID {
		inForce := existing.Status == entities.SubscriptionPending || existing.Status == entities.SubscriptionActive
		if existing.UserID == subscription.UserID && inForce {
			return entities.ErrSubscriptionInForce
		}
	}
	r.byID[subscription.ID] = *subscription
	return nil
}

func (r *inMemoryRepository) FindByID(_ context.Context, id uuid.UUID) (*entities.Subscription, error) {
	subscription, found := r.byID[id]
	if !found {
		return nil, entities.ErrNotFound
	}
	return &subscription, nil
}

func (r *inMemoryRepository) FindLatestByUser(_ context.Context, userID uuid.UUID) (*entities.Subscription, error) {
	var latest *entities.Subscription
	for _, subscription := range r.byID {
		isNewer := latest == nil || subscription.CreatedAt.After(latest.CreatedAt)
		if subscription.UserID == userID && isNewer {
			copied := subscription
			latest = &copied
		}
	}
	if latest == nil {
		return nil, entities.ErrNotFound
	}
	return latest, nil
}

func (r *inMemoryRepository) Activate(_ context.Context, id, chargeID uuid.UUID, _ time.Time) (bool, error) {
	subscription, found := r.byID[id]
	if !found || subscription.Status != entities.SubscriptionPending {
		return false, nil
	}
	subscription.Status = entities.SubscriptionActive
	subscription.ChargeID = &chargeID
	r.byID[id] = subscription
	return true, nil
}

func (r *inMemoryRepository) Close(_ context.Context, id uuid.UUID, to entities.SubscriptionStatus, reason string, _ time.Time) (bool, error) {
	subscription, found := r.byID[id]
	if !found || subscription.Status != entities.SubscriptionPending {
		return false, nil
	}
	subscription.Status = to
	subscription.Reason = reason
	r.byID[id] = subscription
	return true, nil
}

func (r *inMemoryRepository) ExpireOverdue(_ context.Context, now time.Time) ([]entities.Subscription, error) {
	expired := []entities.Subscription{}
	for id, subscription := range r.byID {
		if subscription.Status == entities.SubscriptionPending && subscription.ExpiresAt.Before(now) {
			subscription.Status = entities.SubscriptionExpired
			subscription.Reason = entities.CancellationTimeout
			r.byID[id] = subscription
			expired = append(expired, subscription)
		}
	}
	return expired, nil
}

type passthroughTransactions struct{}

func (passthroughTransactions) WithTx(ctx context.Context, work func(ctx context.Context) error) error {
	return work(ctx)
}

type recordingOutbox struct {
	added []outbox.Event
}

func (o *recordingOutbox) Add(_ context.Context, event outbox.Event) error {
	o.added = append(o.added, event)
	return nil
}

func (o *recordingOutbox) types() []string {
	names := []string{}
	for _, event := range o.added {
		names = append(names, event.Type)
	}
	return names
}

type testEnvironment struct {
	useCase *useCase
	repo    *inMemoryRepository
	outbox  *recordingOutbox
	clock   *time.Time
}

func newTestEnvironment() testEnvironment {
	repo := &inMemoryRepository{byID: map[uuid.UUID]entities.Subscription{}}
	recorder := &recordingOutbox{}
	clock := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	uc := newUseCase(repo, passthroughTransactions{}, recorder, Settings{PaymentTimeout: 30 * time.Second, AmountCents: 1990}).(*useCase)
	uc.now = func() time.Time { return clock }
	return testEnvironment{useCase: uc, repo: repo, outbox: recorder, clock: &clock}
}

func newCaller() *entities.Caller {
	return &entities.Caller{UserID: uuid.New(), Role: entities.RoleUser}
}

func (e testEnvironment) subscribe(t *testing.T, caller *entities.Caller) *entities.Subscription {
	t.Helper()
	subscription, err := e.useCase.Subscribe(context.Background(), caller)
	if err != nil {
		t.Fatal(err)
	}
	return subscription
}

func TestSubscribeCreatesAPendingSubscriptionAndAsksForThePayment(t *testing.T) {
	env := newTestEnvironment()
	caller := newCaller()

	subscription := env.subscribe(t, caller)

	if subscription.Status != entities.SubscriptionPending || subscription.AmountCents != 1990 {
		t.Errorf("unexpected subscription: %+v", subscription)
	}
	if !subscription.ExpiresAt.Equal(env.clock.Add(30 * time.Second)) {
		t.Errorf("must expire after the payment timeout, got %v", subscription.ExpiresAt)
	}
	if got := env.outbox.types(); len(got) != 1 || got[0] != entities.MessagePaymentRequested {
		t.Errorf("want one PaymentRequested, got %v", got)
	}
	payload := env.outbox.added[0].Payload.(entities.PaymentRequestedPayload)
	if payload.SubscriptionID != subscription.ID || payload.UserID != caller.UserID || payload.AmountCents != 1990 {
		t.Errorf("unexpected payload: %+v", payload)
	}
}

func TestSubscribeRefusesASecondSubscriptionInForce(t *testing.T) {
	env := newTestEnvironment()
	caller := newCaller()
	env.subscribe(t, caller)

	second, err := env.useCase.Subscribe(context.Background(), caller)

	if !errors.Is(err, entities.ErrSubscriptionInForce) || second != nil {
		t.Errorf("want ErrSubscriptionInForce, got %v / %v", err, second)
	}
	if len(env.outbox.added) != 1 {
		t.Errorf("the refused request must not publish anything, got %v", env.outbox.types())
	}
}

func TestApprovedPaymentActivatesThePendingSubscription(t *testing.T) {
	env := newTestEnvironment()
	caller := newCaller()
	subscription := env.subscribe(t, caller)
	chargeID := uuid.New()

	err := env.useCase.HandlePaymentApproved(context.Background(),
		entities.PaymentApprovedPayload{SubscriptionID: subscription.ID, UserID: caller.UserID, ChargeID: chargeID})

	stored, _ := env.repo.FindByID(context.Background(), subscription.ID)
	if err != nil || stored.Status != entities.SubscriptionActive || stored.ChargeID == nil || *stored.ChargeID != chargeID {
		t.Fatalf("subscription must be active with the charge, got %+v (err=%v)", stored, err)
	}
	if got := env.outbox.types(); len(got) != 2 || got[1] != entities.MessageSubscriptionActive {
		t.Errorf("want PaymentRequested then SubscriptionActivated, got %v", got)
	}
}

func TestApprovedPaymentDeliveredTwiceActivatesOnlyOnce(t *testing.T) {
	env := newTestEnvironment()
	caller := newCaller()
	subscription := env.subscribe(t, caller)
	approval := entities.PaymentApprovedPayload{SubscriptionID: subscription.ID, UserID: caller.UserID, ChargeID: uuid.New()}

	_ = env.useCase.HandlePaymentApproved(context.Background(), approval)
	err := env.useCase.HandlePaymentApproved(context.Background(), approval)

	if err != nil || len(env.outbox.added) != 2 {
		t.Errorf("the repeated approval must change nothing, got %v (err=%v)", env.outbox.types(), err)
	}
}

func TestLateApprovedPaymentOfAnExpiredSubscriptionAsksForARefund(t *testing.T) {
	env := newTestEnvironment()
	caller := newCaller()
	subscription := env.subscribe(t, caller)
	*env.clock = env.clock.Add(time.Minute)
	if _, err := env.useCase.ExpirePending(context.Background()); err != nil {
		t.Fatal(err)
	}
	chargeID := uuid.New()

	err := env.useCase.HandlePaymentApproved(context.Background(),
		entities.PaymentApprovedPayload{SubscriptionID: subscription.ID, UserID: caller.UserID, ChargeID: chargeID})

	stored, _ := env.repo.FindByID(context.Background(), subscription.ID)
	if err != nil || stored.Status != entities.SubscriptionExpired {
		t.Errorf("an expired subscription must stay expired, got %+v (err=%v)", stored, err)
	}
	got := env.outbox.types()
	if len(got) != 3 || got[2] != entities.MessageRefundRequested {
		t.Fatalf("want PaymentRequested, SubscriptionCanceled, RefundRequested, got %v", got)
	}
	refund := env.outbox.added[2].Payload.(entities.RefundRequestedPayload)
	if refund.ChargeID != chargeID || refund.SubscriptionID != subscription.ID {
		t.Errorf("the refund must name the charge, got %+v", refund)
	}
}

func TestRejectedPaymentCancelsThePendingSubscription(t *testing.T) {
	env := newTestEnvironment()
	caller := newCaller()
	subscription := env.subscribe(t, caller)

	err := env.useCase.HandlePaymentRejected(context.Background(),
		entities.PaymentRejectedPayload{SubscriptionID: subscription.ID, UserID: caller.UserID, Reason: "GATEWAY_TIMEOUT"})

	stored, _ := env.repo.FindByID(context.Background(), subscription.ID)
	if err != nil || stored.Status != entities.SubscriptionCanceled || stored.Reason != entities.CancellationPaymentRejected {
		t.Fatalf("subscription must be canceled by the rejection, got %+v (err=%v)", stored, err)
	}
	canceled := env.outbox.added[1].Payload.(entities.SubscriptionCanceledPayload)
	if canceled.Reason != entities.CancellationPaymentRejected {
		t.Errorf("unexpected cancellation: %+v", canceled)
	}
}

func TestRejectedPaymentOfASubscriptionThatIsNotPendingChangesNothing(t *testing.T) {
	env := newTestEnvironment()
	caller := newCaller()
	subscription := env.subscribe(t, caller)
	_ = env.useCase.HandlePaymentApproved(context.Background(),
		entities.PaymentApprovedPayload{SubscriptionID: subscription.ID, UserID: caller.UserID, ChargeID: uuid.New()})
	publishedBefore := len(env.outbox.added)

	err := env.useCase.HandlePaymentRejected(context.Background(),
		entities.PaymentRejectedPayload{SubscriptionID: subscription.ID, UserID: caller.UserID, Reason: "late"})

	stored, _ := env.repo.FindByID(context.Background(), subscription.ID)
	if err != nil || stored.Status != entities.SubscriptionActive || len(env.outbox.added) != publishedBefore {
		t.Errorf("an active subscription must not be canceled, got %+v / %v (err=%v)", stored, env.outbox.types(), err)
	}
}

func TestExpirePendingClosesOnlyTheOverdueSubscriptions(t *testing.T) {
	env := newTestEnvironment()
	overdue := env.subscribe(t, newCaller())
	*env.clock = env.clock.Add(25 * time.Second)
	fresh := env.subscribe(t, newCaller())
	*env.clock = env.clock.Add(10 * time.Second)

	expiredCount, err := env.useCase.ExpirePending(context.Background())

	overdueStored, _ := env.repo.FindByID(context.Background(), overdue.ID)
	freshStored, _ := env.repo.FindByID(context.Background(), fresh.ID)
	if err != nil || expiredCount != 1 {
		t.Fatalf("want exactly one expired subscription, got %d (err=%v)", expiredCount, err)
	}
	if overdueStored.Status != entities.SubscriptionExpired || freshStored.Status != entities.SubscriptionPending {
		t.Errorf("wrong subscriptions expired: %s / %s", overdueStored.Status, freshStored.Status)
	}
	last := env.outbox.added[len(env.outbox.added)-1].Payload.(entities.SubscriptionCanceledPayload)
	if last.Reason != entities.CancellationTimeout || last.SubscriptionID != overdue.ID {
		t.Errorf("unexpected cancellation: %+v", last)
	}
}

func TestGetByIDIsAllowedForTheOwnerAndTheAdminOnly(t *testing.T) {
	env := newTestEnvironment()
	owner := newCaller()
	subscription := env.subscribe(t, owner)
	admin := &entities.Caller{UserID: uuid.New(), Role: entities.RoleAdmin}

	asOwner, ownerErr := env.useCase.GetByID(context.Background(), owner, subscription.ID)
	asAdmin, adminErr := env.useCase.GetByID(context.Background(), admin, subscription.ID)
	asStranger, strangerErr := env.useCase.GetByID(context.Background(), newCaller(), subscription.ID)

	if ownerErr != nil || asOwner == nil || adminErr != nil || asAdmin == nil {
		t.Errorf("owner and admin must read it: %v / %v", ownerErr, adminErr)
	}
	if !errors.Is(strangerErr, entities.ErrForbidden) || asStranger != nil {
		t.Errorf("a stranger must be forbidden, got %v / %v", strangerErr, asStranger)
	}
}

func TestGetMineReturnsTheLatestSubscriptionAndNotFoundForNewUsers(t *testing.T) {
	env := newTestEnvironment()
	caller := newCaller()
	subscription := env.subscribe(t, caller)

	mine, err := env.useCase.GetMine(context.Background(), caller)
	none, noneErr := env.useCase.GetMine(context.Background(), newCaller())

	if err != nil || mine.ID != subscription.ID {
		t.Errorf("want the subscription, got %+v (err=%v)", mine, err)
	}
	if !errors.Is(noneErr, entities.ErrNotFound) || none != nil {
		t.Errorf("a user without subscriptions must get ErrNotFound, got %v / %v", noneErr, none)
	}
}

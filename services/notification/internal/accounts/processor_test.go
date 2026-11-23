package accounts

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"futhub/notification/internal/live"
	"futhub/notification/internal/messaging"
)

type memoryStore struct {
	premium       map[uuid.UUID]bool
	notifications []Notification
}

func newMemoryStore() *memoryStore {
	return &memoryStore{premium: map[uuid.UUID]bool{}}
}

func (s *memoryStore) SetPremium(_ context.Context, userID uuid.UUID, active bool) error {
	s.premium[userID] = active
	return nil
}

func (s *memoryStore) AddNotification(_ context.Context, notification Notification) error {
	s.notifications = append(s.notifications, notification)
	return nil
}

func (s *memoryStore) ListNotifications(_ context.Context, userID uuid.UUID, _ int) ([]Notification, error) {
	found := []Notification{}
	for _, notification := range s.notifications {
		if notification.UserID == userID {
			found = append(found, notification)
		}
	}
	return found, nil
}

func (s *memoryStore) MarkRead(_ context.Context, userID, notificationID uuid.UUID) (bool, error) {
	for index, notification := range s.notifications {
		if notification.ID == notificationID && notification.UserID == userID {
			s.notifications[index].Read = true
			return true, nil
		}
	}
	return false, nil
}

func envelopeWith(messageType string, payload map[string]any) live.Envelope {
	raw, _ := json.Marshal(payload)
	return live.Envelope{MessageID: uuid.NewString(), Type: messageType, Payload: raw}
}

func TestSubscriptionActivatedMakesTheUserPremiumAndNotifiesThem(t *testing.T) {
	store := newMemoryStore()
	userID := uuid.New()

	err := NewProcessor(store).Process(context.Background(),
		envelopeWith(MessageSubscriptionActivated, map[string]any{"subscriptionId": uuid.NewString(), "userId": userID.String()}))

	if err != nil || !store.premium[userID] {
		t.Fatalf("the user must be premium, got %v (err=%v)", store.premium, err)
	}
	if len(store.notifications) != 1 || store.notifications[0].UserID != userID || store.notifications[0].Type != "SUBSCRIPTION_ACTIVATED" {
		t.Errorf("unexpected notifications: %+v", store.notifications)
	}
}

func TestSubscriptionCanceledKeepsTheUserOutOfPremiumAndExplainsWhy(t *testing.T) {
	scenarios := map[string]string{
		"PAYMENT_REJECTED": "recusado",
		"TIMEOUT":          "demorou",
	}

	for reason, expectedWord := range scenarios {
		t.Run(reason, func(t *testing.T) {
			store := newMemoryStore()
			userID := uuid.New()
			store.premium[userID] = true

			err := NewProcessor(store).Process(context.Background(), envelopeWith(MessageSubscriptionCanceled,
				map[string]any{"subscriptionId": uuid.NewString(), "userId": userID.String(), "reason": reason}))

			if err != nil || store.premium[userID] {
				t.Fatalf("the user must not be premium, got %v (err=%v)", store.premium, err)
			}
			if len(store.notifications) != 1 || !contains(store.notifications[0].Message, expectedWord) {
				t.Errorf("the notification must explain the %s, got %+v", reason, store.notifications)
			}
		})
	}
}

func TestRefundCompletedNotifiesTheUserOnlyWhenTheMessageSaysWho(t *testing.T) {
	store := newMemoryStore()
	processor := NewProcessor(store)
	userID := uuid.New()

	withUser := processor.Process(context.Background(), envelopeWith(MessageRefundCompleted,
		map[string]any{"subscriptionId": uuid.NewString(), "chargeId": uuid.NewString(), "userId": userID.String()}))
	withoutUser := processor.Process(context.Background(), envelopeWith(MessageRefundCompleted,
		map[string]any{"subscriptionId": uuid.NewString(), "chargeId": uuid.NewString()}))

	if withUser != nil || withoutUser != nil || len(store.notifications) != 1 || store.notifications[0].UserID != userID {
		t.Errorf("only the message with a user must notify, got %+v (%v / %v)", store.notifications, withUser, withoutUser)
	}
}

func TestMessagesWithAnInvalidUserAreInvalid(t *testing.T) {
	err := NewProcessor(newMemoryStore()).Process(context.Background(),
		envelopeWith(MessageSubscriptionActivated, map[string]any{"subscriptionId": uuid.NewString(), "userId": "not-a-uuid"}))

	if !errors.Is(err, messaging.ErrInvalidMessage) {
		t.Errorf("want ErrInvalidMessage, got %v", err)
	}
}

func TestTheRouterSendsEachMessageTypeToItsProcessorAndRejectsTheRest(t *testing.T) {
	store := newMemoryStore()
	router := live.NewTypeRouter()
	router.Route(HandledMessageTypes, NewProcessor(store))
	userID := uuid.New()

	handled := router.Process(context.Background(),
		envelopeWith(MessageSubscriptionActivated, map[string]any{"subscriptionId": uuid.NewString(), "userId": userID.String()}))
	unknown := router.Process(context.Background(), envelopeWith("SomethingElse", map[string]any{}))

	if handled != nil || !store.premium[userID] {
		t.Errorf("the known type must be processed: %v", handled)
	}
	if !errors.Is(unknown, messaging.ErrInvalidMessage) {
		t.Errorf("an unknown type must be invalid, got %v", unknown)
	}
}

func contains(text, part string) bool {
	for index := 0; index+len(part) <= len(text); index++ {
		if text[index:index+len(part)] == part {
			return true
		}
	}
	return false
}

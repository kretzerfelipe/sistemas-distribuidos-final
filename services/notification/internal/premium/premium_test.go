package premium

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"futhub/notification/internal/security"
)

type fakeRepository struct{ active map[uuid.UUID]bool }

func (f fakeRepository) IsActive(_ context.Context, userID uuid.UUID) (bool, error) {
	return f.active[userID], nil
}

func TestIsPremium(t *testing.T) {
	subscriber, configured, stranger := uuid.New(), uuid.New(), uuid.New()
	repository := fakeRepository{active: map[uuid.UUID]bool{subscriber: true}}

	tests := []struct {
		name           string
		principal      security.Principal
		adminIsPremium bool
		want           bool
	}{
		{"active projection", security.Principal{UserID: subscriber.String(), Role: "USER"}, true, true},
		{"configured for development", security.Principal{UserID: configured.String(), Role: "USER"}, true, true},
		{"free user", security.Principal{UserID: stranger.String(), Role: "USER"}, true, false},
		{"admin when admins are premium", security.Principal{UserID: stranger.String(), Role: "ADMIN"}, true, true},
		{"admin when admins are not premium", security.Principal{UserID: stranger.String(), Role: "ADMIN"}, false, false},
		{"subject that is not a uuid", security.Principal{UserID: "abc", Role: "USER"}, true, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(repository, []string{configured.String()}, test.adminIsPremium)

			got, err := service.IsPremium(context.Background(), test.principal)

			if err != nil || got != test.want {
				t.Errorf("IsPremium = %v, %v; want %v", got, err, test.want)
			}
		})
	}
}

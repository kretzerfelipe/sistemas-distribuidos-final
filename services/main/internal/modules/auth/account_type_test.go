package auth

import (
	"context"
	"errors"
	"testing"

	"futhub/main/internal/entities"
)

func registerWithAccountType(env testEnvironment, accountType entities.Role) (*entities.User, error) {
	return env.useCase.Register(context.Background(), entities.RegisterInput{
		Name:        "Team Owner",
		Email:       "owner@example.com",
		Password:    validPassword,
		AccountType: accountType,
	})
}

func TestRegisterCreatesATeamUserWhenRequested(t *testing.T) {
	env := newTestEnvironment(t)

	user, err := registerWithAccountType(env, entities.RoleTeam)

	if err != nil || user.Role != entities.RoleTeam {
		t.Errorf("want a TEAM user, got %+v (err=%v)", user, err)
	}
}

func TestRegisterDefaultsToARegularUserWhenNoAccountTypeIsGiven(t *testing.T) {
	env := newTestEnvironment(t)

	user, err := registerWithAccountType(env, "")

	if err != nil || user.Role != entities.RoleUser {
		t.Errorf("want a USER, got %+v (err=%v)", user, err)
	}
}

func TestRegisterNeverCreatesAnAdminOrAnUnknownType(t *testing.T) {
	invalidTypes := map[string]entities.Role{
		"admin":   entities.RoleAdmin,
		"unknown": entities.Role("COACH"),
	}

	for name, accountType := range invalidTypes {
		t.Run(name, func(t *testing.T) {
			env := newTestEnvironment(t)

			user, err := registerWithAccountType(env, accountType)

			if !errors.Is(err, entities.ErrInvalidInput) {
				t.Errorf("want ErrInvalidInput, got %v", err)
			}
			if user != nil {
				t.Error("user must be nil when there is an error")
			}
			if len(env.users.usersByEmail) != 0 {
				t.Error("nothing must be stored")
			}
		})
	}
}

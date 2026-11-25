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

func registerByAdminWithAccountType(env testEnvironment, accountType entities.Role) (*entities.User, error) {
	return env.useCase.RegisterByAdmin(context.Background(), entities.RegisterInput{
		Name:        "Created By Admin",
		Email:       "created@example.com",
		Password:    validPassword,
		AccountType: accountType,
	})
}

func TestRegisterByAdminCreatesAnyKindOfUser(t *testing.T) {
	kinds := map[string]entities.Role{
		"admin": entities.RoleAdmin,
		"team":  entities.RoleTeam,
		"user":  entities.RoleUser,
		"empty": "",
	}

	for name, accountType := range kinds {
		t.Run(name, func(t *testing.T) {
			env := newTestEnvironment(t)

			user, err := registerByAdminWithAccountType(env, accountType)

			wantedRole := accountType
			if accountType == "" {
				wantedRole = entities.RoleUser
			}
			if err != nil || user.Role != wantedRole {
				t.Errorf("want role %s, got %+v (err=%v)", wantedRole, user, err)
			}
		})
	}
}

func TestRegisterByAdminRejectsAnUnknownTypeAndAShortPassword(t *testing.T) {
	env := newTestEnvironment(t)

	_, unknownTypeErr := registerByAdminWithAccountType(env, entities.Role("COACH"))
	_, shortPasswordErr := env.useCase.RegisterByAdmin(context.Background(), entities.RegisterInput{
		Name: "Created By Admin", Email: "created@example.com", Password: "short", AccountType: entities.RoleUser,
	})

	if !errors.Is(unknownTypeErr, entities.ErrInvalidInput) || !errors.Is(shortPasswordErr, entities.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput twice, got %v and %v", unknownTypeErr, shortPasswordErr)
	}
	if len(env.users.usersByEmail) != 0 {
		t.Error("nothing must be stored")
	}
}

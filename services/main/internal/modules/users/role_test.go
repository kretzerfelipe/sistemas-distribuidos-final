package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
)

func (r *inMemoryRepository) UpdateRole(_ context.Context, id uuid.UUID, role entities.Role, _ time.Time) error {
	for email, user := range r.usersByEmail {
		if user.ID == id {
			user.Role = role
			r.usersByEmail[email] = user
			return nil
		}
	}
	return entities.ErrNotFound
}

func createUserForRoleTests(t *testing.T, useCase UseCase) *entities.User {
	t.Helper()
	user, err := useCase.Create(context.Background(), validInput())
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func TestCreateAcceptsTheTeamRole(t *testing.T) {
	input := validInput()
	input.Role = entities.RoleTeam

	user, err := newUseCase(newInMemoryRepository()).Create(context.Background(), input)

	if err != nil || user.Role != entities.RoleTeam {
		t.Errorf("TEAM must be a valid role: user=%+v err=%v", user, err)
	}
}

func TestUpdateRoleChangesTheRoleOfAnotherUser(t *testing.T) {
	useCase := newUseCase(newInMemoryRepository())
	target := createUserForRoleTests(t, useCase)
	admin := &entities.Caller{UserID: uuid.New(), Role: entities.RoleAdmin}

	updated, err := useCase.UpdateRole(context.Background(), admin, target.ID, entities.RoleTeam)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Role != entities.RoleTeam {
		t.Errorf("want TEAM, got %s", updated.Role)
	}
}

func TestUpdateRoleRejectsAnUnknownRole(t *testing.T) {
	useCase := newUseCase(newInMemoryRepository())
	target := createUserForRoleTests(t, useCase)
	admin := &entities.Caller{UserID: uuid.New(), Role: entities.RoleAdmin}

	user, err := useCase.UpdateRole(context.Background(), admin, target.ID, entities.Role("ROOT"))

	if !errors.Is(err, entities.ErrInvalidInput) || user != nil {
		t.Errorf("want ErrInvalidInput and nil user, got %v / %v", err, user)
	}
}

func TestUpdateRoleDoesNotLetAnAdminChangeTheirOwnRole(t *testing.T) {
	useCase := newUseCase(newInMemoryRepository())
	target := createUserForRoleTests(t, useCase)
	callerIsTheTarget := &entities.Caller{UserID: target.ID, Role: entities.RoleAdmin}

	user, err := useCase.UpdateRole(context.Background(), callerIsTheTarget, target.ID, entities.RoleUser)

	if !errors.Is(err, entities.ErrForbidden) || user != nil {
		t.Errorf("want ErrForbidden and nil user, got %v / %v", err, user)
	}
}

func TestUpdateRoleReturnsNotFoundForAnUnknownUser(t *testing.T) {
	useCase := newUseCase(newInMemoryRepository())
	admin := &entities.Caller{UserID: uuid.New(), Role: entities.RoleAdmin}

	user, err := useCase.UpdateRole(context.Background(), admin, uuid.New(), entities.RoleTeam)

	if !errors.Is(err, entities.ErrNotFound) || user != nil {
		t.Errorf("want ErrNotFound and nil user, got %v / %v", err, user)
	}
}

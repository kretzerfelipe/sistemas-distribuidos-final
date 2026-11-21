package players

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
)

type inMemoryRepository struct {
	playersByID map[uuid.UUID]entities.Player
}

func newInMemoryRepository() *inMemoryRepository {
	return &inMemoryRepository{playersByID: map[uuid.UUID]entities.Player{}}
}

func (r *inMemoryRepository) Insert(_ context.Context, player *entities.Player) error {
	for _, existing := range r.playersByID {
		userAlreadyHasAPlayer := existing.UserID == player.UserID
		if userAlreadyHasAPlayer {
			return entities.ErrPlayerAlreadyExists
		}
	}
	r.playersByID[player.ID] = *player
	return nil
}

func (r *inMemoryRepository) FindByID(_ context.Context, id uuid.UUID) (*entities.Player, error) {
	player, found := r.playersByID[id]
	if !found {
		return nil, entities.ErrNotFound
	}
	return &player, nil
}

func (r *inMemoryRepository) FindByUserID(_ context.Context, userID uuid.UUID) (*entities.Player, error) {
	for _, player := range r.playersByID {
		if player.UserID == userID {
			return &player, nil
		}
	}
	return nil, entities.ErrNotFound
}

func (r *inMemoryRepository) FindByIDs(_ context.Context, ids []uuid.UUID) ([]entities.Player, error) {
	found := []entities.Player{}
	for _, id := range ids {
		if player, exists := r.playersByID[id]; exists {
			found = append(found, player)
		}
	}
	return found, nil
}

func (r *inMemoryRepository) List(context.Context) ([]entities.Player, error) {
	all := []entities.Player{}
	for _, player := range r.playersByID {
		all = append(all, player)
	}
	return all, nil
}

func (r *inMemoryRepository) Update(_ context.Context, player *entities.Player) error {
	_, exists := r.playersByID[player.ID]
	if !exists {
		return entities.ErrNotFound
	}
	r.playersByID[player.ID] = *player
	return nil
}

func (r *inMemoryRepository) Delete(_ context.Context, id uuid.UUID) error {
	_, exists := r.playersByID[id]
	if !exists {
		return entities.ErrNotFound
	}
	delete(r.playersByID, id)
	return nil
}

func newCaller() *entities.Caller {
	return &entities.Caller{UserID: uuid.New(), Role: entities.RoleUser}
}

func validInput() entities.SavePlayerInput {
	return entities.SavePlayerInput{Name: "Pele", Position: entities.PositionForward, ShirtNumber: 10}
}

func TestCreateStoresThePlayerForTheCaller(t *testing.T) {
	useCase := newUseCase(newInMemoryRepository())
	caller := newCaller()

	player, err := useCase.Create(context.Background(), caller, validInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if player.UserID != caller.UserID || player.Name != "Pele" || player.ShirtNumber != 10 {
		t.Errorf("unexpected player: %+v", player)
	}
}

func TestCreateTrimsTheName(t *testing.T) {
	input := validInput()
	input.Name = "  Pele  "

	player, err := newUseCase(newInMemoryRepository()).Create(context.Background(), newCaller(), input)

	if err != nil || player.Name != "Pele" {
		t.Errorf("name should be trimmed: %+v err=%v", player, err)
	}
}

func TestCreateAllowsOnlyOnePlayerPerUser(t *testing.T) {
	useCase := newUseCase(newInMemoryRepository())
	caller := newCaller()
	if _, err := useCase.Create(context.Background(), caller, validInput()); err != nil {
		t.Fatal(err)
	}

	second, err := useCase.Create(context.Background(), caller, validInput())

	if !errors.Is(err, entities.ErrPlayerAlreadyExists) || second != nil {
		t.Errorf("want ErrPlayerAlreadyExists and nil player, got %v / %v", err, second)
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	scenarios := map[string]entities.SavePlayerInput{
		"blank name":            {Name: "  ", Position: entities.PositionForward, ShirtNumber: 10},
		"name too long":         {Name: strings.Repeat("a", maxNameLength+1), Position: entities.PositionForward, ShirtNumber: 10},
		"unknown position":      {Name: "Pele", Position: entities.Position("COACH"), ShirtNumber: 10},
		"missing position":      {Name: "Pele", ShirtNumber: 10},
		"shirt number zero":     {Name: "Pele", Position: entities.PositionForward, ShirtNumber: 0},
		"shirt number above":    {Name: "Pele", Position: entities.PositionForward, ShirtNumber: 100},
		"shirt number negative": {Name: "Pele", Position: entities.PositionForward, ShirtNumber: -3},
	}

	for name, input := range scenarios {
		t.Run(name, func(t *testing.T) {
			player, err := newUseCase(newInMemoryRepository()).Create(context.Background(), newCaller(), input)

			if !errors.Is(err, entities.ErrInvalidInput) || player != nil {
				t.Errorf("want ErrInvalidInput and nil player, got %v / %v", err, player)
			}
		})
	}
}

func TestGetMineReturnsNotFoundWhenTheUserHasNoPlayer(t *testing.T) {
	player, err := newUseCase(newInMemoryRepository()).GetMine(context.Background(), newCaller())

	if !errors.Is(err, entities.ErrNotFound) || player != nil {
		t.Errorf("want ErrNotFound and nil player, got %v / %v", err, player)
	}
}

func TestUpdateMineChangesTheFields(t *testing.T) {
	useCase := newUseCase(newInMemoryRepository())
	caller := newCaller()
	if _, err := useCase.Create(context.Background(), caller, validInput()); err != nil {
		t.Fatal(err)
	}

	updated, err := useCase.UpdateMine(context.Background(), caller,
		entities.SavePlayerInput{Name: "Edson", Position: entities.PositionMidfielder, ShirtNumber: 8})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Edson" || updated.Position != entities.PositionMidfielder || updated.ShirtNumber != 8 {
		t.Errorf("unexpected player: %+v", updated)
	}
}

func TestUpdateMineKeepsTheOldDataWhenTheInputIsInvalid(t *testing.T) {
	useCase := newUseCase(newInMemoryRepository())
	caller := newCaller()
	if _, err := useCase.Create(context.Background(), caller, validInput()); err != nil {
		t.Fatal(err)
	}

	updated, err := useCase.UpdateMine(context.Background(), caller,
		entities.SavePlayerInput{Name: "", Position: entities.PositionForward, ShirtNumber: 10})
	stored, _ := useCase.GetMine(context.Background(), caller)

	if !errors.Is(err, entities.ErrInvalidInput) || updated != nil {
		t.Errorf("want ErrInvalidInput and nil player, got %v / %v", err, updated)
	}
	if stored.Name != "Pele" {
		t.Errorf("stored player must be untouched, got %q", stored.Name)
	}
}

func TestUpdateMineReturnsNotFoundWithoutAPlayer(t *testing.T) {
	updated, err := newUseCase(newInMemoryRepository()).UpdateMine(context.Background(), newCaller(), validInput())

	if !errors.Is(err, entities.ErrNotFound) || updated != nil {
		t.Errorf("want ErrNotFound and nil player, got %v / %v", err, updated)
	}
}

func TestDeleteMineRemovesThePlayer(t *testing.T) {
	useCase := newUseCase(newInMemoryRepository())
	caller := newCaller()
	if _, err := useCase.Create(context.Background(), caller, validInput()); err != nil {
		t.Fatal(err)
	}

	err := useCase.DeleteMine(context.Background(), caller)
	_, lookupErr := useCase.GetMine(context.Background(), caller)
	secondDeleteErr := useCase.DeleteMine(context.Background(), caller)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !errors.Is(lookupErr, entities.ErrNotFound) || !errors.Is(secondDeleteErr, entities.ErrNotFound) {
		t.Errorf("the player must be gone: %v / %v", lookupErr, secondDeleteErr)
	}
}

func TestDeleteByIDReturnsNotFoundForAnUnknownPlayer(t *testing.T) {
	err := newUseCase(newInMemoryRepository()).Delete(context.Background(), uuid.New())

	if !errors.Is(err, entities.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestGetByIDsReturnsOnlyTheRequestedPlayers(t *testing.T) {
	useCase := newUseCase(newInMemoryRepository())
	first, _ := useCase.Create(context.Background(), newCaller(), validInput())
	useCase.Create(context.Background(), newCaller(), validInput())

	found, err := useCase.GetByIDs(context.Background(), []uuid.UUID{first.ID})

	if err != nil || len(found) != 1 || found[0].ID != first.ID {
		t.Errorf("want only the first player, got %+v (err=%v)", found, err)
	}
}

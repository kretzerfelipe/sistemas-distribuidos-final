package players

import (
	"context"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/util"
)

type useCase struct {
	repo Repository
}

func newUseCase(repo Repository) UseCase {
	return &useCase{repo: repo}
}

func (uc *useCase) Create(ctx context.Context, caller *entities.Caller, in entities.SavePlayerInput) (*entities.Player, error) {
	validatedInput, err := validate(in)
	if err != nil {
		return nil, util.SendMessageWithError("use case error validating new player", err)
	}

	now := time.Now().UTC()
	player := &entities.Player{
		ID:          uuid.New(),
		UserID:      caller.UserID,
		Name:        validatedInput.Name,
		Position:    validatedInput.Position,
		ShirtNumber: validatedInput.ShirtNumber,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err = uc.repo.Insert(ctx, player)
	if err != nil {
		return nil, util.SendMessageWithError("use case error inserting new player", err)
	}
	return player, nil
}

func (uc *useCase) GetMine(ctx context.Context, caller *entities.Caller) (*entities.Player, error) {
	player, err := uc.repo.FindByUserID(ctx, caller.UserID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding the caller's player", err)
	}
	return player, nil
}

func (uc *useCase) UpdateMine(ctx context.Context, caller *entities.Caller, in entities.SavePlayerInput) (*entities.Player, error) {
	validatedInput, err := validate(in)
	if err != nil {
		return nil, util.SendMessageWithError("use case error validating player update", err)
	}

	player, err := uc.repo.FindByUserID(ctx, caller.UserID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding the caller's player to update", err)
	}

	player.Name = validatedInput.Name
	player.Position = validatedInput.Position
	player.ShirtNumber = validatedInput.ShirtNumber
	player.UpdatedAt = time.Now().UTC()

	err = uc.repo.Update(ctx, player)
	if err != nil {
		return nil, util.SendMessageWithError("use case error updating player", err)
	}
	return player, nil
}

func (uc *useCase) DeleteMine(ctx context.Context, caller *entities.Caller) error {
	player, err := uc.repo.FindByUserID(ctx, caller.UserID)
	if err != nil {
		return util.SendMessageWithError("use case error finding the caller's player to delete", err)
	}

	err = uc.repo.Delete(ctx, player.ID)
	if err != nil {
		return util.SendMessageWithError("use case error deleting the caller's player", err)
	}
	return nil
}

func (uc *useCase) GetByID(ctx context.Context, id uuid.UUID) (*entities.Player, error) {
	player, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding player by id", err)
	}
	return player, nil
}

func (uc *useCase) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.Player, error) {
	player, err := uc.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding player by user id", err)
	}
	return player, nil
}

func (uc *useCase) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]entities.Player, error) {
	players, err := uc.repo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding players by ids", err)
	}
	return players, nil
}

func (uc *useCase) List(ctx context.Context) ([]entities.Player, error) {
	allPlayers, err := uc.repo.List(ctx)
	if err != nil {
		return nil, util.SendMessageWithError("use case error listing players", err)
	}
	return allPlayers, nil
}

func (uc *useCase) Delete(ctx context.Context, id uuid.UUID) error {
	err := uc.repo.Delete(ctx, id)
	if err != nil {
		return util.SendMessageWithError("use case error deleting player", err)
	}
	return nil
}

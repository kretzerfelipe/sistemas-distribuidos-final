package users

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

func (uc *useCase) Create(ctx context.Context, in entities.CreateInput) (*entities.User, error) {
	validatedInput, err := validate(in)
	if err != nil {
		return nil, util.SendMessageWithError("use case error validating new user", err)
	}

	now := time.Now().UTC()
	user := &entities.User{
		ID:           uuid.New(),
		Name:         validatedInput.Name,
		Email:        validatedInput.Email,
		PasswordHash: validatedInput.PasswordHash,
		Role:         validatedInput.Role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = uc.repo.Insert(ctx, user)
	if err != nil {
		return nil, util.SendMessageWithError("use case error inserting new user", err)
	}
	return user, nil
}

func (uc *useCase) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	user, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding user by id", err)
	}
	return user, nil
}

func (uc *useCase) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	user, err := uc.repo.FindByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding user by email", err)
	}
	return user, nil
}

func (uc *useCase) EmailIsRegistered(ctx context.Context, email string) (bool, error) {
	emailIsRegistered, err := uc.repo.ExistsByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		return false, util.SendMessageWithError("use case error checking whether email is registered", err)
	}
	return emailIsRegistered, nil
}

func (uc *useCase) List(ctx context.Context) ([]entities.User, error) {
	allUsers, err := uc.repo.List(ctx)
	if err != nil {
		return nil, util.SendMessageWithError("use case error listing users", err)
	}
	return allUsers, nil
}

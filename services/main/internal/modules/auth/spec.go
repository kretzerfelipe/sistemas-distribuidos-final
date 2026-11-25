package auth

import (
	"context"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
)

type UserService interface {
	Create(ctx context.Context, in entities.CreateInput) (*entities.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	EmailIsRegistered(ctx context.Context, email string) (bool, error)
}

type UseCase interface {
	Register(ctx context.Context, in entities.RegisterInput) (*entities.User, error)
	RegisterByAdmin(ctx context.Context, in entities.RegisterInput) (*entities.User, error)
	Login(ctx context.Context, in entities.LoginInput) (*entities.LoginResult, error)
	Me(ctx context.Context, userID string) (*entities.User, error)
	SeedAdmin(ctx context.Context, name, email, password string) (created bool, err error)
}

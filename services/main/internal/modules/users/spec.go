package users

import (
	"context"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
)

func NewUserResponse(user *entities.User) entities.UserResponse {
	return entities.UserResponse{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, CreatedAt: user.CreatedAt}
}

type UseCase interface {
	Create(ctx context.Context, in entities.CreateInput) (*entities.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	EmailIsRegistered(ctx context.Context, email string) (bool, error)
	List(ctx context.Context) ([]entities.User, error)
}

type Repository interface {
	Insert(ctx context.Context, user *entities.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	FindByEmail(ctx context.Context, email string) (*entities.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	List(ctx context.Context) ([]entities.User, error)
}

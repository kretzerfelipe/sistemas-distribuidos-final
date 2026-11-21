package players

import (
	"context"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
)

func NewPlayerResponse(player *entities.Player) entities.PlayerResponse {
	return entities.PlayerResponse{
		ID:          player.ID,
		Name:        player.Name,
		Position:    player.Position,
		ShirtNumber: player.ShirtNumber,
		CreatedAt:   player.CreatedAt,
	}
}

type UseCase interface {
	Create(ctx context.Context, caller *entities.Caller, in entities.SavePlayerInput) (*entities.Player, error)
	GetMine(ctx context.Context, caller *entities.Caller) (*entities.Player, error)
	UpdateMine(ctx context.Context, caller *entities.Caller, in entities.SavePlayerInput) (*entities.Player, error)
	DeleteMine(ctx context.Context, caller *entities.Caller) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Player, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.Player, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]entities.Player, error)
	List(ctx context.Context) ([]entities.Player, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Repository interface {
	Insert(ctx context.Context, player *entities.Player) error
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Player, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) (*entities.Player, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]entities.Player, error)
	List(ctx context.Context) ([]entities.Player, error)
	Update(ctx context.Context, player *entities.Player) error
	Delete(ctx context.Context, id uuid.UUID) error
}

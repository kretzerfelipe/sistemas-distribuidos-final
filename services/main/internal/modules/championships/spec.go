package championships

import (
	"context"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
)

func NewChampionshipResponse(championship *entities.Championship) entities.ChampionshipResponse {
	return entities.ChampionshipResponse{
		ID:         championship.ID,
		Name:       championship.Name,
		SlotsTotal: championship.SlotsTotal,
		SlotsTaken: championship.SlotsTaken,
		Status:     championship.Status,
		CreatedAt:  championship.CreatedAt,
	}
}

type TeamService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Team, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]entities.Team, error)
}

type UseCase interface {
	Create(ctx context.Context, in entities.SaveChampionshipInput) (*entities.Championship, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Championship, error)
	List(ctx context.Context) ([]entities.Championship, error)
	Update(ctx context.Context, id uuid.UUID, in entities.SaveChampionshipInput) (*entities.Championship, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Start(ctx context.Context, id uuid.UUID) (*entities.Championship, error)
	Finish(ctx context.Context, id uuid.UUID) (*entities.Championship, error)

	Enroll(ctx context.Context, caller *entities.Caller, championshipID, teamID uuid.UUID) (*entities.Championship, error)
	Unenroll(ctx context.Context, caller *entities.Caller, championshipID, teamID uuid.UUID) error
	ListTeams(ctx context.Context, championshipID uuid.UUID) ([]entities.Team, error)
	TeamIsEnrolled(ctx context.Context, championshipID, teamID uuid.UUID) (bool, error)
}

type Repository interface {
	Insert(ctx context.Context, championship *entities.Championship) error
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Championship, error)
	List(ctx context.Context) ([]entities.Championship, error)
	Update(ctx context.Context, championship *entities.Championship) error
	Delete(ctx context.Context, id uuid.UUID) error
	Transition(ctx context.Context, id uuid.UUID, from, to entities.ChampionshipStatus, updatedAt time.Time) error

	Enroll(ctx context.Context, championshipID, teamID uuid.UUID, enrolledAt time.Time) error
	Unenroll(ctx context.Context, championshipID, teamID uuid.UUID) error
	ListEnrolledTeamIDs(ctx context.Context, championshipID uuid.UUID) ([]uuid.UUID, error)
	IsEnrolled(ctx context.Context, championshipID, teamID uuid.UUID) (bool, error)
}

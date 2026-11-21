package matches

import (
	"context"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
)

func NewMatchResponse(match *entities.Match) entities.MatchResponse {
	return entities.MatchResponse{
		ID:             match.ID,
		ChampionshipID: match.ChampionshipID,
		HomeTeamID:     match.HomeTeamID,
		AwayTeamID:     match.AwayTeamID,
		ScheduledAt:    match.ScheduledAt,
		Status:         match.Status,
		HomeScore:      match.HomeScore,
		AwayScore:      match.AwayScore,
		CreatedAt:      match.CreatedAt,
	}
}

type ChampionshipService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Championship, error)
	TeamIsEnrolled(ctx context.Context, championshipID, teamID uuid.UUID) (bool, error)
}

type UseCase interface {
	Create(ctx context.Context, in entities.CreateMatchInput) (*entities.Match, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Match, error)
	ListByChampionship(ctx context.Context, championshipID uuid.UUID) ([]entities.Match, error)
	Reschedule(ctx context.Context, id uuid.UUID, in entities.RescheduleMatchInput) (*entities.Match, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Start(ctx context.Context, id uuid.UUID) (*entities.Match, error)
	Finish(ctx context.Context, id uuid.UUID, in entities.FinishMatchInput) (*entities.Match, error)
}

type Repository interface {
	Insert(ctx context.Context, match *entities.Match) error
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Match, error)
	ListByChampionship(ctx context.Context, championshipID uuid.UUID) ([]entities.Match, error)
	Reschedule(ctx context.Context, id uuid.UUID, scheduledAt, updatedAt time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
	Transition(ctx context.Context, id uuid.UUID, from, to entities.MatchStatus, homeScore, awayScore int, updatedAt time.Time) error
}

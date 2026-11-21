package matches

import (
	"context"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/outbox"
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

func NewMatchEventResponse(event *entities.MatchEvent) entities.MatchEventResponse {
	return entities.MatchEventResponse{
		ID:              event.ID,
		Sequence:        event.Sequence,
		Type:            event.Type,
		Minute:          event.Minute,
		TeamID:          event.TeamID,
		Player:          playerReference(event.PlayerID, event.PlayerName),
		SecondaryPlayer: playerReference(event.SecondaryPlayerID, event.SecondaryPlayerName),
		RegisteredAt:    event.CreatedAt,
	}
}

func playerReference(id *uuid.UUID, name string) *entities.NamedReference {
	playerIsNotInformed := id == nil
	if playerIsNotInformed {
		return nil
	}
	return &entities.NamedReference{ID: *id, Name: name}
}

type ChampionshipService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Championship, error)
	TeamIsEnrolled(ctx context.Context, championshipID, teamID uuid.UUID) (bool, error)
}

type TeamService interface {
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]entities.Team, error)
	PlayerBelongsToTeam(ctx context.Context, teamID, playerID uuid.UUID) (bool, error)
}

type PlayerService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Player, error)
}

type Transactions interface {
	WithTx(ctx context.Context, work func(ctx context.Context) error) error
}

type UseCase interface {
	Create(ctx context.Context, in entities.CreateMatchInput) (*entities.Match, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Match, error)
	ListByChampionship(ctx context.Context, championshipID uuid.UUID) ([]entities.Match, error)
	ListWindow(ctx context.Context, in entities.ListMatchesInput) (*entities.MatchWindow, error)
	ListLive(ctx context.Context) ([]entities.Match, error)
	Reschedule(ctx context.Context, id uuid.UUID, in entities.RescheduleMatchInput) (*entities.Match, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Start(ctx context.Context, id uuid.UUID) (*entities.Match, error)
	Finish(ctx context.Context, id uuid.UUID) (*entities.Match, error)

	RegisterEvent(ctx context.Context, matchID uuid.UUID, in entities.RegisterMatchEventInput) (*entities.MatchEvent, error)
	RemoveEvent(ctx context.Context, matchID, eventID uuid.UUID) error
	ListEvents(ctx context.Context, matchID uuid.UUID) ([]entities.MatchEvent, error)
}

type Repository interface {
	Insert(ctx context.Context, match *entities.Match) error
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Match, error)
	FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*entities.Match, error)
	ListByChampionship(ctx context.Context, championshipID uuid.UUID) ([]entities.Match, error)
	ListScheduledBetween(ctx context.Context, from, to time.Time) ([]entities.Match, error)
	ExistsScheduledFrom(ctx context.Context, from time.Time) (bool, error)
	ListByStatus(ctx context.Context, status entities.MatchStatus) ([]entities.Match, error)
	Reschedule(ctx context.Context, id uuid.UUID, scheduledAt, updatedAt time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
	Transition(ctx context.Context, id uuid.UUID, from, to entities.MatchStatus, updatedAt time.Time) error
	AdvanceSequence(ctx context.Context, id uuid.UUID, homeScoreDelta, awayScoreDelta int, updatedAt time.Time) (*entities.MatchProgress, error)

	InsertEvent(ctx context.Context, event *entities.MatchEvent) error
	FindEvent(ctx context.Context, matchID, eventID uuid.UUID) (*entities.MatchEvent, error)
	ListEvents(ctx context.Context, matchID uuid.UUID) ([]entities.MatchEvent, error)
	DeleteEvent(ctx context.Context, matchID, eventID uuid.UUID) error
	PlayerHasRedCard(ctx context.Context, matchID, playerID uuid.UUID) (bool, error)
}

type Outbox = outbox.Writer

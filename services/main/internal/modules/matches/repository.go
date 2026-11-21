package matches

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/db"
	"futhub/main/internal/platform/util"
)

const (
	matchColumns = "id, championship_id, home_team_id, away_team_id, scheduled_at, status, home_score, away_score, sequence, created_at, updated_at"
	eventColumns = "id, match_id, sequence, type, minute, team_id, player_id, player_name, secondary_player_id, secondary_player_name, created_at"
)

type postgresRepository struct {
	pool *pgxpool.Pool
}

func newPostgresRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) executor(ctx context.Context) db.Executor {
	return db.ExecutorFrom(ctx, r.pool)
}

func (r *postgresRepository) Insert(ctx context.Context, match *entities.Match) error {
	_, err := r.executor(ctx).Exec(ctx,
		`INSERT INTO matches (id, championship_id, home_team_id, away_team_id, scheduled_at, status, home_score, away_score, sequence, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		match.ID, match.ChampionshipID, match.HomeTeamID, match.AwayTeamID, match.ScheduledAt,
		string(match.Status), match.HomeScore, match.AwayScore, match.Sequence, match.CreatedAt, match.UpdatedAt)
	if err != nil {
		if db.IsForeignKeyViolation(err) {
			return fmt.Errorf("%w: championship or team", entities.ErrNotFound)
		}
		return util.SendMessageWithError("repository error inserting match", err)
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Match, error) {
	row := r.executor(ctx).QueryRow(ctx, "SELECT "+matchColumns+" FROM matches WHERE id = $1", id)
	return scanMatch(row)
}

func (r *postgresRepository) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*entities.Match, error) {
	row := r.executor(ctx).QueryRow(ctx, "SELECT "+matchColumns+" FROM matches WHERE id = $1 FOR UPDATE", id)
	return scanMatch(row)
}

func (r *postgresRepository) ListByChampionship(ctx context.Context, championshipID uuid.UUID) ([]entities.Match, error) {
	rows, err := r.executor(ctx).Query(ctx,
		"SELECT "+matchColumns+" FROM matches WHERE championship_id = $1 ORDER BY scheduled_at, id", championshipID)
	if err != nil {
		return nil, util.SendMessageWithError("repository error listing matches", err)
	}
	defer rows.Close()

	championshipMatches := []entities.Match{}
	for rows.Next() {
		match, err := scanMatch(rows)
		if err != nil {
			return nil, err
		}
		championshipMatches = append(championshipMatches, *match)
	}

	err = rows.Err()
	if err != nil {
		return nil, util.SendMessageWithError("repository error reading matches", err)
	}
	return championshipMatches, nil
}

func (r *postgresRepository) Reschedule(ctx context.Context, id uuid.UUID, scheduledAt, updatedAt time.Time) error {
	result, err := r.executor(ctx).Exec(ctx,
		"UPDATE matches SET scheduled_at = $2, updated_at = $3 WHERE id = $1 AND status = 'SCHEDULED'",
		id, scheduledAt, updatedAt)
	if err != nil {
		return util.SendMessageWithError("repository error rescheduling match", err)
	}

	noRowWasUpdated := result.RowsAffected() == 0
	if noRowWasUpdated {
		return fmt.Errorf("%w: only scheduled matches can be rescheduled", entities.ErrInvalidState)
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.executor(ctx).Exec(ctx, "DELETE FROM matches WHERE id = $1 AND status = 'SCHEDULED'", id)
	if err != nil {
		return util.SendMessageWithError("repository error deleting match", err)
	}

	noRowWasDeleted := result.RowsAffected() == 0
	if noRowWasDeleted {
		return fmt.Errorf("%w: only scheduled matches can be deleted", entities.ErrInvalidState)
	}
	return nil
}

func (r *postgresRepository) Transition(ctx context.Context, id uuid.UUID, from, to entities.MatchStatus, updatedAt time.Time) error {
	result, err := r.executor(ctx).Exec(ctx,
		"UPDATE matches SET status = $3, updated_at = $4 WHERE id = $1 AND status = $2",
		id, string(from), string(to), updatedAt)
	if err != nil {
		return util.SendMessageWithError("repository error changing match status", err)
	}

	noRowWasUpdated := result.RowsAffected() == 0
	if noRowWasUpdated {
		return fmt.Errorf("%w: match must be %s", entities.ErrInvalidState, from)
	}
	return nil
}

func (r *postgresRepository) AdvanceSequence(ctx context.Context, id uuid.UUID, homeScoreDelta, awayScoreDelta int, updatedAt time.Time) (*entities.MatchProgress, error) {
	var progress entities.MatchProgress

	err := r.executor(ctx).QueryRow(ctx,
		`UPDATE matches
		 SET sequence = sequence + 1, home_score = home_score + $2, away_score = away_score + $3, updated_at = $4
		 WHERE id = $1
		 RETURNING sequence, home_score, away_score`,
		id, homeScoreDelta, awayScoreDelta, updatedAt).Scan(&progress.Sequence, &progress.HomeScore, &progress.AwayScore)
	if err != nil {
		noRowWasFound := errors.Is(err, pgx.ErrNoRows)
		if noRowWasFound {
			return nil, fmt.Errorf("%w: match", entities.ErrNotFound)
		}
		return nil, util.SendMessageWithError("repository error advancing the match sequence", err)
	}
	return &progress, nil
}

func (r *postgresRepository) InsertEvent(ctx context.Context, event *entities.MatchEvent) error {
	_, err := r.executor(ctx).Exec(ctx,
		`INSERT INTO match_events (id, match_id, sequence, type, minute, team_id, player_id, player_name, secondary_player_id, secondary_player_name, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		event.ID, event.MatchID, event.Sequence, string(event.Type), event.Minute, event.TeamID,
		event.PlayerID, event.PlayerName, event.SecondaryPlayerID, event.SecondaryPlayerName, event.CreatedAt)
	if err != nil {
		if db.IsForeignKeyViolation(err) {
			return fmt.Errorf("%w: match, team or player", entities.ErrNotFound)
		}
		return util.SendMessageWithError("repository error inserting match event", err)
	}
	return nil
}

func (r *postgresRepository) FindEvent(ctx context.Context, matchID, eventID uuid.UUID) (*entities.MatchEvent, error) {
	row := r.executor(ctx).QueryRow(ctx,
		"SELECT "+eventColumns+" FROM match_events WHERE match_id = $1 AND id = $2", matchID, eventID)
	return scanEvent(row)
}

func (r *postgresRepository) ListEvents(ctx context.Context, matchID uuid.UUID) ([]entities.MatchEvent, error) {
	rows, err := r.executor(ctx).Query(ctx,
		"SELECT "+eventColumns+" FROM match_events WHERE match_id = $1 ORDER BY sequence", matchID)
	if err != nil {
		return nil, util.SendMessageWithError("repository error listing match events", err)
	}
	defer rows.Close()

	matchEvents := []entities.MatchEvent{}
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		matchEvents = append(matchEvents, *event)
	}

	err = rows.Err()
	if err != nil {
		return nil, util.SendMessageWithError("repository error reading match events", err)
	}
	return matchEvents, nil
}

func (r *postgresRepository) DeleteEvent(ctx context.Context, matchID, eventID uuid.UUID) error {
	result, err := r.executor(ctx).Exec(ctx, "DELETE FROM match_events WHERE match_id = $1 AND id = $2", matchID, eventID)
	if err != nil {
		return util.SendMessageWithError("repository error deleting match event", err)
	}

	noRowWasDeleted := result.RowsAffected() == 0
	if noRowWasDeleted {
		return fmt.Errorf("%w: match event", entities.ErrNotFound)
	}
	return nil
}

func (r *postgresRepository) PlayerHasRedCard(ctx context.Context, matchID, playerID uuid.UUID) (bool, error) {
	var hasRedCard bool
	err := r.executor(ctx).QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM match_events WHERE match_id = $1 AND player_id = $2 AND type = 'RED_CARD')",
		matchID, playerID).Scan(&hasRedCard)
	if err != nil {
		return false, util.SendMessageWithError("repository error checking red cards", err)
	}
	return hasRedCard, nil
}

func scanMatch(row pgx.Row) (*entities.Match, error) {
	var match entities.Match
	var status string

	err := row.Scan(&match.ID, &match.ChampionshipID, &match.HomeTeamID, &match.AwayTeamID, &match.ScheduledAt,
		&status, &match.HomeScore, &match.AwayScore, &match.Sequence, &match.CreatedAt, &match.UpdatedAt)
	if err != nil {
		noRowWasFound := errors.Is(err, pgx.ErrNoRows)
		if noRowWasFound {
			return nil, fmt.Errorf("%w: match", entities.ErrNotFound)
		}
		return nil, util.SendMessageWithError("repository error scanning match", err)
	}

	match.Status = entities.MatchStatus(status)
	match.ScheduledAt = match.ScheduledAt.UTC()
	match.CreatedAt = match.CreatedAt.UTC()
	match.UpdatedAt = match.UpdatedAt.UTC()
	return &match, nil
}

func scanEvent(row pgx.Row) (*entities.MatchEvent, error) {
	var event entities.MatchEvent
	var eventType string

	err := row.Scan(&event.ID, &event.MatchID, &event.Sequence, &eventType, &event.Minute, &event.TeamID,
		&event.PlayerID, &event.PlayerName, &event.SecondaryPlayerID, &event.SecondaryPlayerName, &event.CreatedAt)
	if err != nil {
		noRowWasFound := errors.Is(err, pgx.ErrNoRows)
		if noRowWasFound {
			return nil, fmt.Errorf("%w: match event", entities.ErrNotFound)
		}
		return nil, util.SendMessageWithError("repository error scanning match event", err)
	}

	event.Type = entities.MatchEventType(eventType)
	event.CreatedAt = event.CreatedAt.UTC()
	return &event, nil
}

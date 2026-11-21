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

const selectColumns = "id, championship_id, home_team_id, away_team_id, scheduled_at, status, home_score, away_score, created_at, updated_at"

type postgresRepository struct {
	pool *pgxpool.Pool
}

func newPostgresRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) Insert(ctx context.Context, match *entities.Match) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO matches (id, championship_id, home_team_id, away_team_id, scheduled_at, status, home_score, away_score, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		match.ID, match.ChampionshipID, match.HomeTeamID, match.AwayTeamID, match.ScheduledAt,
		string(match.Status), match.HomeScore, match.AwayScore, match.CreatedAt, match.UpdatedAt)
	if err != nil {
		if db.IsForeignKeyViolation(err) {
			return fmt.Errorf("%w: championship or team", entities.ErrNotFound)
		}
		return util.SendMessageWithError("repository error inserting match", err)
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Match, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+selectColumns+" FROM matches WHERE id = $1", id)
	return scanMatch(row)
}

func (r *postgresRepository) ListByChampionship(ctx context.Context, championshipID uuid.UUID) ([]entities.Match, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT "+selectColumns+" FROM matches WHERE championship_id = $1 ORDER BY scheduled_at, id", championshipID)
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
	result, err := r.pool.Exec(ctx,
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
	result, err := r.pool.Exec(ctx, "DELETE FROM matches WHERE id = $1 AND status = 'SCHEDULED'", id)
	if err != nil {
		return util.SendMessageWithError("repository error deleting match", err)
	}

	noRowWasDeleted := result.RowsAffected() == 0
	if noRowWasDeleted {
		return fmt.Errorf("%w: only scheduled matches can be deleted", entities.ErrInvalidState)
	}
	return nil
}

func (r *postgresRepository) Transition(ctx context.Context, id uuid.UUID, from, to entities.MatchStatus, homeScore, awayScore int, updatedAt time.Time) error {
	result, err := r.pool.Exec(ctx,
		"UPDATE matches SET status = $3, home_score = $4, away_score = $5, updated_at = $6 WHERE id = $1 AND status = $2",
		id, string(from), string(to), homeScore, awayScore, updatedAt)
	if err != nil {
		return util.SendMessageWithError("repository error changing match status", err)
	}

	noRowWasUpdated := result.RowsAffected() == 0
	if noRowWasUpdated {
		return fmt.Errorf("%w: match must be %s", entities.ErrInvalidState, from)
	}
	return nil
}

func scanMatch(row pgx.Row) (*entities.Match, error) {
	var match entities.Match
	var status string

	err := row.Scan(&match.ID, &match.ChampionshipID, &match.HomeTeamID, &match.AwayTeamID, &match.ScheduledAt,
		&status, &match.HomeScore, &match.AwayScore, &match.CreatedAt, &match.UpdatedAt)
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

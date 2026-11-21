package championships

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

const selectColumns = "id, name, slots_total, slots_taken, status, created_at, updated_at"

type postgresRepository struct {
	pool *pgxpool.Pool
}

func newPostgresRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) Insert(ctx context.Context, championship *entities.Championship) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO championships (id, name, slots_total, slots_taken, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		championship.ID, championship.Name, championship.SlotsTotal, championship.SlotsTaken,
		string(championship.Status), championship.CreatedAt, championship.UpdatedAt)
	if err != nil {
		return util.SendMessageWithError("repository error inserting championship", err)
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Championship, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+selectColumns+" FROM championships WHERE id = $1", id)
	return scanChampionship(row)
}

func (r *postgresRepository) List(ctx context.Context) ([]entities.Championship, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+selectColumns+" FROM championships ORDER BY created_at DESC, id")
	if err != nil {
		return nil, util.SendMessageWithError("repository error listing championships", err)
	}
	defer rows.Close()

	allChampionships := []entities.Championship{}
	for rows.Next() {
		championship, err := scanChampionship(rows)
		if err != nil {
			return nil, err
		}
		allChampionships = append(allChampionships, *championship)
	}

	err = rows.Err()
	if err != nil {
		return nil, util.SendMessageWithError("repository error reading championships", err)
	}
	return allChampionships, nil
}

func (r *postgresRepository) Update(ctx context.Context, championship *entities.Championship) error {
	result, err := r.pool.Exec(ctx,
		"UPDATE championships SET name = $2, slots_total = $3, updated_at = $4 WHERE id = $1",
		championship.ID, championship.Name, championship.SlotsTotal, championship.UpdatedAt)
	if err != nil {
		if db.IsCheckViolation(err) {
			return fmt.Errorf("%w: slotsTotal cannot be lower than the teams already enrolled", entities.ErrInvalidInput)
		}
		return util.SendMessageWithError("repository error updating championship", err)
	}
	noRowWasUpdated := result.RowsAffected() == 0
	if noRowWasUpdated {
		return fmt.Errorf("%w: championship", entities.ErrNotFound)
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM championships WHERE id = $1", id)
	if err != nil {
		if db.IsForeignKeyViolation(err) {
			return entities.ErrChampionshipHasMatches
		}
		return util.SendMessageWithError("repository error deleting championship", err)
	}
	noRowWasDeleted := result.RowsAffected() == 0
	if noRowWasDeleted {
		return fmt.Errorf("%w: championship", entities.ErrNotFound)
	}
	return nil
}

func (r *postgresRepository) Transition(ctx context.Context, id uuid.UUID, from, to entities.ChampionshipStatus, updatedAt time.Time) error {
	result, err := r.pool.Exec(ctx,
		"UPDATE championships SET status = $3, updated_at = $4 WHERE id = $1 AND status = $2",
		id, string(from), string(to), updatedAt)
	if err != nil {
		return util.SendMessageWithError("repository error changing championship status", err)
	}

	noRowWasUpdated := result.RowsAffected() == 0
	if noRowWasUpdated {
		return fmt.Errorf("%w: championship must be %s", entities.ErrInvalidState, from)
	}
	return nil
}

func (r *postgresRepository) Enroll(ctx context.Context, championshipID, teamID uuid.UUID, enrolledAt time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return util.SendMessageWithError("repository error starting enrollment transaction", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result, err := tx.Exec(ctx,
		`UPDATE championships SET slots_taken = slots_taken + 1, updated_at = $2
		 WHERE id = $1 AND status = 'OPEN' AND slots_taken < slots_total`,
		championshipID, enrolledAt)
	if err != nil {
		return util.SendMessageWithError("repository error reserving a championship slot", err)
	}

	noSlotWasReserved := result.RowsAffected() == 0
	if noSlotWasReserved {
		return r.explainWhyNoSlotWasReserved(ctx, tx, championshipID)
	}

	_, err = tx.Exec(ctx,
		"INSERT INTO championship_teams (championship_id, team_id, enrolled_at) VALUES ($1, $2, $3)",
		championshipID, teamID, enrolledAt)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return entities.ErrTeamAlreadyEnrolled
		}
		return util.SendMessageWithError("repository error inserting enrollment", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return util.SendMessageWithError("repository error committing enrollment", err)
	}
	return nil
}

func (r *postgresRepository) explainWhyNoSlotWasReserved(ctx context.Context, tx pgx.Tx, championshipID uuid.UUID) error {
	var status string
	err := tx.QueryRow(ctx, "SELECT status FROM championships WHERE id = $1", championshipID).Scan(&status)
	if err != nil {
		noRowWasFound := errors.Is(err, pgx.ErrNoRows)
		if noRowWasFound {
			return fmt.Errorf("%w: championship", entities.ErrNotFound)
		}
		return util.SendMessageWithError("repository error reading championship status", err)
	}

	championshipIsNotOpen := entities.ChampionshipStatus(status) != entities.ChampionshipOpen
	if championshipIsNotOpen {
		return entities.ErrChampionshipNotOpen
	}
	return entities.ErrChampionshipFull
}

func (r *postgresRepository) Unenroll(ctx context.Context, championshipID, teamID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return util.SendMessageWithError("repository error starting unenrollment transaction", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result, err := tx.Exec(ctx,
		"DELETE FROM championship_teams WHERE championship_id = $1 AND team_id = $2", championshipID, teamID)
	if err != nil {
		return util.SendMessageWithError("repository error deleting enrollment", err)
	}
	teamWasNotEnrolled := result.RowsAffected() == 0
	if teamWasNotEnrolled {
		return entities.ErrTeamNotEnrolled
	}

	_, err = tx.Exec(ctx,
		"UPDATE championships SET slots_taken = slots_taken - 1, updated_at = now() WHERE id = $1", championshipID)
	if err != nil {
		return util.SendMessageWithError("repository error releasing a championship slot", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return util.SendMessageWithError("repository error committing unenrollment", err)
	}
	return nil
}

func (r *postgresRepository) ListEnrolledTeamIDs(ctx context.Context, championshipID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT team_id FROM championship_teams WHERE championship_id = $1 ORDER BY enrolled_at", championshipID)
	if err != nil {
		return nil, util.SendMessageWithError("repository error listing enrolled teams", err)
	}
	defer rows.Close()

	teamIDs := []uuid.UUID{}
	for rows.Next() {
		var teamID uuid.UUID
		err := rows.Scan(&teamID)
		if err != nil {
			return nil, util.SendMessageWithError("repository error scanning enrolled team", err)
		}
		teamIDs = append(teamIDs, teamID)
	}

	err = rows.Err()
	if err != nil {
		return nil, util.SendMessageWithError("repository error reading enrolled teams", err)
	}
	return teamIDs, nil
}

func (r *postgresRepository) IsEnrolled(ctx context.Context, championshipID, teamID uuid.UUID) (bool, error) {
	var isEnrolled bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM championship_teams WHERE championship_id = $1 AND team_id = $2)",
		championshipID, teamID).Scan(&isEnrolled)
	if err != nil {
		return false, util.SendMessageWithError("repository error checking enrollment", err)
	}
	return isEnrolled, nil
}

func scanChampionship(row pgx.Row) (*entities.Championship, error) {
	var championship entities.Championship
	var status string

	err := row.Scan(&championship.ID, &championship.Name, &championship.SlotsTotal, &championship.SlotsTaken,
		&status, &championship.CreatedAt, &championship.UpdatedAt)
	if err != nil {
		noRowWasFound := errors.Is(err, pgx.ErrNoRows)
		if noRowWasFound {
			return nil, fmt.Errorf("%w: championship", entities.ErrNotFound)
		}
		return nil, util.SendMessageWithError("repository error scanning championship", err)
	}

	championship.Status = entities.ChampionshipStatus(status)
	championship.CreatedAt = championship.CreatedAt.UTC()
	championship.UpdatedAt = championship.UpdatedAt.UTC()
	return &championship, nil
}

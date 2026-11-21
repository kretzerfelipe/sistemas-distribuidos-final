package players

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/db"
	"futhub/main/internal/platform/util"
)

const selectColumns = "id, user_id, name, position, shirt_number, created_at, updated_at"

type postgresRepository struct {
	pool *pgxpool.Pool
}

func newPostgresRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) Insert(ctx context.Context, player *entities.Player) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO players (id, user_id, name, position, shirt_number, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		player.ID, player.UserID, player.Name, string(player.Position), player.ShirtNumber, player.CreatedAt, player.UpdatedAt)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return entities.ErrPlayerAlreadyExists
		}
		return util.SendMessageWithError("repository error inserting player", err)
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Player, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+selectColumns+" FROM players WHERE id = $1", id)
	return scanPlayer(row)
}

func (r *postgresRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*entities.Player, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+selectColumns+" FROM players WHERE user_id = $1", userID)
	return scanPlayer(row)
}

func (r *postgresRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]entities.Player, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+selectColumns+" FROM players WHERE id = ANY($1) ORDER BY name, id", ids)
	if err != nil {
		return nil, util.SendMessageWithError("repository error finding players by ids", err)
	}
	return collectPlayers(rows)
}

func (r *postgresRepository) List(ctx context.Context) ([]entities.Player, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+selectColumns+" FROM players ORDER BY name, id")
	if err != nil {
		return nil, util.SendMessageWithError("repository error listing players", err)
	}
	return collectPlayers(rows)
}

func (r *postgresRepository) Update(ctx context.Context, player *entities.Player) error {
	result, err := r.pool.Exec(ctx,
		"UPDATE players SET name = $2, position = $3, shirt_number = $4, updated_at = $5 WHERE id = $1",
		player.ID, player.Name, string(player.Position), player.ShirtNumber, player.UpdatedAt)
	if err != nil {
		return util.SendMessageWithError("repository error updating player", err)
	}
	noRowWasUpdated := result.RowsAffected() == 0
	if noRowWasUpdated {
		return fmt.Errorf("%w: player", entities.ErrNotFound)
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM players WHERE id = $1", id)
	if err != nil {
		return util.SendMessageWithError("repository error deleting player", err)
	}
	noRowWasDeleted := result.RowsAffected() == 0
	if noRowWasDeleted {
		return fmt.Errorf("%w: player", entities.ErrNotFound)
	}
	return nil
}

func collectPlayers(rows pgx.Rows) ([]entities.Player, error) {
	defer rows.Close()

	allPlayers := []entities.Player{}
	for rows.Next() {
		player, err := scanPlayer(rows)
		if err != nil {
			return nil, err
		}
		allPlayers = append(allPlayers, *player)
	}

	err := rows.Err()
	if err != nil {
		return nil, util.SendMessageWithError("repository error reading players", err)
	}
	return allPlayers, nil
}

func scanPlayer(row pgx.Row) (*entities.Player, error) {
	var player entities.Player
	var position string

	err := row.Scan(&player.ID, &player.UserID, &player.Name, &position, &player.ShirtNumber, &player.CreatedAt, &player.UpdatedAt)
	if err != nil {
		noRowWasFound := errors.Is(err, pgx.ErrNoRows)
		if noRowWasFound {
			return nil, fmt.Errorf("%w: player", entities.ErrNotFound)
		}
		return nil, util.SendMessageWithError("repository error scanning player", err)
	}

	player.Position = entities.Position(position)
	player.CreatedAt = player.CreatedAt.UTC()
	player.UpdatedAt = player.UpdatedAt.UTC()
	return &player, nil
}

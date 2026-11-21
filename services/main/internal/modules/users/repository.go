package users

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/util"
)

const (
	pgUniqueViolationCode = "23505"
	selectColumns         = "id, name, email, password_hash, role, created_at, updated_at"
)

type postgresRepository struct {
	pool *pgxpool.Pool
}

func newPostgresRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) Insert(ctx context.Context, user *entities.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, name, email, password_hash, role, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		user.ID, user.Name, user.Email, user.PasswordHash, string(user.Role), user.CreatedAt, user.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return entities.ErrEmailTaken
		}
		return util.SendMessageWithError("repository error inserting user", err)
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+selectColumns+" FROM users WHERE id = $1", id)
	return scanUser(row)
}

func (r *postgresRepository) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+selectColumns+" FROM users WHERE email = $1", email)
	return scanUser(row)
}

func (r *postgresRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		return false, util.SendMessageWithError("repository error checking whether email exists", err)
	}
	return exists, nil
}

func (r *postgresRepository) List(ctx context.Context) ([]entities.User, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+selectColumns+" FROM users ORDER BY created_at, id")
	if err != nil {
		return nil, util.SendMessageWithError("repository error listing users", err)
	}
	defer rows.Close()

	allUsers := []entities.User{}
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		allUsers = append(allUsers, *user)
	}

	err = rows.Err()
	if err != nil {
		return nil, util.SendMessageWithError("repository error reading listed users", err)
	}
	return allUsers, nil
}

func scanUser(row pgx.Row) (*entities.User, error) {
	var user entities.User
	var role string

	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		noRowWasFound := errors.Is(err, pgx.ErrNoRows)
		if noRowWasFound {
			return nil, entities.ErrNotFound
		}
		return nil, util.SendMessageWithError("repository error scanning user", err)
	}

	user.Role = entities.Role(role)
	user.CreatedAt = user.CreatedAt.UTC()
	user.UpdatedAt = user.UpdatedAt.UTC()
	return &user, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolationCode
}

package premium

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"futhub/notification/internal/security"
)

type Repository interface {
	IsActive(ctx context.Context, userID uuid.UUID) (bool, error)
}

type Service struct {
	repository     Repository
	configuredIDs  map[string]struct{}
	adminIsPremium bool
}

func NewService(repository Repository, configuredUserIDs []string, adminIsPremium bool) *Service {
	ids := make(map[string]struct{}, len(configuredUserIDs))
	for _, id := range configuredUserIDs {
		ids[id] = struct{}{}
	}
	return &Service{repository: repository, configuredIDs: ids, adminIsPremium: adminIsPremium}
}

func (s *Service) IsPremium(ctx context.Context, principal security.Principal) (bool, error) {
	if s.adminIsPremium && principal.Role == security.RoleAdmin {
		return true, nil
	}
	if _, configured := s.configuredIDs[principal.UserID]; configured {
		return true, nil
	}

	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return false, nil
	}
	return s.repository.IsActive(ctx, userID)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) IsActive(ctx context.Context, userID uuid.UUID) (bool, error) {
	var active bool
	err := r.pool.QueryRow(ctx, `SELECT active FROM premium_users WHERE user_id = $1`, userID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read premium projection: %w", err)
	}
	return active, nil
}

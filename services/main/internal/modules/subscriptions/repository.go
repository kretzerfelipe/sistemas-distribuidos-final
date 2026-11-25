package subscriptions

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

const subscriptionColumns = "id, user_id, status, COALESCE(reason, ''), charge_id, amount_cents, expires_at, created_at, updated_at"

type postgresRepository struct {
	pool *pgxpool.Pool
}

func newPostgresRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) executor(ctx context.Context) db.Executor {
	return db.ExecutorFrom(ctx, r.pool)
}

func (r *postgresRepository) Insert(ctx context.Context, subscription *entities.Subscription) error {
	_, err := r.executor(ctx).Exec(ctx,
		`INSERT INTO subscriptions (id, user_id, status, amount_cents, expires_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		subscription.ID, subscription.UserID, string(subscription.Status), subscription.AmountCents,
		subscription.ExpiresAt, subscription.CreatedAt, subscription.UpdatedAt)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return entities.ErrSubscriptionInForce
		}
		return util.SendMessageWithError("repository error inserting subscription", err)
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Subscription, error) {
	row := r.executor(ctx).QueryRow(ctx, "SELECT "+subscriptionColumns+" FROM subscriptions WHERE id = $1", id)
	return scanSubscription(row)
}

func (r *postgresRepository) FindLatestByUser(ctx context.Context, userID uuid.UUID) (*entities.Subscription, error) {
	row := r.executor(ctx).QueryRow(ctx,
		"SELECT "+subscriptionColumns+" FROM subscriptions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1", userID)
	return scanSubscription(row)
}

func (r *postgresRepository) Activate(ctx context.Context, id, chargeID uuid.UUID, updatedAt time.Time) (bool, error) {
	result, err := r.executor(ctx).Exec(ctx,
		"UPDATE subscriptions SET status = 'ACTIVE', charge_id = $2, updated_at = $3 WHERE id = $1 AND status = 'PENDING'",
		id, chargeID, updatedAt)
	if err != nil {
		return false, util.SendMessageWithError("repository error activating subscription", err)
	}
	return result.RowsAffected() == 1, nil
}

func (r *postgresRepository) Close(ctx context.Context, id uuid.UUID, to entities.SubscriptionStatus, reason string, updatedAt time.Time) (bool, error) {
	result, err := r.executor(ctx).Exec(ctx,
		"UPDATE subscriptions SET status = $2, reason = $3, updated_at = $4 WHERE id = $1 AND status = 'PENDING'",
		id, string(to), reason, updatedAt)
	if err != nil {
		return false, util.SendMessageWithError("repository error closing subscription", err)
	}
	return result.RowsAffected() == 1, nil
}

func (r *postgresRepository) ExpireOverdue(ctx context.Context, now time.Time) ([]entities.Subscription, error) {
	rows, err := r.executor(ctx).Query(ctx,
		`UPDATE subscriptions SET status = 'EXPIRED', reason = 'TIMEOUT', updated_at = $1
		 WHERE status = 'PENDING' AND expires_at < $1
		 RETURNING `+subscriptionColumns, now)
	if err != nil {
		return nil, util.SendMessageWithError("repository error expiring subscriptions", err)
	}
	defer rows.Close()

	expired := []entities.Subscription{}
	for rows.Next() {
		subscription, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		expired = append(expired, *subscription)
	}

	err = rows.Err()
	if err != nil {
		return nil, util.SendMessageWithError("repository error reading expired subscriptions", err)
	}
	return expired, nil
}

func scanSubscription(row pgx.Row) (*entities.Subscription, error) {
	var subscription entities.Subscription
	var status string

	err := row.Scan(&subscription.ID, &subscription.UserID, &status, &subscription.Reason, &subscription.ChargeID,
		&subscription.AmountCents, &subscription.ExpiresAt, &subscription.CreatedAt, &subscription.UpdatedAt)
	if err != nil {
		noRowWasFound := errors.Is(err, pgx.ErrNoRows)
		if noRowWasFound {
			return nil, fmt.Errorf("%w: subscription", entities.ErrNotFound)
		}
		return nil, util.SendMessageWithError("repository error scanning subscription", err)
	}

	subscription.Status = entities.SubscriptionStatus(status)
	subscription.ExpiresAt = subscription.ExpiresAt.UTC()
	subscription.CreatedAt = subscription.CreatedAt.UTC()
	subscription.UpdatedAt = subscription.UpdatedAt.UTC()
	return &subscription, nil
}

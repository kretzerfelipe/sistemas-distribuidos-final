package accounts

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"futhub/notification/internal/db"
)

type Notification struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"createdAt"`
	UserID    uuid.UUID `json:"-"`
}

type Store interface {
	SetPremium(ctx context.Context, userID uuid.UUID, active bool) error
	AddNotification(ctx context.Context, notification Notification) error
	ListNotifications(ctx context.Context, userID uuid.UUID, limit int) ([]Notification, error)
	MarkRead(ctx context.Context, userID, notificationID uuid.UUID) (bool, error)
}

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) SetPremium(ctx context.Context, userID uuid.UUID, active bool) error {
	_, err := db.ExecutorFrom(ctx, s.pool).Exec(ctx,
		`INSERT INTO premium_users (user_id, active, updated_at) VALUES ($1, $2, now())
		 ON CONFLICT (user_id) DO UPDATE SET active = EXCLUDED.active, updated_at = now()`,
		userID, active)
	if err != nil {
		return fmt.Errorf("save premium projection: %w", err)
	}
	return nil
}

func (s *PostgresStore) AddNotification(ctx context.Context, notification Notification) error {
	_, err := db.ExecutorFrom(ctx, s.pool).Exec(ctx,
		`INSERT INTO notifications (id, user_id, type, message, created_at) VALUES ($1, $2, $3, $4, $5)`,
		notification.ID, notification.UserID, notification.Type, notification.Message, notification.CreatedAt)
	if err != nil {
		return fmt.Errorf("save notification: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListNotifications(ctx context.Context, userID uuid.UUID, limit int) ([]Notification, error) {
	rows, err := db.ExecutorFrom(ctx, s.pool).Query(ctx,
		`SELECT id, type, message, read, created_at FROM notifications
		 WHERE user_id = $1 ORDER BY created_at DESC, id LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	notifications := []Notification{}
	for rows.Next() {
		var notification Notification
		if err := rows.Scan(&notification.ID, &notification.Type, &notification.Message, &notification.Read, &notification.CreatedAt); err != nil {
			return nil, fmt.Errorf("read notification: %w", err)
		}
		notification.CreatedAt = notification.CreatedAt.UTC()
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read notifications: %w", err)
	}
	return notifications, nil
}

func (s *PostgresStore) MarkRead(ctx context.Context, userID, notificationID uuid.UUID) (bool, error) {
	result, err := db.ExecutorFrom(ctx, s.pool).Exec(ctx,
		`UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2`, notificationID, userID)
	if err != nil {
		return false, fmt.Errorf("mark notification as read: %w", err)
	}
	return result.RowsAffected() == 1, nil
}

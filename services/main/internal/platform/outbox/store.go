package outbox

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"futhub/main/internal/platform/util"
)

type Store interface {
	FetchUnpublished(ctx context.Context, limit int) ([]Message, error)
	MarkPublished(ctx context.Context, id uuid.UUID, publishedAt time.Time) error
}

type postgresStore struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) Store {
	return &postgresStore{pool: pool}
}

func (s *postgresStore) FetchUnpublished(ctx context.Context, limit int) ([]Message, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, correlation_id, type, routing_key, payload, created_at
		 FROM outbox WHERE published_at IS NULL ORDER BY position LIMIT $1`, limit)
	if err != nil {
		return nil, util.SendMessageWithError("outbox error fetching unpublished messages", err)
	}
	defer rows.Close()

	pending := []Message{}
	for rows.Next() {
		var message Message
		err := rows.Scan(&message.ID, &message.CorrelationID, &message.Type, &message.RoutingKey, &message.Body, &message.CreatedAt)
		if err != nil {
			return nil, util.SendMessageWithError("outbox error scanning message", err)
		}
		pending = append(pending, message)
	}

	err = rows.Err()
	if err != nil {
		return nil, util.SendMessageWithError("outbox error reading unpublished messages", err)
	}
	return pending, nil
}

func (s *postgresStore) MarkPublished(ctx context.Context, id uuid.UUID, publishedAt time.Time) error {
	_, err := s.pool.Exec(ctx, "UPDATE outbox SET published_at = $2 WHERE id = $1", id, publishedAt)
	if err != nil {
		return util.SendMessageWithError("outbox error marking message as published", err)
	}
	return nil
}

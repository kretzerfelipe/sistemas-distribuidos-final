package outbox

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"futhub/main/internal/platform/db"
	"futhub/main/internal/platform/httpx"
	"futhub/main/internal/platform/util"
)

var ErrNoTransaction = errors.New("outbox: events must be added inside a transaction")

type Writer interface {
	Add(ctx context.Context, event Event) error
}

type postgresWriter struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewWriter(pool *pgxpool.Pool) Writer {
	return &postgresWriter{pool: pool, now: time.Now}
}

func (w *postgresWriter) Add(ctx context.Context, event Event) error {
	_, inTransaction := db.TransactionFrom(ctx)
	if !inTransaction {
		return ErrNoTransaction
	}

	message, err := buildMessage(event, httpx.CorrelationIDFrom(ctx), w.now())
	if err != nil {
		return util.SendMessageWithError("outbox error building message", err)
	}

	_, err = db.ExecutorFrom(ctx, w.pool).Exec(ctx,
		`INSERT INTO outbox (id, correlation_id, type, routing_key, payload, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		message.ID, message.CorrelationID, message.Type, message.RoutingKey, message.Body, message.CreatedAt)
	if err != nil {
		return util.SendMessageWithError("outbox error inserting message", err)
	}
	return nil
}

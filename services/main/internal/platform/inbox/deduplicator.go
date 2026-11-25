package inbox

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"futhub/main/internal/platform/db"
	"futhub/main/internal/platform/util"
)

type Deduplicator struct {
	pool         *pgxpool.Pool
	transactions *db.TransactionManager
}

func NewDeduplicator(pool *pgxpool.Pool, transactions *db.TransactionManager) *Deduplicator {
	return &Deduplicator{pool: pool, transactions: transactions}
}

func (d *Deduplicator) RunOnce(ctx context.Context, messageID string, effect func(ctx context.Context) error) (bool, error) {
	processedNow := false

	err := d.transactions.WithTx(ctx, func(ctx context.Context) error {
		inserted, err := db.ExecutorFrom(ctx, d.pool).Exec(ctx,
			"INSERT INTO processed_messages (message_id) VALUES ($1) ON CONFLICT DO NOTHING", messageID)
		if err != nil {
			return util.SendMessageWithError("inbox error registering message", err)
		}

		messageWasAlreadyProcessed := inserted.RowsAffected() == 0
		if messageWasAlreadyProcessed {
			return nil
		}

		err = effect(ctx)
		if err != nil {
			return err
		}
		processedNow = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return processedNow, nil
}

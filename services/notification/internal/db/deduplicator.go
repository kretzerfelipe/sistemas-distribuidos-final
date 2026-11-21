package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Deduplicator struct {
	pool *pgxpool.Pool
}

func NewDeduplicator(pool *pgxpool.Pool) *Deduplicator {
	return &Deduplicator{pool: pool}
}

func (d *Deduplicator) RunOnce(ctx context.Context, messageID string, effect func(ctx context.Context) error) (bool, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	inserted, err := tx.Exec(ctx,
		`INSERT INTO processed_messages (message_id) VALUES ($1) ON CONFLICT DO NOTHING`, messageID)
	if err != nil {
		return false, fmt.Errorf("register message %s: %w", messageID, err)
	}
	messageWasAlreadyProcessed := inserted.RowsAffected() == 0
	if messageWasAlreadyProcessed {
		return false, nil
	}

	if err := effect(ctx); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit transaction: %w", err)
	}
	return true, nil
}

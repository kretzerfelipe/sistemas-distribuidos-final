package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
}

type transactionKey struct{}

func ExecutorFrom(ctx context.Context, pool *pgxpool.Pool) Executor {
	tx, inTransaction := ctx.Value(transactionKey{}).(pgx.Tx)
	if inTransaction {
		return tx
	}
	return pool
}

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

	if err := effect(context.WithValue(ctx, transactionKey{}, tx)); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit transaction: %w", err)
	}
	return true, nil
}

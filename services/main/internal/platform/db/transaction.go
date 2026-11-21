package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"futhub/main/internal/platform/util"
)

type Executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
}

type transactionKey struct{}

type TransactionManager struct {
	pool *pgxpool.Pool
}

func NewTransactionManager(pool *pgxpool.Pool) *TransactionManager {
	return &TransactionManager{pool: pool}
}

func (m *TransactionManager) WithTx(ctx context.Context, work func(ctx context.Context) error) error {
	_, alreadyInTransaction := TransactionFrom(ctx)
	if alreadyInTransaction {
		return work(ctx)
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return util.SendMessageWithError("db error starting transaction", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = work(context.WithValue(ctx, transactionKey{}, tx))
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return util.SendMessageWithError("db error committing transaction", err)
	}
	return nil
}

func TransactionFrom(ctx context.Context) (pgx.Tx, bool) {
	tx, found := ctx.Value(transactionKey{}).(pgx.Tx)
	return tx, found
}

func ExecutorFrom(ctx context.Context, pool *pgxpool.Pool) Executor {
	tx, inTransaction := TransactionFrom(ctx)
	if inTransaction {
		return tx
	}
	return pool
}

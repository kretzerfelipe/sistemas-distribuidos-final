package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	connectTimeout = 60 * time.Second
	pingTimeout    = 5 * time.Second
	initialBackoff = time.Second
	maxBackoff     = 8 * time.Second
)

func Connect(ctx context.Context, url string, log *slog.Logger) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := waitUntilReachable(ctx, pool, log); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func waitUntilReachable(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) error {
	deadline := time.Now().Add(connectTimeout)
	backoff := initialBackoff

	for {
		err := ping(ctx, pool)
		if err == nil {
			return nil
		}

		deadlineWasReached := time.Now().After(deadline)
		contextWasCancelled := ctx.Err() != nil
		if deadlineWasReached || contextWasCancelled {
			return fmt.Errorf("database unavailable: %w", err)
		}

		log.Warn("database not ready, retrying", "error", err, "retry_in", backoff.String())
		if !sleep(ctx, backoff) {
			return ctx.Err()
		}
		backoff = min(backoff*2, maxBackoff)
	}
}

func ping(ctx context.Context, pool *pgxpool.Pool) error {
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	return pool.Ping(pingCtx)
}

func sleep(ctx context.Context, duration time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(duration):
		return true
	}
}

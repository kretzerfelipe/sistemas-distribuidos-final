package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"

	"futhub/notification/migrations"
)

const (
	connectTimeout   = 60 * time.Second
	pingTimeout      = 5 * time.Second
	initialBackoff   = time.Second
	maxBackoff       = 8 * time.Second
	pgxMigrateScheme = "pgx5://"
)

var postgresSchemes = []string{"postgres://", "postgresql://"}

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

func Migrate(databaseURL string) error {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("load migration files: %w", err)
	}
	migrator, err := migrate.NewWithSourceInstance("iofs", source, withPgxScheme(databaseURL))
	if err != nil {
		return fmt.Errorf("open database for migrations: %w", err)
	}
	defer migrator.Close()

	err = migrator.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func waitUntilReachable(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) error {
	deadline := time.Now().Add(connectTimeout)
	backoff := initialBackoff

	for {
		pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
		err := pool.Ping(pingCtx)
		cancel()
		if err == nil {
			return nil
		}

		if time.Now().After(deadline) || ctx.Err() != nil {
			return fmt.Errorf("database unavailable: %w", err)
		}

		log.Warn("database not ready, retrying", "error", err, "retry_in", backoff.String())
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, maxBackoff)
	}
}

func withPgxScheme(databaseURL string) string {
	for _, scheme := range postgresSchemes {
		if strings.HasPrefix(databaseURL, scheme) {
			return pgxMigrateScheme + strings.TrimPrefix(databaseURL, scheme)
		}
	}
	return databaseURL
}

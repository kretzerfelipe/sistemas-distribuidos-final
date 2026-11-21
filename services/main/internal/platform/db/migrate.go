package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"futhub/main/migrations"
)

const (
	migrationsDirectory = "."
	migrationsSource    = "iofs"
	pgxMigrateScheme    = "pgx5://"
)

var postgresSchemes = []string{"postgres://", "postgresql://"}

func Migrate(databaseURL string) error {
	migrator, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	defer migrator.Close()

	err = migrator.Up()
	nothingToApply := errors.Is(err, migrate.ErrNoChange)
	if err != nil && !nothingToApply {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func newMigrator(databaseURL string) (*migrate.Migrate, error) {
	source, err := iofs.New(migrations.FS, migrationsDirectory)
	if err != nil {
		return nil, fmt.Errorf("load migration files: %w", err)
	}
	migrator, err := migrate.NewWithSourceInstance(migrationsSource, source, withPgxScheme(databaseURL))
	if err != nil {
		return nil, fmt.Errorf("open database for migrations: %w", err)
	}
	return migrator, nil
}

func withPgxScheme(databaseURL string) string {
	for _, scheme := range postgresSchemes {
		urlUsesThisScheme := strings.HasPrefix(databaseURL, scheme)
		if urlUsesThisScheme {
			return pgxMigrateScheme + strings.TrimPrefix(databaseURL, scheme)
		}
	}
	return databaseURL
}

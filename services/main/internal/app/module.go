package app

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"futhub/main/internal/platform/config"
	"futhub/main/internal/platform/db"
	"futhub/main/internal/platform/outbox"
	"futhub/main/internal/platform/security"
	"futhub/main/internal/platform/storage"
)

type Deps struct {
	Config       config.Config
	Logger       *slog.Logger
	DB           *pgxpool.Pool
	Transactions *db.TransactionManager
	Outbox       outbox.Writer
	Storage      storage.Client
	Hasher       security.PasswordHasher
	Tokens       *security.TokenManager
}

type Module interface {
	Name() string
	RegisterRoutes(r chi.Router) error
}

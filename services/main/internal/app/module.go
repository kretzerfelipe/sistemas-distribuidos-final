package app

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"futhub/main/internal/platform/config"
	"futhub/main/internal/platform/security"
)

type Deps struct {
	Config config.Config
	Logger *slog.Logger
	DB     *pgxpool.Pool
	Hasher security.PasswordHasher
	Tokens *security.TokenManager
}

type Module interface {
	Name() string
	RegisterRoutes(r chi.Router) error
}

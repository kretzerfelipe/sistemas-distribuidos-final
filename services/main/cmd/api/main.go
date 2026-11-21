package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"futhub/main/internal/app"
	"futhub/main/internal/modules/auth"
	"futhub/main/internal/modules/championships"
	"futhub/main/internal/modules/matches"
	"futhub/main/internal/modules/players"
	"futhub/main/internal/modules/teams"
	"futhub/main/internal/modules/users"
	"futhub/main/internal/platform/config"
	"futhub/main/internal/platform/db"
	"futhub/main/internal/platform/httpx"
	"futhub/main/internal/platform/logger"
	"futhub/main/internal/platform/security"
)

const (
	readHeaderTimeout  = 5 * time.Second
	shutdownTimeout    = 10 * time.Second
	healthCheckTimeout = 2 * time.Second
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	log := logger.New(cfg.LogLevel)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL, log)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		return err
	}
	log.Info("database ready, migrations applied")

	deps := app.Deps{
		Config: cfg,
		Logger: log,
		DB:     pool,
		Hasher: security.NewBcryptHasher(cfg.BcryptCost),
		Tokens: security.NewTokenManager(cfg.JWTSecret, cfg.JWTExpiration),
	}

	usersModule := users.New(deps)
	authModule, err := auth.New(deps, usersModule.Service())
	if err != nil {
		return err
	}

	playersModule := players.New(deps)
	teamsModule := teams.New(deps, playersModule.Service())
	championshipsModule := championships.New(deps, teamsModule.Service())
	matchesModule := matches.New(deps, championshipsModule.Service())

	router, err := newRouter(log, pool, usersModule, authModule, playersModule, teamsModule, championshipsModule, matchesModule)
	if err != nil {
		return err
	}
	if err := seedAdmin(ctx, cfg, authModule, log); err != nil {
		return err
	}

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	return serve(ctx, server, log)
}

func newRouter(log *slog.Logger, database pinger, modules ...app.Module) (http.Handler, error) {
	router := chi.NewRouter()
	router.Use(httpx.CorrelationID, httpx.RequestLogger(log), httpx.Recover(log))
	router.Get("/health", healthHandler(database, log))

	for _, module := range modules {
		err := module.RegisterRoutes(router)
		if err != nil {
			return nil, fmt.Errorf("register routes of module %s: %w", module.Name(), err)
		}
		log.Info("module registered", "module", module.Name())
	}
	return router, nil
}

func seedAdmin(ctx context.Context, cfg config.Config, authModule *auth.Module, log *slog.Logger) error {
	adminSeedIsConfigured := cfg.AdminEmail != ""
	if !adminSeedIsConfigured {
		return nil
	}

	created, err := authModule.SeedAdmin(ctx, cfg.AdminName, cfg.AdminEmail, cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	log.Info("admin seed checked", "email", cfg.AdminEmail, "created", created)
	return nil
}

func serve(ctx context.Context, server *http.Server, log *slog.Logger) error {
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		serverStoppedUnexpectedly := !errors.Is(err, http.ErrServerClosed)
		if serverStoppedUnexpectedly {
			return err
		}
		return nil
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

type pinger interface {
	Ping(ctx context.Context) error
}

func healthHandler(database pinger, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
		defer cancel()

		err := database.Ping(ctx)
		databaseIsUp := err == nil

		if databaseIsUp {
			writeHealth(w, http.StatusOK, "ok", "up")
			return
		}
		log.Warn("health check: database down", "error", err)
		writeHealth(w, http.StatusServiceUnavailable, "degraded", "down")
	}
}

func writeHealth(w http.ResponseWriter, httpStatus int, serviceStatus, databaseStatus string) {
	httpx.WriteJSON(w, httpStatus, map[string]any{
		"status": serviceStatus,
		"checks": map[string]string{"database": databaseStatus},
	})
}

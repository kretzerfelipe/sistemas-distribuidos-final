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

	"futhub/notification/internal/accounts"
	"futhub/notification/internal/config"
	"futhub/notification/internal/contracts"
	"futhub/notification/internal/db"
	"futhub/notification/internal/httpx"
	"futhub/notification/internal/live"
	"futhub/notification/internal/logger"
	"futhub/notification/internal/messaging"
	"futhub/notification/internal/premium"
	"futhub/notification/internal/security"
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

	validator, err := contracts.NewValidator(cfg.ContractsDirectory)
	if err != nil {
		return err
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL, log)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		return err
	}
	log.Info("database ready, migrations applied")

	store, err := live.NewRedisStore(cfg.RedisURL, cfg.StateTTL, cfg.SnapshotInterval)
	if err != nil {
		return err
	}
	defer store.Close()

	hub := live.NewHub()
	processor := live.NewProcessor(store, hub, log)
	accountsStore := accounts.NewPostgresStore(pool)
	typeRouter := live.NewTypeRouter()
	typeRouter.Route(live.MatchMessageTypes, processor)
	typeRouter.Route(accounts.HandledMessageTypes, accounts.NewProcessor(accountsStore))
	messageHandler := live.NewMessageHandler(validator, db.NewDeduplicator(pool), typeRouter, log)

	consumer := messaging.NewConsumer(messaging.Settings{
		URL:             cfg.RabbitMQURL,
		Queue:           cfg.ConsumerQueue,
		RetryQueue:      cfg.RetryQueue,
		DeadLetterQueue: cfg.DeadLetterQueue,
		Prefetch:        cfg.Prefetch,
		MaxRetries:      cfg.MaxRetries,
	}, messageHandler, log)
	go consumer.Run(ctx)

	go live.NewSnapshotJob(store, cfg.SnapshotInterval, log).Run(ctx)

	premiumService := premium.NewService(premium.NewPostgresRepository(pool), cfg.PremiumUserIDs, cfg.AdminIsPremium)
	liveHandler := live.NewHandler(ctx, store, hub, premiumService, security.NewTokenParser(cfg.JWTSecret), log)

	router := chi.NewRouter()
	router.Use(httpx.CorrelationID, httpx.RequestLogger(log), httpx.Recover(log))
	router.Get("/health", healthHandler(pool, store, consumer, log))
	liveHandler.RegisterRoutes(router)
	accounts.NewHandler(accountsStore, security.NewTokenParser(cfg.JWTSecret), log).RegisterRoutes(router)

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	return serve(ctx, server, log)
}

func serve(ctx context.Context, server *http.Server, log *slog.Logger) error {
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
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

type brokerState interface {
	IsConnected() bool
}

func healthHandler(database, cache pinger, broker brokerState, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
		defer cancel()

		databaseError := database.Ping(ctx)
		cacheError := cache.Ping(ctx)
		databaseIsUp := databaseError == nil
		cacheIsUp := cacheError == nil
		brokerIsUp := broker.IsConnected()

		httpStatus := http.StatusOK
		if !databaseIsUp {
			log.Warn("health check: database down", "error", databaseError)
			httpStatus = http.StatusServiceUnavailable
		}
		if !cacheIsUp {
			log.Warn("health check: redis down", "error", cacheError)
		}

		status := "degraded"
		if databaseIsUp && cacheIsUp && brokerIsUp {
			status = "ok"
		}
		httpx.WriteJSON(w, httpStatus, map[string]any{
			"status": status,
			"checks": map[string]string{
				"database": upOrDown(databaseIsUp),
				"redis":    upOrDown(cacheIsUp),
				"rabbitmq": upOrDown(brokerIsUp),
			},
		})
	}
}

func upOrDown(isUp bool) string {
	if isUp {
		return "up"
	}
	return "down"
}

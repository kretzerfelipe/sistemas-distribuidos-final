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
	"futhub/main/internal/modules/subscriptions"
	"futhub/main/internal/modules/teams"
	"futhub/main/internal/modules/users"
	"futhub/main/internal/platform/config"
	"futhub/main/internal/platform/contracts"
	"futhub/main/internal/platform/db"
	"futhub/main/internal/platform/httpx"
	"futhub/main/internal/platform/inbox"
	"futhub/main/internal/platform/logger"
	"futhub/main/internal/platform/messaging"
	"futhub/main/internal/platform/outbox"
	"futhub/main/internal/platform/security"
	"futhub/main/internal/platform/storage"
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

	broker := messaging.NewBroker(cfg.RabbitMQURL, log)
	go broker.Run(ctx)

	publisher := outbox.NewPublisher(outbox.NewStore(pool), broker, cfg.EventsExchange, cfg.OutboxPollInterval, cfg.OutboxBatchSize, log)
	go publisher.Run(ctx)

	photoStorage, err := storage.NewS3Client(ctx, storage.Settings{
		Endpoint:        cfg.StorageEndpoint,
		Region:          cfg.StorageRegion,
		AccessKeyID:     cfg.StorageAccessKeyID,
		SecretAccessKey: cfg.StorageSecretAccessKey,
		Bucket:          cfg.StorageBucket,
	})
	if err != nil {
		return err
	}

	deps := app.Deps{
		Config:       cfg,
		Logger:       log,
		DB:           pool,
		Transactions: db.NewTransactionManager(pool),
		Outbox:       outbox.NewWriter(pool),
		Storage:      photoStorage,
		Hasher:       security.NewBcryptHasher(cfg.BcryptCost),
		Tokens:       security.NewTokenManager(cfg.JWTSecret, cfg.JWTExpiration),
	}

	usersModule := users.New(deps)
	authModule, err := auth.New(deps, usersModule.Service())
	if err != nil {
		return err
	}

	playersModule := players.New(deps)
	teamsModule := teams.New(deps, playersModule.Service())
	championshipsModule := championships.New(deps, teamsModule.Service())
	matchesModule := matches.New(deps, championshipsModule.Service(), teamsModule.Service(), playersModule.Service())
	subscriptionsModule := subscriptions.New(deps)

	validator, err := contracts.NewValidator(cfg.ContractsDirectory)
	if err != nil {
		return err
	}
	messageHandler := subscriptions.NewMessageHandler(
		validator, inbox.NewDeduplicator(pool, deps.Transactions), subscriptionsModule.Service(), log)
	consumer := messaging.NewConsumer(messaging.ConsumerSettings{
		URL:             cfg.RabbitMQURL,
		Queue:           cfg.ConsumerQueue,
		RetryQueue:      cfg.RetryQueue,
		DeadLetterQueue: cfg.DeadLetterQueue,
		Prefetch:        cfg.Prefetch,
		MaxRetries:      cfg.MaxRetries,
	}, messageHandler, log)
	go consumer.Run(ctx)
	go subscriptionsModule.RunExpiration(ctx, cfg.ExpirationInterval, log)

	router, err := newRouter(log, pool, brokerConnections{broker, consumer},
		usersModule, authModule, playersModule, teamsModule, championshipsModule, matchesModule, subscriptionsModule)
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

func newRouter(log *slog.Logger, database pinger, broker brokerState, modules ...app.Module) (http.Handler, error) {
	router := chi.NewRouter()
	router.Use(httpx.CorrelationID, httpx.RequestLogger(log), httpx.Recover(log))
	router.Get("/health", healthHandler(database, broker, log))

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

type brokerState interface {
	IsConnected() bool
}

type brokerConnections struct {
	publisher brokerState
	consumer  brokerState
}

func (b brokerConnections) IsConnected() bool {
	return b.publisher.IsConnected() && b.consumer.IsConnected()
}

func healthHandler(database pinger, broker brokerState, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
		defer cancel()

		err := database.Ping(ctx)
		databaseIsUp := err == nil
		brokerIsUp := broker.IsConnected()

		httpStatus := http.StatusOK
		if !databaseIsUp {
			log.Warn("health check: database down", "error", err)
			httpStatus = http.StatusServiceUnavailable
		}
		everythingIsUp := databaseIsUp && brokerIsUp

		httpx.WriteJSON(w, httpStatus, map[string]any{
			"status": statusName(everythingIsUp),
			"checks": map[string]string{
				"database": upOrDown(databaseIsUp),
				"rabbitmq": upOrDown(brokerIsUp),
			},
		})
	}
}

func statusName(everythingIsUp bool) string {
	if everythingIsUp {
		return "ok"
	}
	return "degraded"
}

func upOrDown(isUp bool) string {
	if isUp {
		return "up"
	}
	return "down"
}

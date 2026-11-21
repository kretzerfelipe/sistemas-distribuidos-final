package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPPort        = "8080"
	defaultLogLevel        = "info"
	defaultConsumerQueue   = "notification.q"
	defaultRetryQueue      = "notification.retry"
	defaultDeadLetterQueue = "notification.dlq"
	defaultMaxRetries      = 3
	defaultPrefetch        = 5
	defaultSnapshotSeconds = 300
	defaultStateTTLHours   = 24

	minJWTSecretLength = 32
)

var dotEnvCandidatePaths = []string{".env", "../../.env"}

type Config struct {
	HTTPPort    string
	LogLevel    string
	DatabaseURL string
	RedisURL    string

	RabbitMQURL     string
	ConsumerQueue   string
	RetryQueue      string
	DeadLetterQueue string
	MaxRetries      int
	Prefetch        int

	JWTSecret string

	SnapshotInterval time.Duration
	StateTTL         time.Duration

	PremiumUserIDs []string
	AdminIsPremium bool

	ContractsDirectory string
}

func Load() (Config, error) {
	loadDotEnv(dotEnvCandidatePaths...)

	maxRetries, err := getEnvInt("MAX_RETRIES", defaultMaxRetries)
	if err != nil {
		return Config{}, err
	}
	prefetch, err := getEnvInt("PREFETCH", defaultPrefetch)
	if err != nil {
		return Config{}, err
	}
	snapshotSeconds, err := getEnvInt("SNAPSHOT_INTERVAL_SECONDS", defaultSnapshotSeconds)
	if err != nil {
		return Config{}, err
	}
	stateTTLHours, err := getEnvInt("STATE_TTL_HOURS", defaultStateTTLHours)
	if err != nil {
		return Config{}, err
	}
	adminIsPremium, err := getEnvBool("ADMIN_IS_PREMIUM", true)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		HTTPPort:           firstNonEmpty(os.Getenv("NOTIFICATION_HTTP_PORT"), os.Getenv("HTTP_PORT"), defaultHTTPPort),
		LogLevel:           getEnv("LOG_LEVEL", defaultLogLevel),
		DatabaseURL:        firstNonEmpty(os.Getenv("NOTIFICATION_DATABASE_URL"), os.Getenv("DATABASE_URL")),
		RedisURL:           os.Getenv("REDIS_URL"),
		RabbitMQURL:        os.Getenv("RABBITMQ_URL"),
		ConsumerQueue:      getEnv("CONSUMER_QUEUE", defaultConsumerQueue),
		RetryQueue:         getEnv("RETRY_QUEUE", defaultRetryQueue),
		DeadLetterQueue:    getEnv("DEAD_LETTER_QUEUE", defaultDeadLetterQueue),
		MaxRetries:         maxRetries,
		Prefetch:           prefetch,
		JWTSecret:          os.Getenv("JWT_SECRET"),
		SnapshotInterval:   time.Duration(snapshotSeconds) * time.Second,
		StateTTL:           time.Duration(stateTTLHours) * time.Hour,
		PremiumUserIDs:     splitList(os.Getenv("PREMIUM_USER_IDS")),
		AdminIsPremium:     adminIsPremium,
		ContractsDirectory: os.Getenv("CONTRACTS_DIR"),
	}
	return cfg, cfg.validate()
}

func (c Config) validate() error {
	var errs []error
	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if c.RedisURL == "" {
		errs = append(errs, errors.New("REDIS_URL is required"))
	}
	if c.RabbitMQURL == "" {
		errs = append(errs, errors.New("RABBITMQ_URL is required"))
	}
	if len(c.JWTSecret) < minJWTSecretLength {
		errs = append(errs, fmt.Errorf("JWT_SECRET must have at least %d characters", minJWTSecretLength))
	}
	if c.MaxRetries < 0 {
		errs = append(errs, errors.New("MAX_RETRIES must not be negative"))
	}
	if c.Prefetch <= 0 {
		errs = append(errs, errors.New("PREFETCH must be greater than zero"))
	}
	if c.SnapshotInterval <= 0 {
		errs = append(errs, errors.New("SNAPSHOT_INTERVAL_SECONDS must be greater than zero"))
	}
	if c.StateTTL <= 0 {
		errs = append(errs, errors.New("STATE_TTL_HOURS must be greater than zero"))
	}
	return errors.Join(errs...)
}

func getEnv(key, fallback string) string {
	return firstNonEmpty(os.Getenv(key), fallback)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func getEnvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return number, nil
}

func getEnvBool(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false: %w", key, err)
	}
	return parsed, nil
}

func splitList(raw string) []string {
	var items []string
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

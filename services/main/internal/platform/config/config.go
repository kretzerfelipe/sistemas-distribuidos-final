package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultHTTPPort             = "8080"
	defaultLogLevel             = "info"
	defaultAdminName            = "Administrator"
	defaultJWTExpirationMinutes = 60
	defaultBcryptCost           = 12

	minJWTSecretLength = 32
	minBcryptCost      = 4
	maxBcryptCost      = 31
)

var dotEnvCandidatePaths = []string{".env", "../../.env"}

type Config struct {
	HTTPPort    string
	LogLevel    string
	DatabaseURL string

	JWTSecret     string
	JWTExpiration time.Duration
	BcryptCost    int

	AdminName     string
	AdminEmail    string
	AdminPassword string
}

func Load() (Config, error) {
	loadDotEnv(dotEnvCandidatePaths...)

	jwtExpirationMinutes, err := getEnvInt("JWT_EXPIRATION_MINUTES", defaultJWTExpirationMinutes)
	if err != nil {
		return Config{}, err
	}
	bcryptCost, err := getEnvInt("BCRYPT_COST", defaultBcryptCost)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		HTTPPort:      getEnv("HTTP_PORT", defaultHTTPPort),
		LogLevel:      getEnv("LOG_LEVEL", defaultLogLevel),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		JWTExpiration: time.Duration(jwtExpirationMinutes) * time.Minute,
		BcryptCost:    bcryptCost,
		AdminName:     getEnv("ADMIN_NAME", defaultAdminName),
		AdminEmail:    os.Getenv("ADMIN_EMAIL"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
	}
	return cfg, cfg.validate()
}

func (c Config) validate() error {
	databaseURLIsMissing := c.DatabaseURL == ""
	jwtSecretIsTooShort := len(c.JWTSecret) < minJWTSecretLength
	jwtExpirationIsNotPositive := c.JWTExpiration <= 0
	bcryptCostIsOutOfRange := c.BcryptCost < minBcryptCost || c.BcryptCost > maxBcryptCost
	adminPasswordIsMissing := c.AdminEmail != "" && c.AdminPassword == ""

	var errs []error
	if databaseURLIsMissing {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if jwtSecretIsTooShort {
		errs = append(errs, fmt.Errorf("JWT_SECRET must have at least %d characters", minJWTSecretLength))
	}
	if jwtExpirationIsNotPositive {
		errs = append(errs, errors.New("JWT_EXPIRATION_MINUTES must be greater than zero"))
	}
	if bcryptCostIsOutOfRange {
		errs = append(errs, fmt.Errorf("BCRYPT_COST must be between %d and %d", minBcryptCost, maxBcryptCost))
	}
	if adminPasswordIsMissing {
		errs = append(errs, errors.New("ADMIN_PASSWORD is required when ADMIN_EMAIL is set"))
	}
	return errors.Join(errs...)
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	valueIsMissing := value == ""
	if valueIsMissing {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	valueIsMissing := value == ""
	if valueIsMissing {
		return fallback, nil
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return number, nil
}

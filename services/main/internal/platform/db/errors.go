package db

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	uniqueViolationCode     = "23505"
	foreignKeyViolationCode = "23503"
	checkViolationCode      = "23514"
)

func IsUniqueViolation(err error) bool {
	return hasPgCode(err, uniqueViolationCode)
}

func IsForeignKeyViolation(err error) bool {
	return hasPgCode(err, foreignKeyViolationCode)
}

func IsCheckViolation(err error) bool {
	return hasPgCode(err, checkViolationCode)
}

func hasPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

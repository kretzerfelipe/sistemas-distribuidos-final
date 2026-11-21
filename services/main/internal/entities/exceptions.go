package entities

import "futhub/main/internal/platform/util"

var (
	ErrInvalidInput       = util.NewBusinessError("invalid input")
	ErrInvalidCredentials = util.NewBusinessError("invalid email or password")
	ErrNotFound           = util.NewBusinessError("not found")
	ErrEmailTaken         = util.NewBusinessError("email already registered")
)

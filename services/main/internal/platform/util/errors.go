package util

import (
	"errors"
	"fmt"
	"log/slog"
)

type BusinessError struct {
	message string
}

func NewBusinessError(message string) *BusinessError {
	return &BusinessError{message: message}
}

func (e *BusinessError) Error() string {
	return e.message
}

func SendMessageWithError(message string, err error) error {
	if err == nil {
		return nil
	}

	isBusinessError := errors.As(err, new(*BusinessError))
	if isBusinessError {
		return err
	}

	slog.Error(message, "error", err)
	return fmt.Errorf("%s: %w", message, err)
}

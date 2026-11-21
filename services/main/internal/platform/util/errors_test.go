package util

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	output := &bytes.Buffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return output
}

func TestSendMessageWithErrorReturnsNilWhenThereIsNoError(t *testing.T) {
	logs := captureLogs(t)

	err := SendMessageWithError("something failed", nil)

	if err != nil {
		t.Errorf("want nil, got %v", err)
	}
	if logs.Len() != 0 {
		t.Errorf("nothing should be logged, got %q", logs.String())
	}
}

func TestSendMessageWithErrorLogsAndWrapsUnexpectedErrors(t *testing.T) {
	logs := captureLogs(t)
	databaseFailure := errors.New("connection refused")

	err := SendMessageWithError("repository error inserting user", databaseFailure)

	if !errors.Is(err, databaseFailure) {
		t.Error("the original error must stay reachable with errors.Is")
	}
	if !strings.Contains(err.Error(), "repository error inserting user") {
		t.Errorf("error should carry the message, got %q", err.Error())
	}
	loggedMessageAndCause := strings.Contains(logs.String(), "repository error inserting user") &&
		strings.Contains(logs.String(), "connection refused")
	if !loggedMessageAndCause {
		t.Errorf("unexpected errors must be logged with message and cause, got %q", logs.String())
	}
}

func TestSendMessageWithErrorPassesBusinessErrorsThroughWithoutLogging(t *testing.T) {
	logs := captureLogs(t)
	errNotAllowed := NewBusinessError("not allowed")
	detailedBusinessError := fmt.Errorf("%w: reason", errNotAllowed)

	returned := SendMessageWithError("use case error", detailedBusinessError)

	if returned != detailedBusinessError {
		t.Errorf("business errors must be returned untouched, got %v", returned)
	}
	if !errors.Is(returned, errNotAllowed) {
		t.Error("business error must stay reachable with errors.Is")
	}
	if logs.Len() != 0 {
		t.Errorf("business errors are expected outcomes and must not be logged, got %q", logs.String())
	}
}

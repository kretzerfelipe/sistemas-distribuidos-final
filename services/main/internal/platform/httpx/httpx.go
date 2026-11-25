package httpx

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const CorrelationHeader = "X-Correlation-Id"

type ctxKey int

const correlationKey ctxKey = iota

func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get(CorrelationHeader)
		correlationIDWasNotProvided := correlationID == ""
		if correlationIDWasNotProvided {
			correlationID = uuid.NewString()
		}
		w.Header().Set(CorrelationHeader, correlationID)
		ctx := context.WithValue(r.Context(), correlationKey, correlationID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationKey, correlationID)
}

func CorrelationIDFrom(ctx context.Context) string {
	correlationID, _ := ctx.Value(correlationKey).(string)
	return correlationID
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(recorder, r)

			log.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"duration_ms", time.Since(startedAt).Milliseconds(),
				"correlationId", CorrelationIDFrom(r.Context()),
			)
		})
	}
}

func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				panicValue := recover()
				panicHappened := panicValue != nil
				if panicHappened {
					log.Error("panic recovered", "panic", panicValue, "correlationId", CorrelationIDFrom(r.Context()))
					WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	bodyIsPresent := body != nil
	if bodyIsPresent {
		_ = json.NewEncoder(w).Encode(body)
	}
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

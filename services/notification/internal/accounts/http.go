package accounts

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"futhub/notification/internal/httpx"
	"futhub/notification/internal/security"
)

const listLimit = 50

type Handler struct {
	store  Store
	tokens *security.TokenParser
	log    *slog.Logger
}

func NewHandler(store Store, tokens *security.TokenParser, log *slog.Logger) *Handler {
	return &Handler{store: store, tokens: tokens, log: log}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	authenticated := router.With(security.Authenticate(h.tokens, false))
	authenticated.Get("/notifications", h.list)
	authenticated.Post("/notifications/{id}/read", h.markRead)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, valid := callerID(w, r)
	if !valid {
		return
	}

	notifications, err := h.store.ListNotifications(r.Context(), userID, listLimit)
	if err != nil {
		h.log.Error("could not list notifications", "error", err, "correlationId", httpx.CorrelationIDFrom(r.Context()))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, notifications)
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	userID, valid := callerID(w, r)
	if !valid {
		return
	}
	notificationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_input", "notification id must be a valid UUID")
		return
	}

	found, err := h.store.MarkRead(r.Context(), userID, notificationID)
	if err != nil {
		h.log.Error("could not mark notification as read", "error", err, "correlationId", httpx.CorrelationIDFrom(r.Context()))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	if !found {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "notification not found")
		return
	}
	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

func callerID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	principal, authenticated := security.PrincipalFromContext(r.Context())
	if !authenticated {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid user in the token")
		return uuid.Nil, false
	}
	return userID, true
}

package live

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"futhub/notification/internal/httpx"
	"futhub/notification/internal/security"
)

const (
	writeTimeout = 10 * time.Second
	pongTimeout  = 60 * time.Second
	pingInterval = 30 * time.Second
	readLimit    = 1024

	matchNotLiveMessage = "this match is not live; finished results are available at GET /matches/{id} and GET /matches/{id}/events on the main service"
)

type PremiumChecker interface {
	IsPremium(ctx context.Context, principal security.Principal) (bool, error)
}

type Handler struct {
	store    Store
	hub      *Hub
	premium  PremiumChecker
	tokens   *security.TokenParser
	shutdown context.Context
	upgrader websocket.Upgrader
	log      *slog.Logger
}

func NewHandler(shutdown context.Context, store Store, hub *Hub, premium PremiumChecker, tokens *security.TokenParser, log *slog.Logger) *Handler {
	return &Handler{
		store:    store,
		hub:      hub,
		premium:  premium,
		tokens:   tokens,
		shutdown: shutdown,
		upgrader: websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }},
		log:      log,
	}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.With(security.Authenticate(h.tokens, true)).Get("/matches/{id}/stream", h.stream)
	router.With(security.Authenticate(h.tokens, true)).Get("/matches/{id}/summary", h.summary)
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	matchID, valid := matchIDFrom(w, r)
	if !valid {
		return
	}

	finished, err := h.store.WasFinished(r.Context(), matchID)
	if err != nil {
		h.cacheUnavailable(w, r, err)
		return
	}
	summary, err := h.store.LoadSummary(r.Context(), matchID)
	if err != nil {
		h.cacheUnavailable(w, r, err)
		return
	}
	if finished || summary == nil {
		httpx.WriteError(w, http.StatusNotFound, "match_not_live", matchNotLiveMessage)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, summary)
}

func (h *Handler) stream(w http.ResponseWriter, r *http.Request) {
	matchID, valid := matchIDFrom(w, r)
	if !valid {
		return
	}
	principal, _ := security.PrincipalFromContext(r.Context())

	isPremium, err := h.premium.IsPremium(r.Context(), principal)
	if err != nil {
		h.log.Error("could not check premium status", "userId", principal.UserID, "error", err, "correlationId", httpx.CorrelationIDFrom(r.Context()))
		httpx.WriteError(w, http.StatusServiceUnavailable, "premium_unavailable", "could not check the subscription right now, try again")
		return
	}
	if !isPremium {
		httpx.WriteError(w, http.StatusForbidden, "premium_required", "real time updates are available only for premium subscribers; use GET /matches/{id}/summary")
		return
	}

	subscription := h.hub.Subscribe(matchID)
	defer subscription.Close()

	state, err := h.store.LoadState(r.Context(), matchID)
	if err != nil {
		h.cacheUnavailable(w, r, err)
		return
	}
	if state == nil {
		httpx.WriteError(w, http.StatusNotFound, "match_not_live", matchNotLiveMessage)
		return
	}

	connection, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer connection.Close()

	h.log.Info("stream opened", "matchId", matchID, "userId", principal.UserID, "correlationId", httpx.CorrelationIDFrom(r.Context()))
	h.serve(connection, subscription, state)
	h.log.Info("stream closed", "matchId", matchID, "userId", principal.UserID)
}

func (h *Handler) serve(connection *websocket.Conn, subscription *Subscription, state *MatchState) {
	connection.SetReadLimit(readLimit)
	_ = connection.SetReadDeadline(time.Now().Add(pongTimeout))
	connection.SetPongHandler(func(string) error {
		return connection.SetReadDeadline(time.Now().Add(pongTimeout))
	})

	clientGone := make(chan struct{})
	go func() {
		defer close(clientGone)
		for {
			if _, _, err := connection.ReadMessage(); err != nil {
				return
			}
		}
	}()

	payload, err := json.Marshal(state)
	if err != nil {
		return
	}
	initial := Update{Type: UpdateMatchState, MatchID: state.MatchID, Sequence: state.Sequence, Payload: payload}
	if err := writeUpdate(connection, initial); err != nil {
		return
	}

	lastSent := state.Sequence
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-clientGone:
			return
		case <-h.shutdown.Done():
			closeWith(connection, websocket.CloseGoingAway, "service is shutting down")
			return
		case <-ticker.C:
			_ = connection.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case update, open := <-subscription.Updates:
			if !open {
				closeWith(connection, websocket.CloseTryAgainLater, "client too slow, reconnect to resynchronize")
				return
			}
			alreadySentInInitialState := update.Sequence <= lastSent
			if alreadySentInInitialState {
				continue
			}
			if err := writeUpdate(connection, update); err != nil {
				return
			}
			lastSent = update.Sequence
			if update.Final {
				closeWith(connection, websocket.CloseNormalClosure, "match finished")
				return
			}
		}
	}
}

func (h *Handler) cacheUnavailable(w http.ResponseWriter, r *http.Request, err error) {
	h.log.Error("live cache unavailable", "error", err, "correlationId", httpx.CorrelationIDFrom(r.Context()))
	if errors.Is(err, ErrCacheUnavailable) {
		httpx.WriteError(w, http.StatusServiceUnavailable, "cache_unavailable", "live match data is temporarily unavailable, try again shortly")
		return
	}
	httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
}

func matchIDFrom(w http.ResponseWriter, r *http.Request) (string, bool) {
	matchID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_input", "match id must be a valid UUID")
		return "", false
	}
	return matchID.String(), true
}

func writeUpdate(connection *websocket.Conn, update Update) error {
	_ = connection.SetWriteDeadline(time.Now().Add(writeTimeout))
	return connection.WriteJSON(update)
}

func closeWith(connection *websocket.Conn, code int, reason string) {
	_ = connection.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(writeTimeout))
}

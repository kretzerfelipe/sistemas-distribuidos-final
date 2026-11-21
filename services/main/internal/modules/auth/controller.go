package auth

import (
	"errors"
	"net/http"

	"futhub/main/internal/entities"
	"futhub/main/internal/modules/users"
	"futhub/main/internal/platform/httpx"
	"futhub/main/internal/platform/routing"
	"futhub/main/internal/platform/security"
)

type controller struct {
	uc UseCase
}

func newController(uc UseCase) *controller {
	return &controller{uc: uc}
}

func (c *controller) routes() []routing.Route {
	return []routing.Route{
		{Method: http.MethodPost, Path: "/register", Handler: c.register, Auth: routing.Public},
		{Method: http.MethodPost, Path: "/login", Handler: c.login, Auth: routing.Public},
		{Method: http.MethodGet, Path: "/me", Handler: c.me, Auth: routing.Authenticated()},
	}
}

func (c *controller) register(w http.ResponseWriter, r *http.Request) {
	input, err := httpx.ReadBody[entities.RegisterInput](w, r)
	if err != nil {
		writeError(w, err)
		return
	}

	user, err := c.uc.Register(r.Context(), *input)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, users.NewUserResponse(user))
}

func (c *controller) login(w http.ResponseWriter, r *http.Request) {
	input, err := httpx.ReadBody[entities.LoginInput](w, r)
	if err != nil {
		writeError(w, err)
		return
	}

	result, err := c.uc.Login(r.Context(), *input)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (c *controller) me(w http.ResponseWriter, r *http.Request) {
	caller, isAuthenticated := security.PrincipalFromContext(r.Context())
	if !isAuthenticated {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	user, err := c.uc.Me(r.Context(), caller.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, users.NewUserResponse(user))
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, httpx.ErrInvalidRequest):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, entities.ErrInvalidCredentials):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", err.Error())
	case errors.Is(err, entities.ErrInvalidInput):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
	case errors.Is(err, entities.ErrEmailTaken):
		httpx.WriteError(w, http.StatusConflict, "email_taken", err.Error())
	case errors.Is(err, entities.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

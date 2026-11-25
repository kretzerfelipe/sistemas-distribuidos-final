package auth

import (
	"net/http"

	"futhub/main/internal/api"
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
		{Method: http.MethodPost, Path: "/users", Handler: c.registerByAdmin, Auth: routing.Roles(entities.RoleAdmin)},
		{Method: http.MethodPost, Path: "/login", Handler: c.login, Auth: routing.Public},
		{Method: http.MethodGet, Path: "/me", Handler: c.me, Auth: routing.Authenticated()},
	}
}

func (c *controller) register(w http.ResponseWriter, r *http.Request) {
	input, err := httpx.ReadBody[entities.RegisterInput](w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	user, err := c.uc.Register(r.Context(), *input)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, users.NewUserResponse(user))
}

func (c *controller) registerByAdmin(w http.ResponseWriter, r *http.Request) {
	input, err := httpx.ReadBody[entities.RegisterInput](w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	user, err := c.uc.RegisterByAdmin(r.Context(), *input)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, users.NewUserResponse(user))
}

func (c *controller) login(w http.ResponseWriter, r *http.Request) {
	input, err := httpx.ReadBody[entities.LoginInput](w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	result, err := c.uc.Login(r.Context(), *input)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (c *controller) me(w http.ResponseWriter, r *http.Request) {
	principal, isAuthenticated := security.PrincipalFromContext(r.Context())
	if !isAuthenticated {
		api.WriteError(w, entities.ErrUnauthenticated)
		return
	}

	user, err := c.uc.Me(r.Context(), principal.UserID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, users.NewUserResponse(user))
}

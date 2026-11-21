package users

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
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
		{Method: http.MethodGet, Path: "/", Handler: c.list, Auth: routing.Roles(entities.RoleAdmin)},
		{Method: http.MethodGet, Path: "/{id}", Handler: c.getByID, Auth: routing.Authenticated()},
	}
}

func (c *controller) list(w http.ResponseWriter, r *http.Request) {
	allUsers, err := c.uc.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	responses := make([]entities.UserResponse, 0, len(allUsers))
	for i := range allUsers {
		responses = append(responses, NewUserResponse(&allUsers[i]))
	}
	httpx.WriteJSON(w, http.StatusOK, responses)
}

func (c *controller) getByID(w http.ResponseWriter, r *http.Request) {
	requestedID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		writeError(w, err)
		return
	}

	caller, _ := security.PrincipalFromContext(r.Context())
	callerIsAdmin := caller.Role == string(entities.RoleAdmin)
	callerIsRequestedUser := caller.UserID == requestedID.String()
	callerCanAccessUser := callerIsAdmin || callerIsRequestedUser
	if !callerCanAccessUser {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "you can only access your own user")
		return
	}

	user, err := c.uc.GetByID(r.Context(), requestedID)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewUserResponse(user))
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, httpx.ErrInvalidRequest):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, entities.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, entities.ErrEmailTaken):
		httpx.WriteError(w, http.StatusConflict, "email_taken", err.Error())
	case errors.Is(err, entities.ErrInvalidInput):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

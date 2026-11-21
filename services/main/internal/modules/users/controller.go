package users

import (
	"net/http"

	"github.com/google/uuid"

	"futhub/main/internal/api"
	"futhub/main/internal/entities"
	"futhub/main/internal/platform/httpx"
	"futhub/main/internal/platform/routing"
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
		{Method: http.MethodPatch, Path: "/{id}/role", Handler: c.updateRole, Auth: routing.Roles(entities.RoleAdmin)},
	}
}

func (c *controller) list(w http.ResponseWriter, r *http.Request) {
	allUsers, err := c.uc.List(r.Context())
	if err != nil {
		api.WriteError(w, err)
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
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	callerIsAdmin := caller.Role == entities.RoleAdmin
	callerIsRequestedUser := caller.UserID == requestedID
	callerCanAccessUser := callerIsAdmin || callerIsRequestedUser
	if !callerCanAccessUser {
		api.WriteError(w, entities.ErrForbidden)
		return
	}

	user, err := c.uc.GetByID(r.Context(), requestedID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewUserResponse(user))
}

func (c *controller) updateRole(w http.ResponseWriter, r *http.Request) {
	userID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	input, err := httpx.ReadBody[entities.UpdateRoleInput](w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	user, err := c.uc.UpdateRole(r.Context(), caller, userID, input.Role)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewUserResponse(user))
}

package players

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
		{Method: http.MethodPost, Path: "/", Handler: c.create, Auth: routing.Roles(entities.RoleUser)},
		{Method: http.MethodGet, Path: "/me", Handler: c.getMine, Auth: routing.Roles(entities.RoleUser)},
		{Method: http.MethodPut, Path: "/me", Handler: c.updateMine, Auth: routing.Roles(entities.RoleUser)},
		{Method: http.MethodDelete, Path: "/me", Handler: c.deleteMine, Auth: routing.Roles(entities.RoleUser)},
		{Method: http.MethodGet, Path: "/", Handler: c.list, Auth: routing.Roles(entities.RoleTeam, entities.RoleAdmin)},
		{Method: http.MethodGet, Path: "/{id}", Handler: c.getByID, Auth: routing.Roles(entities.RoleTeam, entities.RoleAdmin)},
		{Method: http.MethodDelete, Path: "/{id}", Handler: c.deleteByID, Auth: routing.Roles(entities.RoleAdmin)},
	}
}

func (c *controller) create(w http.ResponseWriter, r *http.Request) {
	input, err := httpx.ReadBody[entities.SavePlayerInput](w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	player, err := c.uc.Create(r.Context(), caller, *input)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, NewPlayerResponse(player))
}

func (c *controller) getMine(w http.ResponseWriter, r *http.Request) {
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	player, err := c.uc.GetMine(r.Context(), caller)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewPlayerResponse(player))
}

func (c *controller) updateMine(w http.ResponseWriter, r *http.Request) {
	input, err := httpx.ReadBody[entities.SavePlayerInput](w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	player, err := c.uc.UpdateMine(r.Context(), caller, *input)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewPlayerResponse(player))
}

func (c *controller) deleteMine(w http.ResponseWriter, r *http.Request) {
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	err = c.uc.DeleteMine(r.Context(), caller)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

func (c *controller) list(w http.ResponseWriter, r *http.Request) {
	allPlayers, err := c.uc.List(r.Context())
	if err != nil {
		api.WriteError(w, err)
		return
	}

	responses := make([]entities.PlayerResponse, 0, len(allPlayers))
	for i := range allPlayers {
		responses = append(responses, NewPlayerResponse(&allPlayers[i]))
	}
	httpx.WriteJSON(w, http.StatusOK, responses)
}

func (c *controller) getByID(w http.ResponseWriter, r *http.Request) {
	playerID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	player, err := c.uc.GetByID(r.Context(), playerID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewPlayerResponse(player))
}

func (c *controller) deleteByID(w http.ResponseWriter, r *http.Request) {
	playerID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	err = c.uc.Delete(r.Context(), playerID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

package matches

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
	adminOnly := routing.Roles(entities.RoleAdmin)

	return []routing.Route{
		{Method: http.MethodPost, Path: "/matches", Handler: c.create, Auth: adminOnly},
		{Method: http.MethodGet, Path: "/matches/{id}", Handler: c.getByID, Auth: routing.Authenticated()},
		{Method: http.MethodGet, Path: "/championships/{id}/matches", Handler: c.listByChampionship, Auth: routing.Authenticated()},
		{Method: http.MethodPut, Path: "/matches/{id}", Handler: c.reschedule, Auth: adminOnly},
		{Method: http.MethodDelete, Path: "/matches/{id}", Handler: c.delete, Auth: adminOnly},
		{Method: http.MethodPost, Path: "/matches/{id}/start", Handler: c.start, Auth: adminOnly},
		{Method: http.MethodPost, Path: "/matches/{id}/finish", Handler: c.finish, Auth: adminOnly},

		{Method: http.MethodPost, Path: "/matches/{id}/events", Handler: c.registerEvent, Auth: adminOnly},
		{Method: http.MethodGet, Path: "/matches/{id}/events", Handler: c.listEvents, Auth: routing.Authenticated()},
		{Method: http.MethodDelete, Path: "/matches/{id}/events/{eventId}", Handler: c.removeEvent, Auth: adminOnly},
	}
}

func (c *controller) create(w http.ResponseWriter, r *http.Request) {
	input, err := httpx.ReadBody[entities.CreateMatchInput](w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	match, err := c.uc.Create(r.Context(), *input)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, NewMatchResponse(match))
}

func (c *controller) getByID(w http.ResponseWriter, r *http.Request) {
	matchID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	match, err := c.uc.GetByID(r.Context(), matchID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewMatchResponse(match))
}

func (c *controller) listByChampionship(w http.ResponseWriter, r *http.Request) {
	championshipID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	championshipMatches, err := c.uc.ListByChampionship(r.Context(), championshipID)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	responses := make([]entities.MatchResponse, 0, len(championshipMatches))
	for i := range championshipMatches {
		responses = append(responses, NewMatchResponse(&championshipMatches[i]))
	}
	httpx.WriteJSON(w, http.StatusOK, responses)
}

func (c *controller) reschedule(w http.ResponseWriter, r *http.Request) {
	matchID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	input, err := httpx.ReadBody[entities.RescheduleMatchInput](w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	match, err := c.uc.Reschedule(r.Context(), matchID, *input)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewMatchResponse(match))
}

func (c *controller) delete(w http.ResponseWriter, r *http.Request) {
	matchID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	err = c.uc.Delete(r.Context(), matchID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

func (c *controller) start(w http.ResponseWriter, r *http.Request) {
	matchID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	match, err := c.uc.Start(r.Context(), matchID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewMatchResponse(match))
}

func (c *controller) finish(w http.ResponseWriter, r *http.Request) {
	matchID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	match, err := c.uc.Finish(r.Context(), matchID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewMatchResponse(match))
}

func (c *controller) registerEvent(w http.ResponseWriter, r *http.Request) {
	matchID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	input, err := httpx.ReadBody[entities.RegisterMatchEventInput](w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	event, err := c.uc.RegisterEvent(r.Context(), matchID, *input)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, NewMatchEventResponse(event))
}

func (c *controller) listEvents(w http.ResponseWriter, r *http.Request) {
	matchID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	matchEvents, err := c.uc.ListEvents(r.Context(), matchID)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	responses := make([]entities.MatchEventResponse, 0, len(matchEvents))
	for i := range matchEvents {
		responses = append(responses, NewMatchEventResponse(&matchEvents[i]))
	}
	httpx.WriteJSON(w, http.StatusOK, responses)
}

func (c *controller) removeEvent(w http.ResponseWriter, r *http.Request) {
	matchID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	eventID, err := httpx.GetVariable[uuid.UUID](r, "eventId")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	err = c.uc.RemoveEvent(r.Context(), matchID, eventID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

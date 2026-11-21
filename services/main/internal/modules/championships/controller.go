package championships

import (
	"net/http"

	"github.com/google/uuid"

	"futhub/main/internal/api"
	"futhub/main/internal/entities"
	"futhub/main/internal/modules/teams"
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
	manageTeams := routing.Roles(entities.RoleTeam, entities.RoleAdmin)

	return []routing.Route{
		{Method: http.MethodPost, Path: "/", Handler: c.create, Auth: adminOnly},
		{Method: http.MethodGet, Path: "/", Handler: c.list, Auth: routing.Authenticated()},
		{Method: http.MethodGet, Path: "/{id}", Handler: c.getByID, Auth: routing.Authenticated()},
		{Method: http.MethodPut, Path: "/{id}", Handler: c.update, Auth: adminOnly},
		{Method: http.MethodDelete, Path: "/{id}", Handler: c.delete, Auth: adminOnly},
		{Method: http.MethodPost, Path: "/{id}/start", Handler: c.start, Auth: adminOnly},
		{Method: http.MethodPost, Path: "/{id}/finish", Handler: c.finish, Auth: adminOnly},

		{Method: http.MethodPost, Path: "/{id}/teams", Handler: c.enroll, Auth: manageTeams},
		{Method: http.MethodGet, Path: "/{id}/teams", Handler: c.listTeams, Auth: routing.Authenticated()},
		{Method: http.MethodDelete, Path: "/{id}/teams/{teamId}", Handler: c.unenroll, Auth: manageTeams},
	}
}

func (c *controller) create(w http.ResponseWriter, r *http.Request) {
	input, err := httpx.ReadBody[entities.SaveChampionshipInput](w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	championship, err := c.uc.Create(r.Context(), *input)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, NewChampionshipResponse(championship))
}

func (c *controller) list(w http.ResponseWriter, r *http.Request) {
	allChampionships, err := c.uc.List(r.Context())
	if err != nil {
		api.WriteError(w, err)
		return
	}

	responses := make([]entities.ChampionshipResponse, 0, len(allChampionships))
	for i := range allChampionships {
		responses = append(responses, NewChampionshipResponse(&allChampionships[i]))
	}
	httpx.WriteJSON(w, http.StatusOK, responses)
}

func (c *controller) getByID(w http.ResponseWriter, r *http.Request) {
	championshipID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	championship, err := c.uc.GetByID(r.Context(), championshipID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewChampionshipResponse(championship))
}

func (c *controller) update(w http.ResponseWriter, r *http.Request) {
	championshipID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	input, err := httpx.ReadBody[entities.SaveChampionshipInput](w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	championship, err := c.uc.Update(r.Context(), championshipID, *input)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewChampionshipResponse(championship))
}

func (c *controller) delete(w http.ResponseWriter, r *http.Request) {
	championshipID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	err = c.uc.Delete(r.Context(), championshipID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

func (c *controller) start(w http.ResponseWriter, r *http.Request) {
	championshipID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	championship, err := c.uc.Start(r.Context(), championshipID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewChampionshipResponse(championship))
}

func (c *controller) finish(w http.ResponseWriter, r *http.Request) {
	championshipID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	championship, err := c.uc.Finish(r.Context(), championshipID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewChampionshipResponse(championship))
}

func (c *controller) enroll(w http.ResponseWriter, r *http.Request) {
	championshipID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	input, err := httpx.ReadBody[entities.EnrollTeamInput](w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	championship, err := c.uc.Enroll(r.Context(), caller, championshipID, input.TeamID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, NewChampionshipResponse(championship))
}

func (c *controller) listTeams(w http.ResponseWriter, r *http.Request) {
	championshipID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	enrolledTeams, err := c.uc.ListTeams(r.Context(), championshipID)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	responses := make([]entities.TeamResponse, 0, len(enrolledTeams))
	for i := range enrolledTeams {
		responses = append(responses, teams.NewTeamResponse(&enrolledTeams[i]))
	}
	httpx.WriteJSON(w, http.StatusOK, responses)
}

func (c *controller) unenroll(w http.ResponseWriter, r *http.Request) {
	championshipID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	teamID, err := httpx.GetVariable[uuid.UUID](r, "teamId")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	err = c.uc.Unenroll(r.Context(), caller, championshipID, teamID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

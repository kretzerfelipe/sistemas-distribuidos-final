package teams

import (
	"net/http"

	"github.com/google/uuid"

	"futhub/main/internal/api"
	"futhub/main/internal/entities"
	"futhub/main/internal/modules/players"
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
	manageTeams := routing.Roles(entities.RoleTeam, entities.RoleAdmin)

	return []routing.Route{
		{Method: http.MethodPost, Path: "/teams", Handler: c.create, Auth: routing.Roles(entities.RoleTeam)},
		{Method: http.MethodGet, Path: "/teams", Handler: c.list, Auth: routing.Authenticated()},
		{Method: http.MethodGet, Path: "/teams/mine", Handler: c.listMine, Auth: routing.Roles(entities.RoleTeam)},
		{Method: http.MethodGet, Path: "/teams/{id}", Handler: c.getByID, Auth: routing.Authenticated()},
		{Method: http.MethodPut, Path: "/teams/{id}", Handler: c.update, Auth: manageTeams},
		{Method: http.MethodDelete, Path: "/teams/{id}", Handler: c.delete, Auth: manageTeams},

		{Method: http.MethodGet, Path: "/teams/{id}/players", Handler: c.listPlayers, Auth: routing.Authenticated()},
		{Method: http.MethodDelete, Path: "/teams/{id}/players/{playerId}", Handler: c.removePlayer, Auth: manageTeams},
		{Method: http.MethodDelete, Path: "/players/me/team", Handler: c.leaveTeam, Auth: routing.Roles(entities.RoleUser)},

		{Method: http.MethodPost, Path: "/teams/{id}/invitations", Handler: c.invite, Auth: routing.Roles(entities.RoleTeam)},
		{Method: http.MethodGet, Path: "/teams/{id}/invitations", Handler: c.listTeamInvitations, Auth: manageTeams},
		{Method: http.MethodDelete, Path: "/teams/{id}/invitations/{invitationId}", Handler: c.cancelInvitation, Auth: manageTeams},

		{Method: http.MethodGet, Path: "/invitations", Handler: c.listMyInvitations, Auth: routing.Roles(entities.RoleUser)},
		{Method: http.MethodPost, Path: "/invitations/{id}/accept", Handler: c.acceptInvitation, Auth: routing.Roles(entities.RoleUser)},
		{Method: http.MethodPost, Path: "/invitations/{id}/decline", Handler: c.declineInvitation, Auth: routing.Roles(entities.RoleUser)},
	}
}

func (c *controller) create(w http.ResponseWriter, r *http.Request) {
	input, err := httpx.ReadBody[entities.SaveTeamInput](w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	team, err := c.uc.Create(r.Context(), caller, *input)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, NewTeamResponse(team))
}

func (c *controller) list(w http.ResponseWriter, r *http.Request) {
	allTeams, err := c.uc.List(r.Context())
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newTeamResponses(allTeams))
}

func (c *controller) listMine(w http.ResponseWriter, r *http.Request) {
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	ownedTeams, err := c.uc.ListMine(r.Context(), caller)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newTeamResponses(ownedTeams))
}

func (c *controller) getByID(w http.ResponseWriter, r *http.Request) {
	teamID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	team, err := c.uc.GetByID(r.Context(), teamID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewTeamResponse(team))
}

func (c *controller) update(w http.ResponseWriter, r *http.Request) {
	teamID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	input, err := httpx.ReadBody[entities.SaveTeamInput](w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	team, err := c.uc.Update(r.Context(), caller, teamID, *input)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewTeamResponse(team))
}

func (c *controller) delete(w http.ResponseWriter, r *http.Request) {
	teamID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	err = c.uc.Delete(r.Context(), caller, teamID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

func (c *controller) listPlayers(w http.ResponseWriter, r *http.Request) {
	teamID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}

	roster, err := c.uc.ListPlayers(r.Context(), teamID)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	responses := make([]entities.PlayerResponse, 0, len(roster))
	for i := range roster {
		responses = append(responses, players.NewPlayerResponse(&roster[i]))
	}
	httpx.WriteJSON(w, http.StatusOK, responses)
}

func (c *controller) removePlayer(w http.ResponseWriter, r *http.Request) {
	teamID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	playerID, err := httpx.GetVariable[uuid.UUID](r, "playerId")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	err = c.uc.RemovePlayer(r.Context(), caller, teamID, playerID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

func (c *controller) leaveTeam(w http.ResponseWriter, r *http.Request) {
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	err = c.uc.LeaveTeam(r.Context(), caller)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

func (c *controller) invite(w http.ResponseWriter, r *http.Request) {
	teamID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	input, err := httpx.ReadBody[entities.InvitePlayerInput](w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	invitation, err := c.uc.Invite(r.Context(), caller, teamID, input.PlayerID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, NewInvitationResponse(invitation))
}

func (c *controller) listTeamInvitations(w http.ResponseWriter, r *http.Request) {
	teamID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	invitations, err := c.uc.ListTeamInvitations(r.Context(), caller, teamID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newInvitationResponses(invitations))
}

func (c *controller) cancelInvitation(w http.ResponseWriter, r *http.Request) {
	teamID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	invitationID, err := httpx.GetVariable[uuid.UUID](r, "invitationId")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	invitation, err := c.uc.CancelInvitation(r.Context(), caller, teamID, invitationID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewInvitationResponse(invitation))
}

func (c *controller) listMyInvitations(w http.ResponseWriter, r *http.Request) {
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	invitations, err := c.uc.ListMyInvitations(r.Context(), caller)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newInvitationResponses(invitations))
}

func (c *controller) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	invitationID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	invitation, err := c.uc.AcceptInvitation(r.Context(), caller, invitationID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewInvitationResponse(invitation))
}

func (c *controller) declineInvitation(w http.ResponseWriter, r *http.Request) {
	invitationID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	invitation, err := c.uc.DeclineInvitation(r.Context(), caller, invitationID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewInvitationResponse(invitation))
}

func newTeamResponses(allTeams []entities.Team) []entities.TeamResponse {
	responses := make([]entities.TeamResponse, 0, len(allTeams))
	for i := range allTeams {
		responses = append(responses, NewTeamResponse(&allTeams[i]))
	}
	return responses
}

func newInvitationResponses(invitations []entities.Invitation) []entities.InvitationResponse {
	responses := make([]entities.InvitationResponse, 0, len(invitations))
	for i := range invitations {
		responses = append(responses, NewInvitationResponse(&invitations[i]))
	}
	return responses
}

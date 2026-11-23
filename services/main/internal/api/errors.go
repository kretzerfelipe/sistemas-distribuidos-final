package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/httpx"
	"futhub/main/internal/platform/security"
)

type errorMapping struct {
	target error
	status int
	code   string
}

var errorMappings = []errorMapping{
	{httpx.ErrInvalidRequest, http.StatusBadRequest, "invalid_request"},
	{entities.ErrUnauthenticated, http.StatusUnauthorized, "unauthorized"},
	{entities.ErrInvalidCredentials, http.StatusUnauthorized, "invalid_credentials"},
	{entities.ErrForbidden, http.StatusForbidden, "forbidden"},
	{entities.ErrNotFound, http.StatusNotFound, "not_found"},
	{entities.ErrPlayerNotInTeam, http.StatusNotFound, "player_not_in_team"},
	{entities.ErrInvalidInput, http.StatusUnprocessableEntity, "invalid_input"},
	{entities.ErrInvalidState, http.StatusConflict, "invalid_state"},
	{entities.ErrEmailTaken, http.StatusConflict, "email_taken"},
	{entities.ErrPlayerAlreadyExists, http.StatusConflict, "player_already_exists"},
	{entities.ErrTeamNameTaken, http.StatusConflict, "team_name_taken"},
	{entities.ErrTeamInUse, http.StatusConflict, "team_in_use"},
	{entities.ErrPlayerAlreadyInTeam, http.StatusConflict, "player_already_in_team"},
	{entities.ErrInvitationAlreadyPending, http.StatusConflict, "invitation_already_pending"},
	{entities.ErrInvitationNotPending, http.StatusConflict, "invitation_not_pending"},
	{entities.ErrChampionshipFull, http.StatusConflict, "championship_full"},
	{entities.ErrChampionshipNotOpen, http.StatusConflict, "championship_not_open"},
	{entities.ErrChampionshipHasMatches, http.StatusConflict, "championship_has_matches"},
	{entities.ErrTeamAlreadyEnrolled, http.StatusConflict, "team_already_enrolled"},
	{entities.ErrTeamNotEnrolled, http.StatusConflict, "team_not_enrolled"},
	{entities.ErrSubscriptionInForce, http.StatusConflict, "subscription_in_force"},
	{entities.ErrUnsupportedPhoto, http.StatusUnsupportedMediaType, "unsupported_photo"},
	{entities.ErrPhotoTooLarge, http.StatusRequestEntityTooLarge, "photo_too_large"},
	{entities.ErrStorageUnavailable, http.StatusServiceUnavailable, "storage_unavailable"},
}

func WriteError(w http.ResponseWriter, err error) {
	for _, mapping := range errorMappings {
		errorMatchesMapping := errors.Is(err, mapping.target)
		if errorMatchesMapping {
			httpx.WriteError(w, mapping.status, mapping.code, err.Error())
			return
		}
	}
	httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
}

func GetCaller(r *http.Request) (*entities.Caller, error) {
	principal, isAuthenticated := security.PrincipalFromContext(r.Context())
	if !isAuthenticated {
		return nil, entities.ErrUnauthenticated
	}

	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return nil, entities.ErrUnauthenticated
	}
	return &entities.Caller{UserID: userID, Role: entities.Role(principal.Role)}, nil
}

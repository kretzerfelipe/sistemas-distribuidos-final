package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/platform/httpx"
	"futhub/main/internal/platform/security"
)

func TestWriteErrorMapsEveryBusinessErrorToItsStatus(t *testing.T) {
	scenarios := map[string]struct {
		err        error
		wantStatus int
	}{
		"invalid request":          {httpx.ErrInvalidRequest, http.StatusBadRequest},
		"unauthenticated":          {entities.ErrUnauthenticated, http.StatusUnauthorized},
		"invalid credentials":      {entities.ErrInvalidCredentials, http.StatusUnauthorized},
		"forbidden":                {entities.ErrForbidden, http.StatusForbidden},
		"not found":                {entities.ErrNotFound, http.StatusNotFound},
		"player not in team":       {entities.ErrPlayerNotInTeam, http.StatusNotFound},
		"invalid input":            {entities.ErrInvalidInput, http.StatusUnprocessableEntity},
		"invalid state":            {entities.ErrInvalidState, http.StatusConflict},
		"email taken":              {entities.ErrEmailTaken, http.StatusConflict},
		"player already exists":    {entities.ErrPlayerAlreadyExists, http.StatusConflict},
		"team name taken":          {entities.ErrTeamNameTaken, http.StatusConflict},
		"team in use":              {entities.ErrTeamInUse, http.StatusConflict},
		"player already in team":   {entities.ErrPlayerAlreadyInTeam, http.StatusConflict},
		"invitation pending":       {entities.ErrInvitationAlreadyPending, http.StatusConflict},
		"invitation not pending":   {entities.ErrInvitationNotPending, http.StatusConflict},
		"championship full":        {entities.ErrChampionshipFull, http.StatusConflict},
		"championship not open":    {entities.ErrChampionshipNotOpen, http.StatusConflict},
		"championship has matches": {entities.ErrChampionshipHasMatches, http.StatusConflict},
		"team already enrolled":    {entities.ErrTeamAlreadyEnrolled, http.StatusConflict},
		"team not enrolled":        {entities.ErrTeamNotEnrolled, http.StatusConflict},
	}

	for name, scenario := range scenarios {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()

			WriteError(response, scenario.err)

			if response.Code != scenario.wantStatus {
				t.Errorf("want %d, got %d", scenario.wantStatus, response.Code)
			}
		})
	}
}

func TestWriteErrorFindsBusinessErrorsInsideWrappedErrors(t *testing.T) {
	response := httptest.NewRecorder()

	WriteError(response, fmt.Errorf("%w: only scheduled matches can be deleted", entities.ErrInvalidState))

	if response.Code != http.StatusConflict {
		t.Errorf("want 409, got %d", response.Code)
	}
}

func TestWriteErrorHidesUnexpectedErrorsBehindA500(t *testing.T) {
	response := httptest.NewRecorder()

	WriteError(response, fmt.Errorf("connection refused to 10.0.0.5:5432"))

	if response.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", response.Code)
	}
	if body := response.Body.String(); strings.Contains(body, "10.0.0.5") {
		t.Errorf("internal details must not leak to the client: %s", body)
	}
}

func TestGetCallerReadsTheAuthenticatedUser(t *testing.T) {
	userID := uuid.New()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(security.ContextWithPrincipal(request.Context(),
		security.Principal{UserID: userID.String(), Role: string(entities.RoleTeam)}))

	caller, err := GetCaller(request)

	if err != nil || caller.UserID != userID || caller.Role != entities.RoleTeam {
		t.Errorf("unexpected caller %+v err=%v", caller, err)
	}
}

func TestGetCallerFailsWithoutAnAuthenticatedUser(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	caller, err := GetCaller(request)

	if err != entities.ErrUnauthenticated || caller != nil {
		t.Errorf("want ErrUnauthenticated and nil caller, got %v / %v", err, caller)
	}
}

func TestGetCallerFailsWhenTheUserIDIsNotAUUID(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(security.ContextWithPrincipal(request.Context(),
		security.Principal{UserID: "not-a-uuid", Role: string(entities.RoleUser)}))

	caller, err := GetCaller(request)

	if err != entities.ErrUnauthenticated || caller != nil {
		t.Errorf("want ErrUnauthenticated and nil caller, got %v / %v", err, caller)
	}
}

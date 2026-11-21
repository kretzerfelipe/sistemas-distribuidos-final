package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"futhub/main/internal/platform/security"
)

type testRole string

const (
	adminRole testRole = "ADMIN"
	userRole  testRole = "USER"

	testSecret    = "test-secret-with-at-least-32-characters!!"
	tokenLifetime = time.Hour
)

func respondOK(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type testEnvironment struct {
	router     chi.Router
	adminToken string
	userToken  string
}

func newTestEnvironment(t *testing.T, basePath string, routes []Route) testEnvironment {
	t.Helper()
	tokens := security.NewTokenManager(testSecret, tokenLifetime)
	adminToken, _, _ := tokens.Issue("admin-1", string(adminRole))
	userToken, _, _ := tokens.Issue("user-1", string(userRole))

	router := chi.NewRouter()
	err := Register(router, basePath, tokens, routes)
	if err != nil {
		t.Fatalf("unexpected error registering routes: %v", err)
	}
	return testEnvironment{router: router, adminToken: adminToken, userToken: userToken}
}

func (e testEnvironment) call(method, target, token string) int {
	request := httptest.NewRequest(method, target, nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	e.router.ServeHTTP(response, request)
	return response.Code
}

func TestPublicRouteNeedsNoToken(t *testing.T) {
	env := newTestEnvironment(t, "/items", []Route{
		{Method: http.MethodGet, Path: "/open", Handler: respondOK, Auth: Public},
	})

	status := env.call(http.MethodGet, "/items/open", "")

	if status != http.StatusOK {
		t.Errorf("want 200, got %d", status)
	}
}

func TestAuthenticatedRouteRequiresAValidTokenButAnyRole(t *testing.T) {
	env := newTestEnvironment(t, "/items", []Route{
		{Method: http.MethodGet, Path: "/mine", Handler: respondOK, Auth: Authenticated()},
	})

	scenarios := map[string]struct {
		token      string
		wantStatus int
	}{
		"no token":      {"", http.StatusUnauthorized},
		"invalid token": {"garbage", http.StatusUnauthorized},
		"regular user":  {env.userToken, http.StatusOK},
		"admin":         {env.adminToken, http.StatusOK},
	}

	for name, scenario := range scenarios {
		t.Run(name, func(t *testing.T) {
			status := env.call(http.MethodGet, "/items/mine", scenario.token)

			if status != scenario.wantStatus {
				t.Errorf("want %d, got %d", scenario.wantStatus, status)
			}
		})
	}
}

func TestRoleRouteOnlyAcceptsTheListedRoles(t *testing.T) {
	env := newTestEnvironment(t, "/items", []Route{
		{Method: http.MethodGet, Path: "/admin-only", Handler: respondOK, Auth: Roles(adminRole)},
		{Method: http.MethodGet, Path: "/staff", Handler: respondOK, Auth: Roles(adminRole, userRole)},
	})

	scenarios := map[string]struct {
		target     string
		token      string
		wantStatus int
	}{
		"admin route without token":   {"/items/admin-only", "", http.StatusUnauthorized},
		"admin route as regular user": {"/items/admin-only", env.userToken, http.StatusForbidden},
		"admin route as admin":        {"/items/admin-only", env.adminToken, http.StatusOK},
		"two roles route as user":     {"/items/staff", env.userToken, http.StatusOK},
		"two roles route as admin":    {"/items/staff", env.adminToken, http.StatusOK},
	}

	for name, scenario := range scenarios {
		t.Run(name, func(t *testing.T) {
			status := env.call(http.MethodGet, scenario.target, scenario.token)

			if status != scenario.wantStatus {
				t.Errorf("want %d, got %d", scenario.wantStatus, status)
			}
		})
	}
}

func TestRoutesAreSeparatedByMethod(t *testing.T) {
	env := newTestEnvironment(t, "/items", []Route{
		{Method: http.MethodGet, Path: "/thing", Handler: respondOK, Auth: Public},
		{Method: http.MethodPost, Path: "/thing", Handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) }, Auth: Public},
	})

	getStatus := env.call(http.MethodGet, "/items/thing", "")
	postStatus := env.call(http.MethodPost, "/items/thing", "")
	deleteStatus := env.call(http.MethodDelete, "/items/thing", "")

	if getStatus != http.StatusOK || postStatus != http.StatusCreated || deleteStatus != http.StatusMethodNotAllowed {
		t.Errorf("want 200/201/405, got %d/%d/%d", getStatus, postStatus, deleteStatus)
	}
}

func TestPathsAreJoinedWithTheBasePath(t *testing.T) {
	routes := []Route{
		{Method: http.MethodGet, Path: "/", Handler: respondOK, Auth: Public},
		{Method: http.MethodGet, Path: "/{id}", Handler: respondOK, Auth: Public},
		{Method: http.MethodGet, Path: "get-by-id/{id}", Handler: respondOK, Auth: Public},
	}
	env := newTestEnvironment(t, "/items", routes)

	scenarios := map[string]string{
		"root of the module":           "/items",
		"path with a variable":         "/items/123",
		"path without a leading slash": "/items/get-by-id/123",
	}

	for name, target := range scenarios {
		t.Run(name, func(t *testing.T) {
			status := env.call(http.MethodGet, target, "")

			if status != http.StatusOK {
				t.Errorf("%s: want 200, got %d", target, status)
			}
		})
	}
}

func TestRegisterRejectsInvalidRoutes(t *testing.T) {
	tokens := security.NewTokenManager(testSecret, tokenLifetime)

	invalidRoutes := map[string]Route{
		"missing method":     {Path: "/x", Handler: respondOK},
		"unsupported method": {Method: http.MethodTrace, Path: "/x", Handler: respondOK},
		"lowercase method":   {Method: "get", Path: "/x", Handler: respondOK},
		"missing handler":    {Method: http.MethodGet, Path: "/x"},
	}

	for name, route := range invalidRoutes {
		t.Run(name, func(t *testing.T) {
			err := Register(chi.NewRouter(), "/items", tokens, []Route{route})

			if err == nil {
				t.Error("an invalid route must be rejected with an error")
			}
		})
	}
}

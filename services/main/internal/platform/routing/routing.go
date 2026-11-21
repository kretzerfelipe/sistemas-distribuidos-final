package routing

import (
	"fmt"
	"net/http"
	"path"
	"slices"

	"github.com/go-chi/chi/v5"

	"futhub/main/internal/platform/security"
)

var supportedMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
}

type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
	Auth    Auth
}

type Auth struct {
	required bool
	roles    []string
}

var Public = Auth{}

func Authenticated() Auth {
	return Auth{required: true}
}

func Roles[R ~string](roles ...R) Auth {
	allowedRoles := make([]string, 0, len(roles))
	for _, role := range roles {
		allowedRoles = append(allowedRoles, string(role))
	}
	return Auth{required: true, roles: allowedRoles}
}

func Register(router chi.Router, basePath string, tokens *security.TokenManager, routes []Route) error {
	authenticate := security.Authenticate(tokens)

	for _, route := range routes {
		err := route.validate()
		if err != nil {
			return fmt.Errorf("invalid route %s %s: %w", route.Method, route.Path, err)
		}
		router.Method(route.Method, fullPath(basePath, route.Path), protect(route, authenticate))
	}
	return nil
}

func (r Route) validate() error {
	methodIsSupported := slices.Contains(supportedMethods, r.Method)
	if !methodIsSupported {
		return fmt.Errorf("method must be one of %v", supportedMethods)
	}
	handlerIsMissing := r.Handler == nil
	if handlerIsMissing {
		return fmt.Errorf("handler is required")
	}
	return nil
}

func fullPath(basePath, routePath string) string {
	return path.Join("/", basePath, routePath)
}

func protect(route Route, authenticate func(http.Handler) http.Handler) http.Handler {
	var handler http.Handler = route.Handler

	routeIsPublic := !route.Auth.required
	if routeIsPublic {
		return handler
	}

	routeRestrictsRoles := len(route.Auth.roles) > 0
	if routeRestrictsRoles {
		handler = security.RequireRole(route.Auth.roles...)(handler)
	}
	return authenticate(handler)
}

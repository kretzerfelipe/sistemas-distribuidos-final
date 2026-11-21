package security

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"futhub/main/internal/platform/httpx"
)

const bearerPrefix = "Bearer "

type principalKey struct{}

func ContextWithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, found := ctx.Value(principalKey{}).(Principal)
	return principal, found
}

func Authenticate(tokens *TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken, hasBearerPrefix := strings.CutPrefix(r.Header.Get("Authorization"), bearerPrefix)
			rawToken = strings.TrimSpace(rawToken)

			tokenIsMissing := !hasBearerPrefix || rawToken == ""
			if tokenIsMissing {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
				return
			}

			principal, err := tokens.Parse(rawToken)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}
			next.ServeHTTP(w, r.WithContext(ContextWithPrincipal(r.Context(), principal)))
		})
	}
}

func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, isAuthenticated := PrincipalFromContext(r.Context())
			if !isAuthenticated {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}

			roleIsAllowed := slices.Contains(allowedRoles, principal.Role)
			if !roleIsAllowed {
				httpx.WriteError(w, http.StatusForbidden, "forbidden", "you do not have permission to access this resource")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

package security

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"futhub/notification/internal/httpx"
)

const (
	RoleAdmin = "ADMIN"

	bearerPrefix    = "Bearer "
	queryTokenParam = "token"
)

var ErrInvalidToken = errors.New("invalid or expired token")

type Principal struct {
	UserID string
	Role   string
}

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type TokenParser struct {
	secret []byte
	now    func() time.Time
}

func NewTokenParser(secret string) *TokenParser {
	return &TokenParser{secret: []byte(secret), now: time.Now}
}

func (t *TokenParser) Parse(rawToken string) (Principal, error) {
	var tokenClaims claims
	token, err := jwt.ParseWithClaims(rawToken, &tokenClaims,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil || !token.Valid || tokenClaims.Subject == "" {
		return Principal{}, ErrInvalidToken
	}
	return Principal{UserID: tokenClaims.Subject, Role: tokenClaims.Role}, nil
}

type principalKey struct{}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, found := ctx.Value(principalKey{}).(Principal)
	return principal, found
}

func Authenticate(parser *TokenParser, allowQueryToken bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken := extractToken(r, allowQueryToken)
			if rawToken == "" {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
				return
			}

			principal, err := parser.Parse(rawToken)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, principal)))
		})
	}
}

func extractToken(r *http.Request, allowQueryToken bool) string {
	headerToken, hasBearerPrefix := strings.CutPrefix(r.Header.Get("Authorization"), bearerPrefix)
	if hasBearerPrefix && strings.TrimSpace(headerToken) != "" {
		return strings.TrimSpace(headerToken)
	}
	if allowQueryToken {
		return strings.TrimSpace(r.URL.Query().Get(queryTokenParam))
	}
	return ""
}

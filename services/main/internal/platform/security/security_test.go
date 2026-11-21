package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testSecret      = "test-secret-with-at-least-32-characters!!"
	otherSecret     = "another-secret-with-at-least-32-chars!!"
	tokenLifetime   = time.Hour
	fastBcryptCost  = 4
	adminRole       = "ADMIN"
	userRole        = "USER"
	someUserID      = "user-1"
	bearerHeaderFor = "Bearer "
)

func TestTokenCarriesUserIDAndRole(t *testing.T) {
	tokens := NewTokenManager(testSecret, tokenLifetime)

	token, expiresAt, err := tokens.Issue(someUserID, adminRole)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := tokens.Parse(token)

	if err != nil || principal.UserID != someUserID || principal.Role != adminRole {
		t.Fatalf("unexpected principal %+v err=%v", principal, err)
	}
	expirationIsInTheFuture := expiresAt.After(time.Now())
	if !expirationIsInTheFuture {
		t.Error("expiration should be in the future")
	}
}

func TestParseRejectsInvalidTokens(t *testing.T) {
	tokens := NewTokenManager(testSecret, tokenLifetime)
	validToken, _, _ := tokens.Issue(someUserID, userRole)

	issuedTwoHoursAgo := NewTokenManager(testSecret, tokenLifetime)
	issuedTwoHoursAgo.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	expiredToken, _, _ := issuedTwoHoursAgo.Issue(someUserID, userRole)

	signedWithOtherSecret, _, _ := NewTokenManager(otherSecret, tokenLifetime).Issue(someUserID, userRole)

	signedWithAlgNone, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": someUserID, "role": adminRole}).
		SignedString(jwt.UnsafeAllowNoneSignatureType)

	tamperedSignature := validToken[:len(validToken)-4] + "AAAA"

	invalidTokens := map[string]string{
		"random text":              "not-a-token",
		"empty":                    "",
		"expired":                  expiredToken,
		"signed with other secret": signedWithOtherSecret,
		"algorithm none":           signedWithAlgNone,
		"tampered signature":       tamperedSignature,
	}

	for name, token := range invalidTokens {
		t.Run(name, func(t *testing.T) {
			_, err := tokens.Parse(token)

			if err == nil {
				t.Error("token should have been rejected")
			}
		})
	}
}

func TestBcryptHasherHashesWithSalt(t *testing.T) {
	hasher := NewBcryptHasher(fastBcryptCost)

	hash, err := hasher.Hash("my-password")
	if err != nil {
		t.Fatal(err)
	}
	otherHash, _ := hasher.Hash("my-password")

	hashIsBcrypt := strings.HasPrefix(hash, "$2")
	if !hashIsBcrypt {
		t.Errorf("expected a bcrypt hash, got %q", hash)
	}
	if hash == otherHash {
		t.Error("two hashes of the same password must differ because of the salt")
	}
}

func TestBcryptHasherComparesPasswords(t *testing.T) {
	hasher := NewBcryptHasher(fastBcryptCost)
	hash, _ := hasher.Hash("my-password")

	rightPasswordMatches := hasher.Compare(hash, "my-password")
	wrongPasswordMatches := hasher.Compare(hash, "other-password")

	if !rightPasswordMatches || wrongPasswordMatches {
		t.Errorf("compare must accept only the right password (right=%v wrong=%v)", rightPasswordMatches, wrongPasswordMatches)
	}
}

func TestAuthenticateAndRequireRole(t *testing.T) {
	tokens := NewTokenManager(testSecret, tokenLifetime)
	adminToken, _, _ := tokens.Issue("admin-1", adminRole)
	userToken, _, _ := tokens.Issue("user-1", userRole)

	adminOnlyRoute := Authenticate(tokens)(RequireRole(adminRole)(respondOK()))

	scenarios := []struct {
		name          string
		authorization string
		wantStatus    int
	}{
		{"no authorization header", "", http.StatusUnauthorized},
		{"scheme other than bearer", "Basic abc", http.StatusUnauthorized},
		{"invalid token", bearerHeaderFor + "nope", http.StatusUnauthorized},
		{"regular user on admin route", bearerHeaderFor + userToken, http.StatusForbidden},
		{"admin on admin route", bearerHeaderFor + adminToken, http.StatusOK},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if scenario.authorization != "" {
				request.Header.Set("Authorization", scenario.authorization)
			}
			response := httptest.NewRecorder()

			adminOnlyRoute.ServeHTTP(response, request)

			if response.Code != scenario.wantStatus {
				t.Errorf("want status %d, got %d", scenario.wantStatus, response.Code)
			}
		})
	}
}

func respondOK() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

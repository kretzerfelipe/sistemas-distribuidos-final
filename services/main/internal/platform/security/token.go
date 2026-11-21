package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

type TokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl, now: time.Now}
}

func (t *TokenManager) Issue(userID, role string) (string, time.Time, error) {
	issuedAt := t.now()
	expiresAt := issuedAt.Add(t.ttl)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})
	signedToken, err := token.SignedString(t.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signedToken, expiresAt, nil
}

func (t *TokenManager) Parse(rawToken string) (Principal, error) {
	var tokenClaims claims
	token, err := jwt.ParseWithClaims(rawToken, &tokenClaims, t.secretKey,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(t.now),
	)

	subjectIsMissing := tokenClaims.Subject == ""
	tokenIsInvalid := err != nil || !token.Valid || subjectIsMissing
	if tokenIsInvalid {
		return Principal{}, ErrInvalidToken
	}
	return Principal{UserID: tokenClaims.Subject, Role: tokenClaims.Role}, nil
}

func (t *TokenManager) secretKey(*jwt.Token) (any, error) {
	return t.secret, nil
}

package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/modules/users"
	"futhub/main/internal/platform/security"
)

const (
	testJWTSecret       = "test-secret-with-at-least-32-characters!!"
	fastBcryptCost      = 4
	validPassword       = "s3cret-pass"
	anaEmail            = "ana@example.com"
	adminEmail          = "admin@example.com"
	tokenLifetime       = time.Hour
	oneCharacterTooMany = maxPasswordLength + 1
)

type inMemoryUserService struct {
	usersByEmail map[string]entities.User
	failure      error
}

func newInMemoryUserService() *inMemoryUserService {
	return &inMemoryUserService{usersByEmail: map[string]entities.User{}}
}

func (s *inMemoryUserService) Create(_ context.Context, in entities.CreateInput) (*entities.User, error) {
	email := users.NormalizeEmail(in.Email)
	_, emailIsTaken := s.usersByEmail[email]
	if emailIsTaken {
		return nil, entities.ErrEmailTaken
	}
	user := entities.User{ID: uuid.New(), Name: in.Name, Email: email, PasswordHash: in.PasswordHash, Role: in.Role}
	s.usersByEmail[email] = user
	return &user, nil
}

func (s *inMemoryUserService) GetByID(_ context.Context, id uuid.UUID) (*entities.User, error) {
	for _, user := range s.usersByEmail {
		if user.ID == id {
			return &user, nil
		}
	}
	return nil, entities.ErrNotFound
}

func (s *inMemoryUserService) GetByEmail(_ context.Context, email string) (*entities.User, error) {
	if s.failure != nil {
		return nil, s.failure
	}
	user, found := s.usersByEmail[users.NormalizeEmail(email)]
	if !found {
		return nil, entities.ErrNotFound
	}
	return &user, nil
}

func (s *inMemoryUserService) EmailIsRegistered(_ context.Context, email string) (bool, error) {
	if s.failure != nil {
		return false, s.failure
	}
	_, found := s.usersByEmail[users.NormalizeEmail(email)]
	return found, nil
}

type testEnvironment struct {
	useCase UseCase
	users   *inMemoryUserService
	tokens  *security.TokenManager
}

func newTestEnvironment(t *testing.T) testEnvironment {
	t.Helper()
	userService := newInMemoryUserService()
	tokens := security.NewTokenManager(testJWTSecret, tokenLifetime)
	useCase, err := newUseCase(userService, security.NewBcryptHasher(fastBcryptCost), tokens)
	if err != nil {
		t.Fatal(err)
	}
	return testEnvironment{useCase: useCase, users: userService, tokens: tokens}
}

func (e testEnvironment) registerAna(t *testing.T) *entities.User {
	t.Helper()
	user, err := e.useCase.Register(context.Background(), entities.RegisterInput{Name: "Ana", Email: anaEmail, Password: validPassword})
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func TestRegisterAlwaysCreatesRegularUser(t *testing.T) {
	env := newTestEnvironment(t)

	user := env.registerAna(t)

	if user.Role != entities.RoleUser {
		t.Errorf("public registration must create role USER, got %s", user.Role)
	}
}

func TestRegisterStoresOnlyTheHashOfThePassword(t *testing.T) {
	env := newTestEnvironment(t)

	env.registerAna(t)

	storedHash := env.users.usersByEmail[anaEmail].PasswordHash
	hashIsPlainPassword := storedHash == validPassword
	hashContainsPlainPassword := strings.Contains(storedHash, validPassword)
	if storedHash == "" || hashIsPlainPassword || hashContainsPlainPassword {
		t.Errorf("password must be stored hashed, got %q", storedHash)
	}
}

func TestRegisterRejectsPasswordOutsideAllowedLength(t *testing.T) {
	passwords := map[string]string{
		"shorter than the minimum": strings.Repeat("a", minPasswordLength-1),
		"longer than the maximum":  strings.Repeat("a", oneCharacterTooMany),
	}
	for name, password := range passwords {
		t.Run(name, func(t *testing.T) {
			env := newTestEnvironment(t)

			user, err := env.useCase.Register(context.Background(), entities.RegisterInput{Name: "Ana", Email: anaEmail, Password: password})

			if !errors.Is(err, entities.ErrInvalidInput) {
				t.Errorf("want ErrInvalidInput, got %v", err)
			}
			if user != nil {
				t.Error("user must be nil when there is an error")
			}
		})
	}
}

func TestRegisterRejectsEmailAlreadyRegistered(t *testing.T) {
	env := newTestEnvironment(t)
	env.registerAna(t)

	user, err := env.useCase.Register(context.Background(), entities.RegisterInput{Name: "Ana", Email: anaEmail, Password: validPassword})

	if !errors.Is(err, entities.ErrEmailTaken) {
		t.Errorf("want ErrEmailTaken, got %v", err)
	}
	if user != nil {
		t.Error("user must be nil when there is an error")
	}
}

func TestLoginReturnsTokenWithUserIDAndRole(t *testing.T) {
	env := newTestEnvironment(t)
	registered := env.registerAna(t)

	result, err := env.useCase.Login(context.Background(), entities.LoginInput{Email: "ANA@example.com", Password: validPassword})

	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	principal, err := env.tokens.Parse(result.AccessToken)
	if err != nil {
		t.Fatalf("token should be valid: %v", err)
	}
	principalIsRegisteredUser := principal.UserID == registered.ID.String() && principal.Role == string(entities.RoleUser)
	if !principalIsRegisteredUser {
		t.Errorf("token carries the wrong principal: %+v", principal)
	}
	if result.TokenType != "Bearer" {
		t.Errorf("unexpected token type %q", result.TokenType)
	}
}

func TestLoginGivesTheSameErrorForWrongPasswordAndUnknownEmail(t *testing.T) {
	env := newTestEnvironment(t)
	env.registerAna(t)

	wrongPasswordResult, wrongPasswordErr := env.useCase.Login(context.Background(), entities.LoginInput{Email: anaEmail, Password: "wrong-password"})
	unknownEmailResult, unknownEmailErr := env.useCase.Login(context.Background(), entities.LoginInput{Email: "nobody@example.com", Password: validPassword})

	bothAreInvalidCredentials := errors.Is(wrongPasswordErr, entities.ErrInvalidCredentials) && errors.Is(unknownEmailErr, entities.ErrInvalidCredentials)
	if !bothAreInvalidCredentials {
		t.Fatalf("both must be ErrInvalidCredentials, got %v and %v", wrongPasswordErr, unknownEmailErr)
	}
	errorsAreIndistinguishable := wrongPasswordErr.Error() == unknownEmailErr.Error()
	if !errorsAreIndistinguishable {
		t.Error("errors must be identical so registered emails cannot be discovered")
	}
	if wrongPasswordResult != nil || unknownEmailResult != nil {
		t.Error("result must be nil when there is an error")
	}
}

func TestLoginKeepsTheCauseOfUnexpectedFailures(t *testing.T) {
	env := newTestEnvironment(t)
	databaseFailure := errors.New("connection refused")
	env.users.failure = databaseFailure

	result, err := env.useCase.Login(context.Background(), entities.LoginInput{Email: anaEmail, Password: validPassword})

	if !errors.Is(err, databaseFailure) {
		t.Errorf("the original failure must stay reachable, got %v", err)
	}
	if errors.Is(err, entities.ErrInvalidCredentials) {
		t.Error("an infrastructure failure must not look like wrong credentials")
	}
	if result != nil {
		t.Error("result must be nil when there is an error")
	}
}

func TestSeedAdminCreatesAnAdmin(t *testing.T) {
	env := newTestEnvironment(t)

	created, err := env.useCase.SeedAdmin(context.Background(), "Admin", adminEmail, "admin-password")

	if err != nil || !created {
		t.Fatalf("seed should create the admin: created=%v err=%v", created, err)
	}
	if env.users.usersByEmail[adminEmail].Role != entities.RoleAdmin {
		t.Error("seeded user must have role ADMIN")
	}
}

func TestSeedAdminDoesNothingWhenTheAdminAlreadyExists(t *testing.T) {
	env := newTestEnvironment(t)
	if _, err := env.useCase.SeedAdmin(context.Background(), "Admin", adminEmail, "admin-password"); err != nil {
		t.Fatal(err)
	}

	created, err := env.useCase.SeedAdmin(context.Background(), "Admin", adminEmail, "another-password")

	if err != nil || created {
		t.Errorf("second seed must be a no-op: created=%v err=%v", created, err)
	}
}

func TestSeedAdminPropagatesFailuresInsteadOfCreatingTheAdmin(t *testing.T) {
	env := newTestEnvironment(t)
	databaseFailure := errors.New("connection refused")
	env.users.failure = databaseFailure

	created, err := env.useCase.SeedAdmin(context.Background(), "Admin", adminEmail, "admin-password")

	if !errors.Is(err, databaseFailure) {
		t.Errorf("the failure must be propagated, got %v", err)
	}
	if created {
		t.Error("nothing must be created when the lookup failed")
	}
	if len(env.users.usersByEmail) != 0 {
		t.Error("no user should have been stored")
	}
}

func TestMeReturnsTheUser(t *testing.T) {
	env := newTestEnvironment(t)
	registered := env.registerAna(t)

	user, err := env.useCase.Me(context.Background(), registered.ID.String())

	if err != nil || user.ID != registered.ID {
		t.Fatalf("Me failed: user=%+v err=%v", user, err)
	}
}

func TestMeReturnsNotFoundForMalformedID(t *testing.T) {
	env := newTestEnvironment(t)

	user, err := env.useCase.Me(context.Background(), "not-a-uuid")

	if !errors.Is(err, entities.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
	if user != nil {
		t.Error("user must be nil when there is an error")
	}
}

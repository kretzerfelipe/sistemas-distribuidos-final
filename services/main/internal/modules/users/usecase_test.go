package users

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
)

type inMemoryRepository struct {
	usersByEmail map[string]entities.User
}

func newInMemoryRepository() *inMemoryRepository {
	return &inMemoryRepository{usersByEmail: map[string]entities.User{}}
}

func (r *inMemoryRepository) Insert(_ context.Context, user *entities.User) error {
	_, emailIsTaken := r.usersByEmail[user.Email]
	if emailIsTaken {
		return entities.ErrEmailTaken
	}
	r.usersByEmail[user.Email] = *user
	return nil
}

func (r *inMemoryRepository) FindByID(_ context.Context, id uuid.UUID) (*entities.User, error) {
	for _, user := range r.usersByEmail {
		if user.ID == id {
			return &user, nil
		}
	}
	return nil, entities.ErrNotFound
}

func (r *inMemoryRepository) FindByEmail(_ context.Context, email string) (*entities.User, error) {
	user, found := r.usersByEmail[email]
	if !found {
		return nil, entities.ErrNotFound
	}
	return &user, nil
}

func (r *inMemoryRepository) ExistsByEmail(_ context.Context, email string) (bool, error) {
	_, found := r.usersByEmail[email]
	return found, nil
}

func (r *inMemoryRepository) List(context.Context) ([]entities.User, error) {
	allUsers := []entities.User{}
	for _, user := range r.usersByEmail {
		allUsers = append(allUsers, user)
	}
	return allUsers, nil
}

type failingRepository struct {
	*inMemoryRepository
	failure error
}

func (r *failingRepository) Insert(context.Context, *entities.User) error { return r.failure }

func validInput() entities.CreateInput {
	return entities.CreateInput{Name: "Ana Silva", Email: "ana@example.com", PasswordHash: "hash", Role: entities.RoleUser}
}

func TestCreateNormalizesNameAndEmail(t *testing.T) {
	input := validInput()
	input.Name = "  Ana Silva "
	input.Email = "  Ana@Example.COM "

	user, err := newUseCase(newInMemoryRepository()).Create(context.Background(), input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Name != "Ana Silva" || user.Email != "ana@example.com" {
		t.Errorf("name and email should be normalized, got %q and %q", user.Name, user.Email)
	}
	if user.ID == uuid.Nil {
		t.Error("id should be generated")
	}
}

func TestCreateRejectsInvalidInputAndReturnsNoUser(t *testing.T) {
	withName := func(name string) entities.CreateInput { in := validInput(); in.Name = name; return in }
	withEmail := func(email string) entities.CreateInput { in := validInput(); in.Email = email; return in }
	withPasswordHash := func(hash string) entities.CreateInput { in := validInput(); in.PasswordHash = hash; return in }
	withRole := func(role entities.Role) entities.CreateInput { in := validInput(); in.Role = role; return in }

	invalidInputs := map[string]entities.CreateInput{
		"blank name":                    withName("   "),
		"name above the maximum length": withName(strings.Repeat("a", maxNameLength+1)),
		"email without domain":          withEmail("not-an-email"),
		"email with display name":       withEmail("Bob <bob@example.com>"),
		"missing password hash":         withPasswordHash(""),
		"unknown role":                  withRole(entities.Role("ROOT")),
	}

	for name, input := range invalidInputs {
		t.Run(name, func(t *testing.T) {
			user, err := newUseCase(newInMemoryRepository()).Create(context.Background(), input)

			if !errors.Is(err, entities.ErrInvalidInput) {
				t.Errorf("want ErrInvalidInput, got %v", err)
			}
			if user != nil {
				t.Error("user must be nil when there is an error")
			}
		})
	}
}

func TestCreateRejectsDuplicateEmailIgnoringCase(t *testing.T) {
	useCase := newUseCase(newInMemoryRepository())
	if _, err := useCase.Create(context.Background(), validInput()); err != nil {
		t.Fatal(err)
	}
	duplicate := validInput()
	duplicate.Email = "ANA@EXAMPLE.COM"

	user, err := useCase.Create(context.Background(), duplicate)

	if !errors.Is(err, entities.ErrEmailTaken) {
		t.Errorf("want ErrEmailTaken, got %v", err)
	}
	if user != nil {
		t.Error("user must be nil when there is an error")
	}
}

func TestCreateKeepsTheCauseOfUnexpectedFailures(t *testing.T) {
	databaseFailure := errors.New("connection refused")
	repo := &failingRepository{inMemoryRepository: newInMemoryRepository(), failure: databaseFailure}

	user, err := newUseCase(repo).Create(context.Background(), validInput())

	if !errors.Is(err, databaseFailure) {
		t.Errorf("the original failure must stay reachable, got %v", err)
	}
	if user != nil {
		t.Error("user must be nil when there is an error")
	}
}

func TestGetByEmailIgnoresCaseAndSpaces(t *testing.T) {
	useCase := newUseCase(newInMemoryRepository())
	if _, err := useCase.Create(context.Background(), validInput()); err != nil {
		t.Fatal(err)
	}

	user, err := useCase.GetByEmail(context.Background(), " ANA@example.com ")

	if err != nil || user == nil {
		t.Errorf("lookup should ignore case and spaces: user=%v err=%v", user, err)
	}
}

func TestGetByIDReturnsNilUserWhenNotFound(t *testing.T) {
	user, err := newUseCase(newInMemoryRepository()).GetByID(context.Background(), uuid.New())

	if !errors.Is(err, entities.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
	if user != nil {
		t.Error("user must be nil when there is an error")
	}
}

func TestEmailIsRegisteredIgnoresCase(t *testing.T) {
	useCase := newUseCase(newInMemoryRepository())
	if _, err := useCase.Create(context.Background(), validInput()); err != nil {
		t.Fatal(err)
	}

	registered, err := useCase.EmailIsRegistered(context.Background(), "ANA@example.com")
	unregistered, otherErr := useCase.EmailIsRegistered(context.Background(), "nobody@example.com")

	if err != nil || otherErr != nil {
		t.Fatalf("unexpected errors: %v and %v", err, otherErr)
	}
	if !registered || unregistered {
		t.Errorf("want registered=true and unregistered=false, got %v and %v", registered, unregistered)
	}
}

func TestUserResponseNeverContainsPasswordHash(t *testing.T) {
	user := &entities.User{Name: "Ana", Email: "ana@example.com", PasswordHash: "secret-hash", Role: entities.RoleUser}

	body, err := json.Marshal(NewUserResponse(user))

	if err != nil {
		t.Fatal(err)
	}
	responseLeaksHash := strings.Contains(string(body), "secret-hash")
	if responseLeaksHash {
		t.Errorf("response must not contain the password hash: %s", body)
	}
}

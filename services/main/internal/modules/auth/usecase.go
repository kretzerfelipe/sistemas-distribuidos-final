package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"futhub/main/internal/entities"
	"futhub/main/internal/modules/users"
	"futhub/main/internal/platform/security"
	"futhub/main/internal/platform/util"
)

const (
	minPasswordLength = 8
	maxPasswordLength = 72
	tokenType         = "Bearer"
	unusedPassword    = "not-a-real-password"
)

type useCase struct {
	users                        UserService
	hasher                       security.PasswordHasher
	tokens                       *security.TokenManager
	hashToCompareForUnknownEmail string
}

func newUseCase(userService UserService, hasher security.PasswordHasher, tokens *security.TokenManager) (UseCase, error) {
	hashToCompareForUnknownEmail, err := hasher.Hash(unusedPassword)
	if err != nil {
		return nil, util.SendMessageWithError("use case error preparing hash for unknown emails", err)
	}
	return &useCase{
		users:                        userService,
		hasher:                       hasher,
		tokens:                       tokens,
		hashToCompareForUnknownEmail: hashToCompareForUnknownEmail,
	}, nil
}

func (uc *useCase) Register(ctx context.Context, in entities.RegisterInput) (*entities.User, error) {
	user, err := uc.createUser(ctx, in.Name, in.Email, in.Password, entities.RoleUser)
	if err != nil {
		return nil, util.SendMessageWithError("use case error registering user", err)
	}
	return user, nil
}

func (uc *useCase) Login(ctx context.Context, in entities.LoginInput) (*entities.LoginResult, error) {
	user, err := uc.users.GetByEmail(ctx, in.Email)
	if err != nil {
		emailIsUnknown := errors.Is(err, entities.ErrNotFound)
		if emailIsUnknown {
			uc.hasher.Compare(uc.hashToCompareForUnknownEmail, in.Password)
			return nil, entities.ErrInvalidCredentials
		}
		return nil, util.SendMessageWithError("use case error finding user by email to log in", err)
	}

	passwordMatches := uc.hasher.Compare(user.PasswordHash, in.Password)
	if !passwordMatches {
		return nil, entities.ErrInvalidCredentials
	}

	accessToken, expiresAt, err := uc.tokens.Issue(user.ID.String(), string(user.Role))
	if err != nil {
		return nil, util.SendMessageWithError("use case error issuing access token", err)
	}
	return &entities.LoginResult{
		AccessToken: accessToken,
		TokenType:   tokenType,
		ExpiresAt:   expiresAt,
		User:        users.NewUserResponse(user),
	}, nil
}

func (uc *useCase) Me(ctx context.Context, userID string) (*entities.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, entities.ErrNotFound
	}

	user, err := uc.users.GetByID(ctx, id)
	if err != nil {
		return nil, util.SendMessageWithError("use case error finding the logged user", err)
	}
	return user, nil
}

func (uc *useCase) SeedAdmin(ctx context.Context, name, email, password string) (bool, error) {
	adminEmailIsRegistered, err := uc.users.EmailIsRegistered(ctx, email)
	if err != nil {
		return false, util.SendMessageWithError("use case error checking whether the admin exists", err)
	}
	if adminEmailIsRegistered {
		return false, nil
	}

	_, err = uc.createUser(ctx, name, email, password, entities.RoleAdmin)
	if err != nil {
		return false, util.SendMessageWithError("use case error creating the admin", err)
	}
	return true, nil
}

func (uc *useCase) createUser(ctx context.Context, name, email, password string, role entities.Role) (*entities.User, error) {
	passwordLengthIsValid := len(password) >= minPasswordLength && len(password) <= maxPasswordLength
	if !passwordLengthIsValid {
		return nil, fmt.Errorf("%w: password must have between %d and %d characters",
			entities.ErrInvalidInput, minPasswordLength, maxPasswordLength)
	}

	passwordHash, err := uc.hasher.Hash(password)
	if err != nil {
		return nil, util.SendMessageWithError("use case error hashing password", err)
	}

	user, err := uc.users.Create(ctx, entities.CreateInput{Name: name, Email: email, PasswordHash: passwordHash, Role: role})
	if err != nil {
		return nil, util.SendMessageWithError("use case error creating user", err)
	}
	return user, nil
}

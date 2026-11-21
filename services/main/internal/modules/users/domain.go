package users

import (
	"fmt"
	"net/mail"
	"strings"

	"futhub/main/internal/entities"
)

const (
	maxNameLength  = 120
	maxEmailLength = 254
)

func roleIsValid(role entities.Role) bool {
	return role == entities.RoleAdmin || role == entities.RoleUser
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validate(in entities.CreateInput) (*entities.CreateInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = NormalizeEmail(in.Email)

	nameIsInvalid := in.Name == "" || len(in.Name) > maxNameLength
	emailIsInvalid := !isPlainEmailAddress(in.Email) || len(in.Email) > maxEmailLength
	passwordHashIsMissing := in.PasswordHash == ""
	roleIsInvalid := !roleIsValid(in.Role)

	switch {
	case nameIsInvalid:
		return nil, fmt.Errorf("%w: name is required and must have at most %d characters", entities.ErrInvalidInput, maxNameLength)
	case emailIsInvalid:
		return nil, fmt.Errorf("%w: email is not valid", entities.ErrInvalidInput)
	case passwordHashIsMissing:
		return nil, fmt.Errorf("%w: password hash is required", entities.ErrInvalidInput)
	case roleIsInvalid:
		return nil, fmt.Errorf("%w: role must be ADMIN or USER", entities.ErrInvalidInput)
	}
	return &in, nil
}

func isPlainEmailAddress(email string) bool {
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email
}

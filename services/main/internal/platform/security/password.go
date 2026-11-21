package security

import "golang.org/x/crypto/bcrypt"

type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) bool
}

type BcryptHasher struct {
	cost int
}

func NewBcryptHasher(cost int) *BcryptHasher {
	return &BcryptHasher{cost: cost}
}

func (h *BcryptHasher) Hash(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (h *BcryptHasher) Compare(hash, plain string) bool {
	passwordMatches := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
	return passwordMatches
}

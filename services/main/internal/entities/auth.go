package entities

import "time"

type RegisterInput struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	AccountType Role   `json:"accountType"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResult struct {
	AccessToken string       `json:"accessToken"`
	TokenType   string       `json:"tokenType"`
	ExpiresAt   time.Time    `json:"expiresAt"`
	User        UserResponse `json:"user"`
}

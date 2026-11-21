package entities

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin Role = "ADMIN"
	RoleUser  Role = "USER"
)

type User struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateInput struct {
	Name         string
	Email        string
	PasswordHash string
	Role         Role
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

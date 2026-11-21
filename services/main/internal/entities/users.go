package entities

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin Role = "ADMIN"
	RoleTeam  Role = "TEAM"
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

type UpdateRoleInput struct {
	Role Role `json:"role"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

type Caller struct {
	UserID uuid.UUID
	Role   Role
}

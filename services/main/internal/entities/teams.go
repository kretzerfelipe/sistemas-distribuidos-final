package entities

import (
	"time"

	"github.com/google/uuid"
)

type Team struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	Name      string
	City      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SaveTeamInput struct {
	Name string `json:"name"`
	City string `json:"city"`
}

type TeamResponse struct {
	ID        uuid.UUID `json:"id"`
	OwnerID   uuid.UUID `json:"ownerId"`
	Name      string    `json:"name"`
	City      string    `json:"city"`
	CreatedAt time.Time `json:"createdAt"`
}

type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "PENDING"
	InvitationAccepted InvitationStatus = "ACCEPTED"
	InvitationDeclined InvitationStatus = "DECLINED"
	InvitationCanceled InvitationStatus = "CANCELED"
)

type Invitation struct {
	ID        uuid.UUID
	TeamID    uuid.UUID
	TeamName  string
	PlayerID  uuid.UUID
	Status    InvitationStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type InvitePlayerInput struct {
	PlayerID uuid.UUID `json:"playerId"`
}

type InvitationResponse struct {
	ID        uuid.UUID        `json:"id"`
	TeamID    uuid.UUID        `json:"teamId"`
	TeamName  string           `json:"teamName"`
	PlayerID  uuid.UUID        `json:"playerId"`
	Status    InvitationStatus `json:"status"`
	CreatedAt time.Time        `json:"createdAt"`
}

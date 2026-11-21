package entities

import (
	"time"

	"github.com/google/uuid"
)

type MatchEventType string

const (
	EventGoal         MatchEventType = "GOAL"
	EventOwnGoal      MatchEventType = "OWN_GOAL"
	EventShot         MatchEventType = "SHOT"
	EventSave         MatchEventType = "SAVE"
	EventFoul         MatchEventType = "FOUL"
	EventYellowCard   MatchEventType = "YELLOW_CARD"
	EventRedCard      MatchEventType = "RED_CARD"
	EventSubstitution MatchEventType = "SUBSTITUTION"
	EventCorner       MatchEventType = "CORNER"
	EventOffside      MatchEventType = "OFFSIDE"
)

type MatchEvent struct {
	ID                  uuid.UUID
	MatchID             uuid.UUID
	Sequence            int
	Type                MatchEventType
	Minute              int
	TeamID              uuid.UUID
	PlayerID            *uuid.UUID
	PlayerName          string
	SecondaryPlayerID   *uuid.UUID
	SecondaryPlayerName string
	CreatedAt           time.Time
}

type RegisterMatchEventInput struct {
	Type              MatchEventType `json:"type"`
	Minute            int            `json:"minute"`
	TeamID            uuid.UUID      `json:"teamId"`
	PlayerID          *uuid.UUID     `json:"playerId"`
	SecondaryPlayerID *uuid.UUID     `json:"secondaryPlayerId"`
}

type NamedReference struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type MatchEventResponse struct {
	ID              uuid.UUID       `json:"id"`
	Sequence        int             `json:"sequence"`
	Type            MatchEventType  `json:"type"`
	Minute          int             `json:"minute"`
	TeamID          uuid.UUID       `json:"teamId"`
	Player          *NamedReference `json:"player"`
	SecondaryPlayer *NamedReference `json:"secondaryPlayer"`
	RegisteredAt    time.Time       `json:"registeredAt"`
}

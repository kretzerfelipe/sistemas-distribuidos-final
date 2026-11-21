package entities

import (
	"time"

	"github.com/google/uuid"
)

type Position string

const (
	PositionGoalkeeper Position = "GOALKEEPER"
	PositionDefender   Position = "DEFENDER"
	PositionMidfielder Position = "MIDFIELDER"
	PositionForward    Position = "FORWARD"
)

type Player struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        string
	Position    Position
	ShirtNumber int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SavePlayerInput struct {
	Name        string   `json:"name"`
	Position    Position `json:"position"`
	ShirtNumber int      `json:"shirtNumber"`
}

type PlayerResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Position    Position  `json:"position"`
	ShirtNumber int       `json:"shirtNumber"`
	CreatedAt   time.Time `json:"createdAt"`
}

package entities

import (
	"time"

	"github.com/google/uuid"
)

type ChampionshipStatus string

const (
	ChampionshipOpen       ChampionshipStatus = "OPEN"
	ChampionshipInProgress ChampionshipStatus = "IN_PROGRESS"
	ChampionshipFinished   ChampionshipStatus = "FINISHED"
)

type Championship struct {
	ID         uuid.UUID
	Name       string
	SlotsTotal int
	SlotsTaken int
	Status     ChampionshipStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type SaveChampionshipInput struct {
	Name       string `json:"name"`
	SlotsTotal int    `json:"slotsTotal"`
}

type EnrollTeamInput struct {
	TeamID uuid.UUID `json:"teamId"`
}

type ChampionshipResponse struct {
	ID         uuid.UUID          `json:"id"`
	Name       string             `json:"name"`
	SlotsTotal int                `json:"slotsTotal"`
	SlotsTaken int                `json:"slotsTaken"`
	Status     ChampionshipStatus `json:"status"`
	CreatedAt  time.Time          `json:"createdAt"`
}

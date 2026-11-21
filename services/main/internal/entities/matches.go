package entities

import (
	"time"

	"github.com/google/uuid"
)

type MatchStatus string

const (
	MatchScheduled  MatchStatus = "SCHEDULED"
	MatchInProgress MatchStatus = "IN_PROGRESS"
	MatchFinished   MatchStatus = "FINISHED"
)

type Match struct {
	ID             uuid.UUID
	ChampionshipID uuid.UUID
	HomeTeamID     uuid.UUID
	AwayTeamID     uuid.UUID
	ScheduledAt    time.Time
	Status         MatchStatus
	HomeScore      int
	AwayScore      int
	Sequence       int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type MatchProgress struct {
	Sequence  int
	HomeScore int
	AwayScore int
}

type CreateMatchInput struct {
	ChampionshipID uuid.UUID `json:"championshipId"`
	HomeTeamID     uuid.UUID `json:"homeTeamId"`
	AwayTeamID     uuid.UUID `json:"awayTeamId"`
	ScheduledAt    time.Time `json:"scheduledAt"`
}

type RescheduleMatchInput struct {
	ScheduledAt time.Time `json:"scheduledAt"`
}

type MatchResponse struct {
	ID             uuid.UUID   `json:"id"`
	ChampionshipID uuid.UUID   `json:"championshipId"`
	HomeTeamID     uuid.UUID   `json:"homeTeamId"`
	AwayTeamID     uuid.UUID   `json:"awayTeamId"`
	ScheduledAt    time.Time   `json:"scheduledAt"`
	Status         MatchStatus `json:"status"`
	HomeScore      int         `json:"homeScore"`
	AwayScore      int         `json:"awayScore"`
	CreatedAt      time.Time   `json:"createdAt"`
}

package entities

import (
	"time"

	"github.com/google/uuid"
)

const (
	MessageMatchStarted         = "MatchStarted"
	MessageMatchEventRegistered = "MatchEventRegistered"
	MessageMatchEventRemoved    = "MatchEventRemoved"
	MessageMatchFinished        = "MatchFinished"

	RoutingKeyMatchStarted         = "match.started"
	RoutingKeyMatchEventRegistered = "match.event-registered"
	RoutingKeyMatchEventRemoved    = "match.event-removed"
	RoutingKeyMatchFinished        = "match.finished"
)

type MatchScore struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

type MatchStartedPayload struct {
	MatchID      uuid.UUID      `json:"matchId"`
	Sequence     int            `json:"sequence"`
	Championship NamedReference `json:"championship"`
	HomeTeam     NamedReference `json:"homeTeam"`
	AwayTeam     NamedReference `json:"awayTeam"`
	ScheduledAt  time.Time      `json:"scheduledAt"`
	StartedAt    time.Time      `json:"startedAt"`
	Score        MatchScore     `json:"score"`
}

type MatchEventRegisteredPayload struct {
	MatchID  uuid.UUID          `json:"matchId"`
	Sequence int                `json:"sequence"`
	Event    MatchEventResponse `json:"event"`
	Score    MatchScore         `json:"score"`
}

type MatchEventRemovedPayload struct {
	MatchID  uuid.UUID  `json:"matchId"`
	Sequence int        `json:"sequence"`
	EventID  uuid.UUID  `json:"eventId"`
	Score    MatchScore `json:"score"`
}

type MatchFinishedPayload struct {
	MatchID    uuid.UUID  `json:"matchId"`
	Sequence   int        `json:"sequence"`
	FinishedAt time.Time  `json:"finishedAt"`
	Score      MatchScore `json:"score"`
}

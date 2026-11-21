package live

import (
	"encoding/json"
	"time"
)

const (
	MessageMatchStarted         = "MatchStarted"
	MessageMatchEventRegistered = "MatchEventRegistered"
	MessageMatchEventRemoved    = "MatchEventRemoved"
	MessageMatchFinished        = "MatchFinished"

	UpdateMatchState = "MatchState"
)

var highlightTypes = map[string]struct{}{
	"GOAL":        {},
	"OWN_GOAL":    {},
	"YELLOW_CARD": {},
	"RED_CARD":    {},
}

type Envelope struct {
	MessageID     string          `json:"messageId"`
	CorrelationID string          `json:"correlationId"`
	Type          string          `json:"type"`
	Version       int             `json:"version"`
	OccurredAt    time.Time       `json:"occurredAt"`
	Payload       json.RawMessage `json:"payload"`
}

type NamedReference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Score struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

type Event struct {
	ID              string          `json:"id"`
	Sequence        int             `json:"sequence"`
	Type            string          `json:"type"`
	Minute          int             `json:"minute"`
	TeamID          string          `json:"teamId"`
	Player          *NamedReference `json:"player"`
	SecondaryPlayer *NamedReference `json:"secondaryPlayer"`
	RegisteredAt    time.Time       `json:"registeredAt"`
}

type MatchState struct {
	MatchID      string         `json:"matchId"`
	Sequence     int            `json:"sequence"`
	Championship NamedReference `json:"championship"`
	HomeTeam     NamedReference `json:"homeTeam"`
	AwayTeam     NamedReference `json:"awayTeam"`
	ScheduledAt  time.Time      `json:"scheduledAt"`
	StartedAt    time.Time      `json:"startedAt"`
	Score        Score          `json:"score"`
	Events       []Event        `json:"events"`
}

type Summary struct {
	MatchID      string         `json:"matchId"`
	Sequence     int            `json:"sequence"`
	Championship NamedReference `json:"championship"`
	HomeTeam     NamedReference `json:"homeTeam"`
	AwayTeam     NamedReference `json:"awayTeam"`
	StartedAt    time.Time      `json:"startedAt"`
	Score        Score          `json:"score"`
	Highlights   []Event        `json:"highlights"`
	GeneratedAt  time.Time      `json:"generatedAt"`
}

type startedPayload struct {
	MatchID      string         `json:"matchId"`
	Sequence     int            `json:"sequence"`
	Championship NamedReference `json:"championship"`
	HomeTeam     NamedReference `json:"homeTeam"`
	AwayTeam     NamedReference `json:"awayTeam"`
	ScheduledAt  time.Time      `json:"scheduledAt"`
	StartedAt    time.Time      `json:"startedAt"`
	Score        Score          `json:"score"`
}

type registeredPayload struct {
	MatchID  string `json:"matchId"`
	Sequence int    `json:"sequence"`
	Event    Event  `json:"event"`
	Score    Score  `json:"score"`
}

type removedPayload struct {
	MatchID  string `json:"matchId"`
	Sequence int    `json:"sequence"`
	EventID  string `json:"eventId"`
	Score    Score  `json:"score"`
}

type finishedPayload struct {
	MatchID    string    `json:"matchId"`
	Sequence   int       `json:"sequence"`
	FinishedAt time.Time `json:"finishedAt"`
	Score      Score     `json:"score"`
}

func newState(payload startedPayload) *MatchState {
	return &MatchState{
		MatchID:      payload.MatchID,
		Sequence:     payload.Sequence,
		Championship: payload.Championship,
		HomeTeam:     payload.HomeTeam,
		AwayTeam:     payload.AwayTeam,
		ScheduledAt:  payload.ScheduledAt,
		StartedAt:    payload.StartedAt,
		Score:        payload.Score,
		Events:       []Event{},
	}
}

func (s *MatchState) withEvent(payload registeredPayload) {
	s.Events = append(s.Events, payload.Event)
	s.Score = payload.Score
	s.Sequence = payload.Sequence
}

func (s *MatchState) withoutEvent(payload removedPayload) {
	kept := make([]Event, 0, len(s.Events))
	for _, event := range s.Events {
		if event.ID != payload.EventID {
			kept = append(kept, event)
		}
	}
	s.Events = kept
	s.Score = payload.Score
	s.Sequence = payload.Sequence
}

func (s *MatchState) summary(generatedAt time.Time) Summary {
	highlights := []Event{}
	for _, event := range s.Events {
		if _, isHighlight := highlightTypes[event.Type]; isHighlight {
			highlights = append(highlights, event)
		}
	}
	return Summary{
		MatchID:      s.MatchID,
		Sequence:     s.Sequence,
		Championship: s.Championship,
		HomeTeam:     s.HomeTeam,
		AwayTeam:     s.AwayTeam,
		StartedAt:    s.StartedAt,
		Score:        s.Score,
		Highlights:   highlights,
		GeneratedAt:  generatedAt,
	}
}

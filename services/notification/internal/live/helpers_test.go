package live

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
)

const contractsDirectory = "../../../../contracts"

var (
	homeTeamID = uuid.NewString()
	awayTeamID = uuid.NewString()
)

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestStore(t *testing.T) (*RedisStore, *miniredis.Miniredis) {
	t.Helper()
	server := miniredis.RunT(t)
	store, err := NewRedisStore("redis://"+server.Addr(), time.Hour, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, server
}

func envelopeOf(t *testing.T, messageType string, payload map[string]any) Envelope {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return Envelope{
		MessageID:     uuid.NewString(),
		CorrelationID: uuid.NewString(),
		Type:          messageType,
		Version:       1,
		OccurredAt:    time.Now().UTC(),
		Payload:       raw,
	}
}

func rawEnvelope(t *testing.T, envelope Envelope) []byte {
	t.Helper()
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func startedMessage(t *testing.T, matchID string) Envelope {
	return envelopeOf(t, MessageMatchStarted, map[string]any{
		"matchId":      matchID,
		"sequence":     1,
		"championship": map[string]any{"id": uuid.NewString(), "name": "Copa Furb"},
		"homeTeam":     map[string]any{"id": homeTeamID, "name": "Home FC"},
		"awayTeam":     map[string]any{"id": awayTeamID, "name": "Away FC"},
		"scheduledAt":  time.Now().UTC().Format(time.RFC3339),
		"startedAt":    time.Now().UTC().Format(time.RFC3339),
		"score":        map[string]any{"home": 0, "away": 0},
	})
}

func goalMessage(t *testing.T, matchID, eventID string, sequence, home, away int) Envelope {
	return eventMessage(t, matchID, eventID, "GOAL", sequence, home, away)
}

func eventMessage(t *testing.T, matchID, eventID, eventType string, sequence, home, away int) Envelope {
	return envelopeOf(t, MessageMatchEventRegistered, map[string]any{
		"matchId":  matchID,
		"sequence": sequence,
		"event": map[string]any{
			"id":              eventID,
			"sequence":        sequence,
			"type":            eventType,
			"minute":          23,
			"teamId":          homeTeamID,
			"player":          map[string]any{"id": uuid.NewString(), "name": "Pele"},
			"secondaryPlayer": nil,
			"registeredAt":    time.Now().UTC().Format(time.RFC3339),
		},
		"score": map[string]any{"home": home, "away": away},
	})
}

func removedMessage(t *testing.T, matchID, eventID string, sequence, home, away int) Envelope {
	return envelopeOf(t, MessageMatchEventRemoved, map[string]any{
		"matchId": matchID, "sequence": sequence, "eventId": eventID,
		"score": map[string]any{"home": home, "away": away},
	})
}

func finishedMessage(t *testing.T, matchID string, sequence, home, away int) Envelope {
	return envelopeOf(t, MessageMatchFinished, map[string]any{
		"matchId": matchID, "sequence": sequence,
		"finishedAt": time.Now().UTC().Format(time.RFC3339),
		"score":      map[string]any{"home": home, "away": away},
	})
}

type memoryDeduplicator struct {
	mutex sync.Mutex
	seen  map[string]bool
}

func newMemoryDeduplicator() *memoryDeduplicator {
	return &memoryDeduplicator{seen: map[string]bool{}}
}

func (d *memoryDeduplicator) RunOnce(ctx context.Context, messageID string, effect func(ctx context.Context) error) (bool, error) {
	d.mutex.Lock()
	alreadyProcessed := d.seen[messageID]
	d.mutex.Unlock()
	if alreadyProcessed {
		return false, nil
	}
	if err := effect(ctx); err != nil {
		return false, err
	}
	d.mutex.Lock()
	d.seen[messageID] = true
	d.mutex.Unlock()
	return true, nil
}

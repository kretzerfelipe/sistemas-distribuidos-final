package entities

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const contractsDirectory = "../../../../contracts/events"

func envelopeAround(messageType string, payload any) []byte {
	body, _ := json.Marshal(map[string]any{
		"messageId":     uuid.NewString(),
		"correlationId": uuid.NewString(),
		"type":          messageType,
		"version":       1,
		"occurredAt":    time.Now().UTC().Format(time.RFC3339Nano),
		"payload":       payload,
	})
	return body
}

func validateAgainstContract(t *testing.T, schemaFile string, document []byte) error {
	t.Helper()
	absolute, err := filepath.Abs(filepath.Join(contractsDirectory, schemaFile))
	if err != nil {
		t.Fatal(err)
	}

	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	schema, err := compiler.Compile(absolute)
	if err != nil {
		t.Fatalf("the contract %s does not compile: %v", schemaFile, err)
	}

	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(document))
	if err != nil {
		t.Fatal(err)
	}
	return schema.Validate(instance)
}

func sampleReference(name string) NamedReference {
	return NamedReference{ID: uuid.New(), Name: name}
}

func sampleEvent(withPlayers bool) MatchEventResponse {
	event := MatchEventResponse{
		ID: uuid.New(), Sequence: 2, Type: EventGoal, Minute: 23, TeamID: uuid.New(), RegisteredAt: time.Now().UTC(),
	}
	if withPlayers {
		player, secondary := sampleReference("Pele"), sampleReference("Garrincha")
		event.Player, event.SecondaryPlayer = &player, &secondary
	}
	return event
}

func TestMatchStartedFollowsItsContract(t *testing.T) {
	payload := MatchStartedPayload{
		MatchID: uuid.New(), Sequence: 1, Championship: sampleReference("Copa Furb"),
		HomeTeam: sampleReference("Home FC"), AwayTeam: sampleReference("Away FC"),
		ScheduledAt: time.Now().UTC(), StartedAt: time.Now().UTC(),
	}

	err := validateAgainstContract(t, "MatchStarted.schema.json", envelopeAround(MessageMatchStarted, payload))

	if err != nil {
		t.Errorf("MatchStarted breaks its contract: %v", err)
	}
}

func TestMatchEventRegisteredFollowsItsContractWithAndWithoutPlayers(t *testing.T) {
	for name, withPlayers := range map[string]bool{"with players": true, "without players": false} {
		t.Run(name, func(t *testing.T) {
			payload := MatchEventRegisteredPayload{
				MatchID: uuid.New(), Sequence: 2, Event: sampleEvent(withPlayers), Score: MatchScore{Home: 1},
			}

			err := validateAgainstContract(t, "MatchEventRegistered.schema.json", envelopeAround(MessageMatchEventRegistered, payload))

			if err != nil {
				t.Errorf("MatchEventRegistered breaks its contract: %v", err)
			}
		})
	}
}

func TestMatchEventRemovedFollowsItsContract(t *testing.T) {
	payload := MatchEventRemovedPayload{MatchID: uuid.New(), Sequence: 4, EventID: uuid.New(), Score: MatchScore{Home: 1, Away: 1}}

	err := validateAgainstContract(t, "MatchEventRemoved.schema.json", envelopeAround(MessageMatchEventRemoved, payload))

	if err != nil {
		t.Errorf("MatchEventRemoved breaks its contract: %v", err)
	}
}

func TestMatchFinishedFollowsItsContract(t *testing.T) {
	payload := MatchFinishedPayload{MatchID: uuid.New(), Sequence: 9, FinishedAt: time.Now().UTC(), Score: MatchScore{Home: 3, Away: 1}}

	err := validateAgainstContract(t, "MatchFinished.schema.json", envelopeAround(MessageMatchFinished, payload))

	if err != nil {
		t.Errorf("MatchFinished breaks its contract: %v", err)
	}
}

func TestEveryEventTypeOfThePlayIsAcceptedByTheContract(t *testing.T) {
	allTypes := []MatchEventType{
		EventGoal, EventOwnGoal, EventShot, EventSave, EventFoul, EventYellowCard,
		EventRedCard, EventSubstitution, EventCorner, EventOffside,
	}

	for _, eventType := range allTypes {
		t.Run(string(eventType), func(t *testing.T) {
			event := sampleEvent(false)
			event.Type = eventType
			payload := MatchEventRegisteredPayload{MatchID: uuid.New(), Sequence: 2, Event: event}

			err := validateAgainstContract(t, "MatchEventRegistered.schema.json", envelopeAround(MessageMatchEventRegistered, payload))

			if err != nil {
				t.Errorf("%s is not accepted by the contract: %v", eventType, err)
			}
		})
	}
}

func TestTheContractRejectsMessagesThatBreakIt(t *testing.T) {
	scenarios := map[string]struct {
		schema   string
		document []byte
	}{
		"missing score": {"MatchFinished.schema.json", envelopeAround(MessageMatchFinished,
			map[string]any{"matchId": uuid.NewString(), "sequence": 3, "finishedAt": time.Now().UTC().Format(time.RFC3339)})},
		"sequence zero": {"MatchEventRemoved.schema.json", envelopeAround(MessageMatchEventRemoved,
			map[string]any{"matchId": uuid.NewString(), "sequence": 0, "eventId": uuid.NewString(), "score": map[string]int{"home": 0, "away": 0}})},
		"negative score": {"MatchFinished.schema.json", envelopeAround(MessageMatchFinished,
			map[string]any{"matchId": uuid.NewString(), "sequence": 3, "finishedAt": time.Now().UTC().Format(time.RFC3339), "score": map[string]int{"home": -1, "away": 0}})},
		"wrong type for the schema": {"MatchStarted.schema.json", envelopeAround(MessageMatchFinished,
			map[string]any{"matchId": uuid.NewString()})},
		"unknown extra field": {"MatchEventRemoved.schema.json", envelopeAround(MessageMatchEventRemoved,
			map[string]any{"matchId": uuid.NewString(), "sequence": 2, "eventId": uuid.NewString(), "score": map[string]int{"home": 0, "away": 0}, "extra": true})},
	}

	for name, scenario := range scenarios {
		t.Run(name, func(t *testing.T) {
			err := validateAgainstContract(t, scenario.schema, scenario.document)

			if err == nil {
				t.Error("the contract must reject this message")
			}
		})
	}
}

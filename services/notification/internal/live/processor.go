package live

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"futhub/notification/internal/messaging"
)

var ErrSequenceGap = errors.New("a previous message of this match has not been applied yet")

type Processor struct {
	store Store
	hub   *Hub
	now   func() time.Time
	log   *slog.Logger
}

func NewProcessor(store Store, hub *Hub, log *slog.Logger) *Processor {
	return &Processor{store: store, hub: hub, now: time.Now, log: log}
}

func (p *Processor) Process(ctx context.Context, envelope Envelope) error {
	var header struct {
		MatchID  string `json:"matchId"`
		Sequence int    `json:"sequence"`
	}
	if err := json.Unmarshal(envelope.Payload, &header); err != nil {
		return fmt.Errorf("%w: payload header: %v", messaging.ErrInvalidMessage, err)
	}

	state, err := p.store.LoadState(ctx, header.MatchID)
	if err != nil {
		return err
	}

	lastApplied := 0
	if state != nil {
		lastApplied = state.Sequence
	} else {
		alreadyFinished, err := p.store.WasFinished(ctx, header.MatchID)
		if err != nil {
			return err
		}
		if alreadyFinished {
			p.logDuplicate(envelope, header.Sequence, lastApplied)
			return nil
		}
	}

	if header.Sequence <= lastApplied {
		p.logDuplicate(envelope, header.Sequence, lastApplied)
		return nil
	}
	messageIsOutOfOrder := header.Sequence != lastApplied+1
	matchHasNotStarted := state == nil && envelope.Type != MessageMatchStarted
	if messageIsOutOfOrder || matchHasNotStarted {
		return fmt.Errorf("%w: match %s, last applied %d, received %d", ErrSequenceGap, header.MatchID, lastApplied, header.Sequence)
	}

	switch envelope.Type {
	case MessageMatchStarted:
		return p.applyStarted(ctx, envelope)
	case MessageMatchEventRegistered:
		return p.applyRegistered(ctx, state, envelope)
	case MessageMatchEventRemoved:
		return p.applyRemoved(ctx, state, envelope)
	case MessageMatchFinished:
		return p.applyFinished(ctx, envelope, header.MatchID, header.Sequence)
	default:
		return fmt.Errorf("%w: unsupported message type %q", messaging.ErrInvalidMessage, envelope.Type)
	}
}

func (p *Processor) applyStarted(ctx context.Context, envelope Envelope) error {
	var payload startedPayload
	if err := decode(envelope, &payload); err != nil {
		return err
	}
	state := newState(payload)
	if err := p.store.SaveState(ctx, state); err != nil {
		return err
	}
	if err := p.store.SaveSummary(ctx, state.summary(p.now())); err != nil {
		return err
	}
	p.hub.Publish(update(envelope, payload.MatchID, payload.Sequence, false))
	return nil
}

func (p *Processor) applyRegistered(ctx context.Context, state *MatchState, envelope Envelope) error {
	var payload registeredPayload
	if err := decode(envelope, &payload); err != nil {
		return err
	}
	state.withEvent(payload)
	if err := p.store.SaveState(ctx, state); err != nil {
		return err
	}
	p.hub.Publish(update(envelope, payload.MatchID, payload.Sequence, false))
	return nil
}

func (p *Processor) applyRemoved(ctx context.Context, state *MatchState, envelope Envelope) error {
	var payload removedPayload
	if err := decode(envelope, &payload); err != nil {
		return err
	}
	state.withoutEvent(payload)
	if err := p.store.SaveState(ctx, state); err != nil {
		return err
	}
	p.hub.Publish(update(envelope, payload.MatchID, payload.Sequence, false))
	return nil
}

func (p *Processor) applyFinished(ctx context.Context, envelope Envelope, matchID string, sequence int) error {
	if err := p.store.FinishMatch(ctx, matchID, sequence); err != nil {
		return err
	}
	p.hub.Publish(update(envelope, matchID, sequence, true))
	return nil
}

func (p *Processor) logDuplicate(envelope Envelope, received, lastApplied int) {
	p.log.Info("duplicate sequence ignored",
		"messageId", envelope.MessageID,
		"correlationId", envelope.CorrelationID,
		"type", envelope.Type,
		"sequence", received,
		"lastApplied", lastApplied,
	)
}

func decode(envelope Envelope, target any) error {
	if err := json.Unmarshal(envelope.Payload, target); err != nil {
		return fmt.Errorf("%w: %s payload: %v", messaging.ErrInvalidMessage, envelope.Type, err)
	}
	return nil
}

func update(envelope Envelope, matchID string, sequence int, final bool) Update {
	return Update{Type: envelope.Type, MatchID: matchID, Sequence: sequence, Payload: envelope.Payload, Final: final}
}

package messaging

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

func newTestBroker(url string) *Broker {
	return NewBroker(url, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestPublishFailsFastWhenTheBrokerIsNotConnected(t *testing.T) {
	broker := newTestBroker("amqp://guest:guest@localhost:1/")

	err := broker.Publish(context.Background(), "futhub.events", "match.started", "id", []byte("{}"))

	if !errors.Is(err, ErrBrokerUnavailable) {
		t.Errorf("want ErrBrokerUnavailable, got %v", err)
	}
	if broker.IsConnected() {
		t.Error("a broker that never connected must report as disconnected")
	}
}

func TestRunKeepsTryingAndStopsWhenTheContextIsCancelled(t *testing.T) {
	broker := newTestBroker("amqp://guest:guest@localhost:1/")
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})

	go func() {
		broker.Run(ctx)
		close(finished)
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("Run must stop soon after the context is cancelled")
	}
	if broker.IsConnected() {
		t.Error("an unreachable broker must never look connected")
	}
}

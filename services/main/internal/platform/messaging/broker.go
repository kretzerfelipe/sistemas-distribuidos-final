package messaging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	initialBackoff = time.Second
	maxBackoff     = 30 * time.Second
	dialTimeout    = 5 * time.Second
	heartbeat      = 10 * time.Second
	confirmTimeout = 5 * time.Second
	returnsBuffer  = 16
)

var (
	ErrBrokerUnavailable = errors.New("message broker is unavailable")
	ErrNotConfirmed      = errors.New("message broker did not confirm the message")
	ErrMessageUnroutable = errors.New("message broker could not route the message to any queue")
)

type session struct {
	connection    *amqp.Connection
	channel       *amqp.Channel
	confirms      <-chan amqp.Confirmation
	returns       <-chan amqp.Return
	connClosed    <-chan *amqp.Error
	channelClosed <-chan *amqp.Error
}

type Broker struct {
	url       string
	log       *slog.Logger
	publishMu sync.Mutex
	sessionMu sync.RWMutex
	current   *session
}

func NewBroker(url string, log *slog.Logger) *Broker {
	return &Broker{url: url, log: log}
}

func (b *Broker) Run(ctx context.Context) {
	backoff := initialBackoff

	for ctx.Err() == nil {
		opened, err := b.open()
		if err != nil {
			b.log.Warn("message broker unavailable, retrying", "error", err, "retry_in", backoff.String())
			if !sleep(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		backoff = initialBackoff
		b.setSession(opened)
		b.log.Info("message broker connected")

		opened.waitUntilClosed(ctx)
		b.setSession(nil)
		opened.close()

		connectionWasLost := ctx.Err() == nil
		if connectionWasLost {
			b.log.Warn("message broker connection lost, reconnecting")
		}
	}
}

func (b *Broker) IsConnected() bool {
	b.sessionMu.RLock()
	defer b.sessionMu.RUnlock()
	return b.current != nil
}

func (b *Broker) Publish(ctx context.Context, exchange, routingKey, messageID string, body []byte) error {
	b.publishMu.Lock()
	defer b.publishMu.Unlock()

	opened := b.currentSession()
	brokerIsDisconnected := opened == nil
	if brokerIsDisconnected {
		return ErrBrokerUnavailable
	}

	opened.discardStaleReturns()

	err := opened.channel.PublishWithContext(ctx, exchange, routingKey, true, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    messageID,
		Timestamp:    time.Now().UTC(),
		Body:         body,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBrokerUnavailable, err)
	}
	return opened.waitForConfirmation(ctx)
}

func (b *Broker) open() (*session, error) {
	connection, err := amqp.DialConfig(b.url, amqp.Config{Heartbeat: heartbeat, Dial: amqp.DefaultDial(dialTimeout)})
	if err != nil {
		return nil, err
	}

	channel, err := connection.Channel()
	if err != nil {
		_ = connection.Close()
		return nil, err
	}

	err = channel.Confirm(false)
	if err != nil {
		_ = connection.Close()
		return nil, err
	}

	return &session{
		connection:    connection,
		channel:       channel,
		confirms:      channel.NotifyPublish(make(chan amqp.Confirmation, 1)),
		returns:       channel.NotifyReturn(make(chan amqp.Return, returnsBuffer)),
		connClosed:    connection.NotifyClose(make(chan *amqp.Error, 1)),
		channelClosed: channel.NotifyClose(make(chan *amqp.Error, 1)),
	}, nil
}

func (b *Broker) setSession(opened *session) {
	b.sessionMu.Lock()
	defer b.sessionMu.Unlock()
	b.current = opened
}

func (b *Broker) currentSession() *session {
	b.sessionMu.RLock()
	defer b.sessionMu.RUnlock()
	return b.current
}

func (s *session) waitUntilClosed(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-s.connClosed:
	case <-s.channelClosed:
	}
}

func (s *session) close() {
	_ = s.connection.Close()
}

func (s *session) discardStaleReturns() {
	for {
		select {
		case <-s.returns:
		default:
			return
		}
	}
}

func (s *session) waitForConfirmation(ctx context.Context) error {
	select {
	case confirmation, open := <-s.confirms:
		confirmationWasNotAcknowledged := !open || !confirmation.Ack
		if confirmationWasNotAcknowledged {
			return ErrNotConfirmed
		}
	case <-time.After(confirmTimeout):
		return ErrNotConfirmed
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case <-s.returns:
		return ErrMessageUnroutable
	default:
		return nil
	}
}

func sleep(ctx context.Context, duration time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(duration):
		return true
	}
}

package messaging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	retryCountHeader    = "x-retry-count"
	failureReasonHeader = "x-failure-reason"
)

var ErrInvalidMessage = errors.New("invalid message")

type Handler interface {
	Handle(ctx context.Context, body []byte) error
}

type ConsumerSettings struct {
	URL             string
	Queue           string
	RetryQueue      string
	DeadLetterQueue string
	Prefetch        int
	MaxRetries      int
}

type Consumer struct {
	settings  ConsumerSettings
	handler   Handler
	log       *slog.Logger
	connected atomic.Bool
}

func NewConsumer(settings ConsumerSettings, handler Handler, log *slog.Logger) *Consumer {
	return &Consumer{settings: settings, handler: handler, log: log}
}

func (c *Consumer) IsConnected() bool {
	return c.connected.Load()
}

func (c *Consumer) Run(ctx context.Context) {
	backoff := initialBackoff

	for ctx.Err() == nil {
		err := c.consumeUntilDisconnected(ctx, &backoff)
		c.connected.Store(false)
		if ctx.Err() != nil {
			return
		}

		c.log.Warn("message consumer disconnected, retrying", "error", err, "retry_in", backoff.String())
		if !sleep(ctx, backoff) {
			return
		}
		backoff = min(backoff*2, maxBackoff)
	}
}

func (c *Consumer) consumeUntilDisconnected(ctx context.Context, backoff *time.Duration) error {
	connection, err := amqp.DialConfig(c.settings.URL, amqp.Config{Heartbeat: heartbeat, Dial: amqp.DefaultDial(dialTimeout)})
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer connection.Close()

	channel, err := connection.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	if err := channel.Qos(c.settings.Prefetch, 0, false); err != nil {
		return fmt.Errorf("set qos: %w", err)
	}
	if err := channel.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirms: %w", err)
	}
	deliveries, err := channel.Consume(c.settings.Queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume %s: %w", c.settings.Queue, err)
	}

	connectionClosed := connection.NotifyClose(make(chan *amqp.Error, 1))
	channelClosed := channel.NotifyClose(make(chan *amqp.Error, 1))

	*backoff = initialBackoff
	c.connected.Store(true)
	c.log.Info("message consumer connected", "queue", c.settings.Queue)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case closeError := <-connectionClosed:
			return fmt.Errorf("connection closed: %v", closeError)
		case closeError := <-channelClosed:
			return fmt.Errorf("channel closed: %v", closeError)
		case delivery, open := <-deliveries:
			if !open {
				return errors.New("delivery channel closed")
			}
			c.process(ctx, channel, delivery)
		}
	}
}

func (c *Consumer) process(ctx context.Context, channel *amqp.Channel, delivery amqp.Delivery) {
	err := c.handler.Handle(ctx, delivery.Body)
	if err == nil {
		_ = delivery.Ack(false)
		return
	}

	if ctx.Err() != nil {
		_ = delivery.Nack(false, true)
		return
	}

	log := c.log.With("messageId", delivery.MessageId, "error", err)

	messageIsInvalid := errors.Is(err, ErrInvalidMessage)
	if messageIsInvalid {
		log.Error("invalid message, sending to the dead letter queue")
		c.moveTo(ctx, channel, delivery, c.settings.DeadLetterQueue, err)
		return
	}

	retries := retryCountOf(delivery)
	retriesAreExhausted := retries >= c.settings.MaxRetries
	if retriesAreExhausted {
		log.Error("message failed too many times, sending to the dead letter queue", "retries", retries)
		c.moveTo(ctx, channel, delivery, c.settings.DeadLetterQueue, err)
		return
	}

	log.Warn("message could not be processed, it will be retried", "retries", retries, "queue", c.settings.RetryQueue)
	c.moveTo(ctx, channel, delivery, c.settings.RetryQueue, err)
}

func (c *Consumer) moveTo(ctx context.Context, channel *amqp.Channel, delivery amqp.Delivery, queue string, reason error) {
	headers := amqp.Table{}
	for key, value := range delivery.Headers {
		headers[key] = value
	}
	headers[retryCountHeader] = int64(retryCountOf(delivery) + 1)
	headers[failureReasonHeader] = reason.Error()

	publishCtx, cancel := context.WithTimeout(ctx, confirmTimeout)
	defer cancel()

	confirmation, err := channel.PublishWithDeferredConfirmWithContext(publishCtx, "", queue, false, false, amqp.Publishing{
		ContentType:  delivery.ContentType,
		DeliveryMode: amqp.Persistent,
		MessageId:    delivery.MessageId,
		Timestamp:    delivery.Timestamp,
		Headers:      headers,
		Body:         delivery.Body,
	})
	if err == nil {
		var acknowledged bool
		acknowledged, err = confirmation.WaitContext(publishCtx)
		brokerDidNotConfirm := err == nil && !acknowledged
		if brokerDidNotConfirm {
			err = ErrNotConfirmed
		}
	}
	if err != nil {
		c.log.Error("could not move message, returning it to the queue", "messageId", delivery.MessageId, "target", queue, "error", err)
		_ = delivery.Nack(false, true)
		return
	}
	_ = delivery.Ack(false)
}

func retryCountOf(delivery amqp.Delivery) int {
	switch count := delivery.Headers[retryCountHeader].(type) {
	case int64:
		return int(count)
	case int32:
		return int(count)
	case int:
		return count
	default:
		return 0
	}
}

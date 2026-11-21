package outbox

import (
	"context"
	"log/slog"
	"time"
)

type MessagePublisher interface {
	Publish(ctx context.Context, exchange, routingKey, messageID string, body []byte) error
}

type Publisher struct {
	store     Store
	broker    MessagePublisher
	exchange  string
	interval  time.Duration
	batchSize int
	log       *slog.Logger
	now       func() time.Time
	isFailing bool
}

func NewPublisher(store Store, broker MessagePublisher, exchange string, interval time.Duration, batchSize int, log *slog.Logger) *Publisher {
	return &Publisher{
		store:     store,
		broker:    broker,
		exchange:  exchange,
		interval:  interval,
		batchSize: batchSize,
		log:       log,
		now:       time.Now,
	}
}

func (p *Publisher) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.drain(ctx)
		}
	}
}

func (p *Publisher) drain(ctx context.Context) {
	for ctx.Err() == nil {
		published, err := p.publishBatch(ctx)
		if err != nil {
			p.reportFailure(err)
			return
		}

		nothingWasPublished := published == 0
		if nothingWasPublished {
			return
		}
		p.reportRecovery()

		batchWasNotFull := published < p.batchSize
		if batchWasNotFull {
			return
		}
	}
}

func (p *Publisher) publishBatch(ctx context.Context) (int, error) {
	pending, err := p.store.FetchUnpublished(ctx, p.batchSize)
	if err != nil {
		return 0, err
	}

	for position, message := range pending {
		err := p.broker.Publish(ctx, p.exchange, message.RoutingKey, message.ID.String(), message.Body)
		if err != nil {
			return position, err
		}

		err = p.store.MarkPublished(ctx, message.ID, p.now().UTC())
		if err != nil {
			return position, err
		}
	}
	return len(pending), nil
}

func (p *Publisher) reportFailure(err error) {
	alreadyReported := p.isFailing
	if alreadyReported {
		return
	}
	p.isFailing = true
	p.log.Warn("outbox could not publish, messages stay stored and will be retried", "error", err)
}

func (p *Publisher) reportRecovery() {
	wasNotFailing := !p.isFailing
	if wasNotFailing {
		return
	}
	p.isFailing = false
	p.log.Info("outbox publishing recovered")
}

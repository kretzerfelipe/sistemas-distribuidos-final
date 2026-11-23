package subscriptions

import (
	"context"
	"log/slog"
	"time"
)

func (m *Module) RunExpiration(ctx context.Context, interval time.Duration, log *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expired, err := m.uc.ExpirePending(ctx)
			if err != nil {
				log.Error("could not expire pending subscriptions", "error", err)
				continue
			}
			if expired > 0 {
				log.Info("pending subscriptions expired", "count", expired)
			}
		}
	}
}

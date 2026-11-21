package live

import (
	"context"
	"log/slog"
	"time"
)

type SnapshotJob struct {
	store    Store
	interval time.Duration
	now      func() time.Time
	log      *slog.Logger
}

func NewSnapshotJob(store Store, interval time.Duration, log *slog.Logger) *SnapshotJob {
	return &SnapshotJob{store: store, interval: interval, now: time.Now, log: log}
}

func (j *SnapshotJob) Run(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.RunOnce(ctx)
		}
	}
}

func (j *SnapshotJob) RunOnce(ctx context.Context) {
	matchIDs, err := j.store.LiveMatchIDs(ctx)
	if err != nil {
		j.log.Warn("snapshot job could not list live matches", "error", err)
		return
	}

	for _, matchID := range matchIDs {
		state, err := j.store.LoadState(ctx, matchID)
		if err != nil {
			j.log.Warn("snapshot job could not read match", "matchId", matchID, "error", err)
			continue
		}
		matchFinishedMeanwhile := state == nil
		if matchFinishedMeanwhile {
			continue
		}
		if err := j.store.SaveSummary(ctx, state.summary(j.now())); err != nil {
			j.log.Warn("snapshot job could not save summary", "matchId", matchID, "error", err)
		}
	}
	j.log.Debug("snapshots refreshed", "matches", len(matchIDs))
}

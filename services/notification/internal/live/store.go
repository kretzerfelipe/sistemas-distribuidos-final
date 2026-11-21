package live

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix       = "notification:match:"
	stateSuffix     = ":state"
	snapshotSuffix  = ":snapshot"
	finishedSuffix  = ":finished"
	finishedTTL     = time.Hour
	scanBatchSize   = 100
	redisDialTimout = 2 * time.Second
	redisIOTimeout  = 2 * time.Second
)

var ErrCacheUnavailable = errors.New("live cache is unavailable")

type Store interface {
	LoadState(ctx context.Context, matchID string) (*MatchState, error)
	SaveState(ctx context.Context, state *MatchState) error
	LoadSummary(ctx context.Context, matchID string) (*Summary, error)
	SaveSummary(ctx context.Context, summary Summary) error
	FinishMatch(ctx context.Context, matchID string, finalSequence int) error
	WasFinished(ctx context.Context, matchID string) (bool, error)
	LiveMatchIDs(ctx context.Context) ([]string, error)
	Ping(ctx context.Context) error
}

type RedisStore struct {
	client      *redis.Client
	stateTTL    time.Duration
	summaryTTL  time.Duration
	finishedTTL time.Duration
}

func NewRedisStore(url string, stateTTL, snapshotInterval time.Duration) (*RedisStore, error) {
	options, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	options.DialTimeout = redisDialTimout
	options.ReadTimeout = redisIOTimeout
	options.WriteTimeout = redisIOTimeout

	return &RedisStore{
		client:      redis.NewClient(options),
		stateTTL:    stateTTL,
		summaryTTL:  3 * snapshotInterval,
		finishedTTL: finishedTTL,
	}, nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

func (s *RedisStore) Ping(ctx context.Context) error {
	return wrapCacheError(s.client.Ping(ctx).Err())
}

func (s *RedisStore) LoadState(ctx context.Context, matchID string) (*MatchState, error) {
	var state MatchState
	found, err := s.getJSON(ctx, stateKey(matchID), &state)
	if err != nil || !found {
		return nil, err
	}
	return &state, nil
}

func (s *RedisStore) SaveState(ctx context.Context, state *MatchState) error {
	return s.setJSON(ctx, stateKey(state.MatchID), state, s.stateTTL)
}

func (s *RedisStore) LoadSummary(ctx context.Context, matchID string) (*Summary, error) {
	var summary Summary
	found, err := s.getJSON(ctx, snapshotKey(matchID), &summary)
	if err != nil || !found {
		return nil, err
	}
	return &summary, nil
}

func (s *RedisStore) SaveSummary(ctx context.Context, summary Summary) error {
	return s.setJSON(ctx, snapshotKey(summary.MatchID), summary, s.summaryTTL)
}

func (s *RedisStore) FinishMatch(ctx context.Context, matchID string, finalSequence int) error {
	pipeline := s.client.TxPipeline()
	pipeline.Del(ctx, stateKey(matchID), snapshotKey(matchID))
	pipeline.Set(ctx, finishedKey(matchID), finalSequence, s.finishedTTL)
	_, err := pipeline.Exec(ctx)
	return wrapCacheError(err)
}

func (s *RedisStore) WasFinished(ctx context.Context, matchID string) (bool, error) {
	exists, err := s.client.Exists(ctx, finishedKey(matchID)).Result()
	if err != nil {
		return false, wrapCacheError(err)
	}
	return exists > 0, nil
}

func (s *RedisStore) LiveMatchIDs(ctx context.Context) ([]string, error) {
	var ids []string
	iterator := s.client.Scan(ctx, 0, keyPrefix+"*"+stateSuffix, scanBatchSize).Iterator()
	for iterator.Next(ctx) {
		id := strings.TrimSuffix(strings.TrimPrefix(iterator.Val(), keyPrefix), stateSuffix)
		ids = append(ids, id)
	}
	if err := iterator.Err(); err != nil {
		return nil, wrapCacheError(err)
	}
	return ids, nil
}

func (s *RedisStore) getJSON(ctx context.Context, key string, target any) (bool, error) {
	raw, err := s.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, wrapCacheError(err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return false, fmt.Errorf("decode %s: %w", key, err)
	}
	return true, nil
}

func (s *RedisStore) setJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode %s: %w", key, err)
	}
	return wrapCacheError(s.client.Set(ctx, key, raw, ttl).Err())
}

func wrapCacheError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", ErrCacheUnavailable, err)
}

func stateKey(matchID string) string    { return keyPrefix + matchID + stateSuffix }
func snapshotKey(matchID string) string { return keyPrefix + matchID + snapshotSuffix }
func finishedKey(matchID string) string { return keyPrefix + matchID + finishedSuffix }

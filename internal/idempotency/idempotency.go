package idempotency

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// IdempotencyLayer prevents duplicate event processing.
//
// Three layers (production-tested):
//  1. Redis fast-path: SETNX, ~1ms, handles 99% of retries
//  2. DB unique index: final guarantee (not shown here)
//  3. T+1 reconciliation: catches edge cases (not shown here)
type IdempotencyLayer struct {
	rdb *redis.Client
}

func New(redisAddr string) (*IdempotencyLayer, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &IdempotencyLayer{rdb: rdb}, nil
}

// Exists checks if an event has already been processed.
// Returns true if it's a duplicate.
func (l *IdempotencyLayer) Exists(ctx context.Context, chainID int64, txHash string, logIndex uint) (bool, error) {
	key := l.key(chainID, txHash, logIndex)

	n, err := l.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}

	return n > 0, nil
}

// MarkProcessed marks an event as processed.
// TTL is 24h — long enough for retries, short enough to not bloat Redis.
func (l *IdempotencyLayer) MarkProcessed(ctx context.Context, chainID int64, txHash string, logIndex uint) error {
	key := l.key(chainID, txHash, logIndex)

	ok, err := l.rdb.SetNX(ctx, key, "1", 24*time.Hour).Result()
	if err != nil {
		return fmt.Errorf("redis setnx: %w", err)
	}

	if !ok {
		return fmt.Errorf("event already processed (race condition): %s", key)
	}

	return nil
}

func (l *IdempotencyLayer) key(chainID int64, txHash string, logIndex uint) string {
	return fmt.Sprintf("idem:%d:%s:%d", chainID, txHash, logIndex)
}

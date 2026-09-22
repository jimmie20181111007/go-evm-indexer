package idempotency

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestExists_NotProcessed(t *testing.T) {
	// Skip if no Redis available
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skip("redis not available, skipping test")
	}

	l := &IdempotencyLayer{rdb: rdb}
	ctx := context.Background()

	// Clean up
	key := l.key(1, "0xtest123", 0)
	rdb.Del(ctx, key)

	exists, err := l.Exists(ctx, 1, "0xtest123", 0)
	if err != nil {
		t.Fatalf("Exists returned error: %v", err)
	}
	if exists {
		t.Error("expected event to not exist")
	}
}

func TestMarkProcessed_AndExists(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skip("redis not available, skipping test")
	}

	l := &IdempotencyLayer{rdb: rdb}
	ctx := context.Background()

	txHash := "0xtest456"

	// Clean up
	key := l.key(1, txHash, 0)
	rdb.Del(ctx, key)

	// Mark processed
	err := l.MarkProcessed(ctx, 1, txHash, 0)
	if err != nil {
		t.Fatalf("MarkProcessed returned error: %v", err)
	}

	// Check exists
	exists, err := l.Exists(ctx, 1, txHash, 0)
	if err != nil {
		t.Fatalf("Exists returned error: %v", err)
	}
	if !exists {
		t.Error("expected event to exist after MarkProcessed")
	}

	// Clean up
	rdb.Del(ctx, key)
}

func TestMarkProcessed_Duplicate(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skip("redis not available, skipping test")
	}

	l := &IdempotencyLayer{rdb: rdb}
	ctx := context.Background()

	txHash := "0xtest789"
	key := l.key(1, txHash, 0)
	rdb.Del(ctx, key)

	// First call should succeed
	err := l.MarkProcessed(ctx, 1, txHash, 0)
	if err != nil {
		t.Fatalf("first MarkProcessed returned error: %v", err)
	}

	// Second call should fail (duplicate)
	err = l.MarkProcessed(ctx, 1, txHash, 0)
	if err == nil {
		t.Error("expected second MarkProcessed to fail with duplicate error")
	}

	// Clean up
	rdb.Del(ctx, key)
}

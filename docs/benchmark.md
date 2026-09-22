# Production Benchmark

These are real numbers from the crypto exchange this indexer was extracted from.

---

## Throughput

| Metric | Value |
|---|---|
| Peak order throughput | 8,200 orders/sec |
| Peak TPS (on-chain events processed) | ~500 events/sec |
| Order API P99 latency | ~35ms |
| Order API P99 before optimization | ~120ms |

## Why P99 matters

Before we optimized, the order API P99 was ~120ms. That means 1% of users were waiting over 120ms to place an order — in a high-frequency trading environment, that's an eternity.

The fix was the same worker pool isolation pattern used in this indexer:
- **Critical pool**: order ingestion, low latency, high priority
- **Normal pool**: trade persistence, medium priority
- **Bulk pool**: reporting, background jobs, low priority

Isolating them meant background jobs couldn't starve the critical path.

---

## Reliability

| Metric | Value |
|---|---|
| Zero duplicate deposits | 18 consecutive months |
| Reorg incidents handled | 3 (all auto-rolled back, zero user impact) |
| On-call pages per month | <1 after first 3 months |
| Indexer uptime | 99.95%+ |

## How we achieved zero duplicates

Three layers:
1. Redis fast-path handles 99% of retries
2. DB unique index catches race conditions
3. T+1 reconciliation catches edge cases

18 months with zero duplicate deposits means all three layers worked together, and the T+1 audit never found anything wrong.

---

## Latency Breakdown

```
User places order
  │
  ├─ API gateway: ~5ms
  ├─ Auth + rate limit: ~3ms
  ├─ Redis cache check: ~1ms
  ├─ Worker pool (critical): ~20ms
  ├─ DB write: ~5ms
  └─ Total P99: ~35ms
```

The biggest win was **pool isolation**. Before, a bulk reporting job running could push P99 from 35ms to 120ms just by consuming all worker goroutines.

---

## Scaling

We scaled from 1,000 orders/day to 8,200 orders/sec without a rewrite — just tuning worker pool sizes and adding Redis caching.

The architecture scales horizontally:
- Multiple indexer instances, each owning a subset of contracts
- DB is the shared state
- Redis is shared for dedup

No single point of failure (except DB, which has replication).

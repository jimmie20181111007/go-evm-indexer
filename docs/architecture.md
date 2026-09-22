# Architecture Deep Dive

## Overview

```
                ┌─────────────────────────────────────────────┐
                │                  Ethereum Node             │
                └──────────────────┬──────────────────────────┘
                                   │
                                   ▼
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  Fetcher │────▶│  Parser  │────▶│Idempotency│────▶│  Writer  │
│ (cursor) │     │ (decode) │     │ (dedup)  │     │   (DB)   │
└──────────┘     └──────────┘     └────┬─────┘     └────┬─────┘
                                        │                 │
                                        ▼                 ▼
                                  ┌──────────┐     ┌──────────┐
                                  │  Redis   │     │ Postgres │
                                  │(fast-path)│     │ (source  │
                                  └──────────┘     │  of truth)│
                                                    └──────────┘
```

## Data Flow

1. **Fetcher** pulls blocks starting from a persisted cursor
2. **Parser** decodes raw event logs into structured events
3. **Idempotency** checks if we've already processed this exact event
4. **Writer** saves new events to Postgres
5. **Reorg Detector** ensures we only process blocks that are N confirmations deep

## Why this design?

### Why three idempotency layers?

In production, things break:
- Redis restarts → fast-path loses state
- Two goroutines race → both pass the check
- Bug in logic → some edge case slips through

You need **defense in depth**:
- Layer 1 (Redis) = speed, handles 99% of retries
- Layer 2 (DB unique index) = correctness, final guarantee
- Layer 3 (T+1 reconciliation) = observability, catches anything weird

### Why cursor-based and not event subscription?

`eth_subscribe` is convenient but unreliable:
- WebSocket connections drop
- You might miss events during disconnection
- No replay guarantee

Cursor-based polling is simpler and crash-safe. If anything goes wrong, just resume from the last saved block height.

### Why safe head N confirmations behind tip?

Ethereum reorgs happen. If you credit a deposit at block 12345 and it gets reorged at block 12346, you now have an unhappy user and an accounting nightmare.

Waiting N confirmations (typically 3-12 depending on amount) means the deposit is "final" before you credit it.

## Throughput

- Fetcher: ~100 blocks/sec (light)
- Parser: ~10k logs/sec (CPU bound)
- Idempotency: ~100k ops/sec (Redis)
- Writer: ~1k inserts/sec (Postgres, batch)

Bottleneck is usually the DB writer, not the indexer itself.

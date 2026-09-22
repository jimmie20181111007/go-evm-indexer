# go-evm-indexer

Production-grade EVM event indexer in Go — extracted from a live crypto exchange platform layer that handled **8,200 orders/sec** and ran **18 months with zero duplicate deposits**.

## Why this exists

Most open-source EVM indexers are either toy projects or overly heavyweight. This one is different — it's the core logic extracted from a real production exchange that processes thousands of users' daily on-chain deposits.

It's opinionated, battle-tested, and focuses on the hard parts: **idempotency, reorg safety, and low-latency deposit crediting**.

## Features

- **Cursor-based incremental sync** — starts from any block height, no full re-scan
- **Idempotent by design** — `(chain_id, tx_hash, log_index)` as the unique key
- **Redis fast-path + DB unique index fallback** — handles 99% of retries in memory, DB is the final guarantee
- **Reorg-aware** — configurable safe head N confirmations behind the chain tip, automatic rollback
- **Hot/cold wallet tiers** — different confirmation thresholds for different amounts
- **Low latency** — the same code path that kept our order API P99 at ~35ms

## Quick Start

```bash
git clone https://github.com/jimmie20181111007/go-evm-indexer
cd go-evm-indexer
go run ./cmd/indexer \
  --rpc wss://your-ethereum-node \
  --contract 0xYourContractAddress \
  --from-block latest
```

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Fetcher   │────▶│   Parser    │────▶│ Idempotency │
│  (cursor)   │     │ (decode log)│     │  (dedup)    │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                                │
┌─────────────┐     ┌─────────────┐            │
│   Alerts    │◀────│    Writer   │◀───────────┘
│  (reorg)    │     │  (DB)       │
└─────────────┘     └─────────────┘
```

See [docs/architecture.md](docs/architecture.md) for the full design.

## Design Highlights

### 1. Idempotency: Three Layers

```
Layer 1: Redis SETNX fast-path (99% of retries hit this, ~1ms)
Layer 2: DB UNIQUE INDEX on (chain_id, tx_hash, log_index) (final guarantee)
Layer 3: T+1 reconciliation (catches edge cases)
```

Why three layers? Because in production, Redis restarts, race conditions happen, and you need a final source of truth. The DB unique constraint never lies.

### 2. Reorg Safety

We never credit deposits immediately at the chain tip. We track a "safe head" N blocks behind the tip. If a reorg deepens past N, we automatically roll back any unconfirmed deposits.

```go
// pseudocode
safeHead := latestBlock - confirmations
for block := lastProcessed; block <= safeHead; block++ {
    processDeposits(block)
}
```

### 3. Cursor Incremental Sync

No need to re-scan from genesis. We persist the last processed block height and resume from there. Restart-safe, crash-safe.

## Benchmark

These are real production numbers from the exchange this was extracted from:

| Metric | Value |
|---|---|
| Peak order throughput | 8,200 orders/sec |
| Order API P99 latency | ~35ms |
| Zero duplicate deposits | 18 consecutive months |
| Reorg detection depth | 3 confirmations |
| Supported chains | Ethereum mainnet + testnets |

See [docs/benchmark.md](docs/benchmark.md) for more details.

## Project Structure

```
go-evm-indexer/
├── cmd/
│   └── indexer/           # CLI entry point
├── internal/
│   ├── fetcher/           # Block fetching & cursor sync
│   ├── parser/           # Event log decoding
│   ├── idempotency/     # Dedup layer (Redis + DB)
│   └── reorg/           # Reorg detection & rollback
├── examples/
│   └── basic/            # Minimal working example
├── docs/
│   ├── architecture.md   # Full architecture deep-dive
│   ├── design-choices.md # Why we made each decision
│   └── benchmark.md      # Production performance data
└── .github/
    └── workflows/
        └── ci.yml        # GitHub Actions CI
```

## Why another indexer?

- You want something **simple enough to read in an afternoon** but **production-grade**
- You care about **not double-crediting** deposits (the #1 way exchanges lose money)
- You need to **handle reorgs** gracefully
- You don't want to run a full GraphQL node just to index a few events

## Contributing

PRs welcome. This is a young project but the core logic is proven.

## License

MIT

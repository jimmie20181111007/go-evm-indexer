package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jimmie20181111007/go-evm-indexer/internal/fetcher"
	"github.com/jimmie20181111007/go-evm-indexer/internal/idempotency"
	"github.com/jimmie20181111007/go-evm-indexer/internal/parser"
	"github.com/jimmie20181111007/go-evm-indexer/internal/reorg"
)

func main() {
	rpcURL := flag.String("rpc", "", "EVM RPC URL (wss:// or https://)")
	contract := flag.String("contract", "", "Contract address to index")
	fromBlock := flag.Int64("from-block", -1, "Start block (-1 = resume from cursor, or latest)")
	confirmations := flag.Int("confirmations", 3, "Safe head confirmations behind tip")
	redisAddr := flag.String("redis", "localhost:6379", "Redis address for fast-path dedup")
	pollInterval := flag.Duration("poll-interval", 2*time.Second, "Polling interval when caught up")
	flag.Parse()

	if *rpcURL == "" || *contract == "" {
		fmt.Println("Usage: go run ./cmd/indexer --rpc <url> --contract <addr>")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutting down gracefully...")
		cancel()
	}()

	// Initialize components
	f, err := fetcher.New(*rpcURL, *contract, *fromBlock)
	if err != nil {
		log.Fatalf("failed to create fetcher: %v", err)
	}

	p := parser.New()

	idem, err := idempotency.New(*redisAddr)
	if err != nil {
		log.Fatalf("failed to create idempotency layer: %v", err)
	}

	rm := reorg.New(*confirmations)

	log.Printf("starting indexer for contract %s, confirmations=%d", *contract, *confirmations)

	// Main loop
	for {
		select {
		case <-ctx.Done():
			log.Println("indexer stopped")
			return
		default:
		}

		// 1. Fetch next block
		block, err := f.Next(ctx)
		if err != nil {
			// Caught up — wait and retry
			log.Printf("caught up, waiting %v...", *pollInterval)
			select {
			case <-ctx.Done():
				return
			case <-time.After(*pollInterval):
			}
			continue
		}

		// 2. Update reorg detector tip
		rm.UpdateTip(block.Number)

		// 3. Check reorg safety
		if !rm.IsSafe(block.Number) {
			log.Printf("block %d not safe yet (need %d confirmations)", block.Number, *confirmations)
			continue
		}

		// 4. Parse logs
		events, err := p.ParseLogs(block.Logs, block.ChainID, block.Number, block.Hash.Hex(), block.Ts)
		if err != nil {
			log.Printf("parse error: %v", err)
			continue
		}

		log.Printf("block %d: parsed %d events", block.Number, len(events))

		// 5. Idempotency check + process
		for _, e := range events {
			exists, err := idem.Exists(ctx, e.ChainID, e.TxHash, e.LogIndex)
			if err != nil {
				log.Printf("idempotency check error: %v", err)
				continue
			}
			if exists {
				log.Printf("duplicate event skipped: tx=%s log=%d", e.TxHash, e.LogIndex)
				continue
			}

			// Process event (TODO: write to DB)
			log.Printf("new event: type=%s amount=%s tx=%s block=%d",
				e.Type, e.Amount.String(), e.TxHash, e.BlockNum)

			// Mark as processed
			if err := idem.MarkProcessed(ctx, e.ChainID, e.TxHash, e.LogIndex); err != nil {
				log.Printf("mark processed error: %v", err)
			}
		}

		// 6. Persist cursor
		if err := f.SaveCursor(ctx); err != nil {
			log.Printf("save cursor error: %v", err)
		}
	}
}

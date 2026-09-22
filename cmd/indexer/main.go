package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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
			log.Printf("fetch error: %v", err)
			continue
		}

		// 2. Check reorg safety
		if !rm.IsSafe(block.NumberU64()) {
			continue
		}

		// 3. Parse logs
		events, err := p.ParseLogs(block.Logs)
		if err != nil {
			log.Printf("parse error: %v", err)
			continue
		}

		// 4. Idempotency check
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

			// TODO: write to DB, mark as processed
			log.Printf("new event: type=%s amount=%s tx=%s", e.Type, e.Amount, e.TxHash)

			if err := idem.MarkProcessed(ctx, e.ChainID, e.TxHash, e.LogIndex); err != nil {
				log.Printf("mark processed error: %v", err)
			}
		}

		// 5. Persist cursor
		if err := f.SaveCursor(ctx); err != nil {
			log.Printf("save cursor error: %v", err)
		}
	}
}

package main

import (
	"fmt"
	"log"

	"github.com/jimmie20181111007/go-evm-indexer/internal/idempotency"
)

func main() {
	// Example: check if an event has been processed
	idem, err := idempotency.New("localhost:6379")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	chainID := int64(1) // Ethereum mainnet
	txHash := "0x1234567890abcdef"
	logIndex := uint(0)

	// Check if already processed
	exists, err := idem.Exists(ctx, chainID, txHash, logIndex)
	if err != nil {
		log.Fatal(err)
	}

	if exists {
		fmt.Println("duplicate event, skipping")
		return
	}

	// Process the event
	fmt.Println("processing new event...")

	// Mark as processed
	if err := idem.MarkProcessed(ctx, chainID, txHash, logIndex); err != nil {
		log.Fatal(err)
	}

	fmt.Println("done")
}

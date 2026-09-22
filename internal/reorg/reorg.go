package reorg

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// Detector handles reorg safety.
//
// We never credit deposits immediately at the chain tip.
// We track a "safe head" N confirmations behind the tip.
// If a reorg deepens past N, we roll back unconfirmed deposits.
type Detector struct {
	mu            sync.RWMutex
	confirmations uint64
	tipBlock      uint64
}

func New(confirmations int) *Detector {
	if confirmations < 1 {
		confirmations = 3
	}
	return &Detector{
		confirmations: uint64(confirmations),
	}
}

// UpdateTip updates the latest known block height.
func (d *Detector) UpdateTip(tip uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.tipBlock = tip
}

// SafeHead returns the highest safe block height.
// A block is safe if it's at least N confirmations behind the tip.
func (d *Detector) SafeHead() uint64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.tipBlock < d.confirmations {
		return 0
	}
	return d.tipBlock - d.confirmations
}

// IsSafe returns true if a block height is safe to process.
func (d *Detector) IsSafe(blockHeight uint64) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	safeHead := d.tipBlock
	if safeHead > d.confirmations {
		safeHead = safeHead - d.confirmations
	} else {
		safeHead = 0
	}

	return blockHeight <= safeHead
}

// Rollback handles a reorg — roll back all deposits in the reorged range.
// This is called when a reorg is detected.
func (d *Detector) Rollback(ctx context.Context, fromBlock, toBlock uint64) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	log.Printf("rolling back deposits from block %d to %d", fromBlock, toBlock)

	// TODO:
	// 1. Find all unconfirmed deposits in [fromBlock, toBlock]
	// 2. Mark them as "rolled back" in DB
	// 3. Send alert to oncall

	return nil
}

// Detect checks if a new block is a reorg.
// Returns true if reorg detected, and the block to roll back to.
func (d *Detector) Detect(ctx context.Context, newBlockHash string, expectedParentHash string) (bool, uint64, error) {
	if newBlockHash == "" || expectedParentHash == "" {
		return false, 0, fmt.Errorf("empty block hashes")
	}

	// In production, you'd compare the new block's parent hash
	// with what you expect. If they don't match, it's a reorg.
	//
	// This is a simplified placeholder — real implementation would
	// fetch the block from the node and compare hashes.

	return false, 0, nil
}

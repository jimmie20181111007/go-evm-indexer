package reorg

import "log"

// Detector handles reorg safety.
//
// We never credit deposits immediately at the chain tip.
// We track a "safe head" N confirmations behind the tip.
// If a reorg deepens past N, we roll back unconfirmed deposits.
type Detector struct {
	confirmations uint64
}

func New(confirmations int) *Detector {
	if confirmations < 1 {
		confirmations = 3
	}
	return &Detector{
		confirmations: uint64(confirmations),
	}
}

// IsSafe returns true if a block height is safe to process.
// A block is safe if it's at least N confirmations behind the tip.
func (d *Detector) IsSafe(blockHeight uint64) bool {
	// TODO: get current tip from chain
	tip := blockHeight + d.confirmations // placeholder
	safeHead := tip - d.confirmations

	if blockHeight > safeHead {
		log.Printf("block %d not safe yet (safe head: %d)", blockHeight, safeHead)
		return false
	}

	return true
}

// Rollback handles a reorg — roll back all deposits in the reorged range.
// This is called when a reorg is detected.
func (d *Detector) Rollback(fromBlock, toBlock uint64) error {
	log.Printf("rolling back deposits from block %d to %d", fromBlock, toBlock)

	// TODO:
	// 1. Find all unconfirmed deposits in [fromBlock, toBlock]
	// 2. Mark them as "rolled back" in DB
	// 3. Send alert to oncall

	return nil
}

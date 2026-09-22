package fetcher

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Fetcher handles block fetching with cursor-based incremental sync.
// It persists the last processed block height and resumes from there.
type Fetcher struct {
	client   *ethclient.Client
	contract common.Address

	mu          sync.Mutex
	cursor      int64 // last processed block height
	initialized bool
}

// New creates a new Fetcher.
// fromBlock = -1 means resume from persisted cursor (or start from latest if none).
func New(rpcURL, contract string, fromBlock int64) (*Fetcher, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial rpc: %w", err)
	}

	return &Fetcher{
		client:   client,
		contract: common.HexToAddress(contract),
		cursor:   fromBlock,
	}, nil
}

// Next returns the next block to process.
// Blocks until a new safe block is available.
func (f *Fetcher) Next(ctx context.Context) (*Block, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		if f.cursor < 0 {
			// TODO: load from persisted cursor (DB/file)
			header, err := f.client.HeaderByNumber(ctx, nil)
			if err != nil {
				return nil, fmt.Errorf("get latest header: %w", err)
			}
			f.cursor = header.Number.Int64()
			log.Printf("starting from latest block: %d", f.cursor)
		}
		f.initialized = true
	}

	f.cursor++

	block, err := f.client.BlockByNumber(ctx, nil /* TODO: f.cursor */)
	if err != nil {
		return nil, fmt.Errorf("get block %d: %w", f.cursor, err)
	}

	// Filter logs for our contract
	logs, err := f.client.FilterLogs(ctx, buildFilterQuery(f.contract, f.cursor, f.cursor))
	if err != nil {
		return nil, fmt.Errorf("filter logs: %w", err)
	}

	return &Block{
		Number: block.NumberU64(),
		Hash:   block.Hash(),
		Logs:   logs,
	}, nil
}

// SaveCursor persists the current cursor to storage.
func (f *Fetcher) SaveCursor(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	// TODO: implement real persistence (DB or file)
	log.Printf("cursor saved: block %d", f.cursor)
	return nil
}

// Block represents a block to process.
type Block struct {
	Number uint64
	Hash   common.Hash
	Logs   []interface{} // types.Log
}

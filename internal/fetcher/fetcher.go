package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Fetcher handles block fetching with cursor-based incremental sync.
// It persists the last processed block height and resumes from there.
type Fetcher struct {
	client   *ethclient.Client
	contract common.Address
	chainID  int64

	mu          sync.Mutex
	cursor      int64
	initialized bool
	cursorFile  string
}

// New creates a new Fetcher.
// fromBlock = -1 means resume from persisted cursor (or start from latest if none).
func New(rpcURL, contract string, fromBlock int64) (*Fetcher, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial rpc: %w", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get chain id: %w", err)
	}

	return &Fetcher{
		client:     client,
		contract:   common.HexToAddress(contract),
		chainID:    chainID.Int64(),
		cursor:     fromBlock,
		cursorFile: ".cursor",
	}, nil
}

// Next returns the next block to process.
// Returns error if we're caught up to the tip.
func (f *Fetcher) Next(ctx context.Context) (*Block, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		if f.cursor < 0 {
			if saved, err := f.loadCursor(); err == nil && saved > 0 {
				f.cursor = saved
				log.Printf("resumed from saved cursor: block %d", f.cursor)
			} else {
				header, err := f.client.HeaderByNumber(ctx, nil)
				if err != nil {
					return nil, fmt.Errorf("get latest header: %w", err)
				}
				f.cursor = header.Number.Int64()
				log.Printf("starting from latest block: %d", f.cursor)
			}
		}
		f.initialized = true
	}

	latest, err := f.client.BlockByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("get latest block: %w", err)
	}

	if f.cursor >= latest.NumberU64() {
		return nil, fmt.Errorf("caught up (current: %d, tip: %d)", f.cursor, latest.NumberU64())
	}

	f.cursor++

	blockNum := big.NewInt(f.cursor)
	block, err := f.client.BlockByNumber(ctx, blockNum)
	if err != nil {
		return nil, fmt.Errorf("get block %d: %w", f.cursor, err)
	}

	query := buildFilterQuery(f.contract, big.NewInt(f.cursor), big.NewInt(f.cursor))
	logs, err := f.client.FilterLogs(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("filter logs: %w", err)
	}

	return &Block{
		Number:  block.NumberU64(),
		Hash:    block.Hash(),
		Ts:      block.Time(),
		ChainID: f.chainID,
		Logs:    logs,
	}, nil
}

// SaveCursor persists the current cursor to disk.
func (f *Fetcher) SaveCursor(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := json.Marshal(map[string]int64{
		"block": f.cursor,
	})
	if err != nil {
		return fmt.Errorf("marshal cursor: %w", err)
	}

	if err := os.WriteFile(f.cursorFile, data, 0644); err != nil {
		return fmt.Errorf("write cursor file: %w", err)
	}

	return nil
}

func (f *Fetcher) loadCursor() (int64, error) {
	data, err := os.ReadFile(f.cursorFile)
	if err != nil {
		return 0, err
	}

	var cursor struct {
		Block int64 `json:"block"`
	}
	if err := json.Unmarshal(data, &cursor); err != nil {
		return 0, err
	}

	return cursor.Block, nil
}

// Block represents a block to process.
type Block struct {
	Number  uint64
	Hash    common.Hash
	Ts      uint64
	ChainID int64
	Logs    []types.Log
}

func buildFilterQuery(contract common.Address, from, to *big.Int) ethereum.FilterQuery {
	return ethereum.FilterQuery{
		FromBlock: from,
		ToBlock:   to,
		Addresses: []common.Address{contract},
	}
}

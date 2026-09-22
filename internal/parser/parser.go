package parser

import (
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Event types
const (
	EventDeposit  = "Deposit"
	EventWithdraw = "Withdraw"
	EventTransfer = "Transfer"
)

// Event represents a parsed on-chain event.
type Event struct {
	Type      string
	TxHash    string
	TxIndex   uint
	LogIndex  uint
	ChainID   int64
	BlockNum  uint64
	BlockHash string
	Ts        uint64
	From      string
	To        string
	Amount    *big.Int
	Recipient string
}

// Parser decodes raw event logs into structured Events.
type Parser struct {
	// Event topic hashes (keccak256 of event signatures)
	depositTopic  common.Hash
	withdrawTopic common.Hash
	transferTopic  common.Hash
}

// New creates a Parser with default event topic hashes.
func New() *Parser {
	return &Parser{
		depositTopic:  common.HexToHash("0xdcbc5413c0798128469f80f11145bd7485fdf0f22c61600e6b5f291745699585"), // Deposit(address,uint256)
		withdrawTopic: common.HexToHash("0x7fcf532c15f0b453db0343b73f4b62d6cdf79f6e627fd3dcfae3c969a6e3b3b6"), // Withdrawal(address,uint256)
		transferTopic: common.HexToHash("0xddf252ad1be2c89b69c3c068fc378daa952ba7f163c4a11628f55a4df523b3ef"), // Transfer(address,address,uint256)
	}
}

// ParseLogs decodes a slice of raw logs into Events.
func (p *Parser) ParseLogs(logs []types.Log, chainID int64, blockNum uint64, blockHash string, ts uint64) ([]*Event, error) {
	var events []*Event

	for _, log := range logs {
		if len(log.Topics) == 0 {
			continue
		}

		topic := log.Topics[0]

		switch {
		case topic == p.depositTopic:
			ev, err := p.parseDeposit(log, chainID, blockNum, blockHash, ts)
			if err != nil {
				log.Printf("failed to parse deposit event: %v", err)
				continue
			}
			events = append(events, ev)

		case topic == p.withdrawTopic:
			ev, err := p.parseWithdraw(log, chainID, blockNum, blockHash, ts)
			if err != nil {
				log.Printf("failed to parse withdraw event: %v", err)
				continue
			}
			events = append(events, ev)

		case topic == p.transferTopic:
			ev, err := p.parseTransfer(log, chainID, blockNum, blockHash, ts)
			if err != nil {
				log.Printf("failed to parse transfer event: %v", err)
				continue
			}
			events = append(events, ev)

		default:
			log.Printf("unknown event topic: %s", topic.Hex())
		}
	}

	return events, nil
}

func (p *Parser) parseDeposit(l types.Log, chainID int64, blockNum uint64, blockHash string, ts uint64) (*Event, error) {
	if len(l.Topics) < 2 {
		return nil, fmt.Errorf("deposit event: not enough topics")
	}

	// Deposit(address indexed user, uint256 amount)
	user := common.BytesToAddress(l.Topics[1].Bytes()).Hex()
	amount := new(big.Int).SetBytes(l.Data)

	return &Event{
		Type:      EventDeposit,
		TxHash:    l.TxHash.Hex(),
		TxIndex:   l.TxIndex,
		LogIndex:  l.Index,
		ChainID:   chainID,
		BlockNum:  blockNum,
		BlockHash: blockHash,
		Ts:        ts,
		To:        user,
		Amount:    amount,
	}, nil
}

func (p *Parser) parseWithdraw(l types.Log, chainID int64, blockNum uint64, blockHash string, ts uint64) (*Event, error) {
	if len(l.Topics) < 2 {
		return nil, fmt.Errorf("withdraw event: not enough topics")
	}

	user := common.BytesToAddress(l.Topics[1].Bytes()).Hex()
	amount := new(big.Int).SetBytes(l.Data)

	return &Event{
		Type:     EventWithdraw,
		TxHash:   l.TxHash.Hex(),
		TxIndex:  l.TxIndex,
		LogIndex: l.Index,
		ChainID:  chainID,
		BlockNum: blockNum,
		Ts:       ts,
		From:     user,
		Amount:   amount,
	}, nil
}

func (p *Parser) parseTransfer(l types.Log, chainID int64, blockNum uint64, blockHash string, ts uint64) (*Event, error) {
	if len(l.Topics) < 3 {
		return nil, fmt.Errorf("transfer event: not enough topics")
	}

	from := common.BytesToAddress(l.Topics[1].Bytes()).Hex()
	to := common.BytesToAddress(l.Topics[2].Bytes()).Hex()
	amount := new(big.Int).SetBytes(l.Data)

	return &Event{
		Type:     EventTransfer,
		TxHash:   l.TxHash.Hex(),
		TxIndex:  l.TxIndex,
		LogIndex: l.Index,
		ChainID:  chainID,
		BlockNum: blockNum,
		Ts:       ts,
		From:     from,
		To:       to,
		Amount:   amount,
	}, nil
}

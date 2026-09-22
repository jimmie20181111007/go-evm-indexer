package parser

import "log"

// Event represents a parsed on-chain event.
type Event struct {
	Type      string
	TxHash    string
	LogIndex  uint
	ChainID   int64
	Amount    string
	Recipient string
}

// Parser decodes raw event logs into structured Events.
type Parser struct{}

func New() *Parser {
	return &Parser{}
}

// ParseLogs decodes a slice of raw logs into Events.
// This is a simplified example — in production you'd use contract ABIs.
func (p *Parser) ParseLogs(logs []interface{}) ([]*Event, error) {
	var events []*Event

	for _, l := range logs {
		log.Printf("parsing log: %+v", l)
		// TODO: decode using contract ABI
		// Example: transfer, deposit, withdraw events
		events = append(events, &Event{
			Type:   "Deposit",
			TxHash: "0x...",
			Amount: "1.5 ETH",
		})
	}

	return events, nil
}

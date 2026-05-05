// Package types defines the normalized data structures shared across the
// indexer, processor, api, and mcp services.
package types

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// ChainEvent is the normalized event produced by the indexer for every
// recognized log on every chain. It is the wire format on the raw_events
// Kafka topic.
type ChainEvent struct {
	ChainID     uint64         `json:"chain_id"`
	BlockNumber uint64         `json:"block_number"`
	TxHash      common.Hash    `json:"tx_hash"`
	LogIndex    uint           `json:"log_index"`
	Contract    common.Address `json:"contract"`
	EventName   string         `json:"event_name"`
	RawTopics   []common.Hash  `json:"raw_topics"`
	RawData     []byte         `json:"raw_data"`
	Timestamp   time.Time      `json:"timestamp"`
}

// DecodedEvent is produced by protocol-specific decoders in the processor.
// It carries the original ChainEvent plus protocol-level meaning.
type DecodedEvent struct {
	ChainEvent
	Protocol  string                 `json:"protocol"`
	EventType string                 `json:"event_type"`
	Params    map[string]interface{} `json:"params"`
}

// WalletPosition is a wallet's aggregated state for a single token within a
// single protocol on a single chain.
type WalletPosition struct {
	Wallet       common.Address `json:"wallet"`
	ChainID      uint64         `json:"chain_id"`
	Protocol     string         `json:"protocol"`
	PositionType string         `json:"position_type"`
	Token        common.Address `json:"token"`
	TokenSymbol  string         `json:"token_symbol"`
	Amount       *big.Int       `json:"amount"`
	ValueUSD     float64        `json:"value_usd"`
	UpdatedAt    time.Time      `json:"updated_at"`
	BlockNumber  uint64         `json:"block_number"`
}

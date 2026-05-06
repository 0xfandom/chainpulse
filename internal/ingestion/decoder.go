// Package ingestion contains the chain listeners, the ABI signature router,
// and the Kafka producer used by the indexer service.
package ingestion

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/0xfandom/chainpulse/internal/types"
)

// Well-known event signatures the indexer recognizes on the hot path.
// Day 1 ships ERC-20 Transfer only. Protocol-specific decoding (Uniswap,
// Aave, Compound) lives in internal/processor/protocols on Day 2.
var (
	ERC20TransferSig = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
)

// ErrUnrecognizedEvent is returned by Decode when no known signature
// matches the log's first topic. Callers should skip the log.
var ErrUnrecognizedEvent = errors.New("unrecognized event signature")

// ABIDecoder routes a raw chain log to a normalized ChainEvent based on its
// event signature. The decoder is stateless and safe for concurrent use.
type ABIDecoder struct{}

// NewABIDecoder builds a decoder. Constructor reserved so future signatures
// can require setup (e.g., loaded ABIs).
func NewABIDecoder() *ABIDecoder {
	return &ABIDecoder{}
}

// Decode normalizes a single ethclient log into a ChainEvent. blockTime is
// the block header's Unix timestamp in seconds. Returns ErrUnrecognizedEvent
// for logs the indexer should skip.
func (d *ABIDecoder) Decode(chainID uint64, vLog ethtypes.Log, blockTime uint64) (*types.ChainEvent, error) {
	if len(vLog.Topics) == 0 {
		return nil, fmt.Errorf("%w: log has no topics", ErrUnrecognizedEvent)
	}

	var eventName string
	switch vLog.Topics[0] {
	case ERC20TransferSig:
		eventName = "Transfer"
	default:
		return nil, ErrUnrecognizedEvent
	}

	return &types.ChainEvent{
		ChainID:     chainID,
		BlockNumber: vLog.BlockNumber,
		TxHash:      vLog.TxHash,
		LogIndex:    vLog.Index,
		Contract:    vLog.Address,
		EventName:   eventName,
		RawTopics:   vLog.Topics,
		RawData:     vLog.Data,
		Timestamp:   time.Unix(int64(blockTime), 0).UTC(),
	}, nil
}

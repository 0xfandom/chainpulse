// Package ingestion contains the chain listeners, the ABI signature router,
// and the Kafka producer used by the indexer service.
package ingestion

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/0xfandom/chainpulse/internal/processor/protocols"
	"github.com/0xfandom/chainpulse/internal/types"
)

// Re-exported for callers / tests that already imported the indexer-side
// constant. Single source of truth lives in internal/processor/protocols.
var ERC20TransferSig = protocols.ERC20TransferSig

// knownSigs maps every event signature the indexer recognizes to a
// human-readable event name. Anything not in this map gets dropped at
// indexer side so we don't pay Kafka + processor cost on logs we can't
// decode anyway. Add new entries here when shipping a new protocol
// decoder under internal/processor/protocols.
var knownSigs map[common.Hash]string

func init() {
	knownSigs = map[common.Hash]string{
		// ERC-20
		protocols.ERC20TransferSig: "Transfer",

		// Uniswap V3 pool events
		protocols.UniswapV3SwapSig:    "Swap",
		protocols.UniswapV3MintSig:    "Mint",
		protocols.UniswapV3BurnSig:    "Burn",
		protocols.UniswapV3CollectSig: "Collect",

		// Aave V3 pool events
		protocols.AaveV3SupplySig:          "Supply",
		protocols.AaveV3WithdrawSig:        "Withdraw",
		protocols.AaveV3BorrowSig:          "Borrow",
		protocols.AaveV3RepaySig:           "Repay",
		protocols.AaveV3LiquidationCallSig: "LiquidationCall",

		// Compound V3 (Comet) events
		protocols.CompoundV3SupplySig:   "Supply",
		protocols.CompoundV3WithdrawSig: "Withdraw",
		protocols.CompoundV3AbsorbSig:   "AbsorbCollateral",

		// Lido stETH events
		protocols.LidoSubmittedSig: "Submitted",

		// Curve Stableswap V1 events
		protocols.CurveTokenExchangeSig: "CurveTokenExchange",
	}
}

// KnownTopics returns the topic0 hashes for every event signature the
// indexer recognizes. Used by the listener's filter-logs subscription
// path so the RPC provider only streams logs we can actually decode.
func KnownTopics() []common.Hash {
	out := make([]common.Hash, 0, len(knownSigs))
	for h := range knownSigs {
		out = append(out, h)
	}
	return out
}

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
//
// Note: when the same signature is shared across protocols (e.g. Aave +
// Compound both fire "Supply" with different ABIs), we tag the event name
// with the matching name from knownSigs but rely on the processor's
// decoder registry to disambiguate by contract address. The
// (sig, contract) tuple is enough downstream.
func (d *ABIDecoder) Decode(chainID uint64, vLog ethtypes.Log, blockTime uint64) (*types.ChainEvent, error) {
	if len(vLog.Topics) == 0 {
		return nil, fmt.Errorf("%w: log has no topics", ErrUnrecognizedEvent)
	}
	eventName, ok := knownSigs[vLog.Topics[0]]
	if !ok {
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

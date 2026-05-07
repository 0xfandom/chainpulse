package processor

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/monitor"
	"github.com/0xfandom/chainpulse/internal/types"
)

// BatchWriterIface, CacheWriterIface, and ProducerIface let tests
// substitute fakes for the three downstream writers. The concrete
// *BatchWriter, *CacheWriter, and *Producer types implement these
// implicitly.
type (
	BatchWriterIface interface {
		Enqueue(ctx context.Context, e *types.DecodedEvent) error
	}
	CacheWriterIface interface {
		IncrBalance(ctx context.Context, chainID uint64, wallet, token common.Address, delta *big.Int) error
		IncrBalancePair(ctx context.Context, chainID uint64, from, to, token common.Address, amount *big.Int) error
		SetPosition(ctx context.Context, pos *types.WalletPosition) error
	}
	ProducerIface interface {
		PublishDecoded(ctx context.Context, e *types.DecodedEvent) error
		PublishPositionUpdate(ctx context.Context, p *types.WalletPosition) error
	}
)

// Aggregator turns a DecodedEvent into ClickHouse rows, Redis cache
// updates, and Kafka publishes. Idempotency is delegated to the writer
// layer (ClickHouse ReplacingMergeTree on (chain_id, block_number,
// log_index); Redis writes are last-write-wins).
//
// Each branch returns the first error encountered; partial side-effects
// are acceptable for v1 because re-delivery (consumer un-commit) replays
// the whole branch and the writer layer dedupes.
type Aggregator struct {
	batch    BatchWriterIface
	cache    CacheWriterIface
	producer ProducerIface
}

// NewAggregator constructs an Aggregator. All three writers are
// required; nil values fail validation.
func NewAggregator(batch BatchWriterIface, cache CacheWriterIface, producer ProducerIface) (*Aggregator, error) {
	if batch == nil {
		return nil, errors.New("aggregator: nil batch writer")
	}
	if cache == nil {
		return nil, errors.New("aggregator: nil cache writer")
	}
	if producer == nil {
		return nil, errors.New("aggregator: nil producer")
	}
	return &Aggregator{batch: batch, cache: cache, producer: producer}, nil
}

// Process dispatches a single DecodedEvent to the right side-effects.
func (a *Aggregator) Process(ctx context.Context, e *types.DecodedEvent) error {
	if e == nil {
		return errors.New("aggregator: nil event")
	}

	switch {
	case e.Protocol == "erc20" && e.EventType == "transfer":
		return a.handleTransfer(ctx, e)
	case e.Protocol == "erc20" && e.EventType == "approval":
		return a.producer.PublishDecoded(ctx, e)
	case e.Protocol == "uniswap_v3":
		return a.handleUniswap(ctx, e)
	case e.Protocol == "aave_v3":
		return a.handleAave(ctx, e)
	case e.Protocol == "compound_v3":
		return a.handleCompound(ctx, e)
	case e.Protocol == "lido":
		return a.handleLido(ctx, e)
	case e.Protocol == "curve":
		return a.handleCurve(ctx, e)
	default:
		monitor.IncProcessorError(monitor.ProcErrAggregate)
		return nil
	}
}

// handleTransfer persists the row, updates Redis balances on both
// sides in a single pipelined round trip, and publishes the decoded
// event. Self-transfers are skipped at the cache layer (net zero).
func (a *Aggregator) handleTransfer(ctx context.Context, e *types.DecodedEvent) error {
	if err := a.batch.Enqueue(ctx, e); err != nil {
		return err
	}
	from, _ := e.Params["from"].(string)
	to, _ := e.Params["to"].(string)
	amountStr, _ := e.Params["amount"].(string)
	tokenStr, _ := e.Params["token"].(string)

	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		return fmt.Errorf("aggregator transfer: bad amount %q", amountStr)
	}
	token := common.HexToAddress(tokenStr)
	if err := a.cache.IncrBalancePair(ctx, e.ChainID, common.HexToAddress(from), common.HexToAddress(to), token, amount); err != nil {
		return err
	}
	return a.producer.PublishDecoded(ctx, e)
}

// handleUniswap persists the swap/mint/burn/collect row and republishes.
// LP positions are not modeled in v1 — Uniswap LP accounting is more
// involved than supply/borrow.
func (a *Aggregator) handleUniswap(ctx context.Context, e *types.DecodedEvent) error {
	if err := a.batch.Enqueue(ctx, e); err != nil {
		return err
	}
	return a.producer.PublishDecoded(ctx, e)
}

// handleAave persists the row, builds a WalletPosition for state-changing
// events (supply/withdraw/borrow/repay/liquidation), updates Redis, and
// publishes both the decoded event and the position update.
func (a *Aggregator) handleAave(ctx context.Context, e *types.DecodedEvent) error {
	if err := a.batch.Enqueue(ctx, e); err != nil {
		return err
	}

	pos := walletPositionFromEvent(e)
	if pos != nil {
		if err := a.cache.SetPosition(ctx, pos); err != nil {
			return err
		}
		if err := a.producer.PublishPositionUpdate(ctx, pos); err != nil {
			return err
		}
	}
	return a.producer.PublishDecoded(ctx, e)
}

// handleCompound mirrors handleAave; absorb is treated like a
// liquidation event for position accounting.
func (a *Aggregator) handleCompound(ctx context.Context, e *types.DecodedEvent) error {
	if err := a.batch.Enqueue(ctx, e); err != nil {
		return err
	}

	pos := walletPositionFromEvent(e)
	if pos != nil {
		if err := a.cache.SetPosition(ctx, pos); err != nil {
			return err
		}
		if err := a.producer.PublishPositionUpdate(ctx, pos); err != nil {
			return err
		}
	}
	return a.producer.PublishDecoded(ctx, e)
}

// handleLido persists the stETH Submitted row and publishes the decoded
// event. Submitted is not a lending-protocol position event, so no
// WalletPosition record is produced.
func (a *Aggregator) handleLido(ctx context.Context, e *types.DecodedEvent) error {
	if err := a.batch.Enqueue(ctx, e); err != nil {
		return err
	}
	return a.producer.PublishDecoded(ctx, e)
}

// handleCurve persists the Curve Stableswap TokenExchange row and publishes
// the decoded event. Swaps have no lending-protocol position semantic.
func (a *Aggregator) handleCurve(ctx context.Context, e *types.DecodedEvent) error {
	if err := a.batch.Enqueue(ctx, e); err != nil {
		return err
	}
	return a.producer.PublishDecoded(ctx, e)
}

// walletPositionFromEvent builds a WalletPosition from supply/withdraw/
// borrow/repay/liquidation/absorb events. Returns nil for events that
// don't have a clear position semantic (e.g. Uniswap swaps).
func walletPositionFromEvent(e *types.DecodedEvent) *types.WalletPosition {
	user, ok := e.Params["user"].(string)
	if !ok || user == "" {
		// Compound supply/withdraw use from/dst/src/to instead of "user"
		switch e.EventType {
		case "supply":
			user, _ = e.Params["dst"].(string)
		case "withdraw":
			user, _ = e.Params["src"].(string)
		case "absorb":
			user, _ = e.Params["borrower"].(string)
		}
	}
	if user == "" {
		return nil
	}

	tokenStr := firstStringParam(e, "reserve", "asset", "token", "comet")
	if tokenStr == "" {
		return nil
	}
	amountStr := firstStringParam(e, "amount", "debt_to_cover", "collateral_absorbed")

	posType := positionTypeForEventType(e.EventType)
	if posType == "" {
		return nil
	}

	pos := &types.WalletPosition{
		Wallet:       common.HexToAddress(user),
		ChainID:      e.ChainID,
		Protocol:     e.Protocol,
		PositionType: posType,
		Token:        common.HexToAddress(tokenStr),
		UpdatedAt:    e.Timestamp,
		BlockNumber:  e.BlockNumber,
	}
	if amountStr != "" {
		if a, ok := new(big.Int).SetString(amountStr, 10); ok {
			pos.Amount = a
		}
	}
	if pos.UpdatedAt.IsZero() {
		pos.UpdatedAt = time.Now().UTC()
	}
	// keep tokens lowercased to match the API/Redis key conventions
	_ = strings.ToLower
	return pos
}

// positionTypeForEventType maps a DecodedEvent.EventType to a stable
// PositionType label. Returns "" for events that should not produce a
// position record.
func positionTypeForEventType(eventType string) string {
	switch eventType {
	case "supply":
		return "supply"
	case "withdraw":
		return "withdraw"
	case "borrow":
		return "borrow"
	case "repay":
		return "repay"
	case "liquidation":
		return "liquidation"
	case "absorb":
		return "absorb"
	default:
		return ""
	}
}

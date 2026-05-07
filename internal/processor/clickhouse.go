package processor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	chainpulselog "github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/monitor"
	"github.com/0xfandom/chainpulse/internal/types"
)

// TransferRow is the persisted shape of a token_transfers row.
type TransferRow struct {
	ChainID     uint64
	BlockNumber uint64
	TxHash      string
	LogIndex    uint32
	Token       string
	FromAddr    string
	ToAddr      string
	Amount      string
	Timestamp   time.Time
}

// DefiEventRow is the persisted shape of a defi_events row.
type DefiEventRow struct {
	ChainID     uint64
	BlockNumber uint64
	TxHash      string
	LogIndex    uint32
	Protocol    string
	EventType   string
	UserAddr    string
	TokenA      string
	TokenB      sql.NullString
	AmountA     string
	AmountB     sql.NullString
	Params      string // JSON blob
	Timestamp   time.Time
}

// BatchConn is the ClickHouse-side dependency. Implementations send each
// row slice as a single batched INSERT. The interface lets unit tests
// substitute a fake without depending on the ClickHouse driver.
type BatchConn interface {
	InsertTransfers(ctx context.Context, rows []TransferRow) error
	InsertDefiEvents(ctx context.Context, rows []DefiEventRow) error
	Close() error
}

// BatchWriterConfig tunes flush behavior.
type BatchWriterConfig struct {
	BatchSize     int
	BatchInterval time.Duration
	// ChainNames maps chain_id to a human-readable label used by the
	// block_to_queryable_seconds histogram. Missing ids fall back to
	// "chain_<id>".
	ChainNames map[uint64]string
}

// BatchWriter buffers decoded events and flushes them to ClickHouse in
// size- or interval-bounded batches.
//
// Idempotency: the schema's ReplacingMergeTree dedups on
// (chain_id, block_number, log_index). Re-inserting a row collapses on
// background merge — no app-side dedup required.
type BatchWriter struct {
	conn       BatchConn
	cfg        BatchWriterConfig
	chainNames map[uint64]string

	mu        sync.Mutex
	transfers []TransferRow
	defi      []DefiEventRow

	stop   chan struct{}
	wg     sync.WaitGroup
	closed bool
}

// NewBatchWriter constructs a BatchWriter and starts its background
// flusher goroutine. Callers must call Close to flush + stop.
func NewBatchWriter(conn BatchConn, cfg BatchWriterConfig) (*BatchWriter, error) {
	if conn == nil {
		return nil, fmt.Errorf("clickhouse batch writer: nil conn")
	}
	if cfg.BatchSize <= 0 {
		return nil, fmt.Errorf("clickhouse batch writer: batch_size must be > 0")
	}
	if cfg.BatchInterval <= 0 {
		return nil, fmt.Errorf("clickhouse batch writer: batch_interval must be > 0")
	}

	w := &BatchWriter{
		conn:       conn,
		cfg:        cfg,
		chainNames: cfg.ChainNames,
		stop:       make(chan struct{}),
	}
	w.wg.Add(1)
	go w.runFlushLoop()
	return w, nil
}

// Enqueue routes an event to the right buffer. ERC-20 transfers go to
// token_transfers; other DeFi events go to defi_events. ERC-20 approvals
// are intentionally skipped (no row, no error).
func (w *BatchWriter) Enqueue(ctx context.Context, e *types.DecodedEvent) error {
	if e == nil {
		return fmt.Errorf("batch writer: nil event")
	}

	switch {
	case e.Protocol == "erc20" && e.EventType == "transfer":
		row, err := mapTransferRow(e)
		if err != nil {
			return err
		}
		w.appendTransfer(ctx, row)
	case e.Protocol == "erc20" && e.EventType == "approval":
		// approvals are not persisted in v1
		return nil
	default:
		row, err := mapDefiRow(e)
		if err != nil {
			return err
		}
		w.appendDefi(ctx, row)
	}
	return nil
}

func (w *BatchWriter) appendTransfer(ctx context.Context, row TransferRow) {
	w.mu.Lock()
	w.transfers = append(w.transfers, row)
	full := len(w.transfers) >= w.cfg.BatchSize
	w.mu.Unlock()
	if full {
		w.flushTransfers(ctx)
	}
}

func (w *BatchWriter) appendDefi(ctx context.Context, row DefiEventRow) {
	w.mu.Lock()
	w.defi = append(w.defi, row)
	full := len(w.defi) >= w.cfg.BatchSize
	w.mu.Unlock()
	if full {
		w.flushDefi(ctx)
	}
}

// runFlushLoop wakes on the configured interval and flushes whatever is
// buffered. Exits on stop signal.
func (w *BatchWriter) runFlushLoop() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.cfg.BatchInterval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			w.flushAll(context.Background())
		}
	}
}

func (w *BatchWriter) flushAll(ctx context.Context) {
	w.flushTransfers(ctx)
	w.flushDefi(ctx)
}

func (w *BatchWriter) flushTransfers(ctx context.Context) {
	w.mu.Lock()
	if len(w.transfers) == 0 {
		w.mu.Unlock()
		return
	}
	rows := w.transfers
	w.transfers = make([]TransferRow, 0, w.cfg.BatchSize)
	w.mu.Unlock()

	start := time.Now()
	if err := w.conn.InsertTransfers(ctx, rows); err != nil {
		l := chainpulselog.Component("clickhouse_writer")
		l.Error().Err(err).Int("rows", len(rows)).Msg("flush transfers failed")
		monitor.IncProcessorError(monitor.ProcErrClickHouseWrite)
		return
	}
	monitor.ObserveClickHouseFlush(time.Since(start))
	w.observeQueryableLag(rows, nil)
}

func (w *BatchWriter) flushDefi(ctx context.Context) {
	w.mu.Lock()
	if len(w.defi) == 0 {
		w.mu.Unlock()
		return
	}
	rows := w.defi
	w.defi = make([]DefiEventRow, 0, w.cfg.BatchSize)
	w.mu.Unlock()

	start := time.Now()
	if err := w.conn.InsertDefiEvents(ctx, rows); err != nil {
		l := chainpulselog.Component("clickhouse_writer")
		l.Error().Err(err).Int("rows", len(rows)).Msg("flush defi_events failed")
		monitor.IncProcessorError(monitor.ProcErrClickHouseWrite)
		return
	}
	monitor.ObserveClickHouseFlush(time.Since(start))
	w.observeQueryableLag(nil, rows)
}

// observeQueryableLag emits one BlockToQueryableSeconds observation per
// row. Either rows slice may be nil; the other is the one that just
// flushed. Time.Now() is sampled once for the whole batch — sub-row
// drift is dwarfed by the multi-second SLO.
func (w *BatchWriter) observeQueryableLag(transfers []TransferRow, defi []DefiEventRow) {
	now := time.Now()
	for i := range transfers {
		monitor.ObserveBlockToQueryable(w.chainLabel(transfers[i].ChainID), now.Sub(transfers[i].Timestamp))
	}
	for i := range defi {
		monitor.ObserveBlockToQueryable(w.chainLabel(defi[i].ChainID), now.Sub(defi[i].Timestamp))
	}
}

// chainLabel renders a chain id into the histogram label, falling back
// to chain_<id> when the name map is missing or unset.
func (w *BatchWriter) chainLabel(id uint64) string {
	if name, ok := w.chainNames[id]; ok && name != "" {
		return name
	}
	return fmt.Sprintf("chain_%d", id)
}

// Close flushes outstanding rows and shuts down the background loop.
func (w *BatchWriter) Close(ctx context.Context) error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.mu.Unlock()

	close(w.stop)
	w.wg.Wait()
	w.flushAll(ctx)
	return w.conn.Close()
}

// mapTransferRow turns an ERC-20 Transfer DecodedEvent into a TransferRow.
// Returns an error if any required Params field is missing.
func mapTransferRow(e *types.DecodedEvent) (TransferRow, error) {
	from, _ := e.Params["from"].(string)
	to, _ := e.Params["to"].(string)
	amount, _ := e.Params["amount"].(string)
	token, _ := e.Params["token"].(string)
	if from == "" || to == "" || amount == "" || token == "" {
		return TransferRow{}, errors.New("transfer row: missing required param")
	}
	return TransferRow{
		ChainID:     e.ChainID,
		BlockNumber: e.BlockNumber,
		TxHash:      e.TxHash.Hex(),
		LogIndex:    uint32(e.LogIndex),
		Token:       token,
		FromAddr:    from,
		ToAddr:      to,
		Amount:      amount,
		Timestamp:   e.Timestamp,
	}, nil
}

// mapDefiRow turns any non-ERC20 DecodedEvent into a DefiEventRow. The
// full Params map is JSON-encoded into the params column; common.Address
// fields are denormalized into user_addr / token_a / amount_a /
// token_b / amount_b heuristically so simple WHERE filters work without
// JSON parsing.
func mapDefiRow(e *types.DecodedEvent) (DefiEventRow, error) {
	paramsJSON, err := json.Marshal(e.Params)
	if err != nil {
		return DefiEventRow{}, fmt.Errorf("defi row: marshal params: %w", err)
	}

	row := DefiEventRow{
		ChainID:     e.ChainID,
		BlockNumber: e.BlockNumber,
		TxHash:      e.TxHash.Hex(),
		LogIndex:    uint32(e.LogIndex),
		Protocol:    e.Protocol,
		EventType:   e.EventType,
		Params:      string(paramsJSON),
		Timestamp:   e.Timestamp,
	}

	row.UserAddr = firstStringParam(e, "user", "owner", "sender", "from", "src", "absorber", "borrower")
	row.TokenA = firstStringParam(e, "reserve", "asset", "token", "pool", "comet")
	row.AmountA = firstStringParam(e, "amount", "amount0", "debt_to_cover", "collateral_absorbed")

	if v := firstStringParam(e, "amount1", "liquidated_collateral_amount", "usd_value"); v != "" {
		row.AmountB = sql.NullString{Valid: true, String: v}
	}
	if v := firstStringParam(e, "token_b"); v != "" {
		row.TokenB = sql.NullString{Valid: true, String: v}
	}

	return row, nil
}

func firstStringParam(e *types.DecodedEvent, keys ...string) string {
	for _, k := range keys {
		if v, ok := e.Params[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

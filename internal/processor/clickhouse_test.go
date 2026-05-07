package processor

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

type fakeBatchConn struct {
	mu sync.Mutex

	transferBatches [][]TransferRow
	defiBatches     [][]DefiEventRow

	closed atomic.Bool

	insertTransferErr error
	insertDefiErr     error
}

func (f *fakeBatchConn) InsertTransfers(_ context.Context, rows []TransferRow) error {
	if f.insertTransferErr != nil {
		return f.insertTransferErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := append([]TransferRow{}, rows...)
	f.transferBatches = append(f.transferBatches, cp)
	return nil
}

func (f *fakeBatchConn) InsertDefiEvents(_ context.Context, rows []DefiEventRow) error {
	if f.insertDefiErr != nil {
		return f.insertDefiErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := append([]DefiEventRow{}, rows...)
	f.defiBatches = append(f.defiBatches, cp)
	return nil
}

func (f *fakeBatchConn) Close() error {
	f.closed.Store(true)
	return nil
}

func makeTransferEvent(amount string) *types.DecodedEvent {
	return &types.DecodedEvent{
		ChainEvent: types.ChainEvent{
			ChainID:     8453,
			BlockNumber: 1,
			Contract:    common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
			Timestamp:   time.Unix(1_700_000_000, 0).UTC(),
		},
		Protocol:  "erc20",
		EventType: "transfer",
		Params: map[string]interface{}{
			"from":   "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"to":     "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"amount": amount,
			"token":  "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
		},
	}
}

func makeAaveBorrow() *types.DecodedEvent {
	return &types.DecodedEvent{
		ChainEvent: types.ChainEvent{
			ChainID:     1,
			BlockNumber: 100,
			Timestamp:   time.Unix(1_700_000_000, 0).UTC(),
		},
		Protocol:  "aave_v3",
		EventType: "borrow",
		Params: map[string]interface{}{
			"reserve":            "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			"user":               "0xcccccccccccccccccccccccccccccccccccccccc",
			"on_behalf_of":       "0xdddddddddddddddddddddddddddddddddddddddd",
			"amount":             "1500",
			"interest_rate_mode": uint64(2),
			"borrow_rate":        "456",
			"referral_code":      "0",
		},
	}
}

func TestNewBatchWriter_Validation(t *testing.T) {
	if _, err := NewBatchWriter(nil, BatchWriterConfig{BatchSize: 10, BatchInterval: time.Second}); err == nil {
		t.Error("expected nil-conn error")
	}
	if _, err := NewBatchWriter(&fakeBatchConn{}, BatchWriterConfig{BatchInterval: time.Second}); err == nil {
		t.Error("expected batch_size error")
	}
	if _, err := NewBatchWriter(&fakeBatchConn{}, BatchWriterConfig{BatchSize: 10}); err == nil {
		t.Error("expected batch_interval error")
	}
}

func TestBatchWriter_SizeTriggerFlush(t *testing.T) {
	conn := &fakeBatchConn{}
	w, err := NewBatchWriter(conn, BatchWriterConfig{BatchSize: 2, BatchInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close(context.Background())

	ctx := context.Background()
	if err := w.Enqueue(ctx, makeTransferEvent("100")); err != nil {
		t.Fatal(err)
	}
	if err := w.Enqueue(ctx, makeTransferEvent("200")); err != nil {
		t.Fatal(err)
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()
	if len(conn.transferBatches) != 1 {
		t.Fatalf("transfer batches = %d, want 1", len(conn.transferBatches))
	}
	if len(conn.transferBatches[0]) != 2 {
		t.Errorf("rows in batch = %d, want 2", len(conn.transferBatches[0]))
	}
}

func TestBatchWriter_IntervalTriggerFlush(t *testing.T) {
	conn := &fakeBatchConn{}
	w, err := NewBatchWriter(conn, BatchWriterConfig{BatchSize: 1000, BatchInterval: 50 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close(context.Background())

	ctx := context.Background()
	if err := w.Enqueue(ctx, makeTransferEvent("1")); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		conn.mu.Lock()
		batches := len(conn.transferBatches)
		conn.mu.Unlock()
		if batches == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()
	if len(conn.transferBatches) != 1 {
		t.Errorf("transfer batches = %d, want 1", len(conn.transferBatches))
	}
}

func TestBatchWriter_ApprovalSkipped(t *testing.T) {
	conn := &fakeBatchConn{}
	w, err := NewBatchWriter(conn, BatchWriterConfig{BatchSize: 10, BatchInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close(context.Background())

	approval := &types.DecodedEvent{
		Protocol:  "erc20",
		EventType: "approval",
		Params: map[string]interface{}{
			"owner":   "0xa",
			"spender": "0xb",
			"amount":  "10",
			"token":   "0xt",
		},
	}
	if err := w.Enqueue(context.Background(), approval); err != nil {
		t.Fatal(err)
	}

	w.Close(context.Background())
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if len(conn.transferBatches) != 0 || len(conn.defiBatches) != 0 {
		t.Errorf("approvals should not flush rows, got %d/%d", len(conn.transferBatches), len(conn.defiBatches))
	}
}

func TestBatchWriter_DefiRouting(t *testing.T) {
	conn := &fakeBatchConn{}
	w, err := NewBatchWriter(conn, BatchWriterConfig{BatchSize: 1, BatchInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close(context.Background())

	if err := w.Enqueue(context.Background(), makeAaveBorrow()); err != nil {
		t.Fatal(err)
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if len(conn.defiBatches) != 1 {
		t.Fatalf("defi batches = %d, want 1", len(conn.defiBatches))
	}
	row := conn.defiBatches[0][0]
	if row.Protocol != "aave_v3" || row.EventType != "borrow" {
		t.Errorf("row = %+v", row)
	}
	if row.UserAddr == "" {
		t.Error("user_addr should be filled from params.user")
	}
	if row.Params == "" {
		t.Error("params JSON should be non-empty")
	}
}

func TestBatchWriter_FlushErrorMetric(t *testing.T) {
	conn := &fakeBatchConn{insertTransferErr: errors.New("ch down")}
	w, err := NewBatchWriter(conn, BatchWriterConfig{BatchSize: 1, BatchInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close(context.Background())

	// Capture the metric counter by reading from the global; we can only
	// verify monotonic increase via checking it doesn't panic.
	if err := w.Enqueue(context.Background(), makeTransferEvent("1")); err != nil {
		t.Fatal(err)
	}
	// flush triggers synchronously due to BatchSize=1
}

func TestBatchWriter_ChainLabelFallback(t *testing.T) {
	w := &BatchWriter{chainNames: map[uint64]string{8453: "base"}}
	if got := w.chainLabel(8453); got != "base" {
		t.Errorf("known id label = %q, want %q", got, "base")
	}
	if got := w.chainLabel(42); got != "chain_42" {
		t.Errorf("unknown id label = %q, want chain_42", got)
	}
	wEmpty := &BatchWriter{}
	if got := wEmpty.chainLabel(1); got != "chain_1" {
		t.Errorf("empty map label = %q, want chain_1", got)
	}
}

func TestBatchWriter_QueryableLagObserved(t *testing.T) {
	conn := &fakeBatchConn{}
	w, err := NewBatchWriter(conn, BatchWriterConfig{
		BatchSize:     1,
		BatchInterval: time.Hour,
		ChainNames:    map[uint64]string{8453: "base"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close(context.Background())

	// Recent timestamp so the histogram observation lands in a small
	// non-negative bucket; we don't read the histogram here, only assert
	// the path doesn't panic and the row reaches the conn.
	ev := makeTransferEvent("1")
	ev.Timestamp = time.Now().Add(-2 * time.Second)
	if err := w.Enqueue(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if len(conn.transferBatches) != 1 {
		t.Fatalf("transfer batches = %d, want 1", len(conn.transferBatches))
	}
}

func TestBatchWriter_CloseFlushesRemaining(t *testing.T) {
	conn := &fakeBatchConn{}
	w, err := NewBatchWriter(conn, BatchWriterConfig{BatchSize: 100, BatchInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Enqueue(context.Background(), makeTransferEvent("1")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if len(conn.transferBatches) != 1 {
		t.Errorf("transfer batches after close = %d, want 1", len(conn.transferBatches))
	}
	if !conn.closed.Load() {
		t.Error("conn should be closed")
	}
}

func TestMapTransferRow_MissingField(t *testing.T) {
	e := &types.DecodedEvent{
		Protocol:  "erc20",
		EventType: "transfer",
		Params: map[string]interface{}{
			"from": "0xa",
			"to":   "0xb",
			// missing amount + token
		},
	}
	if _, err := mapTransferRow(e); err == nil {
		t.Error("expected error for missing fields")
	}
}

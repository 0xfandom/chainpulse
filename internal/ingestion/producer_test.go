package ingestion

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/0xfandom/chainpulse/internal/types"
)

// fakeWriter captures every WriteMessages call so tests can assert
// batching behavior without a live broker.
type fakeWriter struct {
	mu    sync.Mutex
	calls [][]kafka.Message
	err   error
}

func (f *fakeWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	cp := make([]kafka.Message, len(msgs))
	copy(cp, msgs)
	f.calls = append(f.calls, cp)
	return nil
}

func (f *fakeWriter) Close() error { return nil }

func (f *fakeWriter) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func TestNewProducer_Validation(t *testing.T) {
	if _, err := NewProducer(ProducerConfig{}); err == nil {
		t.Fatal("expected error for empty brokers")
	}
	if _, err := NewProducer(ProducerConfig{Brokers: []string{"localhost:9092"}}); err == nil {
		t.Fatal("expected error for empty topic")
	}
}

func TestProducer_ChainLabel(t *testing.T) {
	p := &Producer{chainNames: map[uint64]string{8453: "base"}}
	if got := p.chainLabel(8453); got != "base" {
		t.Errorf("chainLabel(base) = %q", got)
	}
	if got := p.chainLabel(42); got != "chain_42" {
		t.Errorf("chainLabel(unknown) = %q", got)
	}
}

func TestPublish_NilEvent(t *testing.T) {
	p, err := NewProducer(ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "raw_events",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Publish(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestPublishBatch_EmptySlice(t *testing.T) {
	p, err := NewProducer(ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "raw_events",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.PublishBatch(context.Background(), nil); err != nil {
		t.Fatalf("nil slice should be no-op, got %v", err)
	}
	if err := p.PublishBatch(context.Background(), []*types.ChainEvent{}); err != nil {
		t.Fatalf("empty slice should be no-op, got %v", err)
	}
}

func TestPublishBatch_NilEntryRejected(t *testing.T) {
	p, err := NewProducer(ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "raw_events",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = p.PublishBatch(context.Background(), []*types.ChainEvent{
		{ChainID: 8453, EventName: "Transfer"},
		nil,
	})
	if err == nil {
		t.Fatal("expected error for nil entry in batch")
	}
}

// TestPublishBatch_SingleWriteCall is the load-bearing assertion for
// the per-block batching change: regardless of how many events a block
// produces, PublishBatch must issue exactly one WriteMessages call.
func TestPublishBatch_SingleWriteCall(t *testing.T) {
	fw := &fakeWriter{}
	p := newProducerWithWriter(fw, "raw_events", map[uint64]string{8453: "base"})

	events := []*types.ChainEvent{
		{ChainID: 8453, EventName: "Transfer"},
		{ChainID: 8453, EventName: "Swap"},
		{ChainID: 8453, EventName: "Borrow"},
	}
	if err := p.PublishBatch(context.Background(), events); err != nil {
		t.Fatalf("PublishBatch: %v", err)
	}

	if got := fw.callCount(); got != 1 {
		t.Fatalf("WriteMessages calls = %d, want 1", got)
	}
	got := fw.calls[0]
	if len(got) != len(events) {
		t.Fatalf("messages in single call = %d, want %d", len(got), len(events))
	}
	wantKey := strconv.AppendUint(nil, 8453, 10)
	for i, m := range got {
		if string(m.Key) != string(wantKey) {
			t.Errorf("msg %d key = %q, want %q", i, string(m.Key), string(wantKey))
		}
		var ev types.ChainEvent
		if err := json.Unmarshal(m.Value, &ev); err != nil {
			t.Errorf("msg %d value not valid JSON: %v", i, err)
		}
	}
}

func TestPublishBatch_MixedChainsRejected(t *testing.T) {
	fw := &fakeWriter{}
	p := newProducerWithWriter(fw, "raw_events", map[uint64]string{8453: "base", 1: "ethereum"})

	err := p.PublishBatch(context.Background(), []*types.ChainEvent{
		{ChainID: 8453, EventName: "Transfer"},
		{ChainID: 1, EventName: "Transfer"},
	})
	if err == nil {
		t.Fatal("expected error for mixed chain ids in single batch")
	}
	if fw.callCount() != 0 {
		t.Errorf("WriteMessages should not be called on validation failure, got %d calls", fw.callCount())
	}
}

// TestPublishBatch_RebuildsWriterAfterStreak asserts that when the
// writer fails writerRebuildThreshold (=3) consecutive batches the
// Producer closes the failing writer and swaps in a fresh one
// returned by the rebuild closure. Simulates the Kafka-broker-recycle
// case where the cached writer connection is permanently poisoned.
func TestPublishBatch_RebuildsWriterAfterStreak(t *testing.T) {
	bad := &fakeWriter{err: errors.New("connection refused")}
	good := &fakeWriter{}

	rebuilds := 0
	p := &Producer{
		writer:     bad,
		topic:      "raw_events",
		chainNames: map[uint64]string{8453: "base"},
		rebuild: func() (messageWriter, error) {
			rebuilds++
			return good, nil
		},
	}

	ev := &types.ChainEvent{ChainID: 8453, EventName: "Transfer"}

	// First two failures: streak < threshold, no rebuild yet.
	for i := 0; i < 2; i++ {
		if err := p.Publish(context.Background(), ev); err == nil {
			t.Fatalf("expected error on attempt %d", i+1)
		}
	}
	if rebuilds != 0 {
		t.Fatalf("expected 0 rebuilds before threshold, got %d", rebuilds)
	}

	// Third failure crosses the threshold and triggers a rebuild.
	if err := p.Publish(context.Background(), ev); err == nil {
		t.Fatal("expected error on threshold attempt")
	}
	if rebuilds != 1 {
		t.Fatalf("expected 1 rebuild after threshold, got %d", rebuilds)
	}

	// Fourth call hits the freshly-rebuilt good writer; success resets
	// the streak so a future bad-writer streak would have to grow from
	// zero again.
	if err := p.Publish(context.Background(), ev); err != nil {
		t.Fatalf("expected success after rebuild, got %v", err)
	}
	if got := good.callCount(); got != 1 {
		t.Fatalf("good writer call count = %d, want 1", got)
	}
}

// Integration-style test ensuring Publish surfaces a transport error when
// the broker is unreachable. Uses a short context deadline so the test
// stays fast even when no broker is running.
func TestPublish_BrokerUnreachable(t *testing.T) {
	p, err := NewProducer(ProducerConfig{
		Brokers: []string{"127.0.0.1:1"}, // closed port
		Topic:   "raw_events",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err = p.Publish(ctx, &types.ChainEvent{ChainID: 8453, EventName: "Transfer"})
	if err == nil {
		t.Fatal("expected error when broker unreachable")
	}
	// Either the deadline fires or the writer surfaces a connection error;
	// both count as a non-nil error path.
	if errors.Is(err, context.Canceled) {
		t.Errorf("unexpected context.Canceled: %v", err)
	}
}

package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/0xfandom/chainpulse/internal/types"
)

// fakeReader implements MessageReader. It serves a fixed list of messages
// then blocks until ctx is cancelled.
type fakeReader struct {
	mu        sync.Mutex
	messages  []kafka.Message
	committed []kafka.Message
	idx       int
	closed    atomic.Bool
}

func (f *fakeReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	for {
		f.mu.Lock()
		if f.idx < len(f.messages) {
			m := f.messages[f.idx]
			f.idx++
			f.mu.Unlock()
			return m, nil
		}
		f.mu.Unlock()
		select {
		case <-ctx.Done():
			return kafka.Message{}, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func (f *fakeReader) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.committed = append(f.committed, msgs...)
	return nil
}

func (f *fakeReader) Stats() kafka.ReaderStats { return kafka.ReaderStats{} }

func (f *fakeReader) Close() error { f.closed.Store(true); return nil }

func mustJSON(t *testing.T, e *types.ChainEvent) []byte {
	t.Helper()
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestNewConsumer_Validation(t *testing.T) {
	cases := []struct {
		name string
		cfg  ConsumerConfig
	}{
		{"empty brokers", ConsumerConfig{Topic: "t", GroupID: "g"}},
		{"empty topic", ConsumerConfig{Brokers: []string{"localhost:9092"}, GroupID: "g"}},
		{"empty group", ConsumerConfig{Brokers: []string{"localhost:9092"}, Topic: "t"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewConsumer(tc.cfg); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestConsumer_HandlesAndCommits(t *testing.T) {
	msg1 := kafka.Message{
		Topic:     "raw_events",
		Partition: 0,
		Offset:    1,
		Value:     mustJSON(t, &types.ChainEvent{ChainID: 8453, EventName: "Transfer"}),
	}
	msg2 := kafka.Message{
		Topic:     "raw_events",
		Partition: 0,
		Offset:    2,
		Value:     mustJSON(t, &types.ChainEvent{ChainID: 8453, EventName: "Transfer"}),
	}
	r := &fakeReader{messages: []kafka.Message{msg1, msg2}}
	c := newConsumerWithReader(r, "raw_events")

	var got atomic.Int32
	handler := func(ctx context.Context, e *types.ChainEvent) error {
		got.Add(1)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx, handler) }()

	deadline := time.Now().Add(2 * time.Second)
	for got.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done

	if got.Load() != 2 {
		t.Errorf("handler calls = %d, want 2", got.Load())
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.committed) != 2 {
		t.Errorf("committed = %d, want 2", len(r.committed))
	}
}

func TestConsumer_HandlerErrorSkipsCommit(t *testing.T) {
	bad := kafka.Message{
		Topic:     "raw_events",
		Partition: 0,
		Offset:    7,
		Value:     mustJSON(t, &types.ChainEvent{ChainID: 8453, EventName: "Transfer"}),
	}
	r := &fakeReader{messages: []kafka.Message{bad}}
	c := newConsumerWithReader(r, "raw_events")

	calls := atomic.Int32{}
	handler := func(ctx context.Context, e *types.ChainEvent) error {
		calls.Add(1)
		return errors.New("boom")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx, handler) }()
	<-done

	if calls.Load() < 1 {
		t.Errorf("handler should have been called at least once, got %d", calls.Load())
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.committed) != 0 {
		t.Errorf("committed = %d, want 0", len(r.committed))
	}
}

func TestConsumer_DecodeErrorSkipsCommit(t *testing.T) {
	bad := kafka.Message{
		Topic:     "raw_events",
		Partition: 0,
		Offset:    9,
		Value:     []byte("{not json"),
	}
	r := &fakeReader{messages: []kafka.Message{bad}}
	c := newConsumerWithReader(r, "raw_events")

	called := false
	handler := func(ctx context.Context, e *types.ChainEvent) error {
		called = true
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx, handler) }()
	<-done

	if called {
		t.Error("handler should not run for undecodable payload")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.committed) != 0 {
		t.Errorf("committed = %d, want 0", len(r.committed))
	}
}

func TestConsumer_NilHandlerErrors(t *testing.T) {
	c := newConsumerWithReader(&fakeReader{}, "raw_events")
	if err := c.Run(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil handler")
	}
}

// Compile-time guard: the fake satisfies the MessageReader interface.
var _ MessageReader = (*fakeReader)(nil)

func TestConsumer_CloseClosesReader(t *testing.T) {
	r := &fakeReader{}
	c := newConsumerWithReader(r, "raw_events")
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !r.closed.Load() {
		t.Error("reader not closed")
	}

	// covers the nil-receiver branch
	var nilC *Consumer
	if err := nilC.Close(); err != nil {
		t.Errorf("nil consumer Close: %v", err)
	}
	_ = fmt.Sprintf("%v", c.topic) // keep variable from being unused on later edits
}

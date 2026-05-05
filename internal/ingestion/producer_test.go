package ingestion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0xfandom/chainpulse/internal/types"
)

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

package processor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

func TestNewProcessorProducer_Validation(t *testing.T) {
	cases := []struct {
		name string
		cfg  ProducerConfig
	}{
		{"empty brokers", ProducerConfig{TopicDecodedEvents: "d", TopicPositionsUpdate: "p"}},
		{"empty decoded topic", ProducerConfig{Brokers: []string{"localhost:9092"}, TopicPositionsUpdate: "p"}},
		{"empty positions topic", ProducerConfig{Brokers: []string{"localhost:9092"}, TopicDecodedEvents: "d"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewProcessorProducer(tc.cfg); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestProcessorProducer_ChainLabel(t *testing.T) {
	p := &Producer{chainNames: map[uint64]string{8453: "base"}}
	if got := p.chainLabel(8453); got != "base" {
		t.Errorf("chainLabel(known) = %q", got)
	}
	if got := p.chainLabel(7); got != "chain_7" {
		t.Errorf("chainLabel(unknown) = %q", got)
	}
}

func TestProcessorProducer_NilGuards(t *testing.T) {
	p, err := NewProcessorProducer(ProducerConfig{
		Brokers:              []string{"localhost:9092"},
		TopicDecodedEvents:   "decoded_events",
		TopicPositionsUpdate: "positions_update",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if err := p.PublishDecoded(context.Background(), nil); err == nil {
		t.Error("expected error for nil decoded event")
	}
	if err := p.PublishPositionUpdate(context.Background(), nil); err == nil {
		t.Error("expected error for nil position")
	}
}

// TestProcessorProducer_BrokerUnreachable verifies error surfacing when
// broker isn't reachable. Short deadline keeps the test fast.
func TestProcessorProducer_BrokerUnreachable(t *testing.T) {
	p, err := NewProcessorProducer(ProducerConfig{
		Brokers:              []string{"127.0.0.1:1"},
		TopicDecodedEvents:   "decoded_events",
		TopicPositionsUpdate: "positions_update",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err = p.PublishDecoded(ctx, &types.DecodedEvent{
		ChainEvent: types.ChainEvent{ChainID: 8453, EventName: "Transfer"},
		Protocol:   "erc20",
		EventType:  "transfer",
	})
	if err == nil {
		t.Error("expected publish error against unreachable broker")
	}
	if errors.Is(err, context.Canceled) {
		t.Errorf("unexpected context.Canceled: %v", err)
	}

	err = p.PublishPositionUpdate(ctx, &types.WalletPosition{
		Wallet:   common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		ChainID:  1,
		Protocol: "aave_v3",
	})
	if err == nil {
		t.Error("expected publish error for position update")
	}
}

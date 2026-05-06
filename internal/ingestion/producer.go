package ingestion

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"

	"github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/monitor"
	"github.com/0xfandom/chainpulse/internal/types"
)

// Producer wraps a kafka-go writer for the raw_events topic. It JSON-encodes
// ChainEvents and partitions by chain_id so per-chain ordering is preserved
// across consumer instances.
type Producer struct {
	writer     *kafka.Writer
	topic      string
	chainNames map[uint64]string
}

// ProducerConfig is the input to NewProducer.
type ProducerConfig struct {
	Brokers    []string
	Topic      string
	ClientID   string
	ChainNames map[uint64]string
}

// NewProducer constructs a Producer. ChainNames maps chain_id to a
// human-readable chain label used in metrics; missing ids fall back to
// "chain_<id>".
func NewProducer(cfg ProducerConfig) (*Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka producer: brokers must not be empty")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("kafka producer: topic must be set")
	}

	w := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		Topic:                  cfg.Topic,
		Balancer:               &kafka.Hash{},
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: true,
		Async:                  false,
	}
	if cfg.ClientID != "" {
		w.Transport = &kafka.Transport{ClientID: cfg.ClientID}
	}

	return &Producer{
		writer:     w,
		topic:      cfg.Topic,
		chainNames: cfg.ChainNames,
	}, nil
}

// Publish marshals event to JSON and writes it to Kafka, keyed by chain_id.
// Increments the raw_events_produced_total metric on success.
func (p *Producer) Publish(ctx context.Context, event *types.ChainEvent) error {
	if event == nil {
		return fmt.Errorf("publish: nil event")
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(fmt.Sprintf("%d", event.ChainID)),
		Value: payload,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka write: %w", err)
	}

	monitor.IncRawEventsProduced(p.chainLabel(event.ChainID), event.EventName)
	return nil
}

// Close flushes pending writes and closes the underlying Kafka writer.
func (p *Producer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	if err := p.writer.Close(); err != nil {
		l := log.Component("kafka_producer")
		l.Error().Err(err).Msg("close failed")
		return err
	}
	return nil
}

func (p *Producer) chainLabel(id uint64) string {
	if name, ok := p.chainNames[id]; ok && name != "" {
		return name
	}
	return fmt.Sprintf("chain_%d", id)
}

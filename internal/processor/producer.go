package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"

	chainpulselog "github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/monitor"
	"github.com/0xfandom/chainpulse/internal/types"
)

// Decoded-events writer is configured for async batched publish so
// the consumer goroutine is not blocked on a per-event Kafka
// roundtrip. ERC-20 Transfer alone produces hundreds of events per
// second on Polygon-class chains; a single-goroutine consumer that
// waited for each ack ceilinged throughput at ~100 events/sec and
// kafka_lag climbed unbounded. Errors surface via the Completion
// callback set when the writer is constructed.
const (
	decodedBatchSize    = 200
	decodedBatchTimeout = 200 * time.Millisecond
)

// Producer wraps two kafka-go writers — one for decoded_events, one for
// positions_update — sharing a single transport so producers reuse the
// same client_id and connection pool.
type Producer struct {
	decoded   *kafka.Writer
	positions *kafka.Writer

	decodedTopic   string
	positionsTopic string

	chainNames map[uint64]string
}

// ProducerConfig is the input to NewProcessorProducer.
type ProducerConfig struct {
	Brokers              []string
	TopicDecodedEvents   string
	TopicPositionsUpdate string
	ClientID             string
	ChainNames           map[uint64]string
}

// NewProcessorProducer constructs the processor's Kafka producer.
func NewProcessorProducer(cfg ProducerConfig) (*Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("processor producer: brokers must not be empty")
	}
	if cfg.TopicDecodedEvents == "" {
		return nil, fmt.Errorf("processor producer: topic_decoded_events must be set")
	}
	if cfg.TopicPositionsUpdate == "" {
		return nil, fmt.Errorf("processor producer: topic_positions_update must be set")
	}

	transport := &kafka.Transport{}
	if cfg.ClientID != "" {
		transport.ClientID = cfg.ClientID
	}

	mk := func(topic string) *kafka.Writer {
		w := &kafka.Writer{
			Addr:                   kafka.TCP(cfg.Brokers...),
			Topic:                  topic,
			Balancer:               &kafka.Hash{},
			RequiredAcks:           kafka.RequireAll,
			AllowAutoTopicCreation: true,
			Async:                  false,
			Transport:              transport,
		}
		return w
	}

	// decoded_events is the high-volume republish path. Configure the
	// writer for async batched publish so PublishDecoded never blocks
	// the consumer goroutine on a per-event Kafka roundtrip. Errors
	// surface via the Completion callback (one log + one metric bump
	// per failed batch) — at-least-once is preserved for the
	// authoritative copy in raw_events; decoded_events is best-effort.
	decoded := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		Topic:                  cfg.TopicDecodedEvents,
		Balancer:               &kafka.Hash{},
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: true,
		Async:                  true,
		BatchSize:              decodedBatchSize,
		BatchTimeout:           decodedBatchTimeout,
		Transport:              transport,
	}
	decoded.Completion = func(messages []kafka.Message, err error) {
		if err != nil {
			monitor.IncProcessorError(monitor.ProcErrKafkaPublish)
			l := chainpulselog.Component("processor_producer")
			l.Error().Err(err).Int("batch", len(messages)).Str("topic", cfg.TopicDecodedEvents).Msg("async decoded publish failed")
		}
	}

	return &Producer{
		decoded:        decoded,
		positions:      mk(cfg.TopicPositionsUpdate),
		decodedTopic:   cfg.TopicDecodedEvents,
		positionsTopic: cfg.TopicPositionsUpdate,
		chainNames:     cfg.ChainNames,
	}, nil
}

// PublishDecoded marshals event to JSON and enqueues it for the
// decoded_events writer, keyed by chain_id. Returns immediately —
// the underlying writer is configured Async, so the actual broker
// roundtrip happens in background and any failure surfaces via the
// Completion callback set in NewProcessorProducer.
//
// The decoded_events_produced_total metric is incremented on enqueue
// rather than on broker ack; in the failure case the
// processor_errors_total{kafka_publish} counter (bumped from the
// Completion callback) gives the offsetting signal.
func (p *Producer) PublishDecoded(ctx context.Context, event *types.DecodedEvent) error {
	if event == nil {
		return fmt.Errorf("publish decoded: nil event")
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal decoded: %w", err)
	}
	msg := kafka.Message{
		Key:   []byte(fmt.Sprintf("%d", event.ChainID)),
		Value: payload,
	}
	if err := p.decoded.WriteMessages(ctx, msg); err != nil {
		monitor.IncProcessorError(monitor.ProcErrKafkaPublish)
		return fmt.Errorf("kafka write decoded: %w", err)
	}
	monitor.IncDecodedEventsProduced(p.chainLabel(event.ChainID), event.Protocol, event.EventType)
	return nil
}

// PublishPositionUpdate marshals pos to JSON and writes it to
// positions_update, keyed by wallet hex.
func (p *Producer) PublishPositionUpdate(ctx context.Context, pos *types.WalletPosition) error {
	if pos == nil {
		return fmt.Errorf("publish position: nil position")
	}
	payload, err := json.Marshal(pos)
	if err != nil {
		return fmt.Errorf("marshal position: %w", err)
	}
	msg := kafka.Message{
		Key:   []byte(strings.ToLower(pos.Wallet.Hex())),
		Value: payload,
	}
	if err := p.positions.WriteMessages(ctx, msg); err != nil {
		monitor.IncProcessorError(monitor.ProcErrKafkaPublish)
		return fmt.Errorf("kafka write position: %w", err)
	}
	monitor.IncPositionUpdatesProduced(p.chainLabel(pos.ChainID), pos.Protocol)
	return nil
}

// Close flushes pending writes and closes both writers.
func (p *Producer) Close() error {
	if p == nil {
		return nil
	}
	var firstErr error
	if p.decoded != nil {
		if err := p.decoded.Close(); err != nil {
			firstErr = err
			l := chainpulselog.Component("processor_producer")
			l.Error().Err(err).Str("topic", p.decodedTopic).Msg("close decoded writer failed")
		}
	}
	if p.positions != nil {
		if err := p.positions.Close(); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			l := chainpulselog.Component("processor_producer")
			l.Error().Err(err).Str("topic", p.positionsTopic).Msg("close positions writer failed")
		}
	}
	return firstErr
}

func (p *Producer) chainLabel(id uint64) string {
	if name, ok := p.chainNames[id]; ok && name != "" {
		return name
	}
	return fmt.Sprintf("chain_%d", id)
}

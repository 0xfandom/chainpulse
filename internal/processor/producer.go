package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/segmentio/kafka-go"

	chainpulselog "github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/monitor"
	"github.com/0xfandom/chainpulse/internal/types"
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

	return &Producer{
		decoded:        mk(cfg.TopicDecodedEvents),
		positions:      mk(cfg.TopicPositionsUpdate),
		decodedTopic:   cfg.TopicDecodedEvents,
		positionsTopic: cfg.TopicPositionsUpdate,
		chainNames:     cfg.ChainNames,
	}, nil
}

// PublishDecoded marshals event to JSON and writes it to decoded_events,
// keyed by chain_id. Increments decoded_events_produced_total.
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

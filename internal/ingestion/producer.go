package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/monitor"
	"github.com/0xfandom/chainpulse/internal/types"
)

// Producer-side defaults. The hot path is one PublishBatch call per
// block; BatchSize/BatchTimeout only matter if concurrent calls overlap.
// MaxAttempts is intentionally low: retries are bounded so a degraded
// broker cannot inflate the indexer's per-block latency budget.
const (
	defaultBatchSize    = 100
	defaultBatchTimeout = 20 * time.Millisecond
	defaultMaxAttempts  = 3

	// writerRebuildThreshold is the consecutive WriteMessages failure
	// count that triggers a writer reconstruction. segmentio/kafka-go
	// caches broker connections and resolved DNS at the writer level;
	// after a Kafka container recycle or a transient docker DNS
	// NXDOMAIN at indexer startup the cached state stays poisoned for
	// the lifetime of the process. Closing and rebuilding the writer
	// forces a fresh dial.
	writerRebuildThreshold = 3
)

// messageWriter is the subset of *kafka.Writer the Producer uses. The
// indirection lets unit tests intercept WriteMessages calls without
// spinning up a broker.
type messageWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// Producer wraps a kafka-go writer for the raw_events topic. It JSON-encodes
// ChainEvents and partitions by chain_id so per-chain ordering is preserved
// across consumer instances.
type Producer struct {
	mu         sync.Mutex
	writer     messageWriter
	topic      string
	chainNames map[uint64]string

	// rebuild reconstructs the underlying writer when a consecutive
	// publish-failure streak reaches writerRebuildThreshold. nil for
	// test producers built via newProducerWithWriter (no rebuild path,
	// the streak counter still increments but never recreates).
	rebuild    func() (messageWriter, error)
	failStreak int
}

// ProducerConfig is the input to NewProducer.
//
// RequiredAcks defaults to RequireAll for durability. Operators that
// prefer to trade durability for lower per-block latency can override
// to RequireOne, but the trade-off (loss on leader failure pre-replication)
// must be accepted explicitly.
type ProducerConfig struct {
	Brokers      []string
	Topic        string
	ClientID     string
	ChainNames   map[uint64]string
	RequiredAcks kafka.RequiredAcks
	MaxAttempts  int
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

	acks := cfg.RequiredAcks
	if acks == 0 {
		acks = kafka.RequireAll
	}
	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempts
	}

	build := func() (messageWriter, error) {
		w := &kafka.Writer{
			Addr:                   kafka.TCP(cfg.Brokers...),
			Topic:                  cfg.Topic,
			Balancer:               &kafka.Hash{},
			RequiredAcks:           acks,
			AllowAutoTopicCreation: true,
			Async:                  false,
			BatchSize:              defaultBatchSize,
			BatchTimeout:           defaultBatchTimeout,
			MaxAttempts:            maxAttempts,
		}
		if cfg.ClientID != "" {
			w.Transport = &kafka.Transport{ClientID: cfg.ClientID}
		}
		return w, nil
	}
	w, err := build()
	if err != nil {
		return nil, err
	}

	return &Producer{
		writer:     w,
		topic:      cfg.Topic,
		chainNames: cfg.ChainNames,
		rebuild:    build,
	}, nil
}

// newProducerWithWriter is the unexported constructor used by tests. It
// skips broker validation so a fake writer can stand in.
func newProducerWithWriter(w messageWriter, topic string, chainNames map[uint64]string) *Producer {
	return &Producer{writer: w, topic: topic, chainNames: chainNames}
}

// Publish marshals event to JSON and writes it to Kafka, keyed by chain_id.
// Increments the raw_events_produced_total metric on success.
//
// For per-block fan-out callers should prefer PublishBatch — it issues a
// single WriteMessages call and emits a kafka_publish_duration_seconds
// observation labeled by chain.
func (p *Producer) Publish(ctx context.Context, event *types.ChainEvent) error {
	if event == nil {
		return fmt.Errorf("publish: nil event")
	}
	return p.PublishBatch(ctx, []*types.ChainEvent{event})
}

// PublishBatch marshals every event in the slice once and writes the
// resulting kafka.Messages in a single WriteMessages call. Per-chain
// ordering is preserved by partitioning on chain_id; messages within a
// batch sharing a key land on the same partition in order.
//
// All events in a batch must share the same ChainID — the duration
// histogram is labeled by chain, so a mixed batch would produce an
// ambiguous observation. Empty or all-nil slices are a no-op. A nil
// entry inside the slice is an error.
func (p *Producer) PublishBatch(ctx context.Context, events []*types.ChainEvent) error {
	if len(events) == 0 {
		return nil
	}

	chainID := events[0].ChainID
	msgs := make([]kafka.Message, 0, len(events))
	for i, ev := range events {
		if ev == nil {
			return fmt.Errorf("publish batch: nil event at index %d", i)
		}
		if ev.ChainID != chainID {
			return fmt.Errorf("publish batch: mixed chain ids (%d vs %d at index %d)", chainID, ev.ChainID, i)
		}
		payload, err := json.Marshal(ev)
		if err != nil {
			return fmt.Errorf("marshal event %d: %w", i, err)
		}
		msgs = append(msgs, kafka.Message{
			Key:   strconv.AppendUint(nil, ev.ChainID, 10),
			Value: payload,
		})
	}

	chainLabel := p.chainLabel(chainID)
	start := time.Now()
	p.mu.Lock()
	w := p.writer
	p.mu.Unlock()
	if err := w.WriteMessages(ctx, msgs...); err != nil {
		p.recordWriteFailure()
		return fmt.Errorf("kafka write batch (%d msgs): %w", len(msgs), err)
	}
	p.recordWriteSuccess()
	monitor.ObserveKafkaPublish(chainLabel, time.Since(start))

	for _, ev := range events {
		monitor.IncRawEventsProduced(chainLabel, ev.EventName)
	}
	return nil
}

// recordWriteSuccess resets the consecutive-failure streak after any
// successful WriteMessages. Cheap: lock-acquire + zero-store on the
// hot path, contention only with rebuild attempts (very rare).
func (p *Producer) recordWriteSuccess() {
	p.mu.Lock()
	p.failStreak = 0
	p.mu.Unlock()
}

// recordWriteFailure increments the consecutive-failure streak. Once
// the streak hits writerRebuildThreshold the underlying writer is
// closed and reconstructed so DNS resolves fresh and broker
// connections are re-dialed. The streak is reset whether the rebuild
// succeeds or not — a failed rebuild attempt should not pin the
// writer in a permanent rebuild loop on every subsequent batch; the
// next publish failure restarts the count.
//
// Tests using newProducerWithWriter pass nil rebuild and only the
// streak counter increments.
func (p *Producer) recordWriteFailure() {
	p.mu.Lock()
	p.failStreak++
	streak := p.failStreak
	rebuild := p.rebuild
	p.mu.Unlock()

	if rebuild == nil || streak < writerRebuildThreshold {
		return
	}

	l := log.Component("kafka_producer")
	l.Warn().Int("streak", streak).Msg("kafka writer publish streak exceeded; rebuilding writer")

	w, err := rebuild()
	if err != nil {
		l.Error().Err(err).Msg("kafka writer rebuild failed")
		p.mu.Lock()
		p.failStreak = 0
		p.mu.Unlock()
		return
	}

	p.mu.Lock()
	old := p.writer
	p.writer = w
	p.failStreak = 0
	p.mu.Unlock()

	if old != nil {
		_ = old.Close()
	}
}

// Close flushes pending writes and closes the underlying Kafka writer.
func (p *Producer) Close() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	w := p.writer
	p.writer = nil
	p.mu.Unlock()
	if w == nil {
		return nil
	}
	if err := w.Close(); err != nil {
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

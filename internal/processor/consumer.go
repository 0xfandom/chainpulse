package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/segmentio/kafka-go"

	chainpulselog "github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/monitor"
	"github.com/0xfandom/chainpulse/internal/types"
)

// retrySleep is the back-off delay applied when the handler errors. The
// message is left uncommitted so it will be re-delivered after the next
// rebalance / restart. Documented Day-2 limitation: a poison message
// causes a tight retry loop until ops intervenes; a dead-letter topic
// follows in Day 5+.
const retrySleep = 100 * time.Millisecond

// commitInterval is the cadence at which the underlying Reader flushes
// the latest acknowledged offsets to the broker. Manual CommitMessages
// calls update the in-memory offset; the Reader's background loop
// performs the actual broker round-trip every commitInterval. Trades
// at-most-200ms of redelivery on crash for one to two orders of
// magnitude fewer commit RTTs on the hot path.
const commitInterval = 200 * time.Millisecond

// MessageHandler processes a single ChainEvent. Returning an error keeps
// the message un-committed so Kafka will redeliver it.
type MessageHandler func(ctx context.Context, event *types.ChainEvent) error

// MessageReader is the subset of kafka-go's Reader the consumer uses.
// Defined as an interface so unit tests can swap in a fake.
type MessageReader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Stats() kafka.ReaderStats
	Close() error
}

// Consumer wraps a kafka-go consumer-group reader and dispatches each
// message to a MessageHandler.
type Consumer struct {
	reader MessageReader
	topic  string
}

// ConsumerConfig is the input to NewConsumer.
type ConsumerConfig struct {
	Brokers     []string
	Topic       string
	GroupID     string
	ClientID    string
	MaxInFlight int
}

// NewConsumer constructs a Consumer with a real kafka-go Reader.
func NewConsumer(cfg ConsumerConfig) (*Consumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka consumer: brokers must not be empty")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("kafka consumer: topic must be set")
	}
	if cfg.GroupID == "" {
		return nil, fmt.Errorf("kafka consumer: group_id must be set")
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: commitInterval,
	})
	return newConsumerWithReader(r, cfg.Topic), nil
}

// newConsumerWithReader is the unexported constructor used by tests.
func newConsumerWithReader(r MessageReader, topic string) *Consumer {
	return &Consumer{reader: r, topic: topic}
}

// Run drains the consumer until ctx is cancelled. Each fetched message is
// JSON-decoded and handed to handler. On handler success the offset is
// committed; on failure the offset is left uncommitted, an error counter
// is bumped, and the loop sleeps briefly before fetching again.
//
// A separate goroutine refreshes the processor_kafka_lag_messages gauge
// once per second from the underlying reader's Stats.
func (c *Consumer) Run(ctx context.Context, handler MessageHandler) error {
	if handler == nil {
		return fmt.Errorf("consumer: nil handler")
	}

	logger := chainpulselog.Component("kafka_consumer")
	logger.Info().Str("topic", c.topic).Msg("consumer starting")

	statsCtx, cancelStats := context.WithCancel(ctx)
	defer cancelStats()
	go c.runStatsLoop(statsCtx)

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		start := time.Now()
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) {
				return nil
			}
			logger.Error().Err(err).Msg("fetch message failed")
			monitor.IncProcessorError(monitor.ProcErrDecode)
			c.sleep(ctx, retrySleep)
			continue
		}

		var event types.ChainEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			logger.Error().Err(err).Int64("offset", msg.Offset).Msg("decode failed; skipping commit")
			monitor.IncProcessorError(monitor.ProcErrDecode)
			c.sleep(ctx, retrySleep)
			continue
		}

		if err := handler(ctx, &event); err != nil {
			logger.Error().Err(err).Int64("offset", msg.Offset).Msg("handler failed; skipping commit")
			monitor.IncProcessorError(monitor.ProcErrAggregate)
			c.sleep(ctx, retrySleep)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			logger.Error().Err(err).Int64("offset", msg.Offset).Msg("commit failed")
			monitor.IncProcessorError(monitor.ProcErrKafkaPublish)
			continue
		}
		monitor.ObserveEventProcessing(time.Since(start))
	}
}

// Close shuts down the underlying reader.
func (c *Consumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

func (c *Consumer) runStatsLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := c.reader.Stats()
			monitor.SetKafkaConsumerLag(c.topic, stats.Partition, float64(stats.Lag))
		}
	}
}

// sleep is a context-aware time.Sleep wrapper.
func (c *Consumer) sleep(ctx context.Context, d time.Duration) {
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}

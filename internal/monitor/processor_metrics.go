package monitor

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Processor-side metrics. Registered against the default registry on
// import; the existing ServeMetrics handler exposes them at /metrics.
var (
	// DecodedEventsProducedTotal counts events the processor wrote to the
	// decoded_events Kafka topic.
	DecodedEventsProducedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "decoded_events_produced_total",
		Help: "Total number of decoded events produced to Kafka by the processor.",
	}, []string{"chain", "protocol", "event_type"})

	// PositionUpdatesProducedTotal counts position deltas the processor
	// wrote to the positions_update Kafka topic.
	PositionUpdatesProducedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "position_updates_produced_total",
		Help: "Total number of wallet position updates produced by the processor.",
	}, []string{"chain", "protocol"})

	// ProcessorConsumeErrorsTotal counts pipeline failures by stage.
	// Allowed kinds: decode, aggregate, clickhouse_write, redis_write,
	// kafka_publish.
	ProcessorConsumeErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "processor_consume_errors_total",
		Help: "Errors observed by the processor pipeline, labeled by stage.",
	}, []string{"kind"})

	// ClickHouseBatchFlushSeconds tracks batch insert wall-clock time.
	ClickHouseBatchFlushSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "clickhouse_batch_flush_seconds",
		Help:    "Wall-clock time spent flushing a ClickHouse batch insert.",
		Buckets: prometheus.ExponentialBuckets(0.005, 2, 12),
	})

	// RedisWriteSeconds tracks single-op Redis write latency.
	RedisWriteSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "redis_write_seconds",
		Help:    "Wall-clock time of a single Redis write operation.",
		Buckets: prometheus.ExponentialBuckets(0.0005, 2, 12),
	})

	// ProcessorEventProcessingSeconds tracks per-message wall-clock from
	// Kafka fetch to successful aggregate + commit signal.
	ProcessorEventProcessingSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "processor_event_processing_seconds",
		Help:    "Wall-clock time to fetch, decode, aggregate, and acknowledge a single event.",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 14), // 1ms .. ~16s
	})

	// ProcessorKafkaLagMessages is set by the consumer once per second
	// from the underlying reader's stats. Labeled by topic and partition.
	ProcessorKafkaLagMessages = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "processor_kafka_lag_messages",
		Help: "Current Kafka consumer lag in messages, per topic and partition.",
	}, []string{"topic", "partition"})
)

// Allowed values for the processor_consume_errors_total kind label.
const (
	ProcErrDecode          = "decode"
	ProcErrAggregate       = "aggregate"
	ProcErrClickHouseWrite = "clickhouse_write"
	ProcErrRedisWrite      = "redis_write"
	ProcErrKafkaPublish    = "kafka_publish"
)

// IncDecodedEventsProduced bumps the decoded events counter.
func IncDecodedEventsProduced(chain, protocol, eventType string) {
	DecodedEventsProducedTotal.WithLabelValues(chain, protocol, eventType).Inc()
}

// IncPositionUpdatesProduced bumps the position updates counter.
func IncPositionUpdatesProduced(chain, protocol string) {
	PositionUpdatesProducedTotal.WithLabelValues(chain, protocol).Inc()
}

// IncProcessorError bumps the processor errors counter.
func IncProcessorError(kind string) {
	ProcessorConsumeErrorsTotal.WithLabelValues(kind).Inc()
}

// ObserveClickHouseFlush records a single batch flush duration.
func ObserveClickHouseFlush(d time.Duration) {
	ClickHouseBatchFlushSeconds.Observe(d.Seconds())
}

// ObserveRedisWrite records a single Redis write duration.
func ObserveRedisWrite(d time.Duration) {
	RedisWriteSeconds.Observe(d.Seconds())
}

// ObserveEventProcessing records a single per-message processing duration.
func ObserveEventProcessing(d time.Duration) {
	ProcessorEventProcessingSeconds.Observe(d.Seconds())
}

// SetKafkaConsumerLag updates the consumer lag gauge for a partition.
func SetKafkaConsumerLag(topic, partition string, lag float64) {
	ProcessorKafkaLagMessages.WithLabelValues(topic, partition).Set(lag)
}

// Package monitor exposes Prometheus metrics for the indexer service.
//
// Metrics are registered against the default Prometheus registry on import,
// so consumers can call the helpers without any setup beyond ServeMetrics.
package monitor

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// RawEventsProducedTotal counts events the indexer wrote to Kafka,
	// labeled by chain name and event name.
	RawEventsProducedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "raw_events_produced_total",
		Help: "Total number of raw events produced to Kafka by the indexer.",
	}, []string{"chain", "event_name"})

	// BlockProcessingDuration measures the wall-clock time the listener
	// spends turning a block header into Kafka writes.
	BlockProcessingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "block_processing_duration_seconds",
		Help:    "Time spent fetching and decoding logs for a single block.",
		Buckets: prometheus.ExponentialBuckets(0.005, 2, 12), // 5ms .. ~20s
	}, []string{"chain"})

	// ChainListenerConnected is 1 when the chain WebSocket subscription is
	// alive, 0 when disconnected or reconnecting.
	ChainListenerConnected = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "chain_listener_connected",
		Help: "1 if the chain listener has an active WebSocket subscription, 0 otherwise.",
	}, []string{"chain"})

	// KafkaPublishDuration measures the time spent writing a per-block
	// batch of events to the raw_events topic.
	KafkaPublishDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "kafka_publish_duration_seconds",
		Help:    "Time spent writing a per-block batch of events to Kafka.",
		Buckets: prometheus.ExponentialBuckets(0.0005, 2, 14), // 0.5ms .. ~8s
	}, []string{"chain"})
)

// ObserveBlockProcessing records a single block-processing duration sample.
func ObserveBlockProcessing(chain string, dur time.Duration) {
	BlockProcessingDuration.WithLabelValues(chain).Observe(dur.Seconds())
}

// SetListenerConnected toggles the connected gauge for a chain.
func SetListenerConnected(chain string, connected bool) {
	v := 0.0
	if connected {
		v = 1.0
	}
	ChainListenerConnected.WithLabelValues(chain).Set(v)
}

// IncRawEventsProduced bumps the produced-events counter for a chain/event.
func IncRawEventsProduced(chain, eventName string) {
	RawEventsProducedTotal.WithLabelValues(chain, eventName).Inc()
}

// ObserveKafkaPublish records a per-block Kafka batch publish duration.
func ObserveKafkaPublish(chain string, dur time.Duration) {
	KafkaPublishDuration.WithLabelValues(chain).Observe(dur.Seconds())
}

// ServeMetrics starts an HTTP server exposing /metrics on addr. It blocks
// until the server returns or ctx is cancelled. http.ErrServerClosed from
// graceful shutdown is treated as a clean exit.
func ServeMetrics(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

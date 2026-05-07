package monitor

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// API-side metrics. Registered against the default registry so the
// existing ServeMetrics handler exposes them at /metrics.
var (
	APIRequestDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "api_request_duration_seconds",
		Help:    "Wall-clock duration of API requests, labeled by method, path, and status.",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 14), // 1ms .. ~16s
	}, []string{"method", "path", "status"})

	APIRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "api_requests_total",
		Help: "Total number of API requests, labeled by method, path, and status.",
	}, []string{"method", "path", "status"})

	APIWebsocketConnectionsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "api_websocket_connections_active",
		Help: "Number of active WebSocket connections to the api service.",
	})

	APICacheHitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "api_cache_hits_total",
		Help: "Cache hits / misses observed by api handlers.",
	}, []string{"source"}) // source in {l1, redis, clickhouse}

	APICacheHitSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "api_cache_hit_seconds",
		Help:    "Wall-clock time of a cache lookup that resulted in a hit, labeled by source.",
		Buckets: prometheus.ExponentialBuckets(0.0001, 2, 12), // 100us .. ~400ms
	}, []string{"source"})

	APIRateLimitedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "api_rate_limited_total",
		Help: "Number of API requests rejected by the rate limiter.",
	})
)

// Allowed values for the api_cache_hits_total source label.
const (
	APICacheSourceL1         = "l1"
	APICacheSourceRedis      = "redis"
	APICacheSourceClickHouse = "clickhouse"
)

// ObserveAPIRequest records duration + count for a single API request.
func ObserveAPIRequest(method, path, status string, dur time.Duration) {
	APIRequestDurationSeconds.WithLabelValues(method, path, status).Observe(dur.Seconds())
	APIRequestsTotal.WithLabelValues(method, path, status).Inc()
}

// IncAPICacheHit bumps the per-source cache hit counter. Use one of the
// APICacheSource* constants for source.
func IncAPICacheHit(source string) {
	APICacheHitsTotal.WithLabelValues(source).Inc()
}

// ObserveAPICacheHit records the wall-clock duration of a cache lookup
// that resulted in a hit. Misses (which fall through to ClickHouse)
// are bounded by the request-level api_request_duration_seconds.
func ObserveAPICacheHit(source string, dur time.Duration) {
	APICacheHitSeconds.WithLabelValues(source).Observe(dur.Seconds())
}

// IncAPIRateLimited bumps the rate-limit rejection counter.
func IncAPIRateLimited() { APIRateLimitedTotal.Inc() }

// IncAPIWSConnection adjusts the active-WS gauge by delta (+1 on accept,
// -1 on close).
func IncAPIWSConnection(delta int) {
	APIWebsocketConnectionsActive.Add(float64(delta))
}

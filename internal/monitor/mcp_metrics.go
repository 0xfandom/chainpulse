package monitor

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MCP-side metrics. Registered against the default registry so the
// existing ServeMetrics handler exposes them at /metrics.
var (
	MCPToolCallsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mcp_tool_calls_total",
		Help: "MCP tools/call invocations, labeled by tool and status.",
	}, []string{"tool", "status"})

	MCPToolLatencySeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mcp_tool_latency_seconds",
		Help:    "Wall-clock duration of MCP tools/call invocations, labeled by tool.",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 14),
	}, []string{"tool"})

	MCPActiveSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mcp_active_sessions",
		Help: "Active MCP SSE subscribers.",
	})
)

// Allowed values for the mcp_tool_calls_total status label. Bounded so
// the metric's label cardinality stays predictable.
const (
	MCPStatusOK              = "ok"
	MCPStatusError           = "error"
	MCPStatusValidationError = "validation_error"
)

// ObserveMCPToolCall records latency + outcome for one tool dispatch.
// status MUST be one of the MCPStatus* constants.
func ObserveMCPToolCall(tool, status string, dur time.Duration) {
	MCPToolCallsTotal.WithLabelValues(tool, status).Inc()
	MCPToolLatencySeconds.WithLabelValues(tool).Observe(dur.Seconds())
}

// IncMCPSession adjusts the active-session gauge by delta (+1 on connect,
// -1 on close).
func IncMCPSession(delta int) {
	MCPActiveSessions.Add(float64(delta))
}

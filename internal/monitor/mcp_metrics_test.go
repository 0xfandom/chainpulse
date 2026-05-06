package monitor

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestObserveMCPToolCall(t *testing.T) {
	MCPToolCallsTotal.Reset()
	MCPToolLatencySeconds.Reset()

	ObserveMCPToolCall("get_wallet_positions", MCPStatusOK, 25*time.Millisecond)
	ObserveMCPToolCall("get_wallet_positions", MCPStatusError, 5*time.Millisecond)

	if got := testutil.ToFloat64(MCPToolCallsTotal.WithLabelValues("get_wallet_positions", MCPStatusOK)); got != 1 {
		t.Errorf("ok counter = %v, want 1", got)
	}
	if got := testutil.ToFloat64(MCPToolCallsTotal.WithLabelValues("get_wallet_positions", MCPStatusError)); got != 1 {
		t.Errorf("error counter = %v, want 1", got)
	}
	if got := testutil.CollectAndCount(MCPToolLatencySeconds); got == 0 {
		t.Errorf("latency histogram has no observations")
	}
}

func TestIncMCPSession(t *testing.T) {
	MCPActiveSessions.Set(0)
	IncMCPSession(1)
	IncMCPSession(1)
	IncMCPSession(-1)
	if got := testutil.ToFloat64(MCPActiveSessions); got != 1 {
		t.Errorf("active sessions = %v, want 1", got)
	}
}

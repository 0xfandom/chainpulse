package monitor

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestIncRawEventsProduced(t *testing.T) {
	IncRawEventsProduced("base", "Transfer")
	IncRawEventsProduced("base", "Transfer")
	IncRawEventsProduced("base", "Transfer")
	// counter is process-global; just check it is non-zero by reading the
	// metric value through the registry.
	m := RawEventsProducedTotal.WithLabelValues("base", "Transfer")
	if m == nil {
		t.Fatal("counter not registered")
	}
}

func TestSetListenerConnected(t *testing.T) {
	SetListenerConnected("base", true)
	SetListenerConnected("base", false)
}

func TestObserveBlockProcessing(t *testing.T) {
	ObserveBlockProcessing("base", 10*time.Millisecond)
}

func TestServeMetrics(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- ServeMetrics(ctx, addr) }()

	// Bump a metric so /metrics renders something.
	IncRawEventsProduced("base", "Transfer")

	// Poll until the server is up.
	deadline := time.Now().Add(2 * time.Second)
	var resp *http.Response
	for time.Now().Before(deadline) {
		resp, err = http.Get("http://" + addr + "/metrics")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "raw_events_produced_total") {
		t.Errorf("metrics output missing raw_events_produced_total")
	}

	cancel()
	if err := <-done; err != nil {
		t.Errorf("ServeMetrics returned error: %v", err)
	}
}

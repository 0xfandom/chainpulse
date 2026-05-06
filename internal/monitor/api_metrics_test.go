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

func TestAPIMetrics_Helpers(t *testing.T) {
	ObserveAPIRequest("GET", "/v1/wallet/:address/positions", "200", 12*time.Millisecond)
	ObserveAPIRequest("GET", "/v1/wallet/:address/positions", "500", 1*time.Second)
	IncAPICacheHit(APICacheSourceRedis)
	IncAPICacheHit(APICacheSourceClickHouse)
	IncAPIRateLimited()
	IncAPIWSConnection(1)
	IncAPIWSConnection(-1)
}

func TestAPIMetrics_RegisteredOnDefaultHandler(t *testing.T) {
	ObserveAPIRequest("GET", "/health", "200", 0)
	IncAPICacheHit(APICacheSourceRedis)
	IncAPIRateLimited()

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
	body, _ := io.ReadAll(resp.Body)
	for _, want := range []string{
		"api_request_duration_seconds",
		"api_requests_total",
		"api_websocket_connections_active",
		"api_cache_hits_total",
		"api_rate_limited_total",
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("/metrics missing %q", want)
		}
	}
	cancel()
	<-done
}

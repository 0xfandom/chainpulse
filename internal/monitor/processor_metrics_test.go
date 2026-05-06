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

func TestProcessorMetrics_Helpers(t *testing.T) {
	IncDecodedEventsProduced("base", "uniswap_v3", "swap")
	IncPositionUpdatesProduced("base", "aave_v3")
	IncProcessorError(ProcErrDecode)
	IncProcessorError(ProcErrAggregate)
	IncProcessorError(ProcErrClickHouseWrite)
	IncProcessorError(ProcErrRedisWrite)
	IncProcessorError(ProcErrKafkaPublish)
	ObserveClickHouseFlush(20 * time.Millisecond)
	ObserveRedisWrite(2 * time.Millisecond)
	SetKafkaConsumerLag("raw_events", "0", 42)
}

func TestProcessorMetrics_RegisteredOnDefaultHandler(t *testing.T) {
	// Ensure the metrics show up via ServeMetrics (same default registry).
	IncDecodedEventsProduced("base", "erc20", "transfer")
	IncPositionUpdatesProduced("base", "erc20")
	SetKafkaConsumerLag("raw_events", "0", 0)

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
		"decoded_events_produced_total",
		"position_updates_produced_total",
		"processor_consume_errors_total",
		"clickhouse_batch_flush_seconds",
		"redis_write_seconds",
		"processor_kafka_lag_messages",
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("/metrics missing %q", want)
		}
	}
	cancel()
	<-done
}

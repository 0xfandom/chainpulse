package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSubstitute_Replaces(t *testing.T) {
	t.Setenv("FOO", "bar")
	t.Setenv("MIXED_NAME_2", "value42")

	out, err := Substitute("a=${FOO} b=${MIXED_NAME_2}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "a=bar b=value42"
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestSubstitute_MissingVar(t *testing.T) {
	os.Unsetenv("DEFINITELY_UNSET_VAR_XYZ")
	_, err := Substitute("rpc=${DEFINITELY_UNSET_VAR_XYZ}")
	if err == nil {
		t.Fatal("expected error for missing env var, got nil")
	}
	if !errors.Is(err, ErrMissingEnvVar) {
		t.Errorf("err = %v, want ErrMissingEnvVar", err)
	}
	if !strings.Contains(err.Error(), "DEFINITELY_UNSET_VAR_XYZ") {
		t.Errorf("error should name the missing var, got %v", err)
	}
}

func TestSubstitute_NoPlaceholders(t *testing.T) {
	in := "no placeholders here"
	out, err := Substitute(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != in {
		t.Errorf("got %q, want %q", out, in)
	}
}

const validConfigBody = `
[app]
log_level = "info"
metrics_addr = ":9100"

[kafka]
brokers = ["localhost:9092"]
topic_raw_events = "raw_events"
client_id = "chainpulse"

[clickhouse]
dsn = "clickhouse://default:@localhost:9000/chainpulse"

[redis]
addr = "localhost:6379"

[[chains]]
chain_id = 8453
name = "base"
rpc_wss = "${TEST_WSS}"
rpc_http = "${TEST_HTTP}"
`

func TestLoad_ValidConfig(t *testing.T) {
	t.Setenv("TEST_WSS", "wss://example/ws")
	t.Setenv("TEST_HTTP", "https://example/http")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(validConfigBody), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.App.LogLevel != "info" {
		t.Errorf("log_level = %q", cfg.App.LogLevel)
	}
	if len(cfg.Chains) != 1 || cfg.Chains[0].RPCWSS != "wss://example/ws" {
		t.Errorf("chain config not substituted: %+v", cfg.Chains)
	}
}

func TestLoad_AppliesDefaults(t *testing.T) {
	t.Setenv("TEST_WSS", "wss://example/ws")
	t.Setenv("TEST_HTTP", "https://example/http")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(validConfigBody), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ClickHouse.Database != "chainpulse" {
		t.Errorf("ClickHouse.Database = %q, want chainpulse", cfg.ClickHouse.Database)
	}
	if cfg.ClickHouse.BatchSize != 1000 {
		t.Errorf("BatchSize = %d, want 1000", cfg.ClickHouse.BatchSize)
	}
	if got := cfg.ClickHouse.BatchInterval.AsDuration(); got != time.Second {
		t.Errorf("BatchInterval = %v, want 1s", got)
	}
	if got := cfg.Redis.DefaultTTL.AsDuration(); got != 60*time.Second {
		t.Errorf("Redis.DefaultTTL = %v, want 60s", got)
	}
	if cfg.Processor.ConsumerGroup != "chainpulse-processor" {
		t.Errorf("ConsumerGroup = %q", cfg.Processor.ConsumerGroup)
	}
	if cfg.Processor.MaxInFlight != 256 {
		t.Errorf("MaxInFlight = %d, want 256", cfg.Processor.MaxInFlight)
	}
	if cfg.API.Addr != ":8080" {
		t.Errorf("API.Addr default = %q", cfg.API.Addr)
	}
	if cfg.API.GRPCAddr != ":8081" {
		t.Errorf("API.GRPCAddr default = %q", cfg.API.GRPCAddr)
	}
	if got := cfg.API.RequestTimeout.AsDuration(); got != 10*time.Second {
		t.Errorf("API.RequestTimeout = %v", got)
	}
	if cfg.API.RateLimit.PerIPPerMinute != 100 {
		t.Errorf("rate_limit.per_ip_per_minute = %d", cfg.API.RateLimit.PerIPPerMinute)
	}
	if cfg.API.RateLimit.Burst != 20 {
		t.Errorf("rate_limit.burst = %d", cfg.API.RateLimit.Burst)
	}
	if cfg.API.WebSocket.OriginCheck != "strict" {
		t.Errorf("origin_check default = %q", cfg.API.WebSocket.OriginCheck)
	}
	if cfg.MCP.Addr != ":3001" {
		t.Errorf("MCP.Addr default = %q", cfg.MCP.Addr)
	}
	if cfg.MCP.Transport != "sse" {
		t.Errorf("MCP.Transport default = %q", cfg.MCP.Transport)
	}
	if got := cfg.MCP.CacheTTL.AsDuration(); got != 60*time.Second {
		t.Errorf("MCP.CacheTTL default = %v", got)
	}
}

func TestLoad_MCPRejectsUnknownTransport(t *testing.T) {
	t.Setenv("TEST_WSS", "wss://example/ws")
	t.Setenv("TEST_HTTP", "https://example/http")
	body := validConfigBody + `
[mcp]
transport = "websocket"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid mcp.transport")
	}
	if !strings.Contains(err.Error(), "mcp.transport") {
		t.Errorf("error should mention mcp.transport, got %v", err)
	}
}

func TestLoad_MCPOverrides(t *testing.T) {
	t.Setenv("TEST_WSS", "wss://example/ws")
	t.Setenv("TEST_HTTP", "https://example/http")
	body := validConfigBody + `
[mcp]
addr = ":4000"
transport = "stdio"
cache_ttl = "10s"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MCP.Addr != ":4000" {
		t.Errorf("MCP.Addr override = %q", cfg.MCP.Addr)
	}
	if cfg.MCP.Transport != "stdio" {
		t.Errorf("MCP.Transport override = %q", cfg.MCP.Transport)
	}
	if got := cfg.MCP.CacheTTL.AsDuration(); got != 10*time.Second {
		t.Errorf("MCP.CacheTTL override = %v", got)
	}
}

func TestLoad_APIInvalidOriginCheck(t *testing.T) {
	t.Setenv("TEST_WSS", "wss://example/ws")
	t.Setenv("TEST_HTTP", "https://example/http")
	body := validConfigBody + `
[api.websocket]
origin_check = "wide-open"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid origin_check")
	}
}

func TestLoad_OverridesDefaults(t *testing.T) {
	t.Setenv("TEST_WSS", "wss://example/ws")
	t.Setenv("TEST_HTTP", "https://example/http")

	body := `
[app]
log_level = "info"

[kafka]
brokers = ["localhost:9092"]
topic_raw_events = "raw_events"

[clickhouse]
dsn = "clickhouse://override:@host:9000/x"
batch_size = 50
batch_interval = "250ms"

[redis]
addr = "localhost:6379"

[[chains]]
chain_id = 8453
name = "base"
rpc_wss = "${TEST_WSS}"
rpc_http = "${TEST_HTTP}"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ClickHouse.BatchSize != 50 {
		t.Errorf("BatchSize override failed: %d", cfg.ClickHouse.BatchSize)
	}
	if got := cfg.ClickHouse.BatchInterval.AsDuration(); got != 250*time.Millisecond {
		t.Errorf("BatchInterval override failed: %v", got)
	}
}

func TestLoad_MissingChainRPC(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	body := `
[kafka]
brokers = ["localhost:9092"]
topic_raw_events = "raw_events"

[clickhouse]
dsn = "clickhouse://default:@host:9000/x"

[redis]
addr = "localhost:6379"
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected validation error for missing chains, got nil")
	}
}

func TestLoad_MissingClickHouseDSN(t *testing.T) {
	t.Setenv("TEST_WSS", "wss://example/ws")
	t.Setenv("TEST_HTTP", "https://example/http")

	body := `
[kafka]
brokers = ["localhost:9092"]
topic_raw_events = "raw_events"

[redis]
addr = "localhost:6379"

[[chains]]
chain_id = 8453
name = "base"
rpc_wss = "${TEST_WSS}"
rpc_http = "${TEST_HTTP}"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing clickhouse.dsn")
	}
	if !strings.Contains(err.Error(), "clickhouse.dsn") {
		t.Errorf("error should mention clickhouse.dsn, got %v", err)
	}
}

func TestLoad_MissingRedisAddr(t *testing.T) {
	t.Setenv("TEST_WSS", "wss://example/ws")
	t.Setenv("TEST_HTTP", "https://example/http")

	body := `
[kafka]
brokers = ["localhost:9092"]
topic_raw_events = "raw_events"

[clickhouse]
dsn = "clickhouse://default:@host:9000/x"

[[chains]]
chain_id = 8453
name = "base"
rpc_wss = "${TEST_WSS}"
rpc_http = "${TEST_HTTP}"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing redis.addr")
	}
	if !strings.Contains(err.Error(), "redis.addr") {
		t.Errorf("error should mention redis.addr, got %v", err)
	}
}

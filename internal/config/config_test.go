package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestLoad_ValidConfig(t *testing.T) {
	t.Setenv("TEST_WSS", "wss://example/ws")
	t.Setenv("TEST_HTTP", "https://example/http")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	body := `
[app]
log_level = "info"
metrics_addr = ":9100"

[kafka]
brokers = ["localhost:9092"]
topic_raw_events = "raw_events"
client_id = "chainpulse"

[[chains]]
chain_id = 8453
name = "base"
rpc_wss = "${TEST_WSS}"
rpc_http = "${TEST_HTTP}"
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
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

func TestLoad_MissingRequiredField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	body := `
[kafka]
brokers = ["localhost:9092"]
topic_raw_events = "raw_events"
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

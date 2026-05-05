package main

import (
	"strings"
	"testing"
)

func TestRun_MissingConfigFails(t *testing.T) {
	err := run("/path/that/does/not/exist.toml")
	if err == nil {
		t.Fatal("expected error for missing config")
	}
	if !strings.Contains(err.Error(), "load config") {
		t.Errorf("err should mention config load, got %v", err)
	}
}

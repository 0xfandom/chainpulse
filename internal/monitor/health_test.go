package monitor

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestServeHealth_ReturnsOK(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- ServeHealth(ctx, addr) }()

	deadline := time.Now().Add(2 * time.Second)
	var resp *http.Response
	for time.Now().Before(deadline) {
		resp, err = http.Get("http://" + addr + "/health")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("get /health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode: %v body=%s", err, body)
	}
	if payload["status"] != "ok" {
		t.Errorf("status field = %q", payload["status"])
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("ServeHealth returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ServeHealth did not exit after cancel")
	}
}

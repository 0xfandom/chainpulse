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

func startHealthServer(t *testing.T) (addr string, cancel context.CancelFunc, done <-chan error) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr = lis.Addr().String()
	_ = lis.Close()

	ctx, cancelFn := context.WithCancel(context.Background())
	doneCh := make(chan error, 1)
	go func() { doneCh <- ServeHealth(ctx, addr) }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + "/health")
		if err == nil {
			resp.Body.Close()
			return addr, cancelFn, doneCh
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancelFn()
	t.Fatalf("server did not become reachable on %s", addr)
	return "", nil, nil
}

func TestServeHealth_LivenessReturnsOK(t *testing.T) {
	resetReadinessForTest()
	addr, cancel, done := startHealthServer(t)
	defer cancel()

	resp, err := http.Get("http://" + addr + "/health")
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

func TestServeHealth_ReadyDisabledReturns200(t *testing.T) {
	resetReadinessForTest() // readiness off by default
	addr, cancel, _ := startHealthServer(t)
	defer cancel()

	resp, err := http.Get("http://" + addr + "/ready")
	if err != nil {
		t.Fatalf("get /ready: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 (readiness disabled mirrors liveness)", resp.StatusCode)
	}
	var snap ReadinessSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if snap.Enabled || snap.Status != "ok" {
		t.Errorf("snap = %+v, want disabled+ok", snap)
	}
}

func TestServeHealth_ReadyEnabledNoHeadReturns503(t *testing.T) {
	resetReadinessForTest()
	EnableReadiness(time.Second)
	RegisterChain("ethereum")

	addr, cancel, _ := startHealthServer(t)
	defer cancel()

	resp, err := http.Get("http://" + addr + "/ready")
	if err != nil {
		t.Fatalf("get /ready: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
}

func TestServeHealth_ReadyAfterHeadReturns200(t *testing.T) {
	resetReadinessForTest()
	EnableReadiness(10 * time.Second)
	RegisterChain("ethereum")
	RecordHead("ethereum")

	addr, cancel, _ := startHealthServer(t)
	defer cancel()

	resp, err := http.Get("http://" + addr + "/ready")
	if err != nil {
		t.Fatalf("get /ready: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

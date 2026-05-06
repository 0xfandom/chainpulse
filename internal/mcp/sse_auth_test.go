package mcp

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func startSSEWithToken(t *testing.T, token string) (baseURL string, stop func()) {
	t.Helper()
	srv := newTestServer(t)
	tr := NewSSETransport(srv, "127.0.0.1:0")
	tr.BearerToken = token
	ctx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan error, 1)
	go func() { doneCh <- tr.Run(ctx) }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !tr.started.Load() {
		time.Sleep(10 * time.Millisecond)
	}
	if !tr.started.Load() {
		cancel()
		t.Fatal("SSETransport failed to start")
	}
	return "http://" + tr.Addr, func() {
		cancel()
		select {
		case <-doneCh:
		case <-time.After(2 * time.Second):
		}
	}
}

func TestSSEAuth_RejectsMissingHeader(t *testing.T) {
	base, stop := startSSEWithToken(t, "secret-123")
	defer stop()
	resp, err := http.Get(base + "/sse")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestSSEAuth_RejectsWrongToken(t *testing.T) {
	base, stop := startSSEWithToken(t, "secret-123")
	defer stop()
	req, _ := http.NewRequest(http.MethodGet, base+"/sse", nil)
	req.Header.Set("Authorization", "Bearer not-the-right-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestSSEAuth_RejectsWrongScheme(t *testing.T) {
	base, stop := startSSEWithToken(t, "secret-123")
	defer stop()
	req, _ := http.NewRequest(http.MethodGet, base+"/sse", nil)
	req.Header.Set("Authorization", "Basic c2VjcmV0LTEyMw==")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestSSEAuth_AllowsCorrectToken(t *testing.T) {
	base, stop := startSSEWithToken(t, "secret-123")
	defer stop()
	req, _ := http.NewRequest(http.MethodGet, base+"/sse", nil)
	req.Header.Set("Authorization", "Bearer secret-123")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("content-type = %q", ct)
	}
}

func TestSSEAuth_PostMessagesEnforcesAuth(t *testing.T) {
	base, stop := startSSEWithToken(t, "secret-123")
	defer stop()
	req, _ := http.NewRequest(http.MethodPost, base+"/messages?session=x",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestSSEAuth_EmptyTokenDisablesAuth(t *testing.T) {
	base, stop := startSSEWithToken(t, "")
	defer stop()
	resp, err := http.Get(base + "/sse")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("empty token must allow requests; got %d", resp.StatusCode)
	}
}

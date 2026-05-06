package mcp

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// startSSE boots an SSETransport on a random port and returns the bound
// base URL plus a stop function.
func startSSE(t *testing.T) (baseURL string, stop func()) {
	t.Helper()
	srv := newTestServer(t)
	tr := NewSSETransport(srv, "127.0.0.1:0")
	ctx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan error, 1)
	go func() { doneCh <- tr.Run(ctx) }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if tr.started.Load() {
			break
		}
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
			t.Error("SSETransport did not stop in time")
		}
	}
}

// openSSEStream connects to /sse and returns the session id parsed from
// the first event plus the line reader (for additional events).
func openSSEStream(t *testing.T, base string) (sessionID string, body io.ReadCloser, reader *bufio.Reader) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, base+"/sse", nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/sse status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("content-type = %q", ct)
	}
	r := bufio.NewReader(resp.Body)
	// Expect: event: endpoint \n data: /messages?session=<id> \n\n
	var dataLine string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read sse line: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "data: ") {
			dataLine = strings.TrimPrefix(line, "data: ")
		}
	}
	if !strings.Contains(dataLine, "session=") {
		t.Fatalf("first event missing session: %q", dataLine)
	}
	sessionID = dataLine[strings.Index(dataLine, "session=")+len("session="):]
	return sessionID, resp.Body, r
}

// readNextDataFrame reads SSE events until it finds a `data:` line for a
// `message` event, returning the JSON payload.
func readNextDataFrame(t *testing.T, r *bufio.Reader) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("sse read: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(line, "data: ") {
			payload := strings.TrimPrefix(line, "data: ")
			// Skip the keepalive comment lines (start with `:`).
			if strings.HasPrefix(payload, ":") {
				continue
			}
			return payload
		}
	}
	t.Fatal("timed out waiting for sse data frame")
	return ""
}

func TestSSE_OpenStreamAndPostRoundtrip(t *testing.T) {
	base, stop := startSSE(t)
	defer stop()

	sessID, body, reader := openSSEStream(t, base)
	defer body.Close()

	post := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	resp, err := http.Post(base+"/messages?session="+sessID, "application/json", post)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("POST status = %d, want 202", resp.StatusCode)
	}

	frame := readNextDataFrame(t, reader)
	if !strings.Contains(frame, `"echo"`) {
		t.Errorf("frame missing tool name: %s", frame)
	}
}

func TestSSE_PostRejectsUnknownSession(t *testing.T) {
	base, stop := startSSE(t)
	defer stop()
	resp, err := http.Post(base+"/messages?session=does-not-exist", "application/json",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestSSE_PostRejectsMissingSession(t *testing.T) {
	base, stop := startSSE(t)
	defer stop()
	resp, err := http.Post(base+"/messages", "application/json",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestSSE_GetOnlyOnSSEPath(t *testing.T) {
	base, stop := startSSE(t)
	defer stop()
	resp, err := http.Post(base+"/sse", "application/json", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", resp.StatusCode)
	}
}

func TestSSE_NotificationReturns202NoFrame(t *testing.T) {
	base, stop := startSSE(t)
	defer stop()
	sessID, body, _ := openSSEStream(t, base)
	defer body.Close()

	resp, err := http.Post(base+"/messages?session="+sessID, "application/json",
		strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status = %d, want 202", resp.StatusCode)
	}
}

func TestSSE_GracefulShutdown(t *testing.T) {
	srv := newTestServer(t)
	tr := NewSSETransport(srv, "127.0.0.1:0")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- tr.Run(ctx) }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !tr.started.Load() {
		time.Sleep(10 * time.Millisecond)
	}
	if !tr.started.Load() {
		cancel()
		t.Fatal("did not start")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

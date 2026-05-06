package mcp

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	chainpulselog "github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/monitor"
)

const (
	sseHeartbeatInterval = 15 * time.Second
	sseSendTimeout       = 5 * time.Second
	postMaxBytes         = 1 << 20 // 1 MiB per JSON-RPC frame
	sseChannelBuffer     = 32
)

// SSETransport implements the MCP HTTP+SSE transport. Clients GET /sse to
// open the event stream and POST /messages?session=<id> with JSON-RPC
// frames; responses come back over the SSE stream.
//
// When BearerToken is non-empty, both routes require an Authorization
// header of the form `Bearer <token>` and reject mismatches with 401.
// Empty BearerToken disables auth and logs a warning at startup.
type SSETransport struct {
	Server      *Server
	Addr        string
	BearerToken string

	sessions *sessionMap
	srv      *http.Server
	listener net.Listener
	started  atomic.Bool
}

// NewSSETransport constructs an SSETransport bound to addr.
func NewSSETransport(srv *Server, addr string) *SSETransport {
	return &SSETransport{Server: srv, Addr: addr, sessions: newSessionMap()}
}

// Run boots the HTTP listener and serves until ctx is cancelled or the
// server returns an error. Listener address can be inspected via Addr
// after Run is called when a port-zero (":0") config is used.
func (t *SSETransport) Run(ctx context.Context) error {
	if t.Server == nil {
		return errors.New("sse: nil server")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/sse", t.withAuth(t.handleSSE))
	mux.HandleFunc("/messages", t.withAuth(t.handleMessages))

	lis, err := net.Listen("tcp", t.Addr)
	if err != nil {
		return fmt.Errorf("sse listen %s: %w", t.Addr, err)
	}
	t.listener = lis
	t.Addr = lis.Addr().String()
	t.srv = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	t.started.Store(true)

	l := chainpulselog.Component("mcp_sse")
	if t.BearerToken == "" {
		l.Warn().Msg("bearer token not configured; SSE transport runs unauthenticated")
	}
	l.Info().Str("addr", t.Addr).Bool("auth", t.BearerToken != "").Msg("listening")

	errCh := make(chan error, 1)
	go func() { errCh <- t.srv.Serve(lis) }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = t.srv.Shutdown(shutdownCtx)
		<-errCh
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// Listener returns the underlying listener (test helper).
func (t *SSETransport) Listener() net.Listener { return t.listener }

// ActiveSessions reports the current subscriber count.
func (t *SSETransport) ActiveSessions() int { return t.sessions.len() }

// withAuth wraps an HTTP handler with bearer-token enforcement when one
// is configured. Constant-time comparison avoids leaking timing on
// brute-force attempts.
func (t *SSETransport) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if t.BearerToken == "" {
			next(w, r)
			return
		}
		const prefix = "Bearer "
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, prefix) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		got := []byte(header[len(prefix):])
		want := []byte(t.BearerToken)
		if subtle.ConstantTimeCompare(got, want) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// handleSSE upgrades the connection to a Server-Sent Events stream,
// emits the session endpoint as the first event, then forwards every
// frame written to the session channel.
func (t *SSETransport) handleSSE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	sess := &session{
		id:     uuid.NewString(),
		out:    make(chan []byte, sseChannelBuffer),
		closed: make(chan struct{}),
	}
	t.sessions.add(sess)
	monitor.IncMCPSession(1)
	defer func() {
		sess.closeOnce()
		t.sessions.remove(sess.id)
		monitor.IncMCPSession(-1)
	}()

	endpoint := "/messages?session=" + sess.id
	if _, err := fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpoint); err != nil {
		return
	}
	flusher.Flush()

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-sess.closed:
			return
		case <-heartbeat.C:
			if _, err := io.WriteString(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case frame, ok := <-sess.out:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(w, "event: message\ndata: %s\n\n", frame); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// handleMessages accepts a POSTed JSON-RPC frame, dispatches it via the
// shared Server, and pushes the response onto the addressed session's
// stream. Returns 202 Accepted; the actual response travels over SSE.
func (t *SSETransport) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("session")
	if id == "" {
		http.Error(w, "session query param required", http.StatusBadRequest)
		return
	}
	sess, ok := t.sessions.get(id)
	if !ok {
		http.Error(w, "unknown session", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, postMaxBytes))
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	resp := t.Server.Handle(r.Context(), body)
	if resp == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	select {
	case sess.out <- resp:
		w.WriteHeader(http.StatusAccepted)
	case <-time.After(sseSendTimeout):
		http.Error(w, "session backpressured", http.StatusServiceUnavailable)
	case <-sess.closed:
		http.Error(w, "session closed", http.StatusGone)
	}
}

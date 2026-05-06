package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"

	"github.com/0xfandom/chainpulse/internal/types"
)

// fakeFetcher returns a fixed list of messages then blocks until ctx
// cancels.
type fakeFetcher struct {
	messages []kafka.Message
	idx      int32
	closed   atomic.Bool
}

func (f *fakeFetcher) FetchMessage(ctx context.Context) (kafka.Message, error) {
	for {
		i := atomic.AddInt32(&f.idx, 1) - 1
		if int(i) < len(f.messages) {
			return f.messages[i], nil
		}
		select {
		case <-ctx.Done():
			return kafka.Message{}, ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func (f *fakeFetcher) Close() error { f.closed.Store(true); return nil }

func newWSServer(t *testing.T, h *WSHandler) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/v1")
	h.Register(g)
	return httptest.NewServer(r)
}

func dial(t *testing.T, srv *httptest.Server, path string) *websocket.Conn {
	t.Helper()
	u, _ := url.Parse(srv.URL + path)
	u.Scheme = "ws"
	dialer := websocket.DefaultDialer
	header := http.Header{}
	header.Set("Origin", "http://example.com")
	c, _, err := dialer.Dial(u.String(), header)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return c
}

func TestWS_StreamsMessages(t *testing.T) {
	cfg := types.APIWSConfig{
		ReadBufferBytes:  1024,
		WriteBufferBytes: 1024,
		OriginCheck:      "permissive",
	}
	cors := types.APICORSConfig{AllowedOrigins: []string{"*"}}
	h := NewWSHandler(cfg, cors, []string{"localhost:9092"}, "decoded_events")
	fetcher := &fakeFetcher{messages: []kafka.Message{
		{Value: []byte(`{"chain_id":8453,"protocol":"erc20","event_type":"transfer"}`)},
		{Value: []byte(`{"chain_id":1,"protocol":"aave_v3","event_type":"borrow"}`)},
	}}
	h.WithFactory(func(ctx context.Context, groupID string) (WSMessageFetcher, error) {
		return fetcher, nil
	})

	srv := newWSServer(t, h)
	defer srv.Close()

	c := dial(t, srv, "/v1/events/stream")
	defer c.Close()

	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read 1: %v", err)
	}
	if !strings.Contains(string(msg), "8453") {
		t.Errorf("first message wrong: %s", msg)
	}
	_, msg, err = c.ReadMessage()
	if err != nil {
		t.Fatalf("read 2: %v", err)
	}
	if !strings.Contains(string(msg), "aave_v3") {
		t.Errorf("second message wrong: %s", msg)
	}
}

func TestWS_FilterChainID(t *testing.T) {
	cfg := types.APIWSConfig{ReadBufferBytes: 1024, WriteBufferBytes: 1024, OriginCheck: "permissive"}
	cors := types.APICORSConfig{AllowedOrigins: []string{"*"}}
	h := NewWSHandler(cfg, cors, []string{"x"}, "decoded_events")
	fetcher := &fakeFetcher{messages: []kafka.Message{
		{Value: []byte(`{"chain_id":1}`)},
		{Value: []byte(`{"chain_id":8453}`)},
	}}
	h.WithFactory(func(ctx context.Context, groupID string) (WSMessageFetcher, error) {
		return fetcher, nil
	})
	srv := newWSServer(t, h)
	defer srv.Close()

	c := dial(t, srv, "/v1/events/stream?chain_id=8453")
	defer c.Close()

	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(msg), "8453") {
		t.Errorf("expected chain 8453, got: %s", msg)
	}
}

func TestWS_OriginCheckStrictRejectsUnknown(t *testing.T) {
	cfg := types.APIWSConfig{ReadBufferBytes: 1024, WriteBufferBytes: 1024, OriginCheck: "strict"}
	cors := types.APICORSConfig{AllowedOrigins: []string{"https://allowed"}}
	h := NewWSHandler(cfg, cors, []string{"x"}, "decoded_events")
	srv := newWSServer(t, h)
	defer srv.Close()

	u, _ := url.Parse(srv.URL + "/v1/events/stream")
	u.Scheme = "ws"
	header := http.Header{}
	header.Set("Origin", "https://evil")
	_, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err == nil {
		t.Fatal("expected upgrade rejection")
	}
}

func TestWS_FactoryError(t *testing.T) {
	cfg := types.APIWSConfig{ReadBufferBytes: 1024, WriteBufferBytes: 1024, OriginCheck: "permissive"}
	cors := types.APICORSConfig{AllowedOrigins: []string{"*"}}
	h := NewWSHandler(cfg, cors, []string{"x"}, "decoded_events")
	h.WithFactory(func(ctx context.Context, groupID string) (WSMessageFetcher, error) {
		return nil, errors.New("kafka unreachable")
	})
	srv := newWSServer(t, h)
	defer srv.Close()

	c := dial(t, srv, "/v1/events/stream")
	defer c.Close()

	c.SetReadDeadline(time.Now().Add(time.Second))
	_, _, err := c.ReadMessage()
	if err == nil {
		t.Error("expected close after factory error")
	}
}

func TestPassesFilter(t *testing.T) {
	body := []byte(`{"chain_id":8453,"protocol":"erc20"}`)
	if !passesFilter(body, nil, nil) {
		t.Error("no filters should pass")
	}
	if !passesFilter(body, map[uint64]struct{}{8453: {}}, nil) {
		t.Error("matching chain should pass")
	}
	if passesFilter(body, map[uint64]struct{}{1: {}}, nil) {
		t.Error("non-matching chain should fail")
	}
	if !passesFilter(body, nil, map[string]struct{}{"erc20": {}}) {
		t.Error("matching protocol should pass")
	}
	if passesFilter(body, nil, map[string]struct{}{"aave_v3": {}}) {
		t.Error("non-matching protocol should fail")
	}
	if passesFilter([]byte(`not json`), map[uint64]struct{}{1: {}}, nil) {
		t.Error("undecodable payload should fail")
	}
}

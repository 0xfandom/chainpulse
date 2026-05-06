package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"

	chainpulselog "github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/monitor"
	"github.com/0xfandom/chainpulse/internal/types"
)

// WS timing constants — exposed as vars so tests can shorten them.
var (
	wsPingInterval = 30 * time.Second
	wsPongTimeout  = 60 * time.Second
	wsWriteTimeout = 5 * time.Second
)

// WSMessageFetcher is the subset of kafka-go's Reader the WS handler
// uses, factored as an interface so tests can inject a fake.
type WSMessageFetcher interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	Close() error
}

// WSFetcherFactory produces a fresh fetcher per connection. Each
// connection gets its own ephemeral consumer group so subscribers don't
// share offsets.
type WSFetcherFactory func(ctx context.Context, groupID string) (WSMessageFetcher, error)

// WSHandler exposes /events/stream and produces a fan-out of decoded
// events from Kafka over WebSocket.
type WSHandler struct {
	cfg      types.APIWSConfig
	cors     types.APICORSConfig
	brokers  []string
	topic    string
	factory  WSFetcherFactory
	upgrader websocket.Upgrader
}

// NewWSHandler constructs a WSHandler. brokers/topic come from
// cfg.Kafka.Brokers and cfg.Kafka.TopicDecodedEvents at the call site.
func NewWSHandler(cfg types.APIWSConfig, cors types.APICORSConfig, brokers []string, topic string) *WSHandler {
	h := &WSHandler{
		cfg:     cfg,
		cors:    cors,
		brokers: brokers,
		topic:   topic,
	}
	h.factory = h.defaultFactory
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  cfg.ReadBufferBytes,
		WriteBufferSize: cfg.WriteBufferBytes,
		CheckOrigin:     h.checkOrigin,
	}
	return h
}

// WithFactory overrides the kafka fetcher factory for tests.
func (h *WSHandler) WithFactory(f WSFetcherFactory) *WSHandler {
	h.factory = f
	return h
}

// Register attaches the route onto group.
func (h *WSHandler) Register(group *gin.RouterGroup) {
	group.GET("/events/stream", h.Stream)
}

// Stream upgrades the request to WebSocket, opens a per-connection
// kafka reader, and fans decoded events to the client until the
// connection drops or the server context cancels.
func (h *WSHandler) Stream(c *gin.Context) {
	chainFilters := parseUintList(c.QueryArray("chain_id"))
	protocolFilters := stringSet(c.QueryArray("protocol"))

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrader writes its own error response; nothing else to do.
		return
	}

	groupID := "chainpulse-ws-" + uuid.NewString()
	fetcher, err := h.factory(c.Request.Context(), groupID)
	if err != nil {
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "kafka reader failed"))
		_ = conn.Close()
		return
	}

	monitor.IncAPIWSConnection(1)
	defer monitor.IncAPIWSConnection(-1)

	logger := chainpulselog.Component("api_websocket")
	logger.Info().Str("group", groupID).Msg("client connected")

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	defer fetcher.Close()
	defer conn.Close()

	// Heartbeat: read pump owns the read deadline + pong handler.
	conn.SetReadDeadline(time.Now().Add(wsPongTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsPongTimeout))
		return nil
	})

	var wg sync.WaitGroup
	wg.Add(2)
	go h.pingLoop(ctx, cancel, conn, &wg)
	go h.readPump(ctx, cancel, conn, &wg)

	h.writePump(ctx, conn, fetcher, chainFilters, protocolFilters)
	wg.Wait()
	logger.Info().Str("group", groupID).Msg("client disconnected")
}

// pingLoop sends periodic pings; the pong handler resets the read
// deadline. Cancels ctx on write failure.
func (h *WSHandler) pingLoop(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				cancel()
				return
			}
		}
	}
}

// readPump drains client frames so pongs are processed; bails on any
// read error, which signals closed connection or pong timeout.
func (h *WSHandler) readPump(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	defer cancel()
	for {
		if _, _, err := conn.NextReader(); err != nil {
			return
		}
		if ctx.Err() != nil {
			return
		}
	}
}

// writePump consumes Kafka messages, applies filters, writes to the
// client. Returns when ctx is cancelled or the fetcher / write errors.
func (h *WSHandler) writePump(
	ctx context.Context,
	conn *websocket.Conn,
	fetcher WSMessageFetcher,
	chainFilters map[uint64]struct{},
	protocolFilters map[string]struct{},
) {
	for {
		if ctx.Err() != nil {
			return
		}
		msg, err := fetcher.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			return
		}

		if !passesFilter(msg.Value, chainFilters, protocolFilters) {
			continue
		}

		_ = conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
		if err := conn.WriteMessage(websocket.TextMessage, msg.Value); err != nil {
			return
		}
	}
}

// defaultFactory builds a kafka-go reader bound to a unique consumer
// group, starting at LastOffset (live tail only).
func (h *WSHandler) defaultFactory(_ context.Context, groupID string) (WSMessageFetcher, error) {
	if len(h.brokers) == 0 || h.topic == "" {
		return nil, fmt.Errorf("ws factory: brokers / topic missing")
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        h.brokers,
		Topic:          h.topic,
		GroupID:        groupID,
		StartOffset:    kafka.LastOffset,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: 0,
	})
	return r, nil
}

// checkOrigin enforces the cors policy for WS upgrades.
func (h *WSHandler) checkOrigin(r *http.Request) bool {
	if h.cfg.OriginCheck != "strict" {
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	for _, allowed := range h.cors.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// passesFilter reports whether msg.Value matches the chain/protocol
// filters. An empty filter set matches everything.
func passesFilter(payload []byte, chainFilters map[uint64]struct{}, protocolFilters map[string]struct{}) bool {
	if len(chainFilters) == 0 && len(protocolFilters) == 0 {
		return true
	}
	var meta struct {
		ChainID  uint64 `json:"chain_id"`
		Protocol string `json:"protocol"`
	}
	if err := json.Unmarshal(payload, &meta); err != nil {
		return false
	}
	if len(chainFilters) > 0 {
		if _, ok := chainFilters[meta.ChainID]; !ok {
			return false
		}
	}
	if len(protocolFilters) > 0 {
		if _, ok := protocolFilters[meta.Protocol]; !ok {
			return false
		}
	}
	return true
}

// parseUintList converts repeatable query params to a uint64 set.
// Invalid entries are skipped silently — bad input becomes "no filter".
func parseUintList(values []string) map[uint64]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := map[uint64]struct{}{}
	for _, v := range values {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			out[id] = struct{}{}
		}
	}
	return out
}

// stringSet returns the set of values, dropping empties.
func stringSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := map[string]struct{}{}
	for _, v := range values {
		if v != "" {
			out[v] = struct{}{}
		}
	}
	return out
}

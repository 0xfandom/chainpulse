package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/0xfandom/chainpulse/internal/api/store"
	"github.com/0xfandom/chainpulse/internal/mcp"
)

// fakeRows is the no-row driver.Rows impl shared with Day 3 tests.
type fakeRows struct{}

func (f *fakeRows) Next() bool                       { return false }
func (f *fakeRows) Scan(...any) error                { return nil }
func (f *fakeRows) ScanStruct(any) error             { return nil }
func (f *fakeRows) ColumnTypes() []driver.ColumnType { return nil }
func (f *fakeRows) Totals(...any) error              { return nil }
func (f *fakeRows) Columns() []string                { return nil }
func (f *fakeRows) Close() error                     { return nil }
func (f *fakeRows) Err() error                       { return nil }
func (f *fakeRows) HasData() bool                    { return false }

// countingConn records every Query the tool dispatches so cache-hit
// tests can assert the second call did NOT touch the store.
type countingConn struct{ queries int }

func (c *countingConn) Query(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
	c.queries++
	return &fakeRows{}, nil
}
func (c *countingConn) Ping(context.Context) error { return nil }
func (c *countingConn) Close() error               { return nil }

func newDeps(t *testing.T) (*Deps, *countingConn, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := store.NewReadCache(rc, time.Minute)
	conn := &countingConn{}
	rs := store.NewReadStore(conn)
	return &Deps{Store: rs, Cache: cache, TTL: time.Minute}, conn, mr
}

func newServerWithTools(t *testing.T) (*mcp.Server, *countingConn, *miniredis.Miniredis) {
	t.Helper()
	deps, conn, mr := newDeps(t)
	reg := mcp.NewRegistry()
	RegisterWallet(reg, deps)
	RegisterToken(reg, deps)
	return mcp.NewServer(reg), conn, mr
}

func decode(t *testing.T, frame []byte) mcp.Response {
	t.Helper()
	var r mcp.Response
	if err := json.Unmarshal(frame, &r); err != nil {
		t.Fatalf("decode: %v\n%s", err, frame)
	}
	return r
}

func toolCall(name string, args any) []byte {
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": name, "arguments": args},
	})
	return body
}

func TestWalletPositions_BasicEmpty(t *testing.T) {
	s, conn, _ := newServerWithTools(t)
	out := s.Handle(context.Background(), toolCall("get_wallet_positions",
		map[string]any{"wallet": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}))
	resp := decode(t, out)
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	if conn.queries != 1 {
		t.Errorf("expected 1 ClickHouse query, got %d", conn.queries)
	}
}

func TestWalletPositions_RejectsBadAddress(t *testing.T) {
	s, _, _ := newServerWithTools(t)
	out := s.Handle(context.Background(), toolCall("get_wallet_positions",
		map[string]any{"wallet": "not-a-wallet"}))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != mcp.CodeInvalidParams {
		t.Errorf("expected InvalidParams, got %+v", resp.Error)
	}
}

func TestWalletPositions_RejectsUnknownProtocol(t *testing.T) {
	s, _, _ := newServerWithTools(t)
	out := s.Handle(context.Background(), toolCall("get_wallet_positions",
		map[string]any{"wallet": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "protocol": "morpho"}))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != mcp.CodeInvalidParams {
		t.Errorf("expected InvalidParams (schema enum), got %+v", resp.Error)
	}
}

func TestWalletPositions_CachesSecondCall(t *testing.T) {
	s, conn, _ := newServerWithTools(t)
	args := map[string]any{"wallet": "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	first := decode(t, s.Handle(context.Background(), toolCall("get_wallet_positions", args)))
	if first.Error != nil {
		t.Fatalf("first: %+v", first.Error)
	}
	second := decode(t, s.Handle(context.Background(), toolCall("get_wallet_positions", args)))
	if second.Error != nil {
		t.Fatalf("second: %+v", second.Error)
	}
	if conn.queries != 1 {
		t.Errorf("expected 1 query (cache hit on second), got %d", conn.queries)
	}
}

func TestWalletBalances_PerChain(t *testing.T) {
	s, _, mr := newServerWithTools(t)
	wallet := "0xcccccccccccccccccccccccccccccccccccccccc"
	mr.HSet("balance:"+wallet+":1:0xtokenA", "amount", "1000")
	_ = mr // miniredis no longer drives balances; ClickHouse MV does

	out := s.Handle(context.Background(), toolCall("get_wallet_balances",
		map[string]any{"wallet": wallet, "chain_id": 1}))
	payload := unwrapToolCallText(t, out)
	if !strings.Contains(payload, `"chain_id":1`) {
		t.Errorf("expected chain_id=1 in unwrapped payload; got %s", payload)
	}
	if strings.Contains(payload, `"by_chain"`) {
		t.Errorf("chain-scoped query should not include by_chain; got %s", payload)
	}
}

func TestWalletBalances_AcrossChainsEmptyShape(t *testing.T) {
	s, _, _ := newServerWithTools(t)
	wallet := "0xdddddddddddddddddddddddddddddddddddddddd"

	out := s.Handle(context.Background(), toolCall("get_wallet_balances",
		map[string]any{"wallet": wallet}))
	payload := unwrapToolCallText(t, out)
	if !strings.Contains(payload, `"wallet":"`+wallet+`"`) {
		t.Errorf("expected wallet field in payload; got %s", payload)
	}
	if strings.Contains(payload, `"chain_id":`) {
		t.Errorf("cross-chain query should not include chain_id; got %s", payload)
	}
}

// unwrapToolCallText returns the inner JSON string from a tools/call
// response: result.content[0].text. Test helper.
func unwrapToolCallText(t *testing.T, frame []byte) string {
	t.Helper()
	var resp mcp.Response
	if err := json.Unmarshal(frame, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res, _ := json.Marshal(resp.Result)
	var wrap struct {
		Content []struct{ Text string } `json:"content"`
	}
	if err := json.Unmarshal(res, &wrap); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if len(wrap.Content) == 0 {
		t.Fatalf("no content array in result: %s", res)
	}
	return wrap.Content[0].Text
}

func TestWalletHistory_LimitClampedTo500(t *testing.T) {
	s, _, _ := newServerWithTools(t)
	out := s.Handle(context.Background(), toolCall("get_wallet_history",
		map[string]any{"wallet": "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "limit": 9999}))
	resp := decode(t, out)
	// schema caps at 500 -> validation error
	if resp.Error == nil || resp.Error.Code != mcp.CodeInvalidParams {
		t.Errorf("expected InvalidParams (limit > 500), got %+v", resp.Error)
	}
}

func TestWalletHistory_DefaultLimitWorks(t *testing.T) {
	s, conn, _ := newServerWithTools(t)
	out := s.Handle(context.Background(), toolCall("get_wallet_history",
		map[string]any{"wallet": "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}))
	resp := decode(t, out)
	if resp.Error != nil {
		t.Fatalf("err: %+v", resp.Error)
	}
	if conn.queries != 1 {
		t.Errorf("expected 1 query, got %d", conn.queries)
	}
}

func TestTokenTransfers_BasicEmpty(t *testing.T) {
	s, conn, _ := newServerWithTools(t)
	out := s.Handle(context.Background(), toolCall("get_token_transfers",
		map[string]any{"address": "0xffffffffffffffffffffffffffffffffffffffff"}))
	resp := decode(t, out)
	if resp.Error != nil {
		t.Fatalf("err: %+v", resp.Error)
	}
	if conn.queries != 1 {
		t.Errorf("expected 1 query, got %d", conn.queries)
	}
}

func TestTokenTransfers_RejectsBadAddress(t *testing.T) {
	s, _, _ := newServerWithTools(t)
	out := s.Handle(context.Background(), toolCall("get_token_transfers",
		map[string]any{"address": "abc"}))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != mcp.CodeInvalidParams {
		t.Errorf("expected InvalidParams, got %+v", resp.Error)
	}
}

func TestTokenTransfers_LimitClamped(t *testing.T) {
	s, _, _ := newServerWithTools(t)
	out := s.Handle(context.Background(), toolCall("get_token_transfers",
		map[string]any{"address": "0xffffffffffffffffffffffffffffffffffffffff", "limit": 50000}))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != mcp.CodeInvalidParams {
		t.Errorf("expected schema-level limit rejection, got %+v", resp.Error)
	}
}

func TestTokenTransfers_BypassesCache(t *testing.T) {
	// get_token_transfers is a "latest"-class tool: agents asking for
	// the freshest transfers must hit ClickHouse every time, not stale
	// Redis cache. Two calls in a row must produce two queries.
	s, conn, _ := newServerWithTools(t)
	args := map[string]any{"address": "0x1111111111111111111111111111111111111111"}
	_ = decode(t, s.Handle(context.Background(), toolCall("get_token_transfers", args)))
	_ = decode(t, s.Handle(context.Background(), toolCall("get_token_transfers", args)))
	if conn.queries != 2 {
		t.Errorf("expected cache bypass, got queries=%d", conn.queries)
	}
}

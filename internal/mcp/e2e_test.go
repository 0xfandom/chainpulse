package mcp_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/0xfandom/chainpulse/internal/api/store"
	"github.com/0xfandom/chainpulse/internal/mcp"
	"github.com/0xfandom/chainpulse/internal/mcp/tools"
)

// fakeRows is the minimal driver.Rows impl needed for empty-result tests.
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

type fakeConn struct{}

func (fakeConn) Query(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
	return &fakeRows{}, nil
}
func (fakeConn) Ping(context.Context) error { return nil }
func (fakeConn) Close() error               { return nil }

// buildFullServer constructs an MCP server with all six PRD-mandated
// tools registered.
func buildFullServer(t *testing.T) *mcp.Server {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	deps := &tools.Deps{
		Store: store.NewReadStore(fakeConn{}),
		Cache: store.NewReadCache(rc, time.Minute),
		TTL:   time.Minute,
	}
	reg := mcp.NewRegistry()
	tools.RegisterWallet(reg, deps)
	tools.RegisterToken(reg, deps)
	tools.RegisterDeFi(reg, deps)
	tools.RegisterWhale(reg, deps)
	if reg.Len() != 7 {
		t.Fatalf("expected 7 tools, got %d", reg.Len())
	}
	return mcp.NewServer(reg)
}

// runStdioE2E pipes the server to a pair of io.Pipes and returns the
// stream + scanner for the test to drive.
func runStdioE2E(t *testing.T, srv *mcp.Server) (clientWrite *io.PipeWriter, clientRead *bufio.Reader, cancel func()) {
	t.Helper()
	cr, sw := io.Pipe()
	sr, cw := io.Pipe()
	transport := mcp.NewStdioTransport(srv, sr, sw)
	ctx, cancelFn := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = transport.Run(ctx)
		close(done)
	}()
	return cw, bufio.NewReader(cr), func() {
		cancelFn()
		_ = cw.Close()
		_ = sw.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("transport did not shut down")
		}
	}
}

// readFrame reads one newline-terminated JSON-RPC frame.
func readFrame(t *testing.T, r *bufio.Reader) mcp.Response {
	t.Helper()
	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var resp mcp.Response
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		t.Fatalf("decode %q: %v", line, err)
	}
	return resp
}

func TestE2E_StdioFullHandshake(t *testing.T) {
	srv := buildFullServer(t)
	in, out, cancel := runStdioE2E(t, srv)
	defer cancel()

	// 1. initialize
	if _, err := in.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}` + "\n")); err != nil {
		t.Fatal(err)
	}
	resp := readFrame(t, out)
	if resp.Error != nil {
		t.Fatalf("initialize: %+v", resp.Error)
	}
	res, _ := json.Marshal(resp.Result)
	if !strings.Contains(string(res), `"protocolVersion"`) {
		t.Errorf("initialize result missing protocolVersion: %s", res)
	}

	// 2. initialized notification (no response expected)
	if _, err := in.Write([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n")); err != nil {
		t.Fatal(err)
	}

	// 3. tools/list — verify all 6 tools surface
	if _, err := in.Write([]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n")); err != nil {
		t.Fatal(err)
	}
	listResp := readFrame(t, out)
	if listResp.Error != nil {
		t.Fatalf("tools/list: %+v", listResp.Error)
	}
	listJSON, _ := json.Marshal(listResp.Result)
	wantNames := []string{
		"get_wallet_positions", "get_wallet_balances", "get_wallet_history",
		"get_token_transfers", "get_defi_positions", "get_protocol_stats",
		"get_whale_activity",
	}
	for _, name := range wantNames {
		if !strings.Contains(string(listJSON), name) {
			t.Errorf("tools/list missing %q in %s", name, listJSON)
		}
	}

	// 4. tools/call (valid)
	if _, err := in.Write([]byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_wallet_balances","arguments":{"wallet":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","chain_id":1}}}` + "\n")); err != nil {
		t.Fatal(err)
	}
	callResp := readFrame(t, out)
	if callResp.Error != nil {
		t.Fatalf("tools/call: %+v", callResp.Error)
	}

	// 5. tools/call (unknown tool)
	if _, err := in.Write([]byte(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"nope","arguments":{}}}` + "\n")); err != nil {
		t.Fatal(err)
	}
	bad := readFrame(t, out)
	if bad.Error == nil || bad.Error.Code != mcp.CodeInvalidParams {
		t.Errorf("expected InvalidParams for unknown tool, got %+v", bad.Error)
	}

	// 6. ping
	if _, err := in.Write([]byte(`{"jsonrpc":"2.0","id":5,"method":"ping"}` + "\n")); err != nil {
		t.Fatal(err)
	}
	if pong := readFrame(t, out); pong.Error != nil {
		t.Errorf("ping error: %+v", pong.Error)
	}
}

func TestE2E_ToolsListReturnsAllTools(t *testing.T) {
	srv := buildFullServer(t)
	out := srv.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	var resp mcp.Response
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error != nil {
		t.Fatalf("err: %+v", resp.Error)
	}
	res, _ := json.Marshal(resp.Result)
	for _, name := range []string{
		"get_wallet_positions", "get_wallet_balances", "get_wallet_history",
		"get_token_transfers", "get_defi_positions", "get_protocol_stats",
		"get_whale_activity",
	} {
		if !strings.Contains(string(res), name) {
			t.Errorf("missing tool %q", name)
		}
	}
}

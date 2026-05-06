package tools

import (
	"context"
	"testing"

	"github.com/0xfandom/chainpulse/internal/mcp"
)

func newServerWithAllTools(t *testing.T) (*mcp.Server, *countingConn) {
	t.Helper()
	deps, conn, _ := newDeps(t)
	reg := mcp.NewRegistry()
	RegisterWallet(reg, deps)
	RegisterToken(reg, deps)
	RegisterDeFi(reg, deps)
	RegisterWhale(reg, deps)
	return mcp.NewServer(reg), conn
}

func TestDefiPositions_BasicEmpty(t *testing.T) {
	s, conn := newServerWithAllTools(t)
	out := s.Handle(context.Background(), toolCall("get_defi_positions",
		map[string]any{
			"wallet":   "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"protocol": "aave_v3",
		}))
	resp := decode(t, out)
	if resp.Error != nil {
		t.Fatalf("err: %+v", resp.Error)
	}
	if conn.queries != 1 {
		t.Errorf("queries = %d, want 1", conn.queries)
	}
}

func TestDefiPositions_RequiresProtocol(t *testing.T) {
	s, _ := newServerWithAllTools(t)
	out := s.Handle(context.Background(), toolCall("get_defi_positions",
		map[string]any{"wallet": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != mcp.CodeInvalidParams {
		t.Errorf("expected InvalidParams (missing protocol), got %+v", resp.Error)
	}
}

func TestDefiPositions_RejectsUnknownProtocol(t *testing.T) {
	s, _ := newServerWithAllTools(t)
	out := s.Handle(context.Background(), toolCall("get_defi_positions",
		map[string]any{
			"wallet":   "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"protocol": "morpho",
		}))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != mcp.CodeInvalidParams {
		t.Errorf("expected InvalidParams (enum), got %+v", resp.Error)
	}
}

func TestProtocolStats_BasicEmpty(t *testing.T) {
	s, conn := newServerWithAllTools(t)
	out := s.Handle(context.Background(), toolCall("get_protocol_stats",
		map[string]any{"protocol": "uniswap_v3"}))
	resp := decode(t, out)
	if resp.Error != nil {
		t.Fatalf("err: %+v", resp.Error)
	}
	if conn.queries != 1 {
		t.Errorf("queries = %d, want 1", conn.queries)
	}
}

func TestProtocolStats_RejectsUnknownProtocol(t *testing.T) {
	s, _ := newServerWithAllTools(t)
	out := s.Handle(context.Background(), toolCall("get_protocol_stats",
		map[string]any{"protocol": "morpho"}))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != mcp.CodeInvalidParams {
		t.Errorf("expected InvalidParams, got %+v", resp.Error)
	}
}

func TestProtocolStats_CachesSecondCall(t *testing.T) {
	s, conn := newServerWithAllTools(t)
	args := map[string]any{"protocol": "compound_v3"}
	_ = decode(t, s.Handle(context.Background(), toolCall("get_protocol_stats", args)))
	_ = decode(t, s.Handle(context.Background(), toolCall("get_protocol_stats", args)))
	if conn.queries != 1 {
		t.Errorf("expected cache hit on 2nd call, queries=%d", conn.queries)
	}
}

func TestWhale_RequiresHours(t *testing.T) {
	s, _ := newServerWithAllTools(t)
	out := s.Handle(context.Background(), toolCall("get_whale_activity", map[string]any{}))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != mcp.CodeInvalidParams {
		t.Errorf("expected InvalidParams (missing hours), got %+v", resp.Error)
	}
}

func TestWhale_HoursRange(t *testing.T) {
	s, _ := newServerWithAllTools(t)
	out := s.Handle(context.Background(), toolCall("get_whale_activity",
		map[string]any{"hours": 0}))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != mcp.CodeInvalidParams {
		t.Errorf("expected schema rejection on hours=0, got %+v", resp.Error)
	}

	out2 := s.Handle(context.Background(), toolCall("get_whale_activity",
		map[string]any{"hours": 999}))
	resp2 := decode(t, out2)
	if resp2.Error == nil || resp2.Error.Code != mcp.CodeInvalidParams {
		t.Errorf("expected schema rejection on hours>168, got %+v", resp2.Error)
	}
}

func TestWhale_RejectsBadMinAmount(t *testing.T) {
	s, _ := newServerWithAllTools(t)
	out := s.Handle(context.Background(), toolCall("get_whale_activity",
		map[string]any{"hours": 24, "min_amount": "not-a-number"}))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != mcp.CodeInvalidParams {
		t.Errorf("expected InvalidParams (min_amount pattern), got %+v", resp.Error)
	}
}

func TestWhale_RoutesByChain(t *testing.T) {
	s, conn := newServerWithAllTools(t)

	out := s.Handle(context.Background(), toolCall("get_whale_activity",
		map[string]any{"hours": 24}))
	if r := decode(t, out); r.Error != nil {
		t.Fatalf("global: %+v", r.Error)
	}

	out2 := s.Handle(context.Background(), toolCall("get_whale_activity",
		map[string]any{"hours": 24, "chain_id": 1}))
	if r := decode(t, out2); r.Error != nil {
		t.Fatalf("by-chain: %+v", r.Error)
	}

	if conn.queries != 2 {
		t.Errorf("expected 2 distinct queries (global vs by-chain), got %d", conn.queries)
	}
}

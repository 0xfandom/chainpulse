package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/0xfandom/chainpulse/internal/mcp"
)

func newServerWithChainTool(t *testing.T) (*mcp.Server, *countingConn) {
	t.Helper()
	deps, conn, _ := newDeps(t)
	reg := mcp.NewRegistry()
	RegisterChain(reg, deps)
	return mcp.NewServer(reg), conn
}

func TestLatestBlock_AllChainsEmpty(t *testing.T) {
	s, conn := newServerWithChainTool(t)
	out := s.Handle(context.Background(), toolCall("get_latest_block", map[string]any{}))
	resp := decode(t, out)
	if resp.Error != nil {
		t.Fatalf("err: %+v", resp.Error)
	}
	if conn.queries != 1 {
		t.Errorf("queries = %d, want 1", conn.queries)
	}
}

func TestLatestBlock_SingleChain(t *testing.T) {
	s, conn := newServerWithChainTool(t)
	out := s.Handle(context.Background(), toolCall("get_latest_block",
		map[string]any{"chain_id": 8453}))
	resp := decode(t, out)
	if resp.Error != nil {
		t.Fatalf("err: %+v", resp.Error)
	}
	if conn.queries != 1 {
		t.Errorf("queries = %d, want 1", conn.queries)
	}
}

func TestLatestBlock_RejectsZeroChain(t *testing.T) {
	s, _ := newServerWithChainTool(t)
	out := s.Handle(context.Background(), toolCall("get_latest_block",
		map[string]any{"chain_id": 0}))
	resp := decode(t, out)
	// chain_id minimum=1 per schema; 0 must be rejected
	if resp.Error == nil || resp.Error.Code != mcp.CodeInvalidParams {
		t.Errorf("expected InvalidParams for chain_id=0, got %+v", resp.Error)
	}
}

func TestLatestBlock_RejectsAdditionalProps(t *testing.T) {
	s, _ := newServerWithChainTool(t)
	out := s.Handle(context.Background(), toolCall("get_latest_block",
		map[string]any{"unknown": "field"}))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != mcp.CodeInvalidParams {
		t.Errorf("expected InvalidParams (additionalProperties), got %+v", resp.Error)
	}
}

func TestLatestBlock_DescribesUsage(t *testing.T) {
	def := latestBlockDef()
	if def.Name != "get_latest_block" {
		t.Errorf("name = %q", def.Name)
	}
	if !strings.Contains(def.Description, "most recently indexed block") {
		t.Errorf("description missing key phrase: %q", def.Description)
	}
}

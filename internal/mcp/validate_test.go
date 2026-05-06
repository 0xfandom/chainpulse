package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

const addressPattern = "^0x[0-9a-fA-F]{40}$"

func walletSchemaServer(t *testing.T) *Server {
	t.Helper()
	reg := NewRegistry()
	reg.MustRegister(ToolDefinition{
		Name:        "get_x",
		Description: "fetch x for a wallet",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []any{"wallet"},
			"properties": map[string]any{
				"wallet": map[string]any{"type": "string", "pattern": addressPattern},
				"limit":  map[string]any{"type": "integer", "minimum": 1, "maximum": 500},
			},
			"additionalProperties": false,
		},
	}, func(_ context.Context, _ json.RawMessage) (any, error) {
		return map[string]any{"ok": true}, nil
	})
	return NewServer(reg)
}

func TestRegistry_RejectsBadSchema(t *testing.T) {
	r := NewRegistry()
	bad := ToolDefinition{
		Name:        "broken",
		InputSchema: map[string]any{"type": []any{"not-a-real-type"}},
	}
	if err := r.Register(bad, okHandler); err == nil {
		t.Errorf("expected compile error for bad schema")
	}
}

func TestValidate_RejectsMissingRequired(t *testing.T) {
	s := walletSchemaServer(t)
	out := s.Handle(context.Background(),
		[]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_x","arguments":{}}}`))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != CodeInvalidParams {
		t.Fatalf("expected InvalidParams, got %+v", resp.Error)
	}
	if resp.Error.Data == nil {
		t.Errorf("expected structured detail in error.data")
	}
}

func TestValidate_RejectsBadAddressPattern(t *testing.T) {
	s := walletSchemaServer(t)
	out := s.Handle(context.Background(),
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_x","arguments":{"wallet":"not-an-address"}}}`))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != CodeInvalidParams {
		t.Fatalf("expected InvalidParams, got %+v", resp.Error)
	}
}

func TestValidate_RejectsWrongType(t *testing.T) {
	s := walletSchemaServer(t)
	out := s.Handle(context.Background(),
		[]byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_x","arguments":{"wallet":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","limit":"big"}}}`))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != CodeInvalidParams {
		t.Fatalf("expected InvalidParams, got %+v", resp.Error)
	}
}

func TestValidate_RejectsAdditionalProperties(t *testing.T) {
	s := walletSchemaServer(t)
	out := s.Handle(context.Background(),
		[]byte(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"get_x","arguments":{"wallet":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","extra":"nope"}}}`))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != CodeInvalidParams {
		t.Fatalf("expected InvalidParams, got %+v", resp.Error)
	}
}

func TestValidate_AcceptsValidPayload(t *testing.T) {
	s := walletSchemaServer(t)
	out := s.Handle(context.Background(),
		[]byte(`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"get_x","arguments":{"wallet":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","limit":50}}}`))
	resp := decode(t, out)
	if resp.Error != nil {
		t.Fatalf("valid payload errored: %+v", resp.Error)
	}
	res, _ := json.Marshal(resp.Result)
	if !strings.Contains(string(res), `ok`) {
		t.Errorf("missing handler result: %s", res)
	}
}

func TestValidate_EmptyArgsTreatedAsEmptyObject(t *testing.T) {
	r := NewRegistry()
	r.MustRegister(ToolDefinition{
		Name:        "no_args",
		InputSchema: map[string]any{"type": "object"},
	}, func(_ context.Context, _ json.RawMessage) (any, error) {
		return "ok", nil
	})
	s := NewServer(r)
	out := s.Handle(context.Background(),
		[]byte(`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"no_args"}}`))
	resp := decode(t, out)
	if resp.Error != nil {
		t.Errorf("missing args on no-args tool errored: %+v", resp.Error)
	}
}

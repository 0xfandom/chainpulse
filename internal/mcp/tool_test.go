package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

func okHandler(_ context.Context, _ json.RawMessage) (any, error) { return nil, nil }

func TestRegistry_RegisterAndLookup(t *testing.T) {
	r := NewRegistry()
	def := ToolDefinition{Name: "a", Description: "x", InputSchema: map[string]any{"type": "object"}}
	if err := r.Register(def, okHandler); err != nil {
		t.Fatal(err)
	}
	if r.Lookup("a") == nil {
		t.Errorf("Lookup(a) = nil, want handler")
	}
	if r.Lookup("missing") != nil {
		t.Errorf("Lookup(missing) returned non-nil")
	}
}

func TestRegistry_RejectsDuplicate(t *testing.T) {
	r := NewRegistry()
	def := ToolDefinition{Name: "dup", Description: "x", InputSchema: map[string]any{"type": "object"}}
	if err := r.Register(def, okHandler); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(def, okHandler); err == nil {
		t.Errorf("expected error on duplicate Register")
	}
}

func TestRegistry_RejectsEmptyName(t *testing.T) {
	r := NewRegistry()
	err := r.Register(ToolDefinition{Name: "", InputSchema: map[string]any{"type": "object"}}, okHandler)
	if err == nil {
		t.Errorf("expected error for empty name")
	}
}

func TestRegistry_RejectsNilHandler(t *testing.T) {
	r := NewRegistry()
	err := r.Register(ToolDefinition{Name: "x", InputSchema: map[string]any{"type": "object"}}, nil)
	if err == nil {
		t.Errorf("expected error for nil handler")
	}
}

func TestRegistry_RejectsNilSchema(t *testing.T) {
	r := NewRegistry()
	err := r.Register(ToolDefinition{Name: "x"}, okHandler)
	if err == nil {
		t.Errorf("expected error for nil InputSchema")
	}
}

func TestRegistry_ListPreservesOrder(t *testing.T) {
	r := NewRegistry()
	for _, n := range []string{"c", "a", "b"} {
		_ = r.Register(ToolDefinition{Name: n, InputSchema: map[string]any{"type": "object"}}, okHandler)
	}
	got := r.List()
	want := []string{"c", "a", "b"}
	if len(got) != len(want) {
		t.Fatalf("len(List) = %d, want %d", len(got), len(want))
	}
	for i, n := range want {
		if got[i].Name != n {
			t.Errorf("List[%d].Name = %q, want %q", i, got[i].Name, n)
		}
	}
}

func TestToolDefinition_JSONShape(t *testing.T) {
	def := ToolDefinition{
		Name:        "get_x",
		Description: "fetch x",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"id"},
			"properties": map[string]any{
				"id": map[string]any{"type": "string"},
			},
		},
	}
	b, err := json.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	var round map[string]any
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatal(err)
	}
	if round["name"] != "get_x" {
		t.Errorf("name = %v", round["name"])
	}
	if _, ok := round["inputSchema"].(map[string]any); !ok {
		t.Errorf("inputSchema missing or wrong type: %v", round["inputSchema"])
	}
}

func TestRegistry_Len(t *testing.T) {
	r := NewRegistry()
	if r.Len() != 0 {
		t.Errorf("empty Len = %d", r.Len())
	}
	_ = r.Register(ToolDefinition{Name: "a", InputSchema: map[string]any{"type": "object"}}, okHandler)
	if r.Len() != 1 {
		t.Errorf("Len after one Register = %d", r.Len())
	}
}

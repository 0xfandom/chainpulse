package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// ToolDefinition is the schema entry returned by tools/list. JSON shape
// matches the MCP spec: name + description + JSON Schema for inputs.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// ToolHandler is invoked by tools/call. params is the raw arguments object
// from the request. The returned value is JSON-marshalled into the
// response's content[0].text field by the server.
type ToolHandler func(ctx context.Context, params json.RawMessage) (any, error)

// Registry maps tool names to definitions + handlers. Iteration order
// (List) is preserved so tools/list output is deterministic.
type Registry struct {
	mu       sync.RWMutex
	order    []string
	defs     map[string]ToolDefinition
	handlers map[string]ToolHandler
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		defs:     make(map[string]ToolDefinition),
		handlers: make(map[string]ToolHandler),
	}
}

// Register adds a tool. Returns an error if name is empty, schema is nil,
// or the name is already registered.
func (r *Registry) Register(def ToolDefinition, handler ToolHandler) error {
	if def.Name == "" {
		return fmt.Errorf("tool name must not be empty")
	}
	if handler == nil {
		return fmt.Errorf("tool %q: handler must not be nil", def.Name)
	}
	if def.InputSchema == nil {
		return fmt.Errorf("tool %q: inputSchema must not be nil", def.Name)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.defs[def.Name]; exists {
		return fmt.Errorf("tool %q: already registered", def.Name)
	}
	r.defs[def.Name] = def
	r.handlers[def.Name] = handler
	r.order = append(r.order, def.Name)
	return nil
}

// MustRegister panics on registration failure. Use only at boot.
func (r *Registry) MustRegister(def ToolDefinition, handler ToolHandler) {
	if err := r.Register(def, handler); err != nil {
		panic(err)
	}
}

// List returns the tool definitions in registration order.
func (r *Registry) List() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ToolDefinition, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.defs[name])
	}
	return out
}

// Lookup returns the handler for name, or nil if unregistered.
func (r *Registry) Lookup(name string) ToolHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.handlers[name]
}

// Len returns the number of registered tools.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.order)
}

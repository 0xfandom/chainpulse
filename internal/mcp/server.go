package mcp

import (
	"context"
	"encoding/json"
)

// ProtocolVersion is the MCP protocol revision this server implements.
// Bumped only when capability negotiation semantics change.
const ProtocolVersion = "2024-11-05"

// ServerInfo identifies the implementation in the initialize handshake.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Server is the transport-agnostic MCP dispatcher. Wrap with stdio.go or
// sse.go for I/O.
type Server struct {
	Info     ServerInfo
	Registry *Registry
}

// NewServer constructs a Server with the given registry. Info defaults to
// {chainpulse, dev} when zero — overwrite after construction if needed.
func NewServer(reg *Registry) *Server {
	if reg == nil {
		reg = NewRegistry()
	}
	return &Server{
		Info:     ServerInfo{Name: "chainpulse", Version: "dev"},
		Registry: reg,
	}
}

// Handle dispatches one JSON-RPC frame and returns the response frame.
// nil is returned for notifications (no response expected). The bytes
// slice is always either nil or a valid JSON-RPC 2.0 response.
func (s *Server) Handle(ctx context.Context, frame []byte) []byte {
	var req Request
	if err := json.Unmarshal(frame, &req); err != nil {
		return mustMarshal(errorResponse(nil, newError(CodeParseError, "Parse error", err.Error())))
	}
	if req.JSONRPC != JSONRPCVersion {
		return mustMarshal(errorResponse(req.ID, newError(CodeInvalidRequest, "Invalid Request: jsonrpc must be \"2.0\"", nil)))
	}
	if req.Method == "" {
		return mustMarshal(errorResponse(req.ID, newError(CodeInvalidRequest, "Invalid Request: method required", nil)))
	}

	resp := s.dispatch(ctx, &req)

	if req.IsNotification() {
		return nil
	}
	if resp == nil {
		resp = errorResponse(req.ID, newError(CodeInternalError, "Internal error: nil response", nil))
	}
	return mustMarshal(resp)
}

// dispatch resolves method to a handler and returns the Response.
func (s *Server) dispatch(ctx context.Context, req *Request) *Response {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized", "notifications/initialized":
		return nil // notification
	case "ping":
		return resultResponse(req.ID, struct{}{})
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	default:
		return errorResponse(req.ID, newError(CodeMethodNotFound, "Method not found: "+req.Method, nil))
	}
}

// initializeResult is the payload returned to clients during handshake.
type initializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      ServerInfo     `json:"serverInfo"`
}

func (s *Server) handleInitialize(req *Request) *Response {
	return resultResponse(req.ID, initializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities:    map[string]any{"tools": map[string]any{}},
		ServerInfo:      s.Info,
	})
}

// toolsListResult is the payload returned to tools/list.
type toolsListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

func (s *Server) handleToolsList(req *Request) *Response {
	return resultResponse(req.ID, toolsListResult{Tools: s.Registry.List()})
}

// toolsCallParams is the wire shape for tools/call arguments.
type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// toolsCallResult mirrors the MCP spec: each tool returns a content array
// of typed parts. We only emit a single text part with the JSON-encoded
// result — agents that want structured access parse it client-side.
type toolsCallResult struct {
	Content []toolsCallContent `json:"content"`
	IsError bool               `json:"isError,omitempty"`
}

type toolsCallContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *Server) handleToolsCall(ctx context.Context, req *Request) *Response {
	var p toolsCallParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return errorResponse(req.ID, newError(CodeInvalidParams, "Invalid params: "+err.Error(), nil))
	}
	if p.Name == "" {
		return errorResponse(req.ID, newError(CodeInvalidParams, "Invalid params: tool name required", nil))
	}
	handler := s.Registry.Lookup(p.Name)
	if handler == nil {
		return errorResponse(req.ID, newError(CodeInvalidParams, "Unknown tool: "+p.Name, nil))
	}
	result, err := handler(ctx, p.Arguments)
	if err != nil {
		return errorResponse(req.ID, newError(CodeInternalError, err.Error(), nil))
	}
	encoded, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return errorResponse(req.ID, newError(CodeInternalError, "encode result: "+marshalErr.Error(), nil))
	}
	return resultResponse(req.ID, toolsCallResult{
		Content: []toolsCallContent{{Type: "text", Text: string(encoded)}},
	})
}

// mustMarshal serializes a Response. Failure here indicates a programming
// bug — an unmarshalable Result type — so we surface it as an internal
// error frame rather than crashing the transport.
func mustMarshal(resp *Response) []byte {
	b, err := json.Marshal(resp)
	if err != nil {
		fallback := &Response{
			JSONRPC: JSONRPCVersion,
			ID:      resp.ID,
			Error:   newError(CodeInternalError, "marshal response: "+err.Error(), nil),
		}
		b, _ = json.Marshal(fallback)
	}
	return b
}

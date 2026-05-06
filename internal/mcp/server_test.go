package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	reg := NewRegistry()
	reg.MustRegister(ToolDefinition{
		Name:        "echo",
		Description: "echo back the args",
		InputSchema: map[string]any{"type": "object"},
	}, func(_ context.Context, params json.RawMessage) (any, error) {
		return map[string]any{"echo": json.RawMessage(params)}, nil
	})
	return NewServer(reg)
}

func decode(t *testing.T, frame []byte) Response {
	t.Helper()
	var resp Response
	if err := json.Unmarshal(frame, &resp); err != nil {
		t.Fatalf("decode response: %v\nframe=%s", err, frame)
	}
	return resp
}

func TestHandle_Initialize(t *testing.T) {
	s := newTestServer(t)
	frame := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	out := s.Handle(context.Background(), frame)
	resp := decode(t, out)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	res, _ := json.Marshal(resp.Result)
	var got initializeResult
	if err := json.Unmarshal(res, &got); err != nil {
		t.Fatal(err)
	}
	if got.ProtocolVersion != ProtocolVersion {
		t.Errorf("protocolVersion = %q, want %q", got.ProtocolVersion, ProtocolVersion)
	}
	if _, ok := got.Capabilities["tools"]; !ok {
		t.Errorf("capabilities.tools missing: %+v", got.Capabilities)
	}
	if got.ServerInfo.Name != "chainpulse" {
		t.Errorf("serverInfo.name = %q", got.ServerInfo.Name)
	}
}

func TestHandle_Ping(t *testing.T) {
	s := newTestServer(t)
	out := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":7,"method":"ping"}`))
	resp := decode(t, out)
	if resp.Error != nil {
		t.Fatalf("ping returned error: %+v", resp.Error)
	}
}

func TestHandle_ToolsList(t *testing.T) {
	s := newTestServer(t)
	out := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	resp := decode(t, out)
	if resp.Error != nil {
		t.Fatalf("tools/list error: %+v", resp.Error)
	}
	res, _ := json.Marshal(resp.Result)
	var got toolsListResult
	if err := json.Unmarshal(res, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Tools) != 1 || got.Tools[0].Name != "echo" {
		t.Errorf("tools = %+v, want one tool 'echo'", got.Tools)
	}
}

func TestHandle_ToolsCall_Success(t *testing.T) {
	s := newTestServer(t)
	out := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"hello":"world"}}}`))
	resp := decode(t, out)
	if resp.Error != nil {
		t.Fatalf("tools/call error: %+v", resp.Error)
	}
	res, _ := json.Marshal(resp.Result)
	if !strings.Contains(string(res), "hello") {
		t.Errorf("result content missing echoed args: %s", res)
	}
}

func TestHandle_ToolsCall_UnknownTool(t *testing.T) {
	s := newTestServer(t)
	out := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"nope","arguments":{}}}`))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != CodeInvalidParams {
		t.Errorf("expected InvalidParams, got %+v", resp.Error)
	}
}

func TestHandle_MethodNotFound(t *testing.T) {
	s := newTestServer(t)
	out := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":5,"method":"does_not_exist"}`))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != CodeMethodNotFound {
		t.Errorf("expected MethodNotFound, got %+v", resp.Error)
	}
}

func TestHandle_ParseError(t *testing.T) {
	s := newTestServer(t)
	out := s.Handle(context.Background(), []byte(`not-json`))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != CodeParseError {
		t.Errorf("expected ParseError, got %+v", resp.Error)
	}
}

func TestHandle_WrongJSONRPCVersion(t *testing.T) {
	s := newTestServer(t)
	out := s.Handle(context.Background(), []byte(`{"jsonrpc":"1.0","id":6,"method":"ping"}`))
	resp := decode(t, out)
	if resp.Error == nil || resp.Error.Code != CodeInvalidRequest {
		t.Errorf("expected InvalidRequest, got %+v", resp.Error)
	}
}

func TestHandle_NotificationProducesNoOutput(t *testing.T) {
	s := newTestServer(t)
	out := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	if out != nil {
		t.Errorf("notification produced output: %s", out)
	}
}

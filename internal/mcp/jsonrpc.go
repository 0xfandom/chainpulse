// Package mcp implements an MCP (Model Context Protocol) server that
// exposes ChainPulse query tools to AI agents. The package core is
// transport-agnostic: stdio and SSE adapters live alongside but call into
// the same Server.Handle dispatcher.
package mcp

import "encoding/json"

// JSONRPCVersion is the only version this server speaks.
const JSONRPCVersion = "2.0"

// JSON-RPC 2.0 error codes (https://www.jsonrpc.org/specification#error_object).
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// Request is a JSON-RPC 2.0 request frame. ID is a json.RawMessage so the
// caller's number / string / null type is preserved verbatim in the
// response.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// IsNotification reports whether this request has no id (i.e. expects no
// response, per JSON-RPC 2.0 §4.1).
func (r *Request) IsNotification() bool {
	return len(r.ID) == 0 || string(r.ID) == "null"
}

// Response is a JSON-RPC 2.0 response frame. Exactly one of Result / Error
// is set on a non-notification reply.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error is a JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *Error) Error() string { return e.Message }

// newError constructs an Error.
func newError(code int, msg string, data any) *Error {
	return &Error{Code: code, Message: msg, Data: data}
}

// errorResponse builds a Response carrying the given error and id.
func errorResponse(id json.RawMessage, err *Error) *Response {
	if id == nil {
		id = json.RawMessage("null")
	}
	return &Response{JSONRPC: JSONRPCVersion, ID: id, Error: err}
}

// resultResponse builds a Response carrying a successful result.
func resultResponse(id json.RawMessage, result any) *Response {
	return &Response{JSONRPC: JSONRPCVersion, ID: id, Result: result}
}

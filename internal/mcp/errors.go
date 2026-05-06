package mcp

// Named JSON-RPC 2.0 error constructors. Values map to Code* in jsonrpc.go.

// ErrParse builds a -32700 Parse error.
func ErrParse(detail any) *Error {
	return newError(CodeParseError, "Parse error", detail)
}

// ErrInvalidRequest builds a -32600 Invalid Request.
func ErrInvalidRequest(msg string) *Error {
	return newError(CodeInvalidRequest, "Invalid Request: "+msg, nil)
}

// ErrMethodNotFound builds a -32601 Method not found.
func ErrMethodNotFound(method string) *Error {
	return newError(CodeMethodNotFound, "Method not found: "+method, nil)
}

// ErrInvalidParams builds a -32602 Invalid params with optional structured
// detail (e.g. failing field paths from a schema validator).
func ErrInvalidParams(msg string, detail any) *Error {
	return newError(CodeInvalidParams, "Invalid params: "+msg, detail)
}

// ErrInternal builds a -32603 Internal error.
func ErrInternal(msg string) *Error {
	return newError(CodeInternalError, msg, nil)
}

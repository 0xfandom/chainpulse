package mcp

import "testing"

func TestErrorCodesMatchSpec(t *testing.T) {
	cases := []struct {
		err  *Error
		code int
		name string
	}{
		{ErrParse(nil), CodeParseError, "parse"},
		{ErrInvalidRequest("x"), CodeInvalidRequest, "invalid_request"},
		{ErrMethodNotFound("foo"), CodeMethodNotFound, "method_not_found"},
		{ErrInvalidParams("x", nil), CodeInvalidParams, "invalid_params"},
		{ErrInternal("boom"), CodeInternalError, "internal"},
	}
	for _, c := range cases {
		if c.err.Code != c.code {
			t.Errorf("%s: code = %d, want %d", c.name, c.err.Code, c.code)
		}
		if c.err.Message == "" {
			t.Errorf("%s: empty message", c.name)
		}
	}
}

func TestErrorImplementsError(t *testing.T) {
	var err error = ErrInternal("x")
	if err.Error() == "" {
		t.Errorf("Error() returned empty string")
	}
}

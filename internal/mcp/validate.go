package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// compileSchema compiles a JSON Schema given as a generic map. The
// compiler operates on parsed JSON nodes, so the map is round-tripped
// through json.Marshal first.
func compileSchema(name string, schema map[string]any) (*jsonschema.Schema, error) {
	if schema == nil {
		return nil, fmt.Errorf("nil schema")
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}
	c := jsonschema.NewCompiler()
	url := "mem://tools/" + name + ".json"
	if err := c.AddResource(url, doc); err != nil {
		return nil, fmt.Errorf("add schema resource: %w", err)
	}
	compiled, err := c.Compile(url)
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	return compiled, nil
}

// validateArgs runs the precompiled schema against args. Returns nil on
// success or an *Error carrying CodeInvalidParams + a structured detail
// payload describing the failing field paths.
func validateArgs(schema *jsonschema.Schema, args json.RawMessage) *Error {
	if schema == nil {
		return nil
	}
	body := args
	if len(body) == 0 {
		body = json.RawMessage(`{}`)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
	if err != nil {
		return ErrInvalidParams("arguments are not valid JSON", err.Error())
	}
	if err := schema.Validate(doc); err != nil {
		return ErrInvalidParams("arguments do not match tool schema", schemaErrorDetail(err))
	}
	return nil
}

// schemaErrorDetail converts a jsonschema.ValidationError into a compact
// list of {path, message} entries so agents can render the failure
// without parsing free-form text.
func schemaErrorDetail(err error) any {
	verr, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return err.Error()
	}
	type entry struct {
		Path    string `json:"path"`
		Message string `json:"message"`
	}
	var out []entry
	var walk func(e *jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		path := "/" + strings.Join(toStrings(e.InstanceLocation), "/")
		out = append(out, entry{Path: path, Message: fmt.Sprint(e.ErrorKind)})
		for _, c := range e.Causes {
			walk(c)
		}
	}
	walk(verr)
	return out
}

func toStrings(parts []string) []string {
	cp := make([]string, len(parts))
	copy(cp, parts)
	return cp
}

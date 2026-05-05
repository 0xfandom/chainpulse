package mcp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
)

// stdioMaxFrameBytes is the per-frame buffer ceiling. tools/call payloads
// can include sizable result bodies, so the default 64 KiB scanner buffer
// is insufficient.
const stdioMaxFrameBytes = 1 << 20 // 1 MiB

// StdioTransport reads newline-delimited JSON-RPC frames from In and
// writes responses to Out. The Claude Desktop MCP client uses this
// transport over a child-process stdio pipe.
type StdioTransport struct {
	Server *Server
	In     io.Reader
	Out    io.Writer

	mu sync.Mutex // serializes writes to Out
}

// NewStdioTransport constructs a StdioTransport bound to the given
// streams.
func NewStdioTransport(srv *Server, in io.Reader, out io.Writer) *StdioTransport {
	return &StdioTransport{Server: srv, In: in, Out: out}
}

// Run reads frames until the input closes or ctx is cancelled. The
// scanner runs in a goroutine so context cancellation can preempt a
// blocking Read. Returns nil on clean EOF or context cancel.
func (t *StdioTransport) Run(ctx context.Context) error {
	if t.Server == nil {
		return errors.New("stdio: nil server")
	}
	if t.In == nil || t.Out == nil {
		return errors.New("stdio: nil reader or writer")
	}

	scanner := bufio.NewScanner(t.In)
	scanner.Buffer(make([]byte, 0, 64<<10), stdioMaxFrameBytes)

	frames := make(chan []byte)
	scanErr := make(chan error, 1)

	go func() {
		defer close(frames)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			cp := make([]byte, len(line))
			copy(cp, line)
			select {
			case frames <- cp:
			case <-ctx.Done():
				return
			}
		}
		if err := scanner.Err(); err != nil {
			scanErr <- err
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case frame, ok := <-frames:
			if !ok {
				select {
				case err := <-scanErr:
					return err
				default:
					return nil
				}
			}
			resp := t.Server.Handle(ctx, frame)
			if resp == nil {
				continue
			}
			if err := t.write(resp); err != nil {
				return fmt.Errorf("stdio write: %w", err)
			}
		}
	}
}

// write emits one response frame followed by '\n'. Mutex guards against
// interleaved writes if Run is ever called multiple times against the
// same writer.
func (t *StdioTransport) write(frame []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, err := t.Out.Write(frame); err != nil {
		return err
	}
	_, err := t.Out.Write([]byte{'\n'})
	return err
}

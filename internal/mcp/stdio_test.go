package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

func runStdio(t *testing.T) (in *io.PipeWriter, out *bufio.Reader, done <-chan error, cancel context.CancelFunc) {
	t.Helper()
	srv := newTestServer(t)
	pr1, pw1 := io.Pipe()
	pr2, pw2 := io.Pipe()
	tr := NewStdioTransport(srv, pr1, pw2)
	ctx, cancelFn := context.WithCancel(context.Background())
	doneCh := make(chan error, 1)
	go func() { doneCh <- tr.Run(ctx) }()
	return pw1, bufio.NewReader(pr2), doneCh, func() {
		cancelFn()
		_ = pw1.Close()
		_ = pw2.Close()
	}
}

func TestStdio_Roundtrip(t *testing.T) {
	in, out, done, cancel := runStdio(t)
	defer cancel()

	if _, err := in.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")); err != nil {
		t.Fatal(err)
	}
	line, err := out.ReadString('\n')
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if !strings.Contains(line, `"echo"`) {
		t.Errorf("response missing tool name: %s", line)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not exit after cancel")
	}
}

func TestStdio_NotificationProducesNoOutput(t *testing.T) {
	in, out, done, cancel := runStdio(t)
	defer cancel()

	if _, err := in.Write([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n")); err != nil {
		t.Fatal(err)
	}

	// Send a follow-up that does produce output, prove the first wrote nothing
	// before the second's frame appears.
	if _, err := in.Write([]byte(`{"jsonrpc":"2.0","id":2,"method":"ping"}` + "\n")); err != nil {
		t.Fatal(err)
	}

	line, err := out.ReadString('\n')
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var resp Response
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(resp.ID) != "2" {
		t.Errorf("first response id = %s, want 2 (notification produced output)", resp.ID)
	}
	cancel()
	<-done
}

func TestStdio_ContextCancelExits(t *testing.T) {
	in, _, done, cancel := runStdio(t)
	_ = in
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run returned error on cancel: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not exit after cancel")
	}
}

func TestStdio_LargeFrameHandled(t *testing.T) {
	in, out, _, cancel := runStdio(t)
	defer cancel()

	pad := strings.Repeat("x", 200<<10)
	frame := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"big":"` + pad + `"}}}` + "\n"
	if _, err := in.Write([]byte(frame)); err != nil {
		t.Fatal(err)
	}
	line, err := out.ReadString('\n')
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var resp Response
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("large frame errored: %+v", resp.Error)
	}
}

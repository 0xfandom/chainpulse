package monitor

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestWaitWithTimeout_DrainsBeforeTimeout(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		time.Sleep(20 * time.Millisecond)
		wg.Done()
	}()
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	start := time.Now()
	WaitWithTimeout(&wg, time.Second, logger, "test")
	if dur := time.Since(start); dur > 200*time.Millisecond {
		t.Errorf("unexpected wait duration: %v", dur)
	}
	if buf.Len() != 0 {
		t.Errorf("clean drain should not log; got %s", buf.String())
	}
}

func TestWaitWithTimeout_LogsOnTimeout(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Done()

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	WaitWithTimeout(&wg, 50*time.Millisecond, logger, "stuck")
	out := buf.String()
	if out == "" {
		t.Fatalf("expected timeout warning, got empty log")
	}
	if !bytes.Contains(buf.Bytes(), []byte("shutdown timeout exceeded")) {
		t.Errorf("missing timeout message: %s", out)
	}
	if !bytes.Contains(buf.Bytes(), []byte("\"what\":\"stuck\"")) {
		t.Errorf("missing 'what' label: %s", out)
	}
}

func TestWaitWithTimeout_ZeroTimeoutWaitsForever(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		time.Sleep(20 * time.Millisecond)
		wg.Done()
	}()
	var buf bytes.Buffer
	WaitWithTimeout(&wg, 0, zerolog.New(&buf), "no-timeout")
	if buf.Len() != 0 {
		t.Errorf("zero timeout should not log; got %s", buf.String())
	}
}

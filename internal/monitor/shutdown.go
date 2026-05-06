package monitor

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// WaitWithTimeout blocks until wg drains or timeout elapses. On timeout
// it logs a warning so operators see shutdown didn't complete cleanly.
// Pass a zero or negative timeout to wait forever.
func WaitWithTimeout(wg *sync.WaitGroup, timeout time.Duration, logger zerolog.Logger, what string) {
	if timeout <= 0 {
		wg.Wait()
		return
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		logger.Warn().Dur("timeout", timeout).Str("what", what).Msg("shutdown timeout exceeded")
	}
}

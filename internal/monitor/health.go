package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// ServeHealth starts an HTTP server exposing /health and /ready on addr.
//
//   - /health returns 200 OK whenever the process is running. It is a
//     pure liveness probe; use it to detect a crashed or unreachable
//     binary.
//   - /ready reports per-chain head-freshness for the indexer. With
//     readiness disabled (the default for non-indexer binaries) it
//     mirrors /health and always returns 200. With readiness enabled
//     (see EnableReadiness) it returns 503 when no chain has reported a
//     head yet or when any registered chain has gone stale, so docker
//     can autoheal a stuck indexer.
//
// Mirrors ServeMetrics in lifecycle: blocks until ctx is cancelled or
// the listener errors. http.ErrServerClosed from graceful shutdown is
// treated as a clean exit.
func ServeHealth(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleLiveness)
	mux.HandleFunc("/ready", handleReadiness)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func handleLiveness(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleReadiness(w http.ResponseWriter, _ *http.Request) {
	snap := Readiness(time.Now())
	code := http.StatusOK
	if snap.Status != "ok" {
		code = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(snap)
}

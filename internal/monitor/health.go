package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// ServeHealth starts an HTTP server exposing /health on addr. Returns
// 200 OK with a small JSON body when the process is running. Mirrors
// ServeMetrics in lifecycle: blocks until ctx is cancelled or the
// listener errors. http.ErrServerClosed from graceful shutdown is
// treated as a clean exit.
func ServeHealth(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

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

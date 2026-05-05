// Package api wires the Gin REST server. Handlers live under
// internal/api/handlers; storage helpers under internal/api/store;
// middleware under internal/api/middleware.
package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/0xfandom/chainpulse/internal/api/middleware"
	"github.com/0xfandom/chainpulse/internal/api/store"
	chainpulselog "github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/monitor"
	"github.com/0xfandom/chainpulse/internal/types"
)

// Server wraps a Gin router plus its read-side dependencies.
type Server struct {
	cfg       types.APIConfig
	router    *gin.Engine
	store     *store.ReadStore
	cache     *store.ReadCache
	rateLimit *middleware.RateLimiter
	httpSrv   *http.Server
	version   string
}

// NewServer wires global middleware, /health, /metrics, and a /v1 group.
// Endpoint subgroups under /v1 are registered by the handlers layer in
// follow-up PRs via Server.V1Group.
func NewServer(cfg types.APIConfig, st *store.ReadStore, cache *store.ReadCache, version string) *Server {
	gin.SetMode(gin.ReleaseMode)

	rl := middleware.NewRateLimiter(cfg.RateLimit)

	s := &Server{
		cfg:       cfg,
		store:     st,
		cache:     cache,
		rateLimit: rl,
		version:   version,
	}
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(requestLogger())
	r.Use(metricsMiddleware())
	r.Use(middleware.CORS(cfg.CORS))
	r.Use(rl.Middleware())

	r.GET("/health", s.healthHandler)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// v1 group is empty here; later PRs attach handlers.
	r.Group("/v1")

	s.router = r
	return s
}

// Router exposes the Gin engine for tests and external wiring.
func (s *Server) Router() *gin.Engine { return s.router }

// V1Group returns the /v1 routing group so handler packages can attach
// routes without re-creating it.
func (s *Server) V1Group() *gin.RouterGroup {
	return s.router.Group("/v1")
}

// Store / Cache exposed so handlers can read.
func (s *Server) Store() *store.ReadStore { return s.store }
func (s *Server) Cache() *store.ReadCache { return s.cache }

// Run starts the HTTP server and blocks until ctx is cancelled or the
// server returns. Drains for cfg.ShutdownDrain on cancel.
func (s *Server) Run(ctx context.Context) error {
	s.httpSrv = &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           s.router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger := chainpulselog.Component("api_server")
	logger.Info().Str("addr", s.cfg.Addr).Msg("listening")

	errCh := make(chan error, 1)
	go func() { errCh <- s.httpSrv.ListenAndServe() }()

	select {
	case <-ctx.Done():
		drain := s.cfg.ShutdownDrain.AsDuration()
		if drain <= 0 {
			drain = 15 * time.Second
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), drain)
		defer cancel()
		_ = s.httpSrv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": s.version,
	})
}

// requestLogger emits one structured log line per request.
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		l := chainpulselog.Component("api_request")
		l.Info().
			Str("method", c.Request.Method).
			Str("path", c.FullPath()).
			Int("status", c.Writer.Status()).
			Dur("dur", time.Since(start)).
			Str("remote", c.ClientIP()).
			Msg("request")
	}
}

// metricsMiddleware bumps api_request_duration_seconds + api_requests_total
// for every request. FullPath() is used so labels stay bounded — querystring
// values don't blow up cardinality.
func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		monitor.ObserveAPIRequest(c.Request.Method, path, strconv.Itoa(c.Writer.Status()), time.Since(start))
	}
}

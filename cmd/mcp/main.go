// Command mcp serves the Model Context Protocol surface for ChainPulse.
// Read-only — wraps the same ReadStore + ReadCache used by the api binary.
// Stdio transport drives the Claude Desktop integration; SSE transport
// covers HTTP-based agents.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/0xfandom/chainpulse/internal/api/store"
	"github.com/0xfandom/chainpulse/internal/config"
	chainpulselog "github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/mcp"
	"github.com/0xfandom/chainpulse/internal/mcp/tools"
	"github.com/0xfandom/chainpulse/internal/monitor"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "config/config.toml", "path to TOML config file")
	flag.Parse()

	if err := run(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "mcp: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	chainpulselog.Init(cfg.App.LogLevel)
	logger := chainpulselog.Component("mcp")
	logger.Info().
		Str("version", version).
		Str("config", configPath).
		Str("transport", cfg.MCP.Transport).
		Str("addr", cfg.MCP.Addr).
		Msg("starting")

	rootCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	chCtx, chCancel := context.WithTimeout(rootCtx, 10*time.Second)
	rs, err := store.DialReadStore(chCtx, cfg.ClickHouse.DSN)
	chCancel()
	if err != nil {
		return fmt.Errorf("dial clickhouse: %w", err)
	}

	rcCtx, rcCancel := context.WithTimeout(rootCtx, 5*time.Second)
	cache, err := store.DialReadCache(rcCtx, store.ReadCacheConfig{
		Addr:       cfg.Redis.Addr,
		Password:   cfg.Redis.Password,
		DB:         cfg.Redis.DB,
		DefaultTTL: cfg.Redis.DefaultTTL.AsDuration(),
	})
	rcCancel()
	if err != nil {
		_ = rs.Close()
		return fmt.Errorf("dial redis: %w", err)
	}

	deps := &tools.Deps{Store: rs, Cache: cache, TTL: cfg.MCP.CacheTTL.AsDuration()}
	reg := mcp.NewRegistry()
	tools.RegisterWallet(reg, deps)
	tools.RegisterToken(reg, deps)
	tools.RegisterDeFi(reg, deps)
	tools.RegisterWhale(reg, deps)

	server := mcp.NewServer(reg)
	server.Info = mcp.ServerInfo{Name: "chainpulse", Version: version}
	logger.Info().Int("tools", reg.Len()).Msg("tools registered")

	var wg sync.WaitGroup

	if addr := cfg.App.MetricsAddr; addr != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := monitor.ServeMetrics(rootCtx, addr); err != nil {
				logger.Error().Err(err).Str("addr", addr).Msg("metrics server stopped")
			}
		}()
		logger.Info().Str("addr", addr).Msg("metrics server listening")
	}

	// Stdio transport uses stdin/stdout as the protocol channel; binding a
	// health server would still be safe (stderr-only logging) but the
	// canonical flow there is "process is alive iff parent process holds
	// the pipe open." Skip it. SSE transport runs over HTTP and benefits
	// from a separate /health endpoint.
	if cfg.MCP.Transport == "sse" {
		if addr := cfg.App.HealthAddr; addr != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := monitor.ServeHealth(rootCtx, addr); err != nil {
					logger.Error().Err(err).Str("addr", addr).Msg("health server stopped")
				}
			}()
			logger.Info().Str("addr", addr).Msg("health server listening")
		}
	}

	transportErr := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		switch cfg.MCP.Transport {
		case "stdio":
			tr := mcp.NewStdioTransport(server, os.Stdin, os.Stdout)
			transportErr <- tr.Run(rootCtx)
		case "sse":
			tr := mcp.NewSSETransport(server, cfg.MCP.Addr)
			tr.BearerToken = cfg.MCP.BearerToken
			transportErr <- tr.Run(rootCtx)
		default:
			transportErr <- fmt.Errorf("unsupported transport %q", cfg.MCP.Transport)
		}
	}()

	select {
	case <-rootCtx.Done():
		logger.Info().Msg("shutdown signal received")
	case err := <-transportErr:
		if err != nil {
			logger.Error().Err(err).Msg("transport exited")
		}
		cancel()
	}

	monitor.WaitWithTimeout(&wg, cfg.App.ShutdownTimeout.AsDuration(), logger, "transport+metrics+health")

	if err := cache.Close(); err != nil {
		logger.Error().Err(err).Msg("redis close failed")
	}
	if err := rs.Close(); err != nil {
		logger.Error().Err(err).Msg("clickhouse close failed")
	}
	logger.Info().Msg("mcp stopped")
	return nil
}

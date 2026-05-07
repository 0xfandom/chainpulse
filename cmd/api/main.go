// Command api serves the REST + gRPC + WebSocket query layer over
// ClickHouse + Redis. Read-only; the indexer + processor own the writes.
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

	"github.com/0xfandom/chainpulse/internal/api"
	grpcsrv "github.com/0xfandom/chainpulse/internal/api/grpc"
	"github.com/0xfandom/chainpulse/internal/api/handlers"
	"github.com/0xfandom/chainpulse/internal/api/store"
	"github.com/0xfandom/chainpulse/internal/config"
	chainpulselog "github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/monitor"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "config/config.toml", "path to TOML config file")
	flag.Parse()

	if err := run(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "api: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	chainpulselog.Init(cfg.App.LogLevel)
	logger := chainpulselog.Component("api")
	logger.Info().
		Str("version", version).
		Str("config", configPath).
		Str("rest_addr", cfg.API.Addr).
		Str("grpc_addr", cfg.API.GRPCAddr).
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

	server := api.NewServer(cfg.API, rs, cache, version)

	l1 := store.NewL1Cache(store.L1Config{Capacity: 1024, TTL: 2 * time.Second})

	deps := handlers.HandlerDeps{Store: rs, Cache: cache, L1: l1}
	v1 := server.V1Group()
	handlers.NewWalletHandlers(deps).Register(v1)
	handlers.NewTokenHandlers(deps).Register(v1)
	handlers.NewProtocolHandlers(deps).Register(v1)
	handlers.NewChainHandlers(deps).Register(v1)

	wsHandler := api.NewWSHandler(cfg.API.WebSocket, cfg.API.CORS, cfg.Kafka.Brokers, cfg.Kafka.TopicDecodedEvents)
	wsHandler.Register(v1)

	gserver := grpcsrv.New(rs, cache)

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

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := gserver.Serve(cfg.API.GRPCAddr); err != nil {
			logger.Error().Err(err).Msg("grpc server exited")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Run(rootCtx); err != nil {
			logger.Error().Err(err).Msg("rest server exited")
		}
	}()

	<-rootCtx.Done()
	logger.Info().Msg("shutdown signal received")

	gserver.GracefulStop()
	monitor.WaitWithTimeout(&wg, cfg.App.ShutdownTimeout.AsDuration(), logger, "rest+grpc+metrics")

	if err := cache.Close(); err != nil {
		logger.Error().Err(err).Msg("redis close failed")
	}
	if err := rs.Close(); err != nil {
		logger.Error().Err(err).Msg("clickhouse close failed")
	}
	logger.Info().Msg("api stopped")
	return nil
}

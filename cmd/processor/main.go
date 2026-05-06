// Command processor consumes raw_events from Kafka, decodes them through
// protocol-specific decoders, aggregates wallet positions, and writes
// enriched data to ClickHouse + Redis. It also publishes decoded_events
// and positions_update for downstream consumers.
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

	"github.com/rs/zerolog"

	"github.com/0xfandom/chainpulse/internal/config"
	chainpulselog "github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/monitor"
	"github.com/0xfandom/chainpulse/internal/processor"
	"github.com/0xfandom/chainpulse/internal/processor/protocols"
	"github.com/0xfandom/chainpulse/internal/types"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "config/config.toml", "path to TOML config file")
	flag.Parse()

	if err := run(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "processor: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	chainpulselog.Init(cfg.App.LogLevel)
	logger := chainpulselog.Component("processor")
	logger.Info().Str("version", version).Str("config", configPath).Msg("starting")

	rootCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	chCtx, chCancel := context.WithTimeout(rootCtx, 10*time.Second)
	chConn, err := processor.DialClickHouse(chCtx, cfg.ClickHouse.DSN)
	chCancel()
	if err != nil {
		return fmt.Errorf("dial clickhouse: %w", err)
	}
	batch, err := processor.NewBatchWriter(chConn, processor.BatchWriterConfig{
		BatchSize:     cfg.ClickHouse.BatchSize,
		BatchInterval: cfg.ClickHouse.BatchInterval.AsDuration(),
	})
	if err != nil {
		_ = chConn.Close()
		return fmt.Errorf("init batch writer: %w", err)
	}

	redisCtx, redisCancel := context.WithTimeout(rootCtx, 5*time.Second)
	cache, err := processor.NewCacheWriter(redisCtx, processor.CacheWriterConfig{
		Addr:       cfg.Redis.Addr,
		Password:   cfg.Redis.Password,
		DB:         cfg.Redis.DB,
		DefaultTTL: cfg.Redis.DefaultTTL.AsDuration(),
	})
	redisCancel()
	if err != nil {
		_ = batch.Close(rootCtx)
		return fmt.Errorf("init redis: %w", err)
	}

	chainNames := chainNameMap(cfg.Chains)
	prod, err := processor.NewProcessorProducer(processor.ProducerConfig{
		Brokers:              cfg.Kafka.Brokers,
		TopicDecodedEvents:   cfg.Kafka.TopicDecodedEvents,
		TopicPositionsUpdate: cfg.Kafka.TopicPositionsUpdate,
		ClientID:             cfg.Kafka.ClientID,
		ChainNames:           chainNames,
	})
	if err != nil {
		_ = cache.Close()
		_ = batch.Close(rootCtx)
		return fmt.Errorf("init kafka producer: %w", err)
	}

	registry := processor.NewRegistry()
	for _, d := range []processor.ProtocolDecoder{
		protocols.NewERC20Decoder(),
		protocols.NewUniswapV3Decoder(),
		protocols.NewAaveV3Decoder(),
		protocols.NewCompoundV3Decoder(),
	} {
		if err := registry.Register(d); err != nil {
			_ = prod.Close()
			_ = cache.Close()
			_ = batch.Close(rootCtx)
			return fmt.Errorf("register decoder %s: %w", d.Name(), err)
		}
	}

	agg, err := processor.NewAggregator(batch, cache, prod)
	if err != nil {
		_ = prod.Close()
		_ = cache.Close()
		_ = batch.Close(rootCtx)
		return fmt.Errorf("init aggregator: %w", err)
	}

	consumer, err := processor.NewConsumer(processor.ConsumerConfig{
		Brokers:     cfg.Kafka.Brokers,
		Topic:       cfg.Kafka.TopicRawEvents,
		GroupID:     cfg.Processor.ConsumerGroup,
		ClientID:    cfg.Kafka.ClientID,
		MaxInFlight: cfg.Processor.MaxInFlight,
	})
	if err != nil {
		_ = prod.Close()
		_ = cache.Close()
		_ = batch.Close(rootCtx)
		return fmt.Errorf("init consumer: %w", err)
	}

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

	handler := func(ctx context.Context, e *types.ChainEvent) error {
		decoded, err := registry.Decode(e)
		if err != nil {
			// no decoder = silently skip (commit so we don't redeliver)
			return nil
		}
		return agg.Process(ctx, decoded)
	}

	logger.Info().
		Str("topic", cfg.Kafka.TopicRawEvents).
		Str("group", cfg.Processor.ConsumerGroup).
		Strs("decoders", registry.Names()).
		Msg("consumer ready")

	if err := consumer.Run(rootCtx, handler); err != nil {
		logger.Error().Err(err).Msg("consumer exited with error")
	}

	logger.Info().Msg("shutdown signal received; flushing pipeline")

	shutdownTimeout := cfg.App.ShutdownTimeout.AsDuration()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	logStep(logger, "consumer.Close", consumer.Close)
	logStepCtx(logger, "batch.Close", shutdownCtx, batch.Close)
	logStep(logger, "producer.Close", prod.Close)
	logStep(logger, "cache.Close", cache.Close)

	monitor.WaitWithTimeout(&wg, shutdownTimeout, logger, "metrics+health")
	logger.Info().Msg("processor stopped")
	return nil
}

// chainNameMap returns chain_id -> name from the loaded ChainConfigs.
func chainNameMap(chains []types.ChainConfig) map[uint64]string {
	out := make(map[uint64]string, len(chains))
	for _, c := range chains {
		out[c.ChainID] = c.Name
	}
	return out
}

// logStep wraps a no-arg cleanup func with timing + log.
func logStep(logger zerolog.Logger, name string, fn func() error) {
	start := time.Now()
	if err := fn(); err != nil {
		logger.Error().Err(err).Str("step", name).Dur("dur", time.Since(start)).Msg("shutdown step failed")
		return
	}
	logger.Info().Str("step", name).Dur("dur", time.Since(start)).Msg("shutdown step done")
}

// logStepCtx wraps a context-taking cleanup func with timing + log.
func logStepCtx(logger zerolog.Logger, name string, ctx context.Context, fn func(context.Context) error) {
	start := time.Now()
	if err := fn(ctx); err != nil {
		logger.Error().Err(err).Str("step", name).Dur("dur", time.Since(start)).Msg("shutdown step failed")
		return
	}
	logger.Info().Str("step", name).Dur("dur", time.Since(start)).Msg("shutdown step done")
}

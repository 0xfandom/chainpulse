// Command indexer subscribes to chain WebSocket endpoints, decodes raw event
// logs via ABI, and publishes structured events to Kafka.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/0xfandom/chainpulse/internal/config"
	"github.com/0xfandom/chainpulse/internal/ingestion"
	chainpulselog "github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/monitor"
	"github.com/0xfandom/chainpulse/internal/types"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "config/config.toml", "path to TOML config file")
	flag.Parse()

	if err := run(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "indexer: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	chainpulselog.Init(cfg.App.LogLevel)
	logger := chainpulselog.Component("indexer")
	logger.Info().Str("version", version).Str("config", configPath).Int("chains", len(cfg.Chains)).Msg("starting")

	chainNames := make(map[uint64]string, len(cfg.Chains))
	for _, c := range cfg.Chains {
		chainNames[c.ChainID] = c.Name
	}

	producer, err := ingestion.NewProducer(ingestion.ProducerConfig{
		Brokers:    cfg.Kafka.Brokers,
		Topic:      cfg.Kafka.TopicRawEvents,
		ClientID:   cfg.Kafka.ClientID,
		ChainNames: chainNames,
	})
	if err != nil {
		return fmt.Errorf("init kafka producer: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var wg sync.WaitGroup

	if addr := cfg.App.MetricsAddr; addr != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := monitor.ServeMetrics(ctx, addr); err != nil {
				logger.Error().Err(err).Str("addr", addr).Msg("metrics server stopped")
			}
		}()
		logger.Info().Str("addr", addr).Msg("metrics server listening")
	}

	decoder := ingestion.NewABIDecoder()

	for _, ch := range cfg.Chains {
		ch := ch // capture
		wg.Add(1)
		go func(c types.ChainConfig) {
			defer wg.Done()
			listener := ingestion.NewChainListener(c, decoder, producer)
			if err := listener.Run(ctx); err != nil {
				l := chainpulselog.Chain(c.Name, c.ChainID)
				l.Error().Err(err).Msg("listener stopped with error")
			}
		}(ch)
	}

	<-ctx.Done()
	logger.Info().Msg("shutdown signal received, stopping listeners")

	wg.Wait()

	if err := producer.Close(); err != nil {
		logger.Error().Err(err).Msg("kafka producer close failed")
	}

	logger.Info().Msg("indexer stopped")
	return nil
}

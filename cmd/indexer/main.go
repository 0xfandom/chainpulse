// Command indexer subscribes to chain WebSocket endpoints, decodes raw event
// logs via ABI, and publishes structured events to Kafka.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/0xfandom/chainpulse/internal/config"
	"github.com/0xfandom/chainpulse/internal/ingestion"
	chainpulselog "github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/monitor"
	"github.com/0xfandom/chainpulse/internal/types"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "config/config.toml", "path to TOML config file")
	probe := flag.Bool("healthcheck", false, "run readiness probe against -probe-url and exit 0 on 200, 1 otherwise")
	probeURL := flag.String("probe-url", "http://localhost:9180/ready", "URL used by -healthcheck probe")
	flag.Parse()

	if *probe {
		os.Exit(runProbe(*probeURL))
	}

	if err := run(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "indexer: %v\n", err)
		os.Exit(1)
	}
}

// runProbe performs a single GET against url and returns 0 when the
// response status is 2xx, 1 otherwise. Used as the docker healthcheck
// command on the indexer's distroless image, which has no shell or
// wget.
func runProbe(url string) int {
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: GET %s: %v\n", url, err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return 0
	}
	fmt.Fprintf(os.Stderr, "healthcheck: GET %s: status %d\n", url, resp.StatusCode)
	return 1
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

	monitor.EnableReadiness(cfg.App.ReadinessHeadTimeout.AsDuration())
	for _, c := range cfg.Chains {
		monitor.RegisterChain(c.Name)
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

	if addr := cfg.App.HealthAddr; addr != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := monitor.ServeHealth(ctx, addr); err != nil {
				logger.Error().Err(err).Str("addr", addr).Msg("health server stopped")
			}
		}()
		logger.Info().Str("addr", addr).Msg("health server listening")
	}

	decoder := ingestion.NewABIDecoder()

	var kafkaUnhealthy atomic.Bool

	for _, ch := range cfg.Chains {
		ch := ch // capture
		wg.Add(1)
		go func(c types.ChainConfig) {
			defer wg.Done()
			listener := ingestion.NewChainListener(c, decoder, producer).
				WithKafkaFailThreshold(cfg.App.KafkaPublishFailThreshold)
			if err := listener.Run(ctx); err != nil {
				l := chainpulselog.Chain(c.Name, c.ChainID)
				l.Error().Err(err).Msg("listener stopped with error")
				if errors.Is(err, ingestion.ErrKafkaUnhealthy) {
					kafkaUnhealthy.Store(true)
					cancel()
				}
			}
		}(ch)
	}

	<-ctx.Done()
	logger.Info().Msg("shutdown signal received, stopping listeners")

	monitor.WaitWithTimeout(&wg, cfg.App.ShutdownTimeout.AsDuration(), logger, "listeners")

	if err := producer.Close(); err != nil {
		logger.Error().Err(err).Msg("kafka producer close failed")
	}

	if kafkaUnhealthy.Load() {
		logger.Error().Msg("indexer exiting non-zero due to kafka publish unhealthy")
		return ingestion.ErrKafkaUnhealthy
	}

	logger.Info().Msg("indexer stopped")
	return nil
}

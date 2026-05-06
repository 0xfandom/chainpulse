package ingestion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"

	chainpulselog "github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/monitor"
	"github.com/0xfandom/chainpulse/internal/types"
)

const (
	reconnectInitialBackoff = time.Second
	reconnectMaxBackoff     = 30 * time.Second
	reconnectMaxAttempts    = 10
)

// ChainListener subscribes to a single chain's WebSocket head stream,
// fetches logs per block, decodes them, and publishes to Kafka.
type ChainListener struct {
	cfg      types.ChainConfig
	decoder  *ABIDecoder
	producer *Producer
	dialer   Dialer
	log      zerolog.Logger
}

// Dialer abstracts ethclient.DialContext so tests can inject a fake.
type Dialer func(ctx context.Context, url string) (EthClient, error)

// EthClient is the subset of ethclient.Client the listener uses. Defined
// as an interface for test substitution.
type EthClient interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *ethtypes.Header) (ethereum.Subscription, error)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]ethtypes.Log, error)
	Close()
}

// defaultDialer wraps ethclient.DialContext to satisfy the Dialer signature.
func defaultDialer(ctx context.Context, url string) (EthClient, error) {
	c, err := ethclient.DialContext(ctx, url)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// NewChainListener constructs a listener for cfg using the supplied decoder
// and producer. Dialer defaults to ethclient.DialContext.
func NewChainListener(cfg types.ChainConfig, decoder *ABIDecoder, producer *Producer) *ChainListener {
	return &ChainListener{
		cfg:      cfg,
		decoder:  decoder,
		producer: producer,
		dialer:   defaultDialer,
		log:      chainpulselog.Chain(cfg.Name, cfg.ChainID).With().Str(chainpulselog.FieldComponent, "chain_listener").Logger(),
	}
}

// WithDialer overrides the default ethclient dialer. Used for tests.
func (cl *ChainListener) WithDialer(d Dialer) *ChainListener {
	cl.dialer = d
	return cl
}

// Run blocks until ctx is cancelled or the reconnection budget is exhausted.
// Reconnects with exponential backoff (1s -> 30s cap, max 10 attempts) on
// subscription error or initial dial failure.
func (cl *ChainListener) Run(ctx context.Context) error {
	backoff := reconnectInitialBackoff
	for attempt := 0; attempt < reconnectMaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return nil
		}
		err := cl.runOnce(ctx)
		if err == nil || ctx.Err() != nil {
			return nil
		}
		cl.log.Warn().Err(err).Int("attempt", attempt+1).Dur("backoff", backoff).Msg("reconnecting")
		monitor.SetListenerConnected(cl.cfg.Name, false)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return nil
		}
		backoff *= 2
		if backoff > reconnectMaxBackoff {
			backoff = reconnectMaxBackoff
		}
	}
	return fmt.Errorf("listener %s: exhausted %d reconnect attempts", cl.cfg.Name, reconnectMaxAttempts)
}

// runOnce executes a single dial + subscription session. Returns nil if
// ctx is cancelled cleanly, or an error suitable for reconnection.
func (cl *ChainListener) runOnce(ctx context.Context) error {
	client, err := cl.dialer(ctx, cl.cfg.RPCWSS)
	if err != nil {
		return fmt.Errorf("dial %s: %w", cl.cfg.Name, err)
	}
	defer client.Close()

	cl.log.Info().Msg("connected to WebSocket")
	monitor.SetListenerConnected(cl.cfg.Name, true)

	headers := make(chan *ethtypes.Header, 16)
	sub, err := client.SubscribeNewHead(ctx, headers)
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", cl.cfg.Name, err)
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-sub.Err():
			if err == nil {
				return errors.New("subscription closed")
			}
			return fmt.Errorf("subscription error: %w", err)
		case header := <-headers:
			if header == nil {
				continue
			}
			cl.processBlock(ctx, client, header)
		}
	}
}

// processBlock fetches the logs for header.Number, decodes them, and
// publishes recognized events to Kafka.
func (cl *ChainListener) processBlock(ctx context.Context, client EthClient, header *ethtypes.Header) {
	start := time.Now()
	blockNum := header.Number
	if blockNum == nil {
		return
	}

	q := ethereum.FilterQuery{FromBlock: blockNum, ToBlock: blockNum}
	if addrs := parseContracts(cl.cfg.Contracts); len(addrs) > 0 {
		q.Addresses = addrs
	}

	logs, err := client.FilterLogs(ctx, q)
	if err != nil {
		cl.log.Error().Err(err).Uint64(chainpulselog.FieldBlock, blockNum.Uint64()).Msg("filter logs failed")
		return
	}

	for i := range logs {
		event, err := cl.decoder.Decode(cl.cfg.ChainID, logs[i], header.Time)
		if err != nil {
			continue // unrecognized event, skip silently
		}
		if err := cl.producer.Publish(ctx, event); err != nil {
			cl.log.Error().Err(err).
				Uint64(chainpulselog.FieldBlock, blockNum.Uint64()).
				Str("event", event.EventName).
				Msg("kafka publish failed")
		}
	}

	monitor.ObserveBlockProcessing(cl.cfg.Name, time.Since(start))
}

// parseContracts converts string contract addresses from config into
// common.Address values. Empty entries are skipped.
func parseContracts(cs []string) []common.Address {
	if len(cs) == 0 {
		return nil
	}
	out := make([]common.Address, 0, len(cs))
	for _, s := range cs {
		if s == "" {
			continue
		}
		out = append(out, common.HexToAddress(s))
	}
	return out
}

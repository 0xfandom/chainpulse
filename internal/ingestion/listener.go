package ingestion

import (
	"context"
	"errors"
	"fmt"
	"math/big"
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
//
// Reorg safety: emits a block only after `Confirmations` newer heads
// arrive on top of it. With Confirmations = N, each new head H causes
// blocks (lastEmitted, H-N] to be processed via HeaderByNumber. Blocks
// at H-N are buried under N descendants and are very unlikely to be
// reorged out. Confirmations = 0 disables the lag (head emits
// immediately).
type ChainListener struct {
	cfg         types.ChainConfig
	decoder     *ABIDecoder
	producer    *Producer
	dialer      Dialer
	log         zerolog.Logger
	lastEmitted uint64 // last safe block published; 0 = uninitialized
}

// Dialer abstracts ethclient.DialContext so tests can inject a fake.
type Dialer func(ctx context.Context, url string) (EthClient, error)

// EthClient is the subset of ethclient.Client the listener uses. Defined
// as an interface for test substitution.
type EthClient interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *ethtypes.Header) (ethereum.Subscription, error)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]ethtypes.Log, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*ethtypes.Header, error)
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
//
// Cursor lastEmitted is reset on each session: missed-block backfill on
// reconnect is intentionally skipped to avoid re-emitting hours of
// blocks if the listener was offline. Resume from the current safe head
// instead.
func (cl *ChainListener) runOnce(ctx context.Context) error {
	client, err := cl.dialer(ctx, cl.cfg.RPCWSS)
	if err != nil {
		return fmt.Errorf("dial %s: %w", cl.cfg.Name, err)
	}
	defer client.Close()

	cl.log.Info().Msg("connected to WebSocket")
	monitor.SetListenerConnected(cl.cfg.Name, true)

	cl.lastEmitted = 0

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
			cl.onNewHead(ctx, client, header)
		}
	}
}

// onNewHead translates a fresh chain-head event into one or more
// reorg-safe block emissions. For each block in (lastEmitted,
// head-Confirmations] it fetches the block header (or reuses the
// supplied head when Confirmations = 0) and routes it through
// processBlock.
func (cl *ChainListener) onNewHead(ctx context.Context, client EthClient, head *ethtypes.Header) {
	if head.Number == nil {
		return
	}
	headNum := head.Number.Uint64()
	conf := cl.cfg.Confirmations
	if headNum < conf {
		return
	}
	safe := headNum - conf

	if cl.lastEmitted == 0 {
		// First head this session — start streaming from current safe
		// forward; do not backfill arbitrarily deep history.
		if safe == 0 {
			cl.lastEmitted = 0
			cl.processBlock(ctx, client, head)
			return
		}
		cl.lastEmitted = safe - 1
	}

	for n := cl.lastEmitted + 1; n <= safe; n++ {
		var bh *ethtypes.Header
		if n == headNum {
			bh = head
		} else {
			h, err := client.HeaderByNumber(ctx, new(big.Int).SetUint64(n))
			if err != nil {
				cl.log.Error().Err(err).Uint64(chainpulselog.FieldBlock, n).Msg("header by number failed")
				return
			}
			bh = h
		}
		cl.processBlock(ctx, client, bh)
		cl.lastEmitted = n
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

	events := make([]*types.ChainEvent, 0, len(logs))
	for i := range logs {
		event, err := cl.decoder.Decode(cl.cfg.ChainID, logs[i], header.Time)
		if err != nil {
			continue // unrecognized event, skip silently
		}
		events = append(events, event)
	}

	if len(events) > 0 {
		if err := cl.producer.PublishBatch(ctx, events); err != nil {
			cl.log.Error().Err(err).
				Uint64(chainpulselog.FieldBlock, blockNum.Uint64()).
				Int("events", len(events)).
				Msg("kafka publish batch failed")
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

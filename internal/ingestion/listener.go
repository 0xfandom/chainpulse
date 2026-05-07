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
	defaultHeadTimeout        = 90 * time.Second
	defaultKafkaFailThreshold = 5
)

// Reconnect cadence is held in package-level vars (not consts) so tests
// can compress the wall-clock cost of an exhaustion run.
var (
	reconnectInitialBackoff = time.Second
	reconnectMaxBackoff     = 30 * time.Second
	reconnectMaxAttempts    = 10
)

// errHeadTimeout is returned by runOnce when no head arrives within the
// configured watchdog window. Surfaced as an error so the existing
// reconnect loop kicks in.
var errHeadTimeout = errors.New("head watchdog timeout")

// ErrKafkaUnhealthy is returned by Run when the producer has failed to
// publish for too many consecutive blocks. Wrapped errors from main
// short-circuit the reconnect loop so docker can restart the indexer
// instead of cycling WSS while kafka stays broken.
var ErrKafkaUnhealthy = errors.New("kafka publish unhealthy: consecutive block failures exceeded threshold")

// ErrChainListenerDead is returned by Run when the per-chain reconnect
// budget is exhausted (e.g. provider returns 429 on every handshake).
// cmd/indexer escalates this to a process-level exit so docker can
// restart the indexer; otherwise the chain would silently drop out of
// the pipeline for the lifetime of the process.
var ErrChainListenerDead = errors.New("chain listener exhausted reconnect attempts")

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
	cfg                types.ChainConfig
	decoder            *ABIDecoder
	producer           *Producer
	dialer             Dialer
	log                zerolog.Logger
	lastEmitted        uint64 // last safe block published; 0 = uninitialized
	kafkaFailStreak    int    // consecutive blocks for which kafka publish failed
	kafkaFailThreshold int    // streak count that triggers ErrKafkaUnhealthy
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
// and producer. Dialer defaults to ethclient.DialContext. Kafka fail-fast
// threshold defaults to defaultKafkaFailThreshold; override with
// WithKafkaFailThreshold.
func NewChainListener(cfg types.ChainConfig, decoder *ABIDecoder, producer *Producer) *ChainListener {
	return &ChainListener{
		cfg:                cfg,
		decoder:            decoder,
		producer:           producer,
		dialer:             defaultDialer,
		log:                chainpulselog.Chain(cfg.Name, cfg.ChainID).With().Str(chainpulselog.FieldComponent, "chain_listener").Logger(),
		kafkaFailThreshold: defaultKafkaFailThreshold,
	}
}

// WithDialer overrides the default ethclient dialer. Used for tests.
func (cl *ChainListener) WithDialer(d Dialer) *ChainListener {
	cl.dialer = d
	return cl
}

// WithKafkaFailThreshold sets the consecutive-block kafka-publish failure
// count that triggers ErrKafkaUnhealthy. Values <= 0 leave the default
// (5) in place.
func (cl *ChainListener) WithKafkaFailThreshold(n int) *ChainListener {
	if n > 0 {
		cl.kafkaFailThreshold = n
	}
	return cl
}

// Run blocks until ctx is cancelled or the reconnection budget is exhausted.
// Reconnects with exponential backoff (1s -> 30s cap, max 10 attempts) on
// subscription error or initial dial failure.
//
// ErrKafkaUnhealthy short-circuits the reconnect loop: reconnecting WSS
// does not help when kafka is the bug, so the error bubbles up and the
// indexer process exits non-zero for docker to restart.
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
		if errors.Is(err, ErrKafkaUnhealthy) {
			monitor.SetListenerConnected(cl.cfg.Name, false)
			return err
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
	return fmt.Errorf("%w: %s after %d attempts", ErrChainListenerDead, cl.cfg.Name, reconnectMaxAttempts)
}

// runOnce executes a single dial + subscription session. Returns nil if
// ctx is cancelled cleanly, or an error suitable for reconnection.
//
// Cursor lastEmitted is reset on each session: missed-block backfill on
// reconnect is intentionally skipped to avoid re-emitting hours of
// blocks if the listener was offline. Resume from the current safe head
// instead.
//
// A head watchdog covers providers that drop heads without surfacing an
// error on the subscription. If no header arrives within HeadTimeout
// (default 90s), runOnce returns errHeadTimeout and the outer Run loop
// reconnects.
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

	headTimeout := cl.cfg.HeadTimeout.AsDuration()
	if headTimeout <= 0 {
		headTimeout = defaultHeadTimeout
	}
	watchdog := time.NewTimer(headTimeout)
	defer watchdog.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-sub.Err():
			if err == nil {
				return errors.New("subscription closed")
			}
			return fmt.Errorf("subscription error: %w", err)
		case <-watchdog.C:
			cl.log.Warn().Dur("timeout", headTimeout).Msg("no head received within watchdog window; reconnecting")
			return fmt.Errorf("%w after %s", errHeadTimeout, headTimeout)
		case header := <-headers:
			if header == nil {
				continue
			}
			if !watchdog.Stop() {
				select {
				case <-watchdog.C:
				default:
				}
			}
			watchdog.Reset(headTimeout)
			if err := cl.onNewHead(ctx, client, header); err != nil {
				return err
			}
		}
	}
}

// onNewHead translates a fresh chain-head event into one or more
// reorg-safe block emissions. For each block in (lastEmitted,
// head-Confirmations] it fetches the block header (or reuses the
// supplied head when Confirmations = 0) and routes it through
// processBlock.
//
// On a kafka publish failure, lastEmitted is NOT advanced for the
// failing block; the caller's next head will retry from the same point.
// kafkaFailStreak counts consecutive failures across blocks; on
// reaching kafkaFailThreshold the function returns ErrKafkaUnhealthy
// so Run can exit and let docker restart the process. Other transient
// errors (FilterLogs, HeaderByNumber) follow the original behaviour:
// log and return without advancing.
func (cl *ChainListener) onNewHead(ctx context.Context, client EthClient, head *ethtypes.Header) error {
	if head.Number == nil {
		return nil
	}
	headNum := head.Number.Uint64()
	conf := cl.cfg.Confirmations
	if headNum < conf {
		return nil
	}
	safe := headNum - conf

	if cl.lastEmitted == 0 {
		// First head this session — start streaming from current safe
		// forward; do not backfill arbitrarily deep history.
		if safe == 0 {
			cl.lastEmitted = 0
			if err := cl.processBlock(ctx, client, head); err != nil {
				return cl.recordPublishFailure(err)
			}
			cl.kafkaFailStreak = 0
			return nil
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
				return nil
			}
			bh = h
		}
		if err := cl.processBlock(ctx, client, bh); err != nil {
			return cl.recordPublishFailure(err)
		}
		cl.kafkaFailStreak = 0
		cl.lastEmitted = n
	}
	return nil
}

// recordPublishFailure increments the consecutive-failure streak and
// returns ErrKafkaUnhealthy when the threshold is reached. Any prior
// publish error is wrapped in the returned sentinel so callers can
// match it via errors.Is.
func (cl *ChainListener) recordPublishFailure(err error) error {
	cl.kafkaFailStreak++
	if cl.kafkaFailStreak >= cl.kafkaFailThreshold {
		cl.log.Error().
			Int("streak", cl.kafkaFailStreak).
			Int("threshold", cl.kafkaFailThreshold).
			Err(err).
			Msg("kafka publish unhealthy; failing fast for docker restart")
		return fmt.Errorf("%w: %v", ErrKafkaUnhealthy, err)
	}
	return nil
}

// processBlock fetches the logs for header.Number, decodes them, and
// publishes recognized events to Kafka. Returns the publish error so
// the caller can decide whether to advance the cursor or trigger
// fail-fast. Decode/FilterLogs errors are logged and swallowed; the
// caller does not advance but does not count those toward the kafka
// streak.
func (cl *ChainListener) processBlock(ctx context.Context, client EthClient, header *ethtypes.Header) error {
	start := time.Now()
	blockNum := header.Number
	if blockNum == nil {
		return nil
	}

	q := ethereum.FilterQuery{FromBlock: blockNum, ToBlock: blockNum}
	if addrs := parseContracts(cl.cfg.Contracts); len(addrs) > 0 {
		q.Addresses = addrs
	}

	logs, err := client.FilterLogs(ctx, q)
	if err != nil {
		cl.log.Error().Err(err).Uint64(chainpulselog.FieldBlock, blockNum.Uint64()).Msg("filter logs failed")
		return nil
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
			monitor.ObserveBlockProcessing(cl.cfg.Name, time.Since(start))
			return err
		}
	}

	monitor.RecordHead(cl.cfg.Name)
	monitor.ObserveBlockProcessing(cl.cfg.Name, time.Since(start))
	return nil
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

package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"

	chainpulselog "github.com/0xfandom/chainpulse/internal/log"
	"github.com/0xfandom/chainpulse/internal/monitor"
	"github.com/0xfandom/chainpulse/internal/types"
)

// CacheWriter maintains the hot cache of wallet balances and positions.
// It is safe for concurrent use; redis.Client is itself a connection
// pool.
type CacheWriter struct {
	client     *redis.Client
	defaultTTL time.Duration
}

// CacheWriterConfig is the input to NewCacheWriter.
type CacheWriterConfig struct {
	Addr       string
	Password   string
	DB         int
	DefaultTTL time.Duration
}

// NewCacheWriter constructs a CacheWriter and verifies the connection
// with a PING.
func NewCacheWriter(ctx context.Context, cfg CacheWriterConfig) (*CacheWriter, error) {
	if cfg.Addr == "" {
		return nil, errors.New("redis writer: addr must be set")
	}
	if cfg.DefaultTTL <= 0 {
		return nil, errors.New("redis writer: default_ttl must be > 0")
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis writer: ping: %w", err)
	}
	return newCacheWriter(client, cfg.DefaultTTL), nil
}

// newCacheWriter is the unexported constructor used by tests.
func newCacheWriter(client *redis.Client, defaultTTL time.Duration) *CacheWriter {
	return &CacheWriter{client: client, defaultTTL: defaultTTL}
}

// IncrBalance applies delta to the balance hash field for
// (wallet, chainID, token). Day-2 limitation: deltas larger than
// math.MaxInt64 are clamped at int64 boundaries and bump
// processor_consume_errors_total{kind=redis_write}; a Lua script with
// arbitrary-precision math is the planned follow-up.
func (w *CacheWriter) IncrBalance(ctx context.Context, chainID uint64, wallet, token common.Address, delta *big.Int) error {
	if delta == nil || delta.Sign() == 0 {
		return nil
	}
	start := time.Now()
	defer func() { monitor.ObserveRedisWrite(time.Since(start)) }()

	key := balanceKey(wallet, chainID, token)
	d, clamped := clampInt64(delta)
	if clamped {
		l := chainpulselog.Component("redis_writer")
		l.Warn().
			Str("wallet", wallet.Hex()).
			Str("token", token.Hex()).
			Str("delta", delta.String()).
			Msg("incr balance clamped to int64; large-value path not yet implemented")
		monitor.IncProcessorError(monitor.ProcErrRedisWrite)
	}
	return w.client.HIncrBy(ctx, key, "amount", d).Err()
}

// SetPosition writes the JSON-encoded position under the canonical key
// position:{wallet}:{chain_id}:{protocol}:{position_type}:{token} with
// the writer's default TTL.
func (w *CacheWriter) SetPosition(ctx context.Context, pos *types.WalletPosition) error {
	if pos == nil {
		return errors.New("redis writer: nil position")
	}
	start := time.Now()
	defer func() { monitor.ObserveRedisWrite(time.Since(start)) }()

	payload, err := json.Marshal(pos)
	if err != nil {
		return fmt.Errorf("redis writer: marshal position: %w", err)
	}
	key := positionKey(pos.Wallet, pos.ChainID, pos.Protocol, pos.PositionType, pos.Token)
	return w.client.Set(ctx, key, payload, w.defaultTTL).Err()
}

// InvalidatePositions deletes every position:* key for a given wallet
// across chains/protocols. Uses SCAN so it stays non-blocking on large
// keyspaces.
func (w *CacheWriter) InvalidatePositions(ctx context.Context, wallet common.Address) error {
	start := time.Now()
	defer func() { monitor.ObserveRedisWrite(time.Since(start)) }()

	pattern := fmt.Sprintf("position:%s:*", strings.ToLower(wallet.Hex()))
	iter := w.client.Scan(ctx, 0, pattern, 100).Iterator()
	var batch []string
	for iter.Next(ctx) {
		batch = append(batch, iter.Val())
		if len(batch) >= 256 {
			if err := w.client.Del(ctx, batch...).Err(); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(batch) > 0 {
		if err := w.client.Del(ctx, batch...).Err(); err != nil {
			return err
		}
	}
	return nil
}

// Close releases the underlying connection pool.
func (w *CacheWriter) Close() error {
	if w == nil || w.client == nil {
		return nil
	}
	return w.client.Close()
}

// balanceKey returns the canonical hash key for a (wallet, chain, token)
// balance triple. Hex addresses are lowercased so logical equality
// matches across input casings.
func balanceKey(wallet common.Address, chainID uint64, token common.Address) string {
	return fmt.Sprintf("balance:%s:%d:%s", strings.ToLower(wallet.Hex()), chainID, strings.ToLower(token.Hex()))
}

// positionKey returns the canonical key for a wallet position record.
func positionKey(wallet common.Address, chainID uint64, protocol, positionType string, token common.Address) string {
	return fmt.Sprintf("position:%s:%d:%s:%s:%s",
		strings.ToLower(wallet.Hex()), chainID, protocol, positionType, strings.ToLower(token.Hex()))
}

// clampInt64 returns delta as int64 if it fits, else the corresponding
// boundary plus clamped=true.
func clampInt64(delta *big.Int) (int64, bool) {
	if delta.IsInt64() {
		return delta.Int64(), false
	}
	if delta.Sign() < 0 {
		return -1 << 62, true
	}
	return 1<<62 - 1, true
}

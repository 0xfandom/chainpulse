package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// ReadCache is the api's hot-cache reader. Wraps go-redis/v9 *Client.
type ReadCache struct {
	client     *redis.Client
	defaultTTL time.Duration
}

// ReadCacheConfig is the input to DialReadCache.
type ReadCacheConfig struct {
	Addr       string
	Password   string
	DB         int
	DefaultTTL time.Duration
}

// DialReadCache opens a Redis client and PINGs.
func DialReadCache(ctx context.Context, cfg ReadCacheConfig) (*ReadCache, error) {
	if cfg.Addr == "" {
		return nil, errors.New("read cache: addr must be set")
	}
	if cfg.DefaultTTL <= 0 {
		return nil, errors.New("read cache: default_ttl must be > 0")
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("read cache: ping: %w", err)
	}
	return NewReadCache(client, cfg.DefaultTTL), nil
}

// NewReadCache wraps an existing client. Used by tests.
func NewReadCache(client *redis.Client, defaultTTL time.Duration) *ReadCache {
	return &ReadCache{client: client, defaultTTL: defaultTTL}
}

// Close releases the connection pool.
func (c *ReadCache) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

// DefaultTTL exposes the configured default TTL so handlers can opt in.
func (c *ReadCache) DefaultTTL() time.Duration { return c.defaultTTL }

// Client exposes the underlying redis client for advanced use (SCAN,
// pipelining). Handlers should prefer the wrapper methods.
func (c *ReadCache) Client() *redis.Client { return c.client }

// GetBytes returns the raw bytes stored under key. Returns
// (nil, false, nil) on cache miss; (data, true, nil) on hit.
func (c *ReadCache) GetBytes(ctx context.Context, key string) ([]byte, bool, error) {
	v, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return v, true, nil
}

// SetBytes stores raw bytes under key with the given TTL. ttl <= 0 falls
// back to the default TTL.
func (c *ReadCache) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}
	return c.client.Set(ctx, key, value, ttl).Err()
}

// GetBalances reads every token amount under the
// balance:{wallet}:{chain}:* hash family. Returns an empty map (not nil
// error) when nothing is cached for the wallet.
func (c *ReadCache) GetBalances(ctx context.Context, wallet string, chainID uint64) (map[string]string, error) {
	pattern := fmt.Sprintf("balance:%s:%d:*", strings.ToLower(wallet), chainID)
	out := map[string]string{}
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		amount, err := c.client.HGet(ctx, key, "amount").Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			return nil, err
		}
		// key format: balance:<wallet>:<chain>:<token>
		token := key[strings.LastIndex(key, ":")+1:]
		out[token] = amount
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAllBalances scans every cached chain for the wallet and returns a
// {chain_id -> {token -> amount}} map. Empty outer map (not nil) when
// nothing is cached.
func (c *ReadCache) GetAllBalances(ctx context.Context, wallet string) (map[uint64]map[string]string, error) {
	pattern := fmt.Sprintf("balance:%s:*:*", strings.ToLower(wallet))
	out := map[uint64]map[string]string{}
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		// balance:<wallet>:<chain>:<token>
		parts := strings.Split(key, ":")
		if len(parts) != 4 {
			continue
		}
		var chainID uint64
		if _, err := fmt.Sscan(parts[2], &chainID); err != nil {
			continue
		}
		amount, err := c.client.HGet(ctx, key, "amount").Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			return nil, err
		}
		bucket, ok := out[chainID]
		if !ok {
			bucket = map[string]string{}
			out[chainID] = bucket
		}
		bucket[parts[3]] = amount
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

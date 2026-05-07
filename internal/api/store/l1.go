package store

import (
	"time"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
)

// L1Cache is the in-process layer in front of Redis. Holds raw response
// bytes keyed identically to the Redis cache key so writes invalidate
// both layers in lock-step. Capacity-bounded, time-bounded — entries
// older than the configured TTL are evicted on access.
//
// Nil receiver methods are no-ops so handlers and helpers can pass a
// disabled L1 without conditional plumbing.
type L1Cache struct {
	c *lru.LRU[string, []byte]
}

// L1Config tunes the L1 layer.
type L1Config struct {
	Capacity int
	TTL      time.Duration
}

// NewL1Cache constructs an L1Cache. Capacity <= 0 or TTL <= 0 returns a
// nil-but-callable cache (every Get is a miss, every Set a no-op).
func NewL1Cache(cfg L1Config) *L1Cache {
	if cfg.Capacity <= 0 || cfg.TTL <= 0 {
		return nil
	}
	return &L1Cache{c: lru.NewLRU[string, []byte](cfg.Capacity, nil, cfg.TTL)}
}

// Get returns the value cached under key plus a hit/miss flag. The
// returned slice shares its backing array with the cache entry —
// callers must treat it as read-only. If a future caller needs to
// mutate the bytes (append, in-place edit) it must copy first via
// bytes.Clone.
func (l *L1Cache) Get(key string) ([]byte, bool) {
	if l == nil || l.c == nil {
		return nil, false
	}
	return l.c.Get(key)
}

// Set stores value under key, evicting the LRU entry if the cache is
// at capacity.
func (l *L1Cache) Set(key string, value []byte) {
	if l == nil || l.c == nil {
		return
	}
	l.c.Add(key, value)
}

// Purge drops every entry. Useful in tests; not called from production
// paths.
func (l *L1Cache) Purge() {
	if l == nil || l.c == nil {
		return
	}
	l.c.Purge()
}

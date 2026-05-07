package middleware

import (
	"hash/fnv"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/time/rate"

	"github.com/0xfandom/chainpulse/internal/monitor"
	"github.com/0xfandom/chainpulse/internal/types"
)

// rateLimiterShards is the count of independent stripes the per-IP
// bucket map is partitioned across. Power of two so the modulo collapses
// to a bit-mask. Lookups fan out across shards to remove the single
// global mutex as a contention point under load.
const rateLimiterShards = 256

// rateLimiterCacheSize bounds how many distinct IPs we keep buckets for
// across all shards combined. 100k entries at ~64 bytes each is ~6 MB —
// fine for a single api node and prevents an unbounded map from a
// misbehaving client.
const rateLimiterCacheSize = 100_000

// rateLimiterShard is one stripe of the limiter map. Owns its own mutex
// and LRU so contended Allow() calls don't serialize.
type rateLimiterShard struct {
	mu      sync.Mutex
	buckets *lru.Cache[string, *rate.Limiter]
}

// RateLimiter is a per-IP token-bucket limiter backed by a sharded LRU
// map. Sharding by fnv32(ip) & (rateLimiterShards - 1) trades a small
// amount of memory for parallel Allow() throughput.
type RateLimiter struct {
	shards [rateLimiterShards]*rateLimiterShard

	rateLimit rate.Limit
	burst     int
}

// NewRateLimiter constructs a RateLimiter from cfg. PerIPPerMinute=0
// disables limiting (Allow always returns true).
func NewRateLimiter(cfg types.APIRateLimitConfig) *RateLimiter {
	rl := &RateLimiter{burst: cfg.Burst}
	if cfg.PerIPPerMinute > 0 {
		rl.rateLimit = rate.Limit(float64(cfg.PerIPPerMinute) / 60.0)
	}
	if rl.burst <= 0 {
		rl.burst = 1
	}

	perShard := rateLimiterCacheSize / rateLimiterShards
	if perShard < 1 {
		perShard = 1
	}
	for i := range rl.shards {
		cache, _ := lru.New[string, *rate.Limiter](perShard)
		rl.shards[i] = &rateLimiterShard{buckets: cache}
	}
	return rl
}

// Allow reports whether ip may proceed. Disabled limiter (rate==0) always
// allows.
func (r *RateLimiter) Allow(ip string) bool {
	if r.rateLimit == 0 {
		return true
	}
	bucket := r.bucketFor(ip)
	return bucket.Allow()
}

// shardFor maps ip to its owning shard via fnv32a. The hash is stable
// per-process; resharding requires a restart.
func (r *RateLimiter) shardFor(ip string) *rateLimiterShard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(ip))
	return r.shards[h.Sum32()&(rateLimiterShards-1)]
}

func (r *RateLimiter) bucketFor(ip string) *rate.Limiter {
	s := r.shardFor(ip)
	s.mu.Lock()
	defer s.mu.Unlock()
	if b, ok := s.buckets.Get(ip); ok {
		return b
	}
	b := rate.NewLimiter(r.rateLimit, r.burst)
	s.buckets.Add(ip, b)
	return b
}

// Middleware returns the Gin handler that enforces the limiter. Rejected
// requests get 429 + JSON {"error":"rate_limited"} and bump
// api_rate_limited_total.
func (r *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := ClientIP(c.Request)
		if !r.Allow(ip) {
			monitor.IncAPIRateLimited()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
			return
		}
		c.Next()
	}
}

// ClientIP returns the request's source IP. Honors X-Forwarded-For first
// hop when present (typical for requests behind a load balancer); falls
// back to RemoteAddr.
func ClientIP(req *http.Request) string {
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		first := strings.SplitN(xff, ",", 2)[0]
		ip := strings.TrimSpace(first)
		if ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}

// Clear empties the limiter (test helper).
func (r *RateLimiter) Clear() {
	for _, s := range r.shards {
		s.mu.Lock()
		s.buckets.Purge()
		s.mu.Unlock()
	}
	_ = time.Now
}

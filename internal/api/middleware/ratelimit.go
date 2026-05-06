package middleware

import (
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

// rateLimiterCacheSize bounds how many distinct IPs we keep buckets for.
// 100k entries at ~64 bytes each is ~6 MB — fine for a single api node
// and prevents an unbounded map from a misbehaving client.
const rateLimiterCacheSize = 100_000

// RateLimiter is a per-IP token-bucket limiter backed by an LRU map.
type RateLimiter struct {
	mu      sync.Mutex
	buckets *lru.Cache[string, *rate.Limiter]

	rateLimit rate.Limit
	burst     int
}

// NewRateLimiter constructs a RateLimiter from cfg. PerIPPerMinute=0
// disables limiting (Allow always returns true).
func NewRateLimiter(cfg types.APIRateLimitConfig) *RateLimiter {
	cache, _ := lru.New[string, *rate.Limiter](rateLimiterCacheSize)
	rl := &RateLimiter{buckets: cache, burst: cfg.Burst}
	if cfg.PerIPPerMinute > 0 {
		rl.rateLimit = rate.Limit(float64(cfg.PerIPPerMinute) / 60.0)
	}
	if rl.burst <= 0 {
		rl.burst = 1
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

func (r *RateLimiter) bucketFor(ip string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok := r.buckets.Get(ip); ok {
		return b
	}
	b := rate.NewLimiter(r.rateLimit, r.burst)
	r.buckets.Add(ip, b)
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
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buckets.Purge()
	_ = time.Now
}

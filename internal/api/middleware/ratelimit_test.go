package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/0xfandom/chainpulse/internal/types"
)

func newRLRouter(cfg types.APIRateLimitConfig) (*gin.Engine, *RateLimiter) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(cfg)
	r := gin.New()
	r.Use(rl.Middleware())
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })
	return r, rl
}

func TestRateLimit_AllowsUnderLimit(t *testing.T) {
	r, _ := newRLRouter(types.APIRateLimitConfig{PerIPPerMinute: 600, Burst: 5})

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("call %d: status = %d", i, w.Code)
		}
	}
}

func TestRateLimit_BurstThenReject(t *testing.T) {
	r, _ := newRLRouter(types.APIRateLimitConfig{PerIPPerMinute: 1, Burst: 1})

	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req1.RemoteAddr = "5.6.7.8:1234"
	r.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Errorf("first req status = %d", w1.Code)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req2.RemoteAddr = "5.6.7.8:1234"
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second req status = %d, want 429", w2.Code)
	}
}

func TestRateLimit_DisabledWhenZero(t *testing.T) {
	r, _ := newRLRouter(types.APIRateLimitConfig{PerIPPerMinute: 0, Burst: 0})

	for i := 0; i < 50; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "9.9.9.9:1234"
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("call %d rejected with disabled limiter", i)
		}
	}
}

func TestClientIP_XForwardedForFirstHop(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	req.RemoteAddr = "127.0.0.1:1234"
	if got := ClientIP(req); got != "10.0.0.1" {
		t.Errorf("ClientIP = %q", got)
	}
}

func TestClientIP_FallbackRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	if got := ClientIP(req); got != "127.0.0.1" {
		t.Errorf("ClientIP = %q", got)
	}
}

func TestRateLimit_DistinctIPsHitDistinctShards(t *testing.T) {
	rl := NewRateLimiter(types.APIRateLimitConfig{PerIPPerMinute: 600, Burst: 1})
	seen := map[*rateLimiterShard]struct{}{}
	for i := 0; i < 1024; i++ {
		ip := "10.0." + itoa(i>>8) + "." + itoa(i&0xff)
		seen[rl.shardFor(ip)] = struct{}{}
	}
	if len(seen) < rateLimiterShards/2 {
		t.Errorf("only %d/%d shards exercised over 1024 IPs — fnv distribution looks broken", len(seen), rateLimiterShards)
	}
}

func TestRateLimit_PerShardIsolation(t *testing.T) {
	rl := NewRateLimiter(types.APIRateLimitConfig{PerIPPerMinute: 1, Burst: 1})
	// Pick two IPs whose hashes land on different shards. fnv32a is
	// deterministic, so this loop terminates quickly.
	var a, b string
	for i := 0; i < 1024; i++ {
		cand := "10.0.0." + itoa(i)
		if rl.shardFor(cand) != rl.shardFor("1.1.1.1") {
			a, b = "1.1.1.1", cand
			break
		}
	}
	if a == "" || b == "" {
		t.Fatal("could not find two IPs in distinct shards")
	}
	if !rl.Allow(a) || !rl.Allow(b) {
		t.Fatal("first call from each shard should allow")
	}
	if rl.Allow(a) {
		t.Errorf("second call from %s should be rate-limited", a)
	}
	if rl.Allow(b) {
		t.Errorf("second call from %s should be rate-limited", b)
	}
}

// itoa avoids importing strconv just for tests.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [4]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

func BenchmarkRateLimit_Allow(b *testing.B) {
	rl := NewRateLimiter(types.APIRateLimitConfig{PerIPPerMinute: 1_000_000, Burst: 1_000_000})
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ip := "10.0." + itoa((i>>8)&0xff) + "." + itoa(i&0xff)
			rl.Allow(ip)
			i++
		}
	})
}

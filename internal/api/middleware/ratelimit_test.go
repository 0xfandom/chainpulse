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

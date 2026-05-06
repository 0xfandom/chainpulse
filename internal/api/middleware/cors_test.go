package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/0xfandom/chainpulse/internal/types"
)

func newCORSRouter(cfg types.APICORSConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(cfg))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })
	return r
}

func TestCORS_AllowedOriginEchoes(t *testing.T) {
	r := newCORSRouter(types.APICORSConfig{AllowedOrigins: []string{"https://app.example.com"}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://app.example.com")
	r.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("ACAO = %q", got)
	}
}

func TestCORS_DisallowedOriginNoHeader(t *testing.T) {
	r := newCORSRouter(types.APICORSConfig{AllowedOrigins: []string{"https://app.example.com"}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	r.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO unexpectedly set: %q", got)
	}
}

func TestCORS_PreflightReturns204(t *testing.T) {
	r := newCORSRouter(types.APICORSConfig{AllowedOrigins: []string{"*"}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	req.Header.Set("Origin", "https://anywhere")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Allow-Methods missing")
	}
}

func TestCORS_WildcardWithCredentialsRequiresExact(t *testing.T) {
	r := newCORSRouter(types.APICORSConfig{
		AllowedOrigins:   []string{"*", "https://trusted"},
		AllowCredentials: true,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://trusted")
	r.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://trusted" {
		t.Errorf("ACAO = %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("ACAC = %q", got)
	}

	// Untrusted origin — must not echo even with wildcard present, since
	// credentials are on.
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req2.Header.Set("Origin", "https://untrusted")
	r.ServeHTTP(w2, req2)
	if got := w2.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO leaked: %q", got)
	}
}

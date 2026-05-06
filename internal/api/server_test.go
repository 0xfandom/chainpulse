package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xfandom/chainpulse/internal/types"
)

func TestServer_HealthAndMetrics(t *testing.T) {
	cfg := types.APIConfig{
		Addr:      ":0",
		WebSocket: types.APIWSConfig{OriginCheck: "strict"},
	}
	s := NewServer(cfg, nil, nil, "test")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	s.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("health status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Errorf("health body = %s", w.Body.String())
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	s.Router().ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("metrics status = %d", w2.Code)
	}
}

func TestServer_V1GroupExists(t *testing.T) {
	cfg := types.APIConfig{Addr: ":0", WebSocket: types.APIWSConfig{OriginCheck: "strict"}}
	s := NewServer(cfg, nil, nil, "test")
	if g := s.V1Group(); g == nil {
		t.Fatal("V1Group should not be nil")
	}
}

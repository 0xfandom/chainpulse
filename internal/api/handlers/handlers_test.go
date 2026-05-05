package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/0xfandom/chainpulse/internal/api/store"
)

// fakeRows mirrors the one in store_test for handler tests.
type fakeRows struct{}

func (f *fakeRows) Next() bool                       { return false }
func (f *fakeRows) Scan(...any) error                { return nil }
func (f *fakeRows) ScanStruct(any) error             { return nil }
func (f *fakeRows) ColumnTypes() []driver.ColumnType { return nil }
func (f *fakeRows) Totals(...any) error              { return nil }
func (f *fakeRows) Columns() []string                { return nil }
func (f *fakeRows) Close() error                     { return nil }
func (f *fakeRows) Err() error                       { return nil }
func (f *fakeRows) HasData() bool                    { return false }

type fakeQueryConn struct {
	queryErr error
}

func (f *fakeQueryConn) Query(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return &fakeRows{}, nil
}
func (f *fakeQueryConn) Ping(context.Context) error { return nil }
func (f *fakeQueryConn) Close() error               { return nil }

func newHandlerHarness(t *testing.T, conn *fakeQueryConn) (*gin.Engine, *miniredis.Miniredis) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	cache := store.NewReadCache(rc, time.Minute)
	rs := store.NewReadStore(conn)

	deps := HandlerDeps{Store: rs, Cache: cache}
	r := gin.New()
	g := r.Group("/v1")
	NewWalletHandlers(deps).Register(g)
	NewTokenHandlers(deps).Register(g)
	NewProtocolHandlers(deps).Register(g)
	NewChainHandlers(deps).Register(g)
	return r, mr
}

func TestNormalizeAddress(t *testing.T) {
	if _, err := normalizeAddress("not-an-addr"); err == nil {
		t.Error("expected bad-format error")
	}
	got, err := normalizeAddress("0xAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAa")
	if err != nil {
		t.Fatal(err)
	}
	if got != strings.ToLower("0xAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAa") {
		t.Errorf("normalize = %q", got)
	}
}

func TestClampLimit(t *testing.T) {
	if got := clampLimit("", 50, 100); got != 50 {
		t.Errorf("default = %d", got)
	}
	if got := clampLimit("9999", 50, 100); got != 100 {
		t.Errorf("clamp = %d", got)
	}
	if got := clampLimit("0", 50, 100); got != 50 {
		t.Errorf("zero -> default = %d", got)
	}
	if got := clampLimit("nope", 50, 100); got != 50 {
		t.Errorf("garbage -> default = %d", got)
	}
}

func TestParseChainID(t *testing.T) {
	if v, ok := parseChainID(""); !ok || v != 0 {
		t.Errorf("absent: v=%d ok=%v", v, ok)
	}
	if v, ok := parseChainID("8453"); !ok || v != 8453 {
		t.Errorf("valid: v=%d ok=%v", v, ok)
	}
	if _, ok := parseChainID("nope"); ok {
		t.Error("garbage parse should fail")
	}
}

func TestWalletPositions_BadAddress(t *testing.T) {
	r, _ := newHandlerHarness(t, &fakeQueryConn{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/wallet/not-an-addr/positions", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestWalletPositions_OK(t *testing.T) {
	r, _ := newHandlerHarness(t, &fakeQueryConn{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/wallet/0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/positions", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
}

func TestWalletPositions_StoreError(t *testing.T) {
	r, _ := newHandlerHarness(t, &fakeQueryConn{queryErr: errors.New("ch down")})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/wallet/0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/positions", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestWalletBalances_RequiresChainID(t *testing.T) {
	r, _ := newHandlerHarness(t, &fakeQueryConn{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/wallet/0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/balances", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestWalletBalances_FromCache(t *testing.T) {
	r, mr := newHandlerHarness(t, &fakeQueryConn{})

	wallet := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	mr.HSet("balance:"+wallet+":1:0xtoken1", "amount", "1234")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/wallet/"+wallet+"/balances?chain_id=1", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "1234") {
		t.Errorf("body missing balance: %s", w.Body.String())
	}
}

func TestWalletHistory_LimitClamp(t *testing.T) {
	r, _ := newHandlerHarness(t, &fakeQueryConn{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/wallet/0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/history?limit=99999", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	// limit clamped to 500 in body
	if !strings.Contains(w.Body.String(), `"limit":500`) {
		t.Errorf("limit not clamped: %s", w.Body.String())
	}
}

func TestTokenTransfers_BadAddress(t *testing.T) {
	r, _ := newHandlerHarness(t, &fakeQueryConn{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/token/not-addr/transfers", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestTokenTransfers_OK(t *testing.T) {
	r, _ := newHandlerHarness(t, &fakeQueryConn{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/token/0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/transfers", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
}

func TestProtocolStats_UnknownProtocol404(t *testing.T) {
	r, _ := newHandlerHarness(t, &fakeQueryConn{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/protocol/morpho/stats", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d", w.Code)
	}
}

func TestProtocolStats_KnownOK(t *testing.T) {
	r, _ := newHandlerHarness(t, &fakeQueryConn{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/protocol/aave_v3/stats", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
}

func TestChainBlocks_BadID(t *testing.T) {
	r, _ := newHandlerHarness(t, &fakeQueryConn{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/chain/0/blocks", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/v1/chain/abc/blocks", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w2.Code)
	}
}

func TestChainBlocks_OK(t *testing.T) {
	r, _ := newHandlerHarness(t, &fakeQueryConn{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/chain/8453/blocks", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
}

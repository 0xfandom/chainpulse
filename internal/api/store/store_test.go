package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// fakeRows implements driver.Rows just enough for unit tests.
type fakeRows struct{ closed bool }

func (f *fakeRows) Next() bool                       { return false }
func (f *fakeRows) Scan(...any) error                { return nil }
func (f *fakeRows) ScanStruct(any) error             { return nil }
func (f *fakeRows) ColumnTypes() []driver.ColumnType { return nil }
func (f *fakeRows) Totals(...any) error              { return nil }
func (f *fakeRows) Columns() []string                { return nil }
func (f *fakeRows) Close() error                     { f.closed = true; return nil }
func (f *fakeRows) Err() error                       { return nil }
func (f *fakeRows) HasData() bool                    { return false }

// fakeQueryConn captures the last query + args for assertions.
type fakeQueryConn struct {
	gotQuery string
	gotArgs  []any
	queryErr error
}

func (f *fakeQueryConn) Query(_ context.Context, query string, args ...any) (driver.Rows, error) {
	f.gotQuery = query
	f.gotArgs = args
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return &fakeRows{}, nil
}
func (f *fakeQueryConn) Ping(context.Context) error { return nil }
func (f *fakeQueryConn) Close() error               { return nil }

func TestReadStore_QueryConstants(t *testing.T) {
	for name, q := range map[string]string{
		"WalletHistory":       sqlWalletHistory,
		"WalletDefiPositions": sqlWalletDefiPositions,
		"TokenTransfers":      sqlTokenTransfers,
		"ProtocolStats":       sqlProtocolStats,
		"ChainRecentBlocks":   sqlChainRecentBlocks,
	} {
		if !strings.Contains(strings.ToUpper(q), "SELECT") {
			t.Errorf("%s missing SELECT", name)
		}
		if strings.Count(q, "?") == 0 {
			t.Errorf("%s missing parameter placeholders", name)
		}
	}
}

func TestReadStore_RoutesToConn(t *testing.T) {
	conn := &fakeQueryConn{}
	s := NewReadStore(conn)

	cases := []struct {
		name string
		fn   func() error
		want string
	}{
		{"WalletHistory", func() error {
			_, err := s.WalletHistory(context.Background(), "0xa", 50)
			return err
		}, sqlWalletHistory},
		{"WalletDefiPositions", func() error {
			_, err := s.WalletDefiPositions(context.Background(), "0xa", 50)
			return err
		}, sqlWalletDefiPositions},
		{"TokenTransfers", func() error {
			_, err := s.TokenTransfers(context.Background(), "0xt", 50)
			return err
		}, sqlTokenTransfers},
		{"ProtocolStats", func() error {
			_, err := s.ProtocolStats(context.Background(), "aave_v3")
			return err
		}, sqlProtocolStats},
		{"ChainRecentBlocks", func() error {
			_, err := s.ChainRecentBlocks(context.Background(), 8453, 50)
			return err
		}, sqlChainRecentBlocks},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conn.gotQuery = ""
			if err := tc.fn(); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if conn.gotQuery != tc.want {
				t.Errorf("%s sent wrong query", tc.name)
			}
		})
	}
}

func TestReadStore_QueryError(t *testing.T) {
	conn := &fakeQueryConn{queryErr: errors.New("conn closed")}
	s := NewReadStore(conn)
	if _, err := s.WalletHistory(context.Background(), "0xa", 10); err == nil {
		t.Error("expected error to propagate")
	}
}

func TestReadStore_Close(t *testing.T) {
	s := NewReadStore(&fakeQueryConn{})
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	var nilStore *ReadStore
	if err := nilStore.Close(); err != nil {
		t.Errorf("nil close: %v", err)
	}
}

func newTestCache(t *testing.T) (*ReadCache, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewReadCache(client, time.Minute), mr
}

func TestReadCache_GetBytesMiss(t *testing.T) {
	c, _ := newTestCache(t)
	v, hit, err := c.GetBytes(context.Background(), "missing")
	if err != nil || hit || v != nil {
		t.Errorf("expected clean miss, got v=%v hit=%v err=%v", v, hit, err)
	}
}

func TestReadCache_SetGetBytes(t *testing.T) {
	c, _ := newTestCache(t)
	if err := c.SetBytes(context.Background(), "k", []byte("hello"), 0); err != nil {
		t.Fatal(err)
	}
	v, hit, err := c.GetBytes(context.Background(), "k")
	if err != nil {
		t.Fatal(err)
	}
	if !hit || string(v) != "hello" {
		t.Errorf("hit=%v v=%q", hit, v)
	}
}

func TestReadCache_GetBalances(t *testing.T) {
	c, mr := newTestCache(t)
	wallet := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	mr.HSet("balance:"+wallet+":1:0xtoken1", "amount", "100")
	mr.HSet("balance:"+wallet+":1:0xtoken2", "amount", "250")
	// unrelated chain — must not appear
	mr.HSet("balance:"+wallet+":2:0xtoken1", "amount", "999")

	got, err := c.GetBalances(context.Background(), wallet, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got["0xtoken1"] != "100" || got["0xtoken2"] != "250" {
		t.Errorf("got = %v", got)
	}
	if _, ok := got["0xtoken1_chain2"]; ok {
		t.Error("should not include chain 2 entries")
	}
}

func TestAside_HitPath(t *testing.T) {
	c, _ := newTestCache(t)
	if err := c.SetBytes(context.Background(), "k", []byte(`{"v":42}`), time.Minute); err != nil {
		t.Fatal(err)
	}
	type payload struct {
		V int `json:"v"`
	}
	got, hit, err := Aside(context.Background(), c, "k", time.Minute, func(ctx context.Context) (payload, error) {
		t.Fatal("fallback should not run on hit")
		return payload{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hit || got.V != 42 {
		t.Errorf("hit=%v v=%v", hit, got)
	}
}

func TestAside_MissThenFill(t *testing.T) {
	c, mr := newTestCache(t)
	type payload struct {
		V int `json:"v"`
	}
	got, hit, err := Aside(context.Background(), c, "k", time.Minute, func(ctx context.Context) (payload, error) {
		return payload{V: 7}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if hit || got.V != 7 {
		t.Errorf("expected miss + fill, got hit=%v v=%v", hit, got)
	}
	if !mr.Exists("k") {
		t.Error("cache should have been filled")
	}
}

func TestAside_FallbackError(t *testing.T) {
	c, _ := newTestCache(t)
	type payload struct{ V int }
	_, _, err := Aside(context.Background(), c, "k", time.Minute, func(ctx context.Context) (payload, error) {
		return payload{}, errors.New("upstream down")
	})
	if err == nil {
		t.Fatal("expected fallback error")
	}
}

func TestAside_NilCache(t *testing.T) {
	type payload struct{ V int }
	got, hit, err := Aside[payload](context.Background(), nil, "k", time.Minute, func(ctx context.Context) (payload, error) {
		return payload{V: 11}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if hit || got.V != 11 {
		t.Errorf("hit=%v v=%v", hit, got)
	}
}

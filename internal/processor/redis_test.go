package processor

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"

	"github.com/0xfandom/chainpulse/internal/types"
)

func newTestWriter(t *testing.T) (*CacheWriter, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return newCacheWriter(client, time.Minute), mr
}

func TestCacheWriter_NewValidates(t *testing.T) {
	if _, err := NewCacheWriter(context.Background(), CacheWriterConfig{}); err == nil {
		t.Error("expected error for empty addr")
	}
	if _, err := NewCacheWriter(context.Background(), CacheWriterConfig{Addr: "x"}); err == nil {
		t.Error("expected error for zero TTL")
	}
}

func TestCacheWriter_IncrBalance(t *testing.T) {
	w, mr := newTestWriter(t)
	wallet := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	token := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	if err := w.IncrBalance(context.Background(), 8453, wallet, token, big.NewInt(100)); err != nil {
		t.Fatal(err)
	}
	if err := w.IncrBalance(context.Background(), 8453, wallet, token, big.NewInt(-30)); err != nil {
		t.Fatal(err)
	}
	got := mr.HGet(balanceKey(wallet, 8453, token), "amount")
	if got != "70" {
		t.Errorf("balance = %q, want 70", got)
	}
}

func TestCacheWriter_IncrBalance_ZeroNoop(t *testing.T) {
	w, mr := newTestWriter(t)
	wallet := common.HexToAddress("0xa")
	token := common.HexToAddress("0xb")
	if err := w.IncrBalance(context.Background(), 1, wallet, token, big.NewInt(0)); err != nil {
		t.Fatal(err)
	}
	keys := mr.Keys()
	if len(keys) != 0 {
		t.Errorf("zero delta should not create keys, got %v", keys)
	}
}

func TestCacheWriter_IncrBalance_ClampsLargeValue(t *testing.T) {
	w, mr := newTestWriter(t)
	wallet := common.HexToAddress("0xa")
	token := common.HexToAddress("0xb")
	huge := new(big.Int).Lsh(big.NewInt(1), 80) // 2^80
	if err := w.IncrBalance(context.Background(), 1, wallet, token, huge); err != nil {
		t.Fatal(err)
	}
	got := mr.HGet(balanceKey(wallet, 1, token), "amount")
	if got == "" {
		t.Error("expected clamped value to land in the hash")
	}
}

func TestCacheWriter_SetPosition(t *testing.T) {
	w, mr := newTestWriter(t)
	wallet := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	token := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	pos := &types.WalletPosition{
		Wallet:       wallet,
		ChainID:      8453,
		Protocol:     "aave_v3",
		PositionType: "borrow",
		Token:        token,
		Amount:       big.NewInt(1500),
		ValueUSD:     1500.0,
		UpdatedAt:    time.Now(),
		BlockNumber:  100,
	}
	if err := w.SetPosition(context.Background(), pos); err != nil {
		t.Fatal(err)
	}
	key := positionKey(wallet, 8453, "aave_v3", "borrow", token)
	if !mr.Exists(key) {
		t.Fatalf("position key %q not set", key)
	}
	if got := mr.TTL(key); got <= 0 || got > time.Minute {
		t.Errorf("ttl out of range: %v", got)
	}
}

func TestCacheWriter_SetPosition_NilErrors(t *testing.T) {
	w, _ := newTestWriter(t)
	if err := w.SetPosition(context.Background(), nil); err == nil {
		t.Error("expected error for nil position")
	}
}

func TestCacheWriter_InvalidatePositions(t *testing.T) {
	w, mr := newTestWriter(t)
	wallet := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	tokenA := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	tokenB := common.HexToAddress("0xcccccccccccccccccccccccccccccccccccccccc")

	for _, tk := range []common.Address{tokenA, tokenB} {
		_ = w.SetPosition(context.Background(), &types.WalletPosition{
			Wallet: wallet, ChainID: 1, Protocol: "aave_v3", PositionType: "supply", Token: tk,
		})
	}
	// unrelated wallet — must survive
	_ = w.SetPosition(context.Background(), &types.WalletPosition{
		Wallet:  common.HexToAddress("0xdddddddddddddddddddddddddddddddddddddddd"),
		ChainID: 1, Protocol: "aave_v3", PositionType: "supply", Token: tokenA,
	})

	if err := w.InvalidatePositions(context.Background(), wallet); err != nil {
		t.Fatal(err)
	}
	for _, k := range mr.Keys() {
		if strings.HasPrefix(k, "position:"+strings.ToLower(wallet.Hex())) {
			t.Errorf("key %q should have been deleted", k)
		}
	}
	other := positionKey(common.HexToAddress("0xdddddddddddddddddddddddddddddddddddddddd"), 1, "aave_v3", "supply", tokenA)
	if !mr.Exists(other) {
		t.Errorf("unrelated wallet position should remain at %q", other)
	}
}

func TestClampInt64(t *testing.T) {
	if v, c := clampInt64(big.NewInt(5)); v != 5 || c {
		t.Errorf("normal int = %d clamped=%v", v, c)
	}
	huge := new(big.Int).Lsh(big.NewInt(1), 80)
	if _, c := clampInt64(huge); !c {
		t.Error("expected clamp for huge positive")
	}
	negHuge := new(big.Int).Neg(huge)
	if _, c := clampInt64(negHuge); !c {
		t.Error("expected clamp for huge negative")
	}
}

func TestCacheWriter_Close(t *testing.T) {
	w, _ := newTestWriter(t)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var nilW *CacheWriter
	if err := nilW.Close(); err != nil {
		t.Errorf("nil close: %v", err)
	}
}

package processor

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

type fakeBatch struct {
	calls []string
	err   error
}

func (f *fakeBatch) Enqueue(_ context.Context, e *types.DecodedEvent) error {
	f.calls = append(f.calls, e.Protocol+":"+e.EventType)
	return f.err
}

type fakeCache struct {
	incrs     [][3]string // wallet,token,delta
	positions []*types.WalletPosition
	err       error
}

func (f *fakeCache) IncrBalance(_ context.Context, _ uint64, wallet, token common.Address, delta *big.Int) error {
	if f.err != nil {
		return f.err
	}
	f.incrs = append(f.incrs, [3]string{wallet.Hex(), token.Hex(), delta.String()})
	return nil
}

func (f *fakeCache) SetPosition(_ context.Context, p *types.WalletPosition) error {
	if f.err != nil {
		return f.err
	}
	f.positions = append(f.positions, p)
	return nil
}

type fakeProd struct {
	decoded []*types.DecodedEvent
	pos     []*types.WalletPosition
	err     error
}

func (f *fakeProd) PublishDecoded(_ context.Context, e *types.DecodedEvent) error {
	if f.err != nil {
		return f.err
	}
	f.decoded = append(f.decoded, e)
	return nil
}

func (f *fakeProd) PublishPositionUpdate(_ context.Context, p *types.WalletPosition) error {
	if f.err != nil {
		return f.err
	}
	f.pos = append(f.pos, p)
	return nil
}

func newAggHarness(t *testing.T) (*Aggregator, *fakeBatch, *fakeCache, *fakeProd) {
	t.Helper()
	b, c, p := &fakeBatch{}, &fakeCache{}, &fakeProd{}
	agg, err := NewAggregator(b, c, p)
	if err != nil {
		t.Fatal(err)
	}
	return agg, b, c, p
}

func TestNewAggregator_Validation(t *testing.T) {
	if _, err := NewAggregator(nil, &fakeCache{}, &fakeProd{}); err == nil {
		t.Error("expected nil batch error")
	}
	if _, err := NewAggregator(&fakeBatch{}, nil, &fakeProd{}); err == nil {
		t.Error("expected nil cache error")
	}
	if _, err := NewAggregator(&fakeBatch{}, &fakeCache{}, nil); err == nil {
		t.Error("expected nil producer error")
	}
}

func TestAggregator_Transfer(t *testing.T) {
	agg, b, c, p := newAggHarness(t)
	ev := &types.DecodedEvent{
		ChainEvent: types.ChainEvent{ChainID: 8453},
		Protocol:   "erc20",
		EventType:  "transfer",
		Params: map[string]interface{}{
			"from":   "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"to":     "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"amount": "100",
			"token":  "0xcccccccccccccccccccccccccccccccccccccccc",
		},
	}
	if err := agg.Process(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	if len(b.calls) != 1 {
		t.Errorf("batch enqueue calls = %d", len(b.calls))
	}
	if len(c.incrs) != 2 {
		t.Errorf("balance updates = %d, want 2", len(c.incrs))
	}
	if c.incrs[0][2] != "-100" || c.incrs[1][2] != "100" {
		t.Errorf("deltas = %v / %v", c.incrs[0][2], c.incrs[1][2])
	}
	if len(p.decoded) != 1 || len(p.pos) != 0 {
		t.Errorf("publishes = %d decoded / %d positions", len(p.decoded), len(p.pos))
	}
}

func TestAggregator_Approval(t *testing.T) {
	agg, b, c, p := newAggHarness(t)
	ev := &types.DecodedEvent{
		Protocol:  "erc20",
		EventType: "approval",
		Params:    map[string]interface{}{"owner": "0xa", "spender": "0xb", "amount": "10", "token": "0xt"},
	}
	if err := agg.Process(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	if len(b.calls) != 0 {
		t.Errorf("approvals should not enqueue, got %d", len(b.calls))
	}
	if len(c.incrs) != 0 || len(c.positions) != 0 {
		t.Error("approvals should not touch cache")
	}
	if len(p.decoded) != 1 {
		t.Errorf("approvals should still publish decoded, got %d", len(p.decoded))
	}
}

func TestAggregator_AaveBorrow(t *testing.T) {
	agg, b, c, p := newAggHarness(t)
	ev := &types.DecodedEvent{
		ChainEvent: types.ChainEvent{ChainID: 1, BlockNumber: 100},
		Protocol:   "aave_v3",
		EventType:  "borrow",
		Params: map[string]interface{}{
			"reserve": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			"user":    "0xcccccccccccccccccccccccccccccccccccccccc",
			"amount":  "1500",
		},
	}
	if err := agg.Process(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	if len(b.calls) != 1 {
		t.Errorf("batch calls = %d", len(b.calls))
	}
	if len(c.positions) != 1 {
		t.Errorf("set position calls = %d, want 1", len(c.positions))
	}
	if c.positions[0].PositionType != "borrow" {
		t.Errorf("position type = %q", c.positions[0].PositionType)
	}
	if len(p.decoded) != 1 || len(p.pos) != 1 {
		t.Errorf("publishes = %d/%d", len(p.decoded), len(p.pos))
	}
}

func TestAggregator_CompoundAbsorb(t *testing.T) {
	agg, b, c, p := newAggHarness(t)
	ev := &types.DecodedEvent{
		ChainEvent: types.ChainEvent{ChainID: 1, BlockNumber: 5},
		Protocol:   "compound_v3",
		EventType:  "absorb",
		Params: map[string]interface{}{
			"absorber":            "0xabsorber",
			"borrower":            "0xdddddddddddddddddddddddddddddddddddddddd",
			"asset":               "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			"collateral_absorbed": "5000",
			"usd_value":           "1500",
		},
	}
	if err := agg.Process(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	if len(b.calls) != 1 {
		t.Error("batch enqueue missing")
	}
	if len(c.positions) != 1 {
		t.Errorf("set position calls = %d", len(c.positions))
	}
	if c.positions[0].PositionType != "absorb" {
		t.Errorf("position type = %q", c.positions[0].PositionType)
	}
	if len(p.pos) != 1 {
		t.Errorf("position publishes = %d", len(p.pos))
	}
}

func TestAggregator_UniswapSwap(t *testing.T) {
	agg, b, c, p := newAggHarness(t)
	ev := &types.DecodedEvent{
		Protocol:  "uniswap_v3",
		EventType: "swap",
		Params:    map[string]interface{}{"sender": "0xa", "recipient": "0xb", "pool": "0xp"},
	}
	if err := agg.Process(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	if len(b.calls) != 1 || len(p.decoded) != 1 {
		t.Errorf("expected 1 enqueue + 1 publish, got %d / %d", len(b.calls), len(p.decoded))
	}
	if len(c.positions) != 0 || len(p.pos) != 0 {
		t.Error("uniswap should not produce positions in v1")
	}
}

func TestAggregator_UnknownProtocolMetric(t *testing.T) {
	agg, b, c, p := newAggHarness(t)
	ev := &types.DecodedEvent{Protocol: "morpho", EventType: "supply"}
	if err := agg.Process(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	if len(b.calls) != 0 || len(c.positions) != 0 || len(p.decoded) != 0 {
		t.Error("unknown protocol should be a noop other than the metric")
	}
}

func TestAggregator_TransferBadAmount(t *testing.T) {
	agg, _, _, _ := newAggHarness(t)
	ev := &types.DecodedEvent{
		Protocol:  "erc20",
		EventType: "transfer",
		Params: map[string]interface{}{
			"from": "0xa", "to": "0xb", "amount": "not-a-number", "token": "0xt",
		},
	}
	if err := agg.Process(context.Background(), ev); err == nil {
		t.Fatal("expected error for non-numeric amount")
	}
}

func TestAggregator_PropagatesBatchError(t *testing.T) {
	b := &fakeBatch{err: errors.New("boom")}
	agg, err := NewAggregator(b, &fakeCache{}, &fakeProd{})
	if err != nil {
		t.Fatal(err)
	}
	ev := &types.DecodedEvent{
		Protocol: "erc20", EventType: "transfer",
		Params: map[string]interface{}{"from": "0xa", "to": "0xb", "amount": "1", "token": "0xt"},
	}
	if err := agg.Process(context.Background(), ev); err == nil {
		t.Error("expected error to propagate")
	}
}

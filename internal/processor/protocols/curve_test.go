package protocols

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

var (
	// Curve 3pool on Ethereum mainnet
	curve3pool = common.HexToAddress("0xbEbc44782C7dB0a1A60Cb6fe97d0b483032FF1C7")
	curveBuyer = common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
)

func TestCurve_TokenExchange(t *testing.T) {
	d := NewCurveDecoder()

	// sold_id=0 (DAI), tokens_sold=1000e18, bought_id=1 (USDC), tokens_bought=999e6
	soldID := big.NewInt(0)
	tokensSold := new(big.Int).Mul(big.NewInt(1000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	boughtID := big.NewInt(1)
	tokensBought := new(big.Int).Mul(big.NewInt(999), new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil))

	args := curveABI.Events["TokenExchange"].Inputs.NonIndexed()
	data, err := args.Pack(soldID, tokensSold, boughtID, tokensBought)
	if err != nil {
		t.Fatal(err)
	}

	ev := &types.ChainEvent{
		Contract: curve3pool,
		RawTopics: []common.Hash{
			CurveTokenExchangeSig,
			addrTopic(curveBuyer),
		},
		RawData: data,
	}

	got, err := d.Decode(ev)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Protocol != "curve" {
		t.Errorf("Protocol = %q, want %q", got.Protocol, "curve")
	}
	if got.EventType != "swap" {
		t.Errorf("EventType = %q, want %q", got.EventType, "swap")
	}
	if got.Params["buyer"].(string) != curveBuyer.Hex() {
		t.Errorf("buyer = %v, want %v", got.Params["buyer"], curveBuyer.Hex())
	}
	if got.Params["sold_id"].(string) != "0" {
		t.Errorf("sold_id = %v, want 0", got.Params["sold_id"])
	}
	if got.Params["tokens_sold"].(string) != tokensSold.String() {
		t.Errorf("tokens_sold = %v, want %v", got.Params["tokens_sold"], tokensSold.String())
	}
	if got.Params["bought_id"].(string) != "1" {
		t.Errorf("bought_id = %v, want 1", got.Params["bought_id"])
	}
	if got.Params["tokens_bought"].(string) != tokensBought.String() {
		t.Errorf("tokens_bought = %v, want %v", got.Params["tokens_bought"], tokensBought.String())
	}
}

func TestCurve_BasicSupport(t *testing.T) {
	d := NewCurveDecoder()
	if d.Name() != "curve" {
		t.Errorf("Name = %q", d.Name())
	}
	for _, sig := range d.SupportedEvents() {
		if !d.CanDecode(sig) {
			t.Errorf("CanDecode rejected %s", sig.Hex())
		}
	}
	if d.CanDecode(common.HexToHash("0xdead000000000000000000000000000000000000000000000000000000000000")) {
		t.Error("CanDecode should reject unknown sig")
	}
}

func TestCurve_Malformed(t *testing.T) {
	d := NewCurveDecoder()

	// nil event
	if _, err := d.Decode(nil); err == nil {
		t.Error("expected error for nil event")
	}
	// no topics
	if _, err := d.Decode(&types.ChainEvent{}); err == nil {
		t.Error("expected error for empty topics")
	}
	// only topic[0] — buyer topic[1] is missing
	if _, err := d.Decode(&types.ChainEvent{
		RawTopics: []common.Hash{CurveTokenExchangeSig},
	}); err == nil {
		t.Error("expected error for too-few topics")
	}
	// wrong topic[0]
	if _, err := d.Decode(&types.ChainEvent{
		RawTopics: []common.Hash{
			common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000"),
			addrTopic(curveBuyer),
		},
		RawData: make([]byte, 128),
	}); err == nil {
		t.Error("expected error for unsupported sig")
	}
}

package protocols

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

var (
	usdcReserve  = common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48") // USDC mainnet
	wethDebt     = common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2") // WETH mainnet
	userA        = common.HexToAddress("0xaaaaAAaaaaAaAaAaaaaAaaaaaAaAAAaaaaAaAAAa")
	userB        = common.HexToAddress("0xbBBBBBBBbbBBBbbBBBBbBbBBBBbBBBbBBBBbbBBb")
	liquidatorAd = common.HexToAddress("0xCCCccCcccCcCcCcCccCcCcCccCCcCCcccCcccCcc")
)

// uint16Topic right-aligns a uint16 referral code into a 32-byte topic.
func uint16Topic(v uint16) common.Hash {
	var h common.Hash
	h[30] = byte(v >> 8)
	h[31] = byte(v)
	return h
}

func TestAaveV3_Supply(t *testing.T) {
	d := NewAaveV3Decoder()
	args := aaveV3ABI.Events["Supply"].Inputs.NonIndexed()
	data, err := args.Pack(userA, big.NewInt(1_000_000))
	if err != nil {
		t.Fatal(err)
	}
	ev := &types.ChainEvent{
		RawTopics: []common.Hash{
			AaveV3SupplySig,
			addrTopic(usdcReserve),
			addrTopic(userB),
			uint16Topic(123),
		},
		RawData: data,
	}
	got, err := d.Decode(ev)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Protocol != "aave_v3" || got.EventType != "supply" {
		t.Errorf("proto/type = %q/%q", got.Protocol, got.EventType)
	}
	if got.Params["amount"].(string) != "1000000" {
		t.Errorf("amount = %v", got.Params["amount"])
	}
	if got.Params["reserve"].(string) != usdcReserve.Hex() {
		t.Errorf("reserve = %v", got.Params["reserve"])
	}
	if got.Params["referral_code"].(string) != "123" {
		t.Errorf("referral_code = %v", got.Params["referral_code"])
	}
	if got.Params["user"].(string) != userA.Hex() {
		t.Errorf("user = %v", got.Params["user"])
	}
}

func TestAaveV3_Withdraw(t *testing.T) {
	d := NewAaveV3Decoder()
	args := aaveV3ABI.Events["Withdraw"].Inputs.NonIndexed()
	data, err := args.Pack(big.NewInt(500))
	if err != nil {
		t.Fatal(err)
	}
	ev := &types.ChainEvent{
		RawTopics: []common.Hash{
			AaveV3WithdrawSig,
			addrTopic(usdcReserve),
			addrTopic(userA),
			addrTopic(userB),
		},
		RawData: data,
	}
	got, err := d.Decode(ev)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.EventType != "withdraw" {
		t.Errorf("type = %q", got.EventType)
	}
	if got.Params["to"].(string) != userB.Hex() {
		t.Errorf("to = %v", got.Params["to"])
	}
	if got.Params["amount"].(string) != "500" {
		t.Errorf("amount = %v", got.Params["amount"])
	}
}

func TestAaveV3_Borrow(t *testing.T) {
	d := NewAaveV3Decoder()
	args := aaveV3ABI.Events["Borrow"].Inputs.NonIndexed()
	data, err := args.Pack(userA, big.NewInt(2_000), uint8(2), big.NewInt(456))
	if err != nil {
		t.Fatal(err)
	}
	ev := &types.ChainEvent{
		RawTopics: []common.Hash{
			AaveV3BorrowSig,
			addrTopic(wethDebt),
			addrTopic(userB),
			uint16Topic(0),
		},
		RawData: data,
	}
	got, err := d.Decode(ev)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.EventType != "borrow" {
		t.Errorf("type = %q", got.EventType)
	}
	if got.Params["interest_rate_mode"].(uint64) != 2 {
		t.Errorf("rate mode = %v", got.Params["interest_rate_mode"])
	}
	if got.Params["borrow_rate"].(string) != "456" {
		t.Errorf("borrow_rate = %v", got.Params["borrow_rate"])
	}
}

func TestAaveV3_Repay(t *testing.T) {
	d := NewAaveV3Decoder()
	args := aaveV3ABI.Events["Repay"].Inputs.NonIndexed()
	data, err := args.Pack(big.NewInt(99), true)
	if err != nil {
		t.Fatal(err)
	}
	ev := &types.ChainEvent{
		RawTopics: []common.Hash{
			AaveV3RepaySig,
			addrTopic(wethDebt),
			addrTopic(userA),
			addrTopic(userB),
		},
		RawData: data,
	}
	got, err := d.Decode(ev)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.EventType != "repay" {
		t.Errorf("type = %q", got.EventType)
	}
	if got.Params["use_atokens"].(bool) != true {
		t.Errorf("use_atokens = %v", got.Params["use_atokens"])
	}
}

func TestAaveV3_Liquidation(t *testing.T) {
	d := NewAaveV3Decoder()
	args := aaveV3ABI.Events["LiquidationCall"].Inputs.NonIndexed()
	data, err := args.Pack(big.NewInt(7_000), big.NewInt(8_000), liquidatorAd, false)
	if err != nil {
		t.Fatal(err)
	}
	ev := &types.ChainEvent{
		RawTopics: []common.Hash{
			AaveV3LiquidationCallSig,
			addrTopic(usdcReserve),
			addrTopic(wethDebt),
			addrTopic(userA),
		},
		RawData: data,
	}
	got, err := d.Decode(ev)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.EventType != "liquidation" {
		t.Errorf("type = %q", got.EventType)
	}
	if got.Params["liquidator"].(string) != liquidatorAd.Hex() {
		t.Errorf("liquidator = %v", got.Params["liquidator"])
	}
	if got.Params["receive_atoken"].(bool) != false {
		t.Errorf("receive_atoken = %v", got.Params["receive_atoken"])
	}
	if got.Params["debt_to_cover"].(string) != "7000" {
		t.Errorf("debt_to_cover = %v", got.Params["debt_to_cover"])
	}
}

func TestAaveV3_BasicSupport(t *testing.T) {
	d := NewAaveV3Decoder()
	if d.Name() != "aave_v3" {
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

func TestAaveV3_Malformed(t *testing.T) {
	d := NewAaveV3Decoder()
	if _, err := d.Decode(nil); err == nil {
		t.Error("expected error for nil event")
	}
	if _, err := d.Decode(&types.ChainEvent{}); err == nil {
		t.Error("expected error for empty topics")
	}
	if _, err := d.Decode(&types.ChainEvent{
		RawTopics: []common.Hash{AaveV3SupplySig, {}},
	}); err == nil {
		t.Error("expected error for too-few topics")
	}
}

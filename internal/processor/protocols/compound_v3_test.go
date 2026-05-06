package protocols

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

var (
	cometUSDC    = common.HexToAddress("0xc3d688B66703497DAA19211EEdff47f25384cdc3") // cUSDCv3 mainnet
	srcAddr      = common.HexToAddress("0x1111111111111111111111111111111111111111")
	dstAddr      = common.HexToAddress("0x2222222222222222222222222222222222222222")
	borrowerAddr = common.HexToAddress("0x3333333333333333333333333333333333333333")
)

func TestCompoundV3_Supply(t *testing.T) {
	d := NewCompoundV3Decoder()
	args := compoundV3ABI.Events["Supply"].Inputs.NonIndexed()
	data, err := args.Pack(big.NewInt(123_456))
	if err != nil {
		t.Fatal(err)
	}
	ev := &types.ChainEvent{
		Contract: cometUSDC,
		RawTopics: []common.Hash{
			CompoundV3SupplySig,
			addrTopic(srcAddr),
			addrTopic(dstAddr),
		},
		RawData: data,
	}
	got, err := d.Decode(ev)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Protocol != "compound_v3" || got.EventType != "supply" {
		t.Errorf("proto/type = %q/%q", got.Protocol, got.EventType)
	}
	if got.Params["amount"].(string) != "123456" {
		t.Errorf("amount = %v", got.Params["amount"])
	}
	if got.Params["from"].(string) != srcAddr.Hex() {
		t.Errorf("from = %v", got.Params["from"])
	}
	if got.Params["comet"].(string) != cometUSDC.Hex() {
		t.Errorf("comet = %v", got.Params["comet"])
	}
}

func TestCompoundV3_Withdraw(t *testing.T) {
	d := NewCompoundV3Decoder()
	args := compoundV3ABI.Events["Withdraw"].Inputs.NonIndexed()
	data, err := args.Pack(big.NewInt(42))
	if err != nil {
		t.Fatal(err)
	}
	ev := &types.ChainEvent{
		Contract: cometUSDC,
		RawTopics: []common.Hash{
			CompoundV3WithdrawSig,
			addrTopic(srcAddr),
			addrTopic(dstAddr),
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
	if got.Params["src"].(string) != srcAddr.Hex() {
		t.Errorf("src = %v", got.Params["src"])
	}
}

func TestCompoundV3_Absorb(t *testing.T) {
	d := NewCompoundV3Decoder()
	args := compoundV3ABI.Events["AbsorbCollateral"].Inputs.NonIndexed()
	data, err := args.Pack(big.NewInt(5_000_000), big.NewInt(1_500))
	if err != nil {
		t.Fatal(err)
	}
	ev := &types.ChainEvent{
		Contract: cometUSDC,
		RawTopics: []common.Hash{
			CompoundV3AbsorbSig,
			addrTopic(srcAddr),
			addrTopic(borrowerAddr),
			addrTopic(usdcReserve),
		},
		RawData: data,
	}
	got, err := d.Decode(ev)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.EventType != "absorb" {
		t.Errorf("type = %q", got.EventType)
	}
	if got.Params["asset"].(string) != usdcReserve.Hex() {
		t.Errorf("asset = %v", got.Params["asset"])
	}
	if got.Params["collateral_absorbed"].(string) != "5000000" {
		t.Errorf("collateral = %v", got.Params["collateral_absorbed"])
	}
	if got.Params["usd_value"].(string) != "1500" {
		t.Errorf("usd = %v", got.Params["usd_value"])
	}
}

func TestCompoundV3_BasicSupport(t *testing.T) {
	d := NewCompoundV3Decoder()
	if d.Name() != "compound_v3" {
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

func TestCompoundV3_Malformed(t *testing.T) {
	d := NewCompoundV3Decoder()
	if _, err := d.Decode(nil); err == nil {
		t.Error("expected error for nil event")
	}
	if _, err := d.Decode(&types.ChainEvent{}); err == nil {
		t.Error("expected error for empty topics")
	}
	if _, err := d.Decode(&types.ChainEvent{
		RawTopics: []common.Hash{CompoundV3SupplySig, {}},
	}); err == nil {
		t.Error("expected error for too-few topics")
	}
}

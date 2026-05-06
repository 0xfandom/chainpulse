package protocols

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

var (
	// stETH canonical contract on Ethereum mainnet
	stETHContract = common.HexToAddress("0xae7ab96520de3a18e5e111b5eaab095312d7fe84")
	lidoSender    = common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	lidoReferral  = common.HexToAddress("0x0000000000000000000000000000000000000000")
)

func TestLido_Submitted(t *testing.T) {
	d := NewLidoDecoder()

	args := lidoABI.Events["Submitted"].Inputs.NonIndexed()
	data, err := args.Pack(big.NewInt(1_000_000_000_000_000_000), lidoReferral)
	if err != nil {
		t.Fatal(err)
	}

	ev := &types.ChainEvent{
		Contract: stETHContract,
		RawTopics: []common.Hash{
			LidoSubmittedSig,
			addrTopic(lidoSender),
		},
		RawData: data,
	}

	got, err := d.Decode(ev)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Protocol != "lido" {
		t.Errorf("Protocol = %q, want %q", got.Protocol, "lido")
	}
	if got.EventType != "submitted" {
		t.Errorf("EventType = %q, want %q", got.EventType, "submitted")
	}
	if got.Params["sender"].(string) != lidoSender.Hex() {
		t.Errorf("sender = %v, want %v", got.Params["sender"], lidoSender.Hex())
	}
	if got.Params["amount"].(string) != "1000000000000000000" {
		t.Errorf("amount = %v, want 1000000000000000000", got.Params["amount"])
	}
	if got.Params["referral"].(string) != lidoReferral.Hex() {
		t.Errorf("referral = %v, want %v", got.Params["referral"], lidoReferral.Hex())
	}
}

func TestLido_BasicSupport(t *testing.T) {
	d := NewLidoDecoder()
	if d.Name() != "lido" {
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

func TestLido_Malformed(t *testing.T) {
	d := NewLidoDecoder()

	if _, err := d.Decode(nil); err == nil {
		t.Error("expected error for nil event")
	}
	if _, err := d.Decode(&types.ChainEvent{}); err == nil {
		t.Error("expected error for empty topics")
	}
	// Only topic[0] present — sender topic[1] is missing.
	if _, err := d.Decode(&types.ChainEvent{
		RawTopics: []common.Hash{LidoSubmittedSig},
	}); err == nil {
		t.Error("expected error for too-few topics")
	}
}

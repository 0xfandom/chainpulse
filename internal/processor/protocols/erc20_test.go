package protocols

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/0xfandom/chainpulse/internal/types"
)

// Real Base-mainnet USDC contract.
var baseUSDC = common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")

func newTransferEvent() *types.ChainEvent {
	return &types.ChainEvent{
		ChainID:  8453,
		Contract: baseUSDC,
		RawTopics: []common.Hash{
			ERC20TransferSig,
			common.HexToHash("0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			common.HexToHash("0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		},
		RawData: hexutil.MustDecode("0x00000000000000000000000000000000000000000000000000000000000f4240"), // 1_000_000
	}
}

func newApprovalEvent() *types.ChainEvent {
	return &types.ChainEvent{
		ChainID:  8453,
		Contract: baseUSDC,
		RawTopics: []common.Hash{
			ERC20ApprovalSig,
			common.HexToHash("0x000000000000000000000000cccccccccccccccccccccccccccccccccccccccc"),
			common.HexToHash("0x000000000000000000000000dddddddddddddddddddddddddddddddddddddddd"),
		},
		RawData: hexutil.MustDecode("0x00000000000000000000000000000000000000000000000000000000000186a0"), // 100_000
	}
}

func TestERC20Decoder_BasicSupport(t *testing.T) {
	d := NewERC20Decoder()
	if d.Name() != "erc20" {
		t.Errorf("Name = %q", d.Name())
	}
	if !d.CanDecode(ERC20TransferSig) || !d.CanDecode(ERC20ApprovalSig) {
		t.Error("CanDecode should accept Transfer and Approval")
	}
	if d.CanDecode(common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000")) {
		t.Error("CanDecode should reject unknown sig")
	}
	if got := d.SupportedEvents(); len(got) != 2 {
		t.Errorf("SupportedEvents len = %d", len(got))
	}
}

func TestERC20Decoder_DecodeTransfer(t *testing.T) {
	d := NewERC20Decoder()
	got, err := d.Decode(newTransferEvent())
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.EventType != "transfer" || got.Protocol != "erc20" {
		t.Errorf("type/proto = %q/%q", got.EventType, got.Protocol)
	}
	if got.Params["amount"].(string) != "1000000" {
		t.Errorf("amount = %v", got.Params["amount"])
	}
	if got.Params["token"].(string) != baseUSDC.Hex() {
		t.Errorf("token = %v", got.Params["token"])
	}
	for _, k := range []string{"from", "to", "amount", "token"} {
		if got.Params[k] == nil {
			t.Errorf("missing param %q", k)
		}
	}
}

func TestERC20Decoder_DecodeApproval(t *testing.T) {
	d := NewERC20Decoder()
	got, err := d.Decode(newApprovalEvent())
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.EventType != "approval" {
		t.Errorf("type = %q", got.EventType)
	}
	if got.Params["amount"].(string) != "100000" {
		t.Errorf("amount = %v", got.Params["amount"])
	}
	for _, k := range []string{"owner", "spender", "amount", "token"} {
		if got.Params[k] == nil {
			t.Errorf("missing param %q", k)
		}
	}
}

func TestERC20Decoder_Malformed(t *testing.T) {
	d := NewERC20Decoder()

	// missing topics
	bad := &types.ChainEvent{RawTopics: []common.Hash{ERC20TransferSig}}
	if _, err := d.Decode(bad); err == nil {
		t.Error("expected error for too few topics")
	}

	// short data
	short := &types.ChainEvent{
		RawTopics: []common.Hash{ERC20TransferSig, {}, {}},
		RawData:   []byte{0x01, 0x02},
	}
	if _, err := d.Decode(short); err == nil {
		t.Error("expected error for short data")
	}

	// unsupported sig
	wrong := &types.ChainEvent{
		RawTopics: []common.Hash{common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000"), {}, {}},
		RawData:   make([]byte, 32),
	}
	if _, err := d.Decode(wrong); err == nil {
		t.Error("expected error for unsupported sig")
	}

	// nil event
	if _, err := d.Decode(nil); err == nil {
		t.Error("expected error for nil event")
	}
}

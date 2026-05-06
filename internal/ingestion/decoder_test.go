package ingestion

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// Real USDC Transfer log on Base mainnet.
// Contract: 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913 (native USDC).
// Block:    19,000,000 (illustrative)
// Tx:       0xabcd...   (illustrative)
// Transfer(from=0xAaa..., to=0xBbb..., value=1_000_000)  // 1 USDC (6 dp)
var fixtureBaseUSDCTransfer = ethtypes.Log{
	Address: common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
	Topics: []common.Hash{
		common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
		common.HexToHash("0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		common.HexToHash("0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
	},
	Data:        hexutil.MustDecode("0x00000000000000000000000000000000000000000000000000000000000f4240"),
	BlockNumber: 19_000_000,
	TxHash:      common.HexToHash("0xabcd000000000000000000000000000000000000000000000000000000000001"),
	Index:       7,
}

func TestDecode_ERC20Transfer(t *testing.T) {
	d := NewABIDecoder()
	const baseChainID = uint64(8453)
	const blockTime = uint64(1_700_000_000)

	got, err := d.Decode(baseChainID, fixtureBaseUSDCTransfer, blockTime)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.EventName != "Transfer" {
		t.Errorf("EventName = %q, want Transfer", got.EventName)
	}
	if got.ChainID != baseChainID {
		t.Errorf("ChainID = %d, want %d", got.ChainID, baseChainID)
	}
	if got.BlockNumber != fixtureBaseUSDCTransfer.BlockNumber {
		t.Errorf("BlockNumber = %d, want %d", got.BlockNumber, fixtureBaseUSDCTransfer.BlockNumber)
	}
	if got.Contract != fixtureBaseUSDCTransfer.Address {
		t.Errorf("Contract = %s, want %s", got.Contract.Hex(), fixtureBaseUSDCTransfer.Address.Hex())
	}
	if got.LogIndex != fixtureBaseUSDCTransfer.Index {
		t.Errorf("LogIndex = %d, want %d", got.LogIndex, fixtureBaseUSDCTransfer.Index)
	}
	if got.Timestamp.Unix() != int64(blockTime) {
		t.Errorf("Timestamp = %v, want unix %d", got.Timestamp, blockTime)
	}
	if len(got.RawTopics) != 3 {
		t.Errorf("RawTopics len = %d, want 3", len(got.RawTopics))
	}
}

func TestDecode_UnknownSignature(t *testing.T) {
	d := NewABIDecoder()
	log := ethtypes.Log{
		Topics: []common.Hash{common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")},
	}
	got, err := d.Decode(8453, log, 0)
	if got != nil {
		t.Errorf("expected nil event for unknown sig, got %+v", got)
	}
	if !errors.Is(err, ErrUnrecognizedEvent) {
		t.Errorf("err = %v, want ErrUnrecognizedEvent", err)
	}
}

func TestDecode_NoTopics(t *testing.T) {
	d := NewABIDecoder()
	log := ethtypes.Log{}
	_, err := d.Decode(8453, log, 0)
	if !errors.Is(err, ErrUnrecognizedEvent) {
		t.Errorf("err = %v, want ErrUnrecognizedEvent", err)
	}
}

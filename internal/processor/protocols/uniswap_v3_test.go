package protocols

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

// helpers for building topic values
func addrTopic(a common.Address) common.Hash {
	var h common.Hash
	copy(h[12:], a.Bytes())
	return h
}

// int24Topic packs a signed int24 into a topic, two's-complement
// sign-extended to 32 bytes (matching how Solidity emits indexed int24).
func int24Topic(n int64) common.Hash {
	v := big.NewInt(n)
	if n < 0 {
		// 256-bit two's complement: add 2^256.
		twoPow256 := new(big.Int).Lsh(big.NewInt(1), 256)
		v = new(big.Int).Add(twoPow256, v)
	}
	var h common.Hash
	bs := v.Bytes()
	copy(h[32-len(bs):], bs)
	if n < 0 {
		// Ensure leading 0xff bytes for sign extension above the 24-bit value
		for i := 0; i < 32-3; i++ {
			h[i] = 0xff
		}
	}
	return h
}

var (
	uniswapPool = common.HexToAddress("0x88e6A0c2dDD26FEEb64F039a2c41296FcB3f5640") // USDC/WETH on Ethereum
	senderAddr  = common.HexToAddress("0xCccCCccCcCCCcCCcCCcCCccCCcCCcccccccccccc")
	recipAddr   = common.HexToAddress("0xDDdDDdDdDdDDDDdDdDDDDDDDdDDdddDdDDdddddD")
	ownerAddr   = common.HexToAddress("0xeeEeEEEeeEEEeEeEEEEeEEeeeeeEeEeEEeeeeEee")
)

func packSwapData(t *testing.T, amount0, amount1, sqrt, liquidity *big.Int, tick *big.Int) []byte {
	t.Helper()
	args := uniswapV3ABI.Events["Swap"].Inputs.NonIndexed()
	out, err := args.Pack(amount0, amount1, sqrt, liquidity, tick)
	if err != nil {
		t.Fatalf("pack swap: %v", err)
	}
	return out
}

func TestUniswapV3_Swap(t *testing.T) {
	d := NewUniswapV3Decoder()

	amount0 := big.NewInt(1_000_000)
	amount1 := big.NewInt(-500_000) // negative -> recipient receiving token1
	sqrt := new(big.Int).Lsh(big.NewInt(1), 96)
	liquidity := big.NewInt(123_456)
	tick := big.NewInt(-887272) // a real tick range value

	data := packSwapData(t, amount0, amount1, sqrt, liquidity, tick)

	ev := &types.ChainEvent{
		ChainID:  1,
		Contract: uniswapPool,
		RawTopics: []common.Hash{
			UniswapV3SwapSig,
			addrTopic(senderAddr),
			addrTopic(recipAddr),
		},
		RawData: data,
	}

	got, err := d.Decode(ev)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Protocol != "uniswap_v3" || got.EventType != "swap" {
		t.Errorf("proto/type = %q/%q", got.Protocol, got.EventType)
	}
	if got.Params["amount0"].(string) != amount0.String() {
		t.Errorf("amount0 = %v", got.Params["amount0"])
	}
	if got.Params["amount1"].(string) != amount1.String() {
		t.Errorf("amount1 = %v", got.Params["amount1"])
	}
	if got.Params["tick"].(string) != tick.String() {
		t.Errorf("tick = %v", got.Params["tick"])
	}
	if got.Params["pool"].(string) != uniswapPool.Hex() {
		t.Errorf("pool = %v", got.Params["pool"])
	}
}

func TestUniswapV3_Mint(t *testing.T) {
	d := NewUniswapV3Decoder()
	args := uniswapV3ABI.Events["Mint"].Inputs.NonIndexed()
	data, err := args.Pack(senderAddr, big.NewInt(42), big.NewInt(1000), big.NewInt(2000))
	if err != nil {
		t.Fatal(err)
	}
	ev := &types.ChainEvent{
		Contract: uniswapPool,
		RawTopics: []common.Hash{
			UniswapV3MintSig,
			addrTopic(ownerAddr),
			int24Topic(-100),
			int24Topic(200),
		},
		RawData: data,
	}
	got, err := d.Decode(ev)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.EventType != "mint" {
		t.Errorf("type = %q", got.EventType)
	}
	if got.Params["tick_lower"].(string) != "-100" {
		t.Errorf("tick_lower = %v", got.Params["tick_lower"])
	}
	if got.Params["tick_upper"].(string) != "200" {
		t.Errorf("tick_upper = %v", got.Params["tick_upper"])
	}
	if got.Params["sender"].(string) != senderAddr.Hex() {
		t.Errorf("sender = %v", got.Params["sender"])
	}
}

func TestUniswapV3_Burn(t *testing.T) {
	d := NewUniswapV3Decoder()
	args := uniswapV3ABI.Events["Burn"].Inputs.NonIndexed()
	data, err := args.Pack(big.NewInt(11), big.NewInt(22), big.NewInt(33))
	if err != nil {
		t.Fatal(err)
	}
	ev := &types.ChainEvent{
		Contract: uniswapPool,
		RawTopics: []common.Hash{
			UniswapV3BurnSig,
			addrTopic(ownerAddr),
			int24Topic(-50),
			int24Topic(50),
		},
		RawData: data,
	}
	got, err := d.Decode(ev)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.EventType != "burn" {
		t.Errorf("type = %q", got.EventType)
	}
	if got.Params["amount0"].(string) != "22" {
		t.Errorf("amount0 = %v", got.Params["amount0"])
	}
}

func TestUniswapV3_Collect(t *testing.T) {
	d := NewUniswapV3Decoder()
	args := uniswapV3ABI.Events["Collect"].Inputs.NonIndexed()
	data, err := args.Pack(recipAddr, big.NewInt(5), big.NewInt(7))
	if err != nil {
		t.Fatal(err)
	}
	ev := &types.ChainEvent{
		Contract: uniswapPool,
		RawTopics: []common.Hash{
			UniswapV3CollectSig,
			addrTopic(ownerAddr),
			int24Topic(0),
			int24Topic(100),
		},
		RawData: data,
	}
	got, err := d.Decode(ev)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.EventType != "collect" {
		t.Errorf("type = %q", got.EventType)
	}
	if got.Params["recipient"].(string) != recipAddr.Hex() {
		t.Errorf("recipient = %v", got.Params["recipient"])
	}
}

func TestUniswapV3_BasicSupport(t *testing.T) {
	d := NewUniswapV3Decoder()
	if d.Name() != "uniswap_v3" {
		t.Errorf("Name = %q", d.Name())
	}
	for _, sig := range d.SupportedEvents() {
		if !d.CanDecode(sig) {
			t.Errorf("CanDecode rejected supported sig %s", sig.Hex())
		}
	}
	if d.CanDecode(common.HexToHash("0xdead000000000000000000000000000000000000000000000000000000000000")) {
		t.Error("CanDecode should reject unknown sig")
	}
}

func TestUniswapV3_Malformed(t *testing.T) {
	d := NewUniswapV3Decoder()
	if _, err := d.Decode(nil); err == nil {
		t.Error("expected error for nil event")
	}
	if _, err := d.Decode(&types.ChainEvent{}); err == nil {
		t.Error("expected error for empty topics")
	}
	if _, err := d.Decode(&types.ChainEvent{
		RawTopics: []common.Hash{UniswapV3SwapSig},
	}); err == nil {
		t.Error("expected error for too-few topics")
	}
}

package protocols

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

// uniswapV3PoolABIJSON is the minimal Uniswap V3 Pool ABI covering the
// four events we decode. Embedding it inline avoids an extra build step
// and keeps the decoder self-contained.
const uniswapV3PoolABIJSON = `[
  {"anonymous":false,"name":"Swap","type":"event","inputs":[
    {"indexed":true,"name":"sender","type":"address"},
    {"indexed":true,"name":"recipient","type":"address"},
    {"indexed":false,"name":"amount0","type":"int256"},
    {"indexed":false,"name":"amount1","type":"int256"},
    {"indexed":false,"name":"sqrtPriceX96","type":"uint160"},
    {"indexed":false,"name":"liquidity","type":"uint128"},
    {"indexed":false,"name":"tick","type":"int24"}
  ]},
  {"anonymous":false,"name":"Mint","type":"event","inputs":[
    {"indexed":false,"name":"sender","type":"address"},
    {"indexed":true,"name":"owner","type":"address"},
    {"indexed":true,"name":"tickLower","type":"int24"},
    {"indexed":true,"name":"tickUpper","type":"int24"},
    {"indexed":false,"name":"amount","type":"uint128"},
    {"indexed":false,"name":"amount0","type":"uint256"},
    {"indexed":false,"name":"amount1","type":"uint256"}
  ]},
  {"anonymous":false,"name":"Burn","type":"event","inputs":[
    {"indexed":true,"name":"owner","type":"address"},
    {"indexed":true,"name":"tickLower","type":"int24"},
    {"indexed":true,"name":"tickUpper","type":"int24"},
    {"indexed":false,"name":"amount","type":"uint128"},
    {"indexed":false,"name":"amount0","type":"uint256"},
    {"indexed":false,"name":"amount1","type":"uint256"}
  ]},
  {"anonymous":false,"name":"Collect","type":"event","inputs":[
    {"indexed":true,"name":"owner","type":"address"},
    {"indexed":false,"name":"recipient","type":"address"},
    {"indexed":true,"name":"tickLower","type":"int24"},
    {"indexed":true,"name":"tickUpper","type":"int24"},
    {"indexed":false,"name":"amount0","type":"uint128"},
    {"indexed":false,"name":"amount1","type":"uint128"}
  ]}
]`

// Uniswap V3 Pool event signatures.
var (
	UniswapV3SwapSig    common.Hash
	UniswapV3MintSig    common.Hash
	UniswapV3BurnSig    common.Hash
	UniswapV3CollectSig common.Hash

	uniswapV3ABI abi.ABI
)

func init() {
	parsed, err := abi.JSON(strings.NewReader(uniswapV3PoolABIJSON))
	if err != nil {
		panic(fmt.Sprintf("uniswap_v3: parse ABI: %v", err))
	}
	uniswapV3ABI = parsed
	UniswapV3SwapSig = parsed.Events["Swap"].ID
	UniswapV3MintSig = parsed.Events["Mint"].ID
	UniswapV3BurnSig = parsed.Events["Burn"].ID
	UniswapV3CollectSig = parsed.Events["Collect"].ID
}

// UniswapV3Decoder implements processor.ProtocolDecoder for Uniswap V3
// Pool events.
type UniswapV3Decoder struct{}

// NewUniswapV3Decoder returns a stateless decoder.
func NewUniswapV3Decoder() *UniswapV3Decoder { return &UniswapV3Decoder{} }

// Name implements ProtocolDecoder.
func (d *UniswapV3Decoder) Name() string { return "uniswap_v3" }

// CanDecode implements ProtocolDecoder.
func (d *UniswapV3Decoder) CanDecode(sig common.Hash) bool {
	switch sig {
	case UniswapV3SwapSig, UniswapV3MintSig, UniswapV3BurnSig, UniswapV3CollectSig:
		return true
	}
	return false
}

// SupportedEvents implements ProtocolDecoder.
func (d *UniswapV3Decoder) SupportedEvents() []common.Hash {
	return []common.Hash{
		UniswapV3SwapSig, UniswapV3MintSig, UniswapV3BurnSig, UniswapV3CollectSig,
	}
}

// Decode implements ProtocolDecoder.
func (d *UniswapV3Decoder) Decode(event *types.ChainEvent) (*types.DecodedEvent, error) {
	if event == nil {
		return nil, fmt.Errorf("uniswap_v3: nil event")
	}
	if len(event.RawTopics) == 0 {
		return nil, fmt.Errorf("uniswap_v3: no topics")
	}
	pool := event.Contract.Hex()

	switch event.RawTopics[0] {
	case UniswapV3SwapSig:
		return d.decodeSwap(event, pool)
	case UniswapV3MintSig:
		return d.decodeMint(event, pool)
	case UniswapV3BurnSig:
		return d.decodeBurn(event, pool)
	case UniswapV3CollectSig:
		return d.decodeCollect(event, pool)
	default:
		return nil, fmt.Errorf("uniswap_v3: unsupported signature %s", event.RawTopics[0].Hex())
	}
}

func (d *UniswapV3Decoder) decodeSwap(event *types.ChainEvent, pool string) (*types.DecodedEvent, error) {
	if len(event.RawTopics) < 3 {
		return nil, fmt.Errorf("uniswap_v3 swap: expected 3 topics, got %d", len(event.RawTopics))
	}
	sender := common.BytesToAddress(event.RawTopics[1].Bytes())
	recipient := common.BytesToAddress(event.RawTopics[2].Bytes())

	out, err := uniswapV3ABI.Events["Swap"].Inputs.NonIndexed().Unpack(event.RawData)
	if err != nil {
		return nil, fmt.Errorf("uniswap_v3 swap: unpack: %w", err)
	}
	if len(out) != 5 {
		return nil, fmt.Errorf("uniswap_v3 swap: unexpected unpack arity %d", len(out))
	}
	amount0, _ := out[0].(*big.Int)
	amount1, _ := out[1].(*big.Int)
	sqrtPriceX96, _ := out[2].(*big.Int)
	liquidity, _ := out[3].(*big.Int)
	tick, _ := out[4].(*big.Int)

	return &types.DecodedEvent{
		ChainEvent: *event,
		Protocol:   d.Name(),
		EventType:  "swap",
		Params: map[string]interface{}{
			"sender":         sender.Hex(),
			"recipient":      recipient.Hex(),
			"amount0":        bigString(amount0),
			"amount1":        bigString(amount1),
			"sqrt_price_x96": bigString(sqrtPriceX96),
			"liquidity":      bigString(liquidity),
			"tick":           bigString(tick),
			"pool":           pool,
		},
	}, nil
}

func (d *UniswapV3Decoder) decodeMint(event *types.ChainEvent, pool string) (*types.DecodedEvent, error) {
	if len(event.RawTopics) < 4 {
		return nil, fmt.Errorf("uniswap_v3 mint: expected 4 topics, got %d", len(event.RawTopics))
	}
	owner := common.BytesToAddress(event.RawTopics[1].Bytes())
	tickLower := topicToInt24(event.RawTopics[2])
	tickUpper := topicToInt24(event.RawTopics[3])

	out, err := uniswapV3ABI.Events["Mint"].Inputs.NonIndexed().Unpack(event.RawData)
	if err != nil {
		return nil, fmt.Errorf("uniswap_v3 mint: unpack: %w", err)
	}
	if len(out) != 4 {
		return nil, fmt.Errorf("uniswap_v3 mint: unexpected unpack arity %d", len(out))
	}
	sender, _ := out[0].(common.Address)
	amount, _ := out[1].(*big.Int)
	amount0, _ := out[2].(*big.Int)
	amount1, _ := out[3].(*big.Int)

	return &types.DecodedEvent{
		ChainEvent: *event,
		Protocol:   d.Name(),
		EventType:  "mint",
		Params: map[string]interface{}{
			"sender":     sender.Hex(),
			"owner":      owner.Hex(),
			"tick_lower": tickLower.String(),
			"tick_upper": tickUpper.String(),
			"amount":     bigString(amount),
			"amount0":    bigString(amount0),
			"amount1":    bigString(amount1),
			"pool":       pool,
		},
	}, nil
}

func (d *UniswapV3Decoder) decodeBurn(event *types.ChainEvent, pool string) (*types.DecodedEvent, error) {
	if len(event.RawTopics) < 4 {
		return nil, fmt.Errorf("uniswap_v3 burn: expected 4 topics, got %d", len(event.RawTopics))
	}
	owner := common.BytesToAddress(event.RawTopics[1].Bytes())
	tickLower := topicToInt24(event.RawTopics[2])
	tickUpper := topicToInt24(event.RawTopics[3])

	out, err := uniswapV3ABI.Events["Burn"].Inputs.NonIndexed().Unpack(event.RawData)
	if err != nil {
		return nil, fmt.Errorf("uniswap_v3 burn: unpack: %w", err)
	}
	if len(out) != 3 {
		return nil, fmt.Errorf("uniswap_v3 burn: unexpected unpack arity %d", len(out))
	}
	amount, _ := out[0].(*big.Int)
	amount0, _ := out[1].(*big.Int)
	amount1, _ := out[2].(*big.Int)

	return &types.DecodedEvent{
		ChainEvent: *event,
		Protocol:   d.Name(),
		EventType:  "burn",
		Params: map[string]interface{}{
			"owner":      owner.Hex(),
			"tick_lower": tickLower.String(),
			"tick_upper": tickUpper.String(),
			"amount":     bigString(amount),
			"amount0":    bigString(amount0),
			"amount1":    bigString(amount1),
			"pool":       pool,
		},
	}, nil
}

func (d *UniswapV3Decoder) decodeCollect(event *types.ChainEvent, pool string) (*types.DecodedEvent, error) {
	if len(event.RawTopics) < 4 {
		return nil, fmt.Errorf("uniswap_v3 collect: expected 4 topics, got %d", len(event.RawTopics))
	}
	owner := common.BytesToAddress(event.RawTopics[1].Bytes())
	tickLower := topicToInt24(event.RawTopics[2])
	tickUpper := topicToInt24(event.RawTopics[3])

	out, err := uniswapV3ABI.Events["Collect"].Inputs.NonIndexed().Unpack(event.RawData)
	if err != nil {
		return nil, fmt.Errorf("uniswap_v3 collect: unpack: %w", err)
	}
	if len(out) != 3 {
		return nil, fmt.Errorf("uniswap_v3 collect: unexpected unpack arity %d", len(out))
	}
	recipient, _ := out[0].(common.Address)
	amount0, _ := out[1].(*big.Int)
	amount1, _ := out[2].(*big.Int)

	return &types.DecodedEvent{
		ChainEvent: *event,
		Protocol:   d.Name(),
		EventType:  "collect",
		Params: map[string]interface{}{
			"owner":      owner.Hex(),
			"recipient":  recipient.Hex(),
			"tick_lower": tickLower.String(),
			"tick_upper": tickUpper.String(),
			"amount0":    bigString(amount0),
			"amount1":    bigString(amount1),
			"pool":       pool,
		},
	}, nil
}

// topicToInt24 reads a signed int24 from a 32-byte topic. Solidity packs
// int24 right-aligned with sign extension, so we interpret the top byte
// as sign.
func topicToInt24(t common.Hash) *big.Int {
	b := t.Bytes()
	v := new(big.Int).SetBytes(b)
	// If the top bit of the int24 is set, the topic is sign-extended with
	// 0xff in the high bytes; convert two's-complement to a signed value.
	if b[0] == 0xff {
		// 256-bit two's complement: subtract 2^256.
		twoPow256 := new(big.Int).Lsh(big.NewInt(1), 256)
		v.Sub(v, twoPow256)
	}
	return v
}

// bigString returns the decimal string of v, or "0" when v is nil.
func bigString(v *big.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
}

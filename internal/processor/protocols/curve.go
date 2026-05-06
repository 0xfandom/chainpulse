package protocols

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

// curveStableswapABIJSON is the Curve Stableswap V1 ABI for TokenExchange.
// Covers 3pool, FRAX/USDC, sUSD, MIM, and other stableswap pools.
// The cryptopool variant uses uint256 indexes and has a different topic0;
// it is intentionally excluded here.
const curveStableswapABIJSON = `[
  {"anonymous":false,"name":"TokenExchange","type":"event","inputs":[
    {"indexed":true,"name":"buyer","type":"address"},
    {"indexed":false,"name":"sold_id","type":"int128"},
    {"indexed":false,"name":"tokens_sold","type":"uint256"},
    {"indexed":false,"name":"bought_id","type":"int128"},
    {"indexed":false,"name":"tokens_bought","type":"uint256"}
  ]}
]`

// CurveTokenExchangeSig is the topic0 for
// TokenExchange(address,int128,uint256,int128,uint256).
// keccak256 = 0x8b3e96f2b889fa771c53c981b40daf005f63f637f1869f707052d15a3dd97140
var CurveTokenExchangeSig common.Hash

var curveABI abi.ABI

func init() {
	parsed, err := abi.JSON(strings.NewReader(curveStableswapABIJSON))
	if err != nil {
		panic(fmt.Sprintf("curve: parse ABI: %v", err))
	}
	curveABI = parsed
	CurveTokenExchangeSig = parsed.Events["TokenExchange"].ID
}

// CurveDecoder implements processor.ProtocolDecoder for Curve Stableswap V1
// TokenExchange events.
type CurveDecoder struct{}

// NewCurveDecoder returns a stateless decoder.
func NewCurveDecoder() *CurveDecoder { return &CurveDecoder{} }

// Name implements ProtocolDecoder.
func (d *CurveDecoder) Name() string { return "curve" }

// CanDecode implements ProtocolDecoder.
func (d *CurveDecoder) CanDecode(sig common.Hash) bool {
	return sig == CurveTokenExchangeSig
}

// SupportedEvents implements ProtocolDecoder.
func (d *CurveDecoder) SupportedEvents() []common.Hash {
	return []common.Hash{CurveTokenExchangeSig}
}

// Decode implements ProtocolDecoder.
func (d *CurveDecoder) Decode(event *types.ChainEvent) (*types.DecodedEvent, error) {
	if event == nil {
		return nil, fmt.Errorf("curve: nil event")
	}
	if len(event.RawTopics) == 0 {
		return nil, fmt.Errorf("curve: no topics")
	}
	switch event.RawTopics[0] {
	case CurveTokenExchangeSig:
		return d.decodeTokenExchange(event)
	default:
		return nil, fmt.Errorf("curve: unsupported signature %s", event.RawTopics[0].Hex())
	}
}

func (d *CurveDecoder) decodeTokenExchange(event *types.ChainEvent) (*types.DecodedEvent, error) {
	// topic[1] = buyer (indexed address)
	if len(event.RawTopics) < 2 {
		return nil, fmt.Errorf("decode TokenExchange: expected 2 topics, got %d", len(event.RawTopics))
	}
	buyer := common.BytesToAddress(event.RawTopics[1].Bytes())

	out, err := curveABI.Events["TokenExchange"].Inputs.NonIndexed().Unpack(event.RawData)
	if err != nil {
		return nil, fmt.Errorf("decode TokenExchange: unpack: %w", err)
	}
	if len(out) != 4 {
		return nil, fmt.Errorf("decode TokenExchange: arity %d", len(out))
	}
	// go-ethereum ABI decodes int128 as *big.Int (signed).
	soldID, _ := out[0].(*big.Int)
	tokensSold, _ := out[1].(*big.Int)
	boughtID, _ := out[2].(*big.Int)
	tokensBought, _ := out[3].(*big.Int)

	return &types.DecodedEvent{
		ChainEvent: *event,
		Protocol:   d.Name(),
		EventType:  "swap",
		Params: map[string]interface{}{
			"buyer":         buyer.Hex(),
			"sold_id":       bigString(soldID),
			"tokens_sold":   bigString(tokensSold),
			"bought_id":     bigString(boughtID),
			"tokens_bought": bigString(tokensBought),
		},
	}, nil
}

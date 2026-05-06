// Package protocols holds concrete ProtocolDecoder implementations.
package protocols

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

// lidoStETHABIJSON is the stETH contract ABI for the Submitted event.
// Sourced from the canonical stETH contract at
// 0xae7ab96520de3a18e5e111b5eaab095312d7fe84 on Ethereum mainnet.
const lidoStETHABIJSON = `[
  {"anonymous":false,"name":"Submitted","type":"event","inputs":[
    {"indexed":true,"name":"sender","type":"address"},
    {"indexed":false,"name":"amount","type":"uint256"},
    {"indexed":false,"name":"referral","type":"address"}
  ]}
]`

// LidoSubmittedSig is the topic0 for Submitted(address,uint256,address).
// keccak256("Submitted(address,uint256,address)") =
// 0x96a25c8ce0baabc1fdefd93e9ed25d8e092a3332f3aa9a41722b5697231d1d1a
var LidoSubmittedSig common.Hash

var lidoABI abi.ABI

func init() {
	parsed, err := abi.JSON(strings.NewReader(lidoStETHABIJSON))
	if err != nil {
		panic(fmt.Sprintf("lido: parse ABI: %v", err))
	}
	lidoABI = parsed
	LidoSubmittedSig = parsed.Events["Submitted"].ID
}

// LidoDecoder implements processor.ProtocolDecoder for the Lido stETH
// Submitted event on Ethereum mainnet.
type LidoDecoder struct{}

// NewLidoDecoder returns a stateless decoder.
func NewLidoDecoder() *LidoDecoder { return &LidoDecoder{} }

// Name implements ProtocolDecoder.
func (d *LidoDecoder) Name() string { return "lido" }

// CanDecode implements ProtocolDecoder.
func (d *LidoDecoder) CanDecode(sig common.Hash) bool {
	return sig == LidoSubmittedSig
}

// SupportedEvents implements ProtocolDecoder.
func (d *LidoDecoder) SupportedEvents() []common.Hash {
	return []common.Hash{LidoSubmittedSig}
}

// Decode implements ProtocolDecoder.
func (d *LidoDecoder) Decode(event *types.ChainEvent) (*types.DecodedEvent, error) {
	if event == nil {
		return nil, fmt.Errorf("lido: nil event")
	}
	if len(event.RawTopics) == 0 {
		return nil, fmt.Errorf("lido: no topics")
	}
	switch event.RawTopics[0] {
	case LidoSubmittedSig:
		return d.decodeSubmitted(event)
	default:
		return nil, fmt.Errorf("lido: unsupported signature %s", event.RawTopics[0].Hex())
	}
}

func (d *LidoDecoder) decodeSubmitted(event *types.ChainEvent) (*types.DecodedEvent, error) {
	// topic[1] = sender (indexed address)
	if len(event.RawTopics) < 2 {
		return nil, fmt.Errorf("decode submitted: expected 2 topics, got %d", len(event.RawTopics))
	}
	sender := common.BytesToAddress(event.RawTopics[1].Bytes())

	out, err := lidoABI.Events["Submitted"].Inputs.NonIndexed().Unpack(event.RawData)
	if err != nil {
		return nil, fmt.Errorf("decode submitted: unpack: %w", err)
	}
	if len(out) != 2 {
		return nil, fmt.Errorf("decode submitted: arity %d", len(out))
	}
	amount, _ := out[0].(*big.Int)
	referral, _ := out[1].(common.Address)

	return &types.DecodedEvent{
		ChainEvent: *event,
		Protocol:   d.Name(),
		EventType:  "submitted",
		Params: map[string]interface{}{
			"sender":   sender.Hex(),
			"amount":   bigString(amount),
			"referral": referral.Hex(),
		},
	}, nil
}

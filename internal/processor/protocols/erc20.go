package protocols

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

// ERC20Decoder implements processor.ProtocolDecoder for the ERC-20
// Transfer and Approval events.
type ERC20Decoder struct{}

// NewERC20Decoder returns a stateless ERC20Decoder.
func NewERC20Decoder() *ERC20Decoder { return &ERC20Decoder{} }

// Name implements ProtocolDecoder.
func (d *ERC20Decoder) Name() string { return "erc20" }

// CanDecode implements ProtocolDecoder.
func (d *ERC20Decoder) CanDecode(sig common.Hash) bool {
	return sig == ERC20TransferSig || sig == ERC20ApprovalSig
}

// SupportedEvents implements ProtocolDecoder.
func (d *ERC20Decoder) SupportedEvents() []common.Hash {
	return []common.Hash{ERC20TransferSig, ERC20ApprovalSig}
}

// Decode implements ProtocolDecoder. Both Transfer and Approval share the
// shape (indexed addr, indexed addr, uint256 value), so the decoding path
// is identical apart from the field names.
func (d *ERC20Decoder) Decode(event *types.ChainEvent) (*types.DecodedEvent, error) {
	if event == nil {
		return nil, fmt.Errorf("erc20: nil event")
	}
	if len(event.RawTopics) < 3 {
		return nil, fmt.Errorf("erc20: expected 3 topics, got %d", len(event.RawTopics))
	}
	if len(event.RawData) < 32 {
		return nil, fmt.Errorf("erc20: expected 32 bytes of data, got %d", len(event.RawData))
	}

	a := common.BytesToAddress(event.RawTopics[1].Bytes())
	b := common.BytesToAddress(event.RawTopics[2].Bytes())
	amount := new(big.Int).SetBytes(event.RawData[:32])
	token := event.Contract.Hex()

	switch event.RawTopics[0] {
	case ERC20TransferSig:
		return &types.DecodedEvent{
			ChainEvent: *event,
			Protocol:   d.Name(),
			EventType:  "transfer",
			Params: map[string]interface{}{
				"from":   a.Hex(),
				"to":     b.Hex(),
				"amount": amount.String(),
				"token":  token,
			},
		}, nil
	case ERC20ApprovalSig:
		return &types.DecodedEvent{
			ChainEvent: *event,
			Protocol:   d.Name(),
			EventType:  "approval",
			Params: map[string]interface{}{
				"owner":   a.Hex(),
				"spender": b.Hex(),
				"amount":  amount.String(),
				"token":   token,
			},
		}, nil
	default:
		return nil, fmt.Errorf("erc20: unsupported signature %s", event.RawTopics[0].Hex())
	}
}

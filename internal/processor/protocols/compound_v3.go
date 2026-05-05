package protocols

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

// compoundV3CometABIJSON covers the Comet events we decode on Day 2.
// SupplyCollateral / TransferCollateral / Withdrawals between users are
// out of scope until the API needs them.
const compoundV3CometABIJSON = `[
  {"anonymous":false,"name":"Supply","type":"event","inputs":[
    {"indexed":true,"name":"from","type":"address"},
    {"indexed":true,"name":"dst","type":"address"},
    {"indexed":false,"name":"amount","type":"uint256"}
  ]},
  {"anonymous":false,"name":"Withdraw","type":"event","inputs":[
    {"indexed":true,"name":"src","type":"address"},
    {"indexed":true,"name":"to","type":"address"},
    {"indexed":false,"name":"amount","type":"uint256"}
  ]},
  {"anonymous":false,"name":"AbsorbCollateral","type":"event","inputs":[
    {"indexed":true,"name":"absorber","type":"address"},
    {"indexed":true,"name":"borrower","type":"address"},
    {"indexed":true,"name":"asset","type":"address"},
    {"indexed":false,"name":"collateralAbsorbed","type":"uint256"},
    {"indexed":false,"name":"usdValue","type":"uint256"}
  ]}
]`

var (
	CompoundV3SupplySig   common.Hash
	CompoundV3WithdrawSig common.Hash
	CompoundV3AbsorbSig   common.Hash

	compoundV3ABI abi.ABI
)

func init() {
	parsed, err := abi.JSON(strings.NewReader(compoundV3CometABIJSON))
	if err != nil {
		panic(fmt.Sprintf("compound_v3: parse ABI: %v", err))
	}
	compoundV3ABI = parsed
	CompoundV3SupplySig = parsed.Events["Supply"].ID
	CompoundV3WithdrawSig = parsed.Events["Withdraw"].ID
	CompoundV3AbsorbSig = parsed.Events["AbsorbCollateral"].ID
}

// CompoundV3Decoder implements processor.ProtocolDecoder for the Compound
// V3 Comet contract.
type CompoundV3Decoder struct{}

// NewCompoundV3Decoder returns a stateless decoder.
func NewCompoundV3Decoder() *CompoundV3Decoder { return &CompoundV3Decoder{} }

// Name implements ProtocolDecoder.
func (d *CompoundV3Decoder) Name() string { return "compound_v3" }

// CanDecode implements ProtocolDecoder.
func (d *CompoundV3Decoder) CanDecode(sig common.Hash) bool {
	switch sig {
	case CompoundV3SupplySig, CompoundV3WithdrawSig, CompoundV3AbsorbSig:
		return true
	}
	return false
}

// SupportedEvents implements ProtocolDecoder.
func (d *CompoundV3Decoder) SupportedEvents() []common.Hash {
	return []common.Hash{CompoundV3SupplySig, CompoundV3WithdrawSig, CompoundV3AbsorbSig}
}

// Decode implements ProtocolDecoder.
func (d *CompoundV3Decoder) Decode(event *types.ChainEvent) (*types.DecodedEvent, error) {
	if event == nil {
		return nil, fmt.Errorf("compound_v3: nil event")
	}
	if len(event.RawTopics) == 0 {
		return nil, fmt.Errorf("compound_v3: no topics")
	}
	comet := event.Contract.Hex()

	switch event.RawTopics[0] {
	case CompoundV3SupplySig:
		return d.decodeSupply(event, comet)
	case CompoundV3WithdrawSig:
		return d.decodeWithdraw(event, comet)
	case CompoundV3AbsorbSig:
		return d.decodeAbsorb(event, comet)
	default:
		return nil, fmt.Errorf("compound_v3: unsupported signature %s", event.RawTopics[0].Hex())
	}
}

func (d *CompoundV3Decoder) decodeSupply(event *types.ChainEvent, comet string) (*types.DecodedEvent, error) {
	if len(event.RawTopics) < 3 {
		return nil, fmt.Errorf("compound_v3 supply: expected 3 topics, got %d", len(event.RawTopics))
	}
	from := common.BytesToAddress(event.RawTopics[1].Bytes())
	dst := common.BytesToAddress(event.RawTopics[2].Bytes())

	out, err := compoundV3ABI.Events["Supply"].Inputs.NonIndexed().Unpack(event.RawData)
	if err != nil {
		return nil, fmt.Errorf("compound_v3 supply: unpack: %w", err)
	}
	if len(out) != 1 {
		return nil, fmt.Errorf("compound_v3 supply: arity %d", len(out))
	}
	amount, _ := out[0].(*big.Int)

	return &types.DecodedEvent{
		ChainEvent: *event,
		Protocol:   d.Name(),
		EventType:  "supply",
		Params: map[string]interface{}{
			"from":   from.Hex(),
			"dst":    dst.Hex(),
			"amount": bigString(amount),
			"comet":  comet,
		},
	}, nil
}

func (d *CompoundV3Decoder) decodeWithdraw(event *types.ChainEvent, comet string) (*types.DecodedEvent, error) {
	if len(event.RawTopics) < 3 {
		return nil, fmt.Errorf("compound_v3 withdraw: expected 3 topics, got %d", len(event.RawTopics))
	}
	src := common.BytesToAddress(event.RawTopics[1].Bytes())
	to := common.BytesToAddress(event.RawTopics[2].Bytes())

	out, err := compoundV3ABI.Events["Withdraw"].Inputs.NonIndexed().Unpack(event.RawData)
	if err != nil {
		return nil, fmt.Errorf("compound_v3 withdraw: unpack: %w", err)
	}
	if len(out) != 1 {
		return nil, fmt.Errorf("compound_v3 withdraw: arity %d", len(out))
	}
	amount, _ := out[0].(*big.Int)

	return &types.DecodedEvent{
		ChainEvent: *event,
		Protocol:   d.Name(),
		EventType:  "withdraw",
		Params: map[string]interface{}{
			"src":    src.Hex(),
			"to":     to.Hex(),
			"amount": bigString(amount),
			"comet":  comet,
		},
	}, nil
}

func (d *CompoundV3Decoder) decodeAbsorb(event *types.ChainEvent, comet string) (*types.DecodedEvent, error) {
	if len(event.RawTopics) < 4 {
		return nil, fmt.Errorf("compound_v3 absorb: expected 4 topics, got %d", len(event.RawTopics))
	}
	absorber := common.BytesToAddress(event.RawTopics[1].Bytes())
	borrower := common.BytesToAddress(event.RawTopics[2].Bytes())
	asset := common.BytesToAddress(event.RawTopics[3].Bytes())

	out, err := compoundV3ABI.Events["AbsorbCollateral"].Inputs.NonIndexed().Unpack(event.RawData)
	if err != nil {
		return nil, fmt.Errorf("compound_v3 absorb: unpack: %w", err)
	}
	if len(out) != 2 {
		return nil, fmt.Errorf("compound_v3 absorb: arity %d", len(out))
	}
	collateralAbsorbed, _ := out[0].(*big.Int)
	usdValue, _ := out[1].(*big.Int)

	return &types.DecodedEvent{
		ChainEvent: *event,
		Protocol:   d.Name(),
		EventType:  "absorb",
		Params: map[string]interface{}{
			"absorber":            absorber.Hex(),
			"borrower":            borrower.Hex(),
			"asset":               asset.Hex(),
			"collateral_absorbed": bigString(collateralAbsorbed),
			"usd_value":           bigString(usdValue),
			"comet":               comet,
		},
	}, nil
}

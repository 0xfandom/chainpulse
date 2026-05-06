package protocols

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

// aaveV3PoolABIJSON is the Aave V3 Pool ABI for the events we decode.
// Sourced from the canonical Pool.sol; embedded inline so the decoder is
// self-contained.
const aaveV3PoolABIJSON = `[
  {"anonymous":false,"name":"Supply","type":"event","inputs":[
    {"indexed":true,"name":"reserve","type":"address"},
    {"indexed":false,"name":"user","type":"address"},
    {"indexed":true,"name":"onBehalfOf","type":"address"},
    {"indexed":false,"name":"amount","type":"uint256"},
    {"indexed":true,"name":"referralCode","type":"uint16"}
  ]},
  {"anonymous":false,"name":"Withdraw","type":"event","inputs":[
    {"indexed":true,"name":"reserve","type":"address"},
    {"indexed":true,"name":"user","type":"address"},
    {"indexed":true,"name":"to","type":"address"},
    {"indexed":false,"name":"amount","type":"uint256"}
  ]},
  {"anonymous":false,"name":"Borrow","type":"event","inputs":[
    {"indexed":true,"name":"reserve","type":"address"},
    {"indexed":false,"name":"user","type":"address"},
    {"indexed":true,"name":"onBehalfOf","type":"address"},
    {"indexed":false,"name":"amount","type":"uint256"},
    {"indexed":false,"name":"interestRateMode","type":"uint8"},
    {"indexed":false,"name":"borrowRate","type":"uint256"},
    {"indexed":true,"name":"referralCode","type":"uint16"}
  ]},
  {"anonymous":false,"name":"Repay","type":"event","inputs":[
    {"indexed":true,"name":"reserve","type":"address"},
    {"indexed":true,"name":"user","type":"address"},
    {"indexed":true,"name":"repayer","type":"address"},
    {"indexed":false,"name":"amount","type":"uint256"},
    {"indexed":false,"name":"useATokens","type":"bool"}
  ]},
  {"anonymous":false,"name":"LiquidationCall","type":"event","inputs":[
    {"indexed":true,"name":"collateralAsset","type":"address"},
    {"indexed":true,"name":"debtAsset","type":"address"},
    {"indexed":true,"name":"user","type":"address"},
    {"indexed":false,"name":"debtToCover","type":"uint256"},
    {"indexed":false,"name":"liquidatedCollateralAmount","type":"uint256"},
    {"indexed":false,"name":"liquidator","type":"address"},
    {"indexed":false,"name":"receiveAToken","type":"bool"}
  ]}
]`

var (
	AaveV3SupplySig          common.Hash
	AaveV3WithdrawSig        common.Hash
	AaveV3BorrowSig          common.Hash
	AaveV3RepaySig           common.Hash
	AaveV3LiquidationCallSig common.Hash

	aaveV3ABI abi.ABI
)

func init() {
	parsed, err := abi.JSON(strings.NewReader(aaveV3PoolABIJSON))
	if err != nil {
		panic(fmt.Sprintf("aave_v3: parse ABI: %v", err))
	}
	aaveV3ABI = parsed
	AaveV3SupplySig = parsed.Events["Supply"].ID
	AaveV3WithdrawSig = parsed.Events["Withdraw"].ID
	AaveV3BorrowSig = parsed.Events["Borrow"].ID
	AaveV3RepaySig = parsed.Events["Repay"].ID
	AaveV3LiquidationCallSig = parsed.Events["LiquidationCall"].ID
}

// AaveV3Decoder implements processor.ProtocolDecoder for Aave V3 Pool
// events.
type AaveV3Decoder struct{}

// NewAaveV3Decoder returns a stateless decoder.
func NewAaveV3Decoder() *AaveV3Decoder { return &AaveV3Decoder{} }

// Name implements ProtocolDecoder.
func (d *AaveV3Decoder) Name() string { return "aave_v3" }

// CanDecode implements ProtocolDecoder.
func (d *AaveV3Decoder) CanDecode(sig common.Hash) bool {
	switch sig {
	case AaveV3SupplySig, AaveV3WithdrawSig, AaveV3BorrowSig, AaveV3RepaySig, AaveV3LiquidationCallSig:
		return true
	}
	return false
}

// SupportedEvents implements ProtocolDecoder.
func (d *AaveV3Decoder) SupportedEvents() []common.Hash {
	return []common.Hash{
		AaveV3SupplySig, AaveV3WithdrawSig, AaveV3BorrowSig, AaveV3RepaySig, AaveV3LiquidationCallSig,
	}
}

// Decode implements ProtocolDecoder.
func (d *AaveV3Decoder) Decode(event *types.ChainEvent) (*types.DecodedEvent, error) {
	if event == nil {
		return nil, fmt.Errorf("aave_v3: nil event")
	}
	if len(event.RawTopics) == 0 {
		return nil, fmt.Errorf("aave_v3: no topics")
	}
	switch event.RawTopics[0] {
	case AaveV3SupplySig:
		return d.decodeSupply(event)
	case AaveV3WithdrawSig:
		return d.decodeWithdraw(event)
	case AaveV3BorrowSig:
		return d.decodeBorrow(event)
	case AaveV3RepaySig:
		return d.decodeRepay(event)
	case AaveV3LiquidationCallSig:
		return d.decodeLiquidation(event)
	default:
		return nil, fmt.Errorf("aave_v3: unsupported signature %s", event.RawTopics[0].Hex())
	}
}

func (d *AaveV3Decoder) decodeSupply(event *types.ChainEvent) (*types.DecodedEvent, error) {
	if len(event.RawTopics) < 4 {
		return nil, fmt.Errorf("aave_v3 supply: expected 4 topics, got %d", len(event.RawTopics))
	}
	reserve := common.BytesToAddress(event.RawTopics[1].Bytes())
	onBehalfOf := common.BytesToAddress(event.RawTopics[2].Bytes())
	referralCode := new(big.Int).SetBytes(event.RawTopics[3].Bytes())

	out, err := aaveV3ABI.Events["Supply"].Inputs.NonIndexed().Unpack(event.RawData)
	if err != nil {
		return nil, fmt.Errorf("aave_v3 supply: unpack: %w", err)
	}
	if len(out) != 2 {
		return nil, fmt.Errorf("aave_v3 supply: arity %d", len(out))
	}
	user, _ := out[0].(common.Address)
	amount, _ := out[1].(*big.Int)

	return &types.DecodedEvent{
		ChainEvent: *event,
		Protocol:   d.Name(),
		EventType:  "supply",
		Params: map[string]interface{}{
			"reserve":       reserve.Hex(),
			"user":          user.Hex(),
			"on_behalf_of":  onBehalfOf.Hex(),
			"amount":        bigString(amount),
			"referral_code": referralCode.String(),
		},
	}, nil
}

func (d *AaveV3Decoder) decodeWithdraw(event *types.ChainEvent) (*types.DecodedEvent, error) {
	if len(event.RawTopics) < 4 {
		return nil, fmt.Errorf("aave_v3 withdraw: expected 4 topics, got %d", len(event.RawTopics))
	}
	reserve := common.BytesToAddress(event.RawTopics[1].Bytes())
	user := common.BytesToAddress(event.RawTopics[2].Bytes())
	to := common.BytesToAddress(event.RawTopics[3].Bytes())

	out, err := aaveV3ABI.Events["Withdraw"].Inputs.NonIndexed().Unpack(event.RawData)
	if err != nil {
		return nil, fmt.Errorf("aave_v3 withdraw: unpack: %w", err)
	}
	if len(out) != 1 {
		return nil, fmt.Errorf("aave_v3 withdraw: arity %d", len(out))
	}
	amount, _ := out[0].(*big.Int)

	return &types.DecodedEvent{
		ChainEvent: *event,
		Protocol:   d.Name(),
		EventType:  "withdraw",
		Params: map[string]interface{}{
			"reserve": reserve.Hex(),
			"user":    user.Hex(),
			"to":      to.Hex(),
			"amount":  bigString(amount),
		},
	}, nil
}

func (d *AaveV3Decoder) decodeBorrow(event *types.ChainEvent) (*types.DecodedEvent, error) {
	if len(event.RawTopics) < 4 {
		return nil, fmt.Errorf("aave_v3 borrow: expected 4 topics, got %d", len(event.RawTopics))
	}
	reserve := common.BytesToAddress(event.RawTopics[1].Bytes())
	onBehalfOf := common.BytesToAddress(event.RawTopics[2].Bytes())
	referralCode := new(big.Int).SetBytes(event.RawTopics[3].Bytes())

	out, err := aaveV3ABI.Events["Borrow"].Inputs.NonIndexed().Unpack(event.RawData)
	if err != nil {
		return nil, fmt.Errorf("aave_v3 borrow: unpack: %w", err)
	}
	if len(out) != 4 {
		return nil, fmt.Errorf("aave_v3 borrow: arity %d", len(out))
	}
	user, _ := out[0].(common.Address)
	amount, _ := out[1].(*big.Int)
	interestRateMode, _ := out[2].(uint8)
	borrowRate, _ := out[3].(*big.Int)

	return &types.DecodedEvent{
		ChainEvent: *event,
		Protocol:   d.Name(),
		EventType:  "borrow",
		Params: map[string]interface{}{
			"reserve":            reserve.Hex(),
			"user":               user.Hex(),
			"on_behalf_of":       onBehalfOf.Hex(),
			"amount":             bigString(amount),
			"interest_rate_mode": uint64(interestRateMode),
			"borrow_rate":        bigString(borrowRate),
			"referral_code":      referralCode.String(),
		},
	}, nil
}

func (d *AaveV3Decoder) decodeRepay(event *types.ChainEvent) (*types.DecodedEvent, error) {
	if len(event.RawTopics) < 4 {
		return nil, fmt.Errorf("aave_v3 repay: expected 4 topics, got %d", len(event.RawTopics))
	}
	reserve := common.BytesToAddress(event.RawTopics[1].Bytes())
	user := common.BytesToAddress(event.RawTopics[2].Bytes())
	repayer := common.BytesToAddress(event.RawTopics[3].Bytes())

	out, err := aaveV3ABI.Events["Repay"].Inputs.NonIndexed().Unpack(event.RawData)
	if err != nil {
		return nil, fmt.Errorf("aave_v3 repay: unpack: %w", err)
	}
	if len(out) != 2 {
		return nil, fmt.Errorf("aave_v3 repay: arity %d", len(out))
	}
	amount, _ := out[0].(*big.Int)
	useATokens, _ := out[1].(bool)

	return &types.DecodedEvent{
		ChainEvent: *event,
		Protocol:   d.Name(),
		EventType:  "repay",
		Params: map[string]interface{}{
			"reserve":     reserve.Hex(),
			"user":        user.Hex(),
			"repayer":     repayer.Hex(),
			"amount":      bigString(amount),
			"use_atokens": useATokens,
		},
	}, nil
}

func (d *AaveV3Decoder) decodeLiquidation(event *types.ChainEvent) (*types.DecodedEvent, error) {
	if len(event.RawTopics) < 4 {
		return nil, fmt.Errorf("aave_v3 liquidation: expected 4 topics, got %d", len(event.RawTopics))
	}
	collateralAsset := common.BytesToAddress(event.RawTopics[1].Bytes())
	debtAsset := common.BytesToAddress(event.RawTopics[2].Bytes())
	user := common.BytesToAddress(event.RawTopics[3].Bytes())

	out, err := aaveV3ABI.Events["LiquidationCall"].Inputs.NonIndexed().Unpack(event.RawData)
	if err != nil {
		return nil, fmt.Errorf("aave_v3 liquidation: unpack: %w", err)
	}
	if len(out) != 4 {
		return nil, fmt.Errorf("aave_v3 liquidation: arity %d", len(out))
	}
	debtToCover, _ := out[0].(*big.Int)
	liquidatedCollateralAmount, _ := out[1].(*big.Int)
	liquidator, _ := out[2].(common.Address)
	receiveAToken, _ := out[3].(bool)

	return &types.DecodedEvent{
		ChainEvent: *event,
		Protocol:   d.Name(),
		EventType:  "liquidation",
		Params: map[string]interface{}{
			"collateral_asset":             collateralAsset.Hex(),
			"debt_asset":                   debtAsset.Hex(),
			"user":                         user.Hex(),
			"debt_to_cover":                bigString(debtToCover),
			"liquidated_collateral_amount": bigString(liquidatedCollateralAmount),
			"liquidator":                   liquidator.Hex(),
			"receive_atoken":               receiveAToken,
		},
	}, nil
}

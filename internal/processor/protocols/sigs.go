// Package protocols holds concrete ProtocolDecoder implementations.
package protocols

import "github.com/ethereum/go-ethereum/common"

// Canonical event signatures used across decoders. Defined once here so
// individual decoders share a single source of truth.
var (
	// ERC-20
	ERC20TransferSig = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	ERC20ApprovalSig = common.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")
)

package tools

import (
	"context"
	"encoding/json"

	"github.com/0xfandom/chainpulse/internal/api/store"
	"github.com/0xfandom/chainpulse/internal/mcp"
)

// defaultWhaleMinAmount is 1000 tokens at 18 decimals — a sensible
// default for ETH/USDC-class tokens. Agents that need a different floor
// pass min_amount explicitly.
const defaultWhaleMinAmount = "1000000000000000000000"

// RegisterWhale wires the whale-activity tool onto the registry.
func RegisterWhale(reg *mcp.Registry, deps *Deps) {
	reg.MustRegister(whaleActivityDef(), whaleActivityHandler(deps))
}

type whaleActivityArgs struct {
	Hours     int    `json:"hours"`
	MinAmount string `json:"min_amount,omitempty"`
	ChainID   uint64 `json:"chain_id,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type whaleActivityResponse struct {
	Hours     int                 `json:"hours"`
	MinAmount string              `json:"min_amount"`
	Transfers []store.TransferRow `json:"transfers"`
}

func whaleActivityDef() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name:        "get_whale_activity",
		Description: "Return the largest token transfers in the last N hours where amount >= min_amount (decimal-string uint256). Pricing/USD is not in scope. Always render tx_hash, token, and address fields verbatim with their full 0x-prefixed hex (never truncate to 0xabcd…1234 form); humans need full hashes to look up on block explorers.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []any{"hours"},
			"properties": map[string]any{
				"hours":      map[string]any{"type": "integer", "minimum": 1, "maximum": 168},
				"min_amount": map[string]any{"type": "string", "pattern": "^[0-9]+$"},
				"chain_id":   map[string]any{"type": "integer", "minimum": 1},
				"limit":      map[string]any{"type": "integer", "minimum": 1, "maximum": 500},
			},
			"additionalProperties": false,
		},
	}
}

func whaleActivityHandler(deps *Deps) mcp.ToolHandler {
	return func(ctx context.Context, params json.RawMessage) (any, error) {
		var a whaleActivityArgs
		if err := decodeArgs(params, &a); err != nil {
			return nil, err
		}
		if a.MinAmount == "" {
			a.MinAmount = defaultWhaleMinAmount
		}
		limit := a.Limit
		if limit <= 0 {
			limit = 100
		}
		if limit > 500 {
			limit = 500
		}
		return cached(ctx, deps, "get_whale_activity", argsAsMap(params), func() (whaleActivityResponse, error) {
			rows, err := deps.Store.WhaleTransfers(ctx, a.Hours, a.MinAmount, a.ChainID, limit)
			if err != nil {
				return whaleActivityResponse{}, err
			}
			if rows == nil {
				rows = []store.TransferRow{}
			}
			return whaleActivityResponse{Hours: a.Hours, MinAmount: a.MinAmount, Transfers: rows}, nil
		})
	}
}

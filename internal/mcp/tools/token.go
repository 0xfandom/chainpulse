package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/0xfandom/chainpulse/internal/api/store"
	"github.com/0xfandom/chainpulse/internal/mcp"
)

// RegisterToken wires the token-scoped tools onto the registry.
func RegisterToken(reg *mcp.Registry, deps *Deps) {
	reg.MustRegister(tokenTransfersDef(), tokenTransfersHandler(deps))
}

type tokenTransfersArgs struct {
	Address string `json:"address"`
	Limit   int    `json:"limit,omitempty"`
	ChainID uint64 `json:"chain_id,omitempty"`
}

// tokenTransfersResponse pre-filters the address-side store result down
// to the requested chain (when given).
type tokenTransfersResponse struct {
	Address   string              `json:"address"`
	Transfers []store.TransferRow `json:"transfers"`
}

func tokenTransfersDef() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name:        "get_token_transfers",
		Description: "Return recent ERC-20 transfers where the address is the token contract or one of the parties (from/to). Always render tx_hash, token, and address fields verbatim with their full 0x-prefixed hex (never truncate to 0xabcd…1234 form); humans need full hashes to look up on block explorers.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []any{"address"},
			"properties": map[string]any{
				"address":  map[string]any{"type": "string", "pattern": addressPattern},
				"limit":    map[string]any{"type": "integer", "minimum": 1, "maximum": 500},
				"chain_id": map[string]any{"type": "integer", "minimum": 1},
			},
			"additionalProperties": false,
		},
	}
}

func tokenTransfersHandler(deps *Deps) mcp.ToolHandler {
	return func(ctx context.Context, params json.RawMessage) (any, error) {
		var a tokenTransfersArgs
		if err := decodeArgs(params, &a); err != nil {
			return nil, err
		}
		a.Address = strings.ToLower(a.Address)
		limit := a.Limit
		if limit <= 0 {
			limit = 50
		}
		if limit > 500 {
			limit = 500
		}
		return cachedWithTTL(ctx, deps, "get_token_transfers", argsAsMap(params), 0, func() (tokenTransfersResponse, error) {
			rows, err := deps.Store.TokenTransfersByAddress(ctx, a.Address, limit)
			if err != nil {
				return tokenTransfersResponse{}, err
			}
			out := make([]store.TransferRow, 0, len(rows))
			for _, r := range rows {
				if a.ChainID != 0 && r.ChainID != a.ChainID {
					continue
				}
				out = append(out, r)
			}
			return tokenTransfersResponse{Address: a.Address, Transfers: out}, nil
		})
	}
}

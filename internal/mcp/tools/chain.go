package tools

import (
	"context"
	"encoding/json"

	"github.com/0xfandom/chainpulse/internal/api/store"
	"github.com/0xfandom/chainpulse/internal/mcp"
)

// RegisterChain wires the chain-scoped tools onto the registry.
func RegisterChain(reg *mcp.Registry, deps *Deps) {
	reg.MustRegister(latestBlockDef(), latestBlockHandler(deps))
}

type latestBlockArgs struct {
	ChainID uint64 `json:"chain_id,omitempty"`
}

type latestBlockResponse struct {
	Chains []store.LatestBlockRow `json:"chains"`
}

func latestBlockDef() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name:        "get_latest_block",
		Description: "Return the most recently indexed block per chain (chain_id, block_number, timestamp). With chain_id, scopes to one chain; without, returns every chain currently in storage. Always render block numbers and timestamps verbatim; humans use them to verify indexer freshness.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"chain_id": map[string]any{"type": "integer", "minimum": 1},
			},
			"additionalProperties": false,
		},
	}
}

func latestBlockHandler(deps *Deps) mcp.ToolHandler {
	return func(ctx context.Context, params json.RawMessage) (any, error) {
		var a latestBlockArgs
		if err := decodeArgs(params, &a); err != nil {
			return nil, err
		}
		return cached(ctx, deps, "get_latest_block", argsAsMap(params), func() (latestBlockResponse, error) {
			if a.ChainID != 0 {
				row, err := deps.Store.LatestBlock(ctx, a.ChainID)
				if err != nil {
					return latestBlockResponse{}, err
				}
				if row == nil {
					return latestBlockResponse{Chains: []store.LatestBlockRow{}}, nil
				}
				return latestBlockResponse{Chains: []store.LatestBlockRow{*row}}, nil
			}
			rows, err := deps.Store.LatestBlocksAllChains(ctx)
			if err != nil {
				return latestBlockResponse{}, err
			}
			if rows == nil {
				rows = []store.LatestBlockRow{}
			}
			return latestBlockResponse{Chains: rows}, nil
		})
	}
}

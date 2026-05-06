package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/0xfandom/chainpulse/internal/api/store"
	"github.com/0xfandom/chainpulse/internal/mcp"
)

// RegisterDeFi wires the protocol-scoped tools onto the registry.
func RegisterDeFi(reg *mcp.Registry, deps *Deps) {
	reg.MustRegister(defiPositionsDef(), defiPositionsHandler(deps))
	reg.MustRegister(protocolStatsDef(), protocolStatsHandler(deps))
}

// ---------- get_defi_positions ----------

type defiPositionsArgs struct {
	Wallet   string `json:"wallet"`
	Protocol string `json:"protocol"`
	ChainID  uint64 `json:"chain_id,omitempty"`
}

func defiPositionsDef() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name:        "get_defi_positions",
		Description: "Return positions for a wallet inside a single protocol; optional chain_id scope.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []any{"wallet", "protocol"},
			"properties": map[string]any{
				"wallet":   map[string]any{"type": "string", "pattern": addressPattern},
				"protocol": map[string]any{"type": "string", "enum": []any{"erc20", "uniswap_v3", "aave_v3", "compound_v3"}},
				"chain_id": map[string]any{"type": "integer", "minimum": 1},
			},
			"additionalProperties": false,
		},
	}
}

func defiPositionsHandler(deps *Deps) mcp.ToolHandler {
	return func(ctx context.Context, params json.RawMessage) (any, error) {
		var a defiPositionsArgs
		if err := decodeArgs(params, &a); err != nil {
			return nil, err
		}
		a.Wallet = strings.ToLower(a.Wallet)
		if _, ok := validProtocols[a.Protocol]; !ok {
			return nil, fmt.Errorf("unknown protocol %q", a.Protocol)
		}
		return cached(ctx, deps, "get_defi_positions", argsAsMap(params), func() ([]store.PositionRow, error) {
			rows, err := deps.Store.WalletDefiPositions(ctx, a.Wallet, 200)
			if err != nil {
				return nil, err
			}
			out := make([]store.PositionRow, 0, len(rows))
			for _, r := range rows {
				if r.Protocol != a.Protocol {
					continue
				}
				if a.ChainID != 0 && r.ChainID != a.ChainID {
					continue
				}
				out = append(out, r)
			}
			return out, nil
		})
	}
}

// ---------- get_protocol_stats ----------

type protocolStatsArgs struct {
	Protocol string `json:"protocol"`
	ChainID  uint64 `json:"chain_id,omitempty"`
}

func protocolStatsDef() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name:        "get_protocol_stats",
		Description: "Return 24h volume + unique-user counts per chain for a protocol.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []any{"protocol"},
			"properties": map[string]any{
				"protocol": map[string]any{"type": "string", "enum": []any{"erc20", "uniswap_v3", "aave_v3", "compound_v3"}},
				"chain_id": map[string]any{"type": "integer", "minimum": 1},
			},
			"additionalProperties": false,
		},
	}
}

func protocolStatsHandler(deps *Deps) mcp.ToolHandler {
	return func(ctx context.Context, params json.RawMessage) (any, error) {
		var a protocolStatsArgs
		if err := decodeArgs(params, &a); err != nil {
			return nil, err
		}
		if _, ok := validProtocols[a.Protocol]; !ok {
			return nil, fmt.Errorf("unknown protocol %q", a.Protocol)
		}
		return cached(ctx, deps, "get_protocol_stats", argsAsMap(params), func() ([]store.ProtocolStat, error) {
			rows, err := deps.Store.ProtocolStats(ctx, a.Protocol)
			if err != nil {
				return nil, err
			}
			out := make([]store.ProtocolStat, 0, len(rows))
			for _, r := range rows {
				if a.ChainID != 0 && r.ChainID != a.ChainID {
					continue
				}
				out = append(out, r)
			}
			return out, nil
		})
	}
}

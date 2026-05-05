package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/0xfandom/chainpulse/internal/api/store"
	"github.com/0xfandom/chainpulse/internal/mcp"
)

const addressPattern = "^0x[0-9a-fA-F]{40}$"

// validProtocols is the whitelist shared with the REST + gRPC layers.
var validProtocols = map[string]struct{}{
	"erc20": {}, "uniswap_v3": {}, "aave_v3": {}, "compound_v3": {},
}

// RegisterWallet wires the three wallet-scoped tools onto the registry.
func RegisterWallet(reg *mcp.Registry, deps *Deps) {
	reg.MustRegister(walletPositionsDef(), walletPositionsHandler(deps))
	reg.MustRegister(walletBalancesDef(), walletBalancesHandler(deps))
	reg.MustRegister(walletHistoryDef(), walletHistoryHandler(deps))
}

// ---------- get_wallet_positions ----------

type walletPositionsArgs struct {
	Wallet   string `json:"wallet"`
	ChainID  uint64 `json:"chain_id,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

func walletPositionsDef() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name:        "get_wallet_positions",
		Description: "Return active DeFi positions (supply, borrow, withdraw, repay) for a wallet across indexed chains.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []any{"wallet"},
			"properties": map[string]any{
				"wallet":   map[string]any{"type": "string", "pattern": addressPattern},
				"chain_id": map[string]any{"type": "integer", "minimum": 1},
				"protocol": map[string]any{"type": "string", "enum": []any{"erc20", "uniswap_v3", "aave_v3", "compound_v3"}},
			},
			"additionalProperties": false,
		},
	}
}

func walletPositionsHandler(deps *Deps) mcp.ToolHandler {
	return func(ctx context.Context, params json.RawMessage) (any, error) {
		var a walletPositionsArgs
		if err := decodeArgs(params, &a); err != nil {
			return nil, err
		}
		a.Wallet = strings.ToLower(a.Wallet)
		if a.Protocol != "" {
			if _, ok := validProtocols[a.Protocol]; !ok {
				return nil, fmt.Errorf("unknown protocol %q", a.Protocol)
			}
		}
		return cached(ctx, deps, "get_wallet_positions", argsAsMap(params), func() ([]store.PositionRow, error) {
			rows, err := deps.Store.WalletDefiPositions(ctx, a.Wallet, 200)
			if err != nil {
				return nil, err
			}
			out := make([]store.PositionRow, 0, len(rows))
			for _, r := range rows {
				if a.ChainID != 0 && r.ChainID != a.ChainID {
					continue
				}
				if a.Protocol != "" && r.Protocol != a.Protocol {
					continue
				}
				out = append(out, r)
			}
			return out, nil
		})
	}
}

// ---------- get_wallet_balances ----------

type walletBalancesArgs struct {
	Wallet  string `json:"wallet"`
	ChainID uint64 `json:"chain_id,omitempty"`
}

func walletBalancesDef() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name:        "get_wallet_balances",
		Description: "Return per-token balances for a wallet. With chain_id, scopes to that chain; without, returns balances across every cached chain.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []any{"wallet"},
			"properties": map[string]any{
				"wallet":   map[string]any{"type": "string", "pattern": addressPattern},
				"chain_id": map[string]any{"type": "integer", "minimum": 1},
			},
			"additionalProperties": false,
		},
	}
}

// walletBalancesResponse is the JSON shape returned to the agent.
type walletBalancesResponse struct {
	Wallet   string                       `json:"wallet"`
	ChainID  uint64                       `json:"chain_id,omitempty"`
	Balances map[string]string            `json:"balances,omitempty"`
	ByChain  map[uint64]map[string]string `json:"by_chain,omitempty"`
}

func walletBalancesHandler(deps *Deps) mcp.ToolHandler {
	return func(ctx context.Context, params json.RawMessage) (any, error) {
		var a walletBalancesArgs
		if err := decodeArgs(params, &a); err != nil {
			return nil, err
		}
		a.Wallet = strings.ToLower(a.Wallet)
		if deps.Cache == nil {
			if a.ChainID != 0 {
				return walletBalancesResponse{Wallet: a.Wallet, ChainID: a.ChainID, Balances: map[string]string{}}, nil
			}
			return walletBalancesResponse{Wallet: a.Wallet, ByChain: map[uint64]map[string]string{}}, nil
		}
		return cached(ctx, deps, "get_wallet_balances", argsAsMap(params), func() (walletBalancesResponse, error) {
			if a.ChainID != 0 {
				balances, err := deps.Cache.GetBalances(ctx, a.Wallet, a.ChainID)
				if err != nil {
					return walletBalancesResponse{}, err
				}
				if balances == nil {
					balances = map[string]string{}
				}
				return walletBalancesResponse{Wallet: a.Wallet, ChainID: a.ChainID, Balances: balances}, nil
			}
			byChain, err := deps.Cache.GetAllBalances(ctx, a.Wallet)
			if err != nil {
				return walletBalancesResponse{}, err
			}
			if byChain == nil {
				byChain = map[uint64]map[string]string{}
			}
			return walletBalancesResponse{Wallet: a.Wallet, ByChain: byChain}, nil
		})
	}
}

// ---------- get_wallet_history ----------

type walletHistoryArgs struct {
	Wallet string `json:"wallet"`
	Limit  int    `json:"limit,omitempty"`
}

func walletHistoryDef() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name:        "get_wallet_history",
		Description: "Return the most recent decoded events involving a wallet across all indexed chains.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []any{"wallet"},
			"properties": map[string]any{
				"wallet": map[string]any{"type": "string", "pattern": addressPattern},
				"limit":  map[string]any{"type": "integer", "minimum": 1, "maximum": 500},
			},
			"additionalProperties": false,
		},
	}
}

func walletHistoryHandler(deps *Deps) mcp.ToolHandler {
	return func(ctx context.Context, params json.RawMessage) (any, error) {
		var a walletHistoryArgs
		if err := decodeArgs(params, &a); err != nil {
			return nil, err
		}
		a.Wallet = strings.ToLower(a.Wallet)
		limit := a.Limit
		if limit <= 0 {
			limit = 100
		}
		if limit > 500 {
			limit = 500
		}
		return cached(ctx, deps, "get_wallet_history", argsAsMap(params), func() ([]store.HistoryRow, error) {
			rows, err := deps.Store.WalletHistory(ctx, a.Wallet, limit)
			if err != nil {
				return nil, err
			}
			if rows == nil {
				rows = []store.HistoryRow{}
			}
			return rows, nil
		})
	}
}

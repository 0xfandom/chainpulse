package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/0xfandom/chainpulse/internal/api/store"
)

const jsonContentType = "application/json; charset=utf-8"

// Wallet handler TTLs / limits.
const (
	walletPositionsTTL = 30 * time.Second
	walletBalancesTTL  = 10 * time.Second
	walletHistoryTTL   = 30 * time.Second
	walletPositionsLim = 200
	walletHistoryDef   = 100
	walletHistoryMax   = 500
)

// WalletHandlers exposes the /wallet/* endpoints.
type WalletHandlers struct {
	deps HandlerDeps
}

// NewWalletHandlers constructs a WalletHandlers.
func NewWalletHandlers(deps HandlerDeps) *WalletHandlers { return &WalletHandlers{deps: deps} }

// Register attaches the routes onto group.
func (h *WalletHandlers) Register(group *gin.RouterGroup) {
	group.GET("/wallet/:address/positions", h.Positions)
	group.GET("/wallet/:address/balances", h.Balances)
	group.GET("/wallet/:address/history", h.History)
}

// Positions: GET /v1/wallet/:address/positions
//
// Returns DeFi positions (supply/borrow/withdraw/repay events) for the
// wallet. Cached for 30s.
func (h *WalletHandlers) Positions(c *gin.Context) {
	addr, err := normalizeAddress(c.Param("address"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	limit := walletPositionsLim
	cacheKey := fmt.Sprintf("api:positions:%s:%d", addr, limit)

	raw, _, err := store.AsideRaw(c.Request.Context(), h.deps.L1, h.deps.Cache, cacheKey, walletPositionsTTL,
		func(ctx context.Context) ([]byte, error) {
			rows, qErr := h.deps.Store.WalletDefiPositions(ctx, addr, limit)
			if qErr != nil {
				return nil, qErr
			}
			return json.Marshal(gin.H{"wallet": addr, "positions": rows})
		})
	if err != nil {
		respondInternal(c, err, "wallet positions failed")
		return
	}
	c.Data(http.StatusOK, jsonContentType, raw)
}

// Balances: GET /v1/wallet/:address/balances?chain_id=
//
// Reads the ClickHouse wallet_balances materialized view (Int256 deltas)
// — authoritative; no int64 clamping. The Redis HINCRBY-backed hot path
// is unreliable for tokens whose raw amounts exceed int64 (any 18-decimal
// token); replacing it is a Day-2 follow-up.
func (h *WalletHandlers) Balances(c *gin.Context) {
	addr, err := normalizeAddress(c.Param("address"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	chainID, ok := parseChainID(c.Query("chain_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain_id"})
		return
	}
	if chainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chain_id is required for balances"})
		return
	}
	balances, err := h.deps.Store.WalletBalancesByChain(c.Request.Context(), addr, chainID)
	if err != nil {
		respondInternal(c, err, "wallet balances failed")
		return
	}
	if balances == nil {
		balances = map[string]string{}
	}
	c.JSON(http.StatusOK, gin.H{"wallet": addr, "chain_id": chainID, "balances": balances})
}

// History: GET /v1/wallet/:address/history?limit=
//
// Returns the most recent decoded events for the wallet. limit defaults
// to 100, max 500.
func (h *WalletHandlers) History(c *gin.Context) {
	addr, err := normalizeAddress(c.Param("address"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	limit := clampLimit(c.Query("limit"), walletHistoryDef, walletHistoryMax)
	cacheKey := fmt.Sprintf("api:history:%s:%d", addr, limit)

	raw, _, err := store.AsideRaw(c.Request.Context(), h.deps.L1, h.deps.Cache, cacheKey, walletHistoryTTL,
		func(ctx context.Context) ([]byte, error) {
			rows, qErr := h.deps.Store.WalletHistory(ctx, addr, limit)
			if qErr != nil {
				return nil, qErr
			}
			return json.Marshal(gin.H{"wallet": addr, "limit": limit, "events": rows})
		})
	if err != nil {
		respondInternal(c, err, "wallet history failed")
		return
	}
	c.Data(http.StatusOK, jsonContentType, raw)
}

// respondInternal logs and responds with 500. Cache write failures don't
// surface to clients — they are visible in logs / metrics.
func respondInternal(c *gin.Context, err error, msg string) {
	if errors.Is(err, context.Canceled) {
		c.AbortWithStatus(499) // client closed request
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
}

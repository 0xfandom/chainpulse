package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/0xfandom/chainpulse/internal/api/store"
)

const (
	tokenTransfersTTL = 60 * time.Second
	tokenTransfersDef = 100
	tokenTransfersMax = 500
)

// TokenHandlers exposes the /token/* endpoints.
type TokenHandlers struct{ deps HandlerDeps }

// NewTokenHandlers constructs a TokenHandlers.
func NewTokenHandlers(deps HandlerDeps) *TokenHandlers { return &TokenHandlers{deps: deps} }

// Register attaches the routes onto group.
func (h *TokenHandlers) Register(group *gin.RouterGroup) {
	group.GET("/token/:address/transfers", h.Transfers)
}

// Transfers: GET /v1/token/:address/transfers?limit=
func (h *TokenHandlers) Transfers(c *gin.Context) {
	addr, err := normalizeAddress(c.Param("address"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	limit := clampLimit(c.Query("limit"), tokenTransfersDef, tokenTransfersMax)
	cacheKey := fmt.Sprintf("api:token_transfers:%s:%d", addr, limit)

	raw, _, err := store.AsideRaw(c.Request.Context(), h.deps.L1, h.deps.Cache, cacheKey, tokenTransfersTTL,
		func(ctx context.Context) ([]byte, error) {
			rows, qErr := h.deps.Store.TokenTransfers(ctx, addr, limit)
			if qErr != nil {
				return nil, qErr
			}
			return json.Marshal(gin.H{"token": addr, "limit": limit, "transfers": rows})
		})
	if err != nil {
		respondInternal(c, err, "token transfers failed")
		return
	}
	c.Data(http.StatusOK, jsonContentType, raw)
}

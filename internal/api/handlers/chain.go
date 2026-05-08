package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/0xfandom/chainpulse/internal/api/store"
)

const (
	chainBlocksTTL = 0 * time.Second // bypass cache; latest-block must always be fresh
	chainBlocksDef = 50
	chainBlocksMax = 200
)

// ChainHandlers exposes the /chain/* endpoints.
type ChainHandlers struct{ deps HandlerDeps }

// NewChainHandlers constructs a ChainHandlers.
func NewChainHandlers(deps HandlerDeps) *ChainHandlers { return &ChainHandlers{deps: deps} }

// Register attaches the routes onto group.
func (h *ChainHandlers) Register(group *gin.RouterGroup) {
	group.GET("/chain/:id/blocks", h.Blocks)
}

// Blocks: GET /v1/chain/:id/blocks?limit=
func (h *ChainHandlers) Blocks(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain id"})
		return
	}
	limit := clampLimit(c.Query("limit"), chainBlocksDef, chainBlocksMax)
	cacheKey := fmt.Sprintf("api:chain_blocks:%d:%d", id, limit)

	raw, _, err := store.AsideRaw(c.Request.Context(), h.deps.L1, h.deps.Cache, cacheKey, chainBlocksTTL,
		func(ctx context.Context) ([]byte, error) {
			rows, qErr := h.deps.Store.ChainRecentBlocks(ctx, id, limit)
			if qErr != nil {
				return nil, qErr
			}
			return json.Marshal(gin.H{"chain_id": id, "limit": limit, "blocks": rows})
		})
	if err != nil {
		respondInternal(c, err, "chain blocks failed")
		return
	}
	c.Data(http.StatusOK, jsonContentType, raw)
}

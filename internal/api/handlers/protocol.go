package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/0xfandom/chainpulse/internal/api/store"
)

const (
	protocolStatsTTL = 5 * time.Minute
)

// validProtocols is the closed set the API exposes. Unknown names return
// 404 so clients can tell "we don't index this protocol" from "we have no
// data for this name yet".
var validProtocols = map[string]struct{}{
	"erc20":       {},
	"uniswap_v3":  {},
	"aave_v3":     {},
	"compound_v3": {},
}

// ProtocolHandlers exposes the /protocol/* endpoints.
type ProtocolHandlers struct{ deps HandlerDeps }

// NewProtocolHandlers constructs a ProtocolHandlers.
func NewProtocolHandlers(deps HandlerDeps) *ProtocolHandlers {
	return &ProtocolHandlers{deps: deps}
}

// Register attaches the routes onto group.
func (h *ProtocolHandlers) Register(group *gin.RouterGroup) {
	group.GET("/protocol/:name/stats", h.Stats)
}

// Stats: GET /v1/protocol/:name/stats?chain_id=
func (h *ProtocolHandlers) Stats(c *gin.Context) {
	name := c.Param("name")
	if _, ok := validProtocols[name]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown protocol"})
		return
	}
	chainID, ok := parseChainID(c.Query("chain_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain_id"})
		return
	}
	cacheKey := fmt.Sprintf("api:protocol_stats:%s:%d", name, chainID)

	stats, _, err := store.Aside(c.Request.Context(), h.deps.Cache, cacheKey, protocolStatsTTL,
		func(ctx context.Context) ([]store.ProtocolStat, error) {
			return h.deps.Store.ProtocolStats(ctx, name)
		})
	if err != nil {
		respondInternal(c, err, "protocol stats failed")
		return
	}
	if chainID > 0 {
		filtered := stats[:0]
		for _, s := range stats {
			if s.ChainID == chainID {
				filtered = append(filtered, s)
			}
		}
		stats = filtered
	}
	c.JSON(http.StatusOK, gin.H{"protocol": name, "stats": stats})
}

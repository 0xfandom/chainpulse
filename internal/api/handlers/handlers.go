// Package handlers contains the api binary's REST handlers, organized by
// resource: wallet, token, protocol, chain. Each handler delegates to
// ReadStore (analytical) + ReadCache (hot path) via the raw-bytes
// cache-aside helper in store.AsideRaw, with an in-process L1 LRU
// (store.L1Cache) keyed identically to Redis.
package handlers

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/0xfandom/chainpulse/internal/api/store"
)

// addressRegex matches 0x + 40 lowercase hex (case-insensitive at input
// time; we lowercase before lookup so cache keys stay consistent).
var addressRegex = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

// errBadAddress is returned by the validators below.
var errBadAddress = errors.New("invalid address format (expected 0x + 40 hex)")

// normalizeAddress validates + lowercases.
func normalizeAddress(addr string) (string, error) {
	if !addressRegex.MatchString(addr) {
		return "", errBadAddress
	}
	return strings.ToLower(addr), nil
}

// clampLimit parses ?limit= and clamps to [1, max]. Returns def if absent.
func clampLimit(raw string, def, max int) int {
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		return def
	}
	if v > max {
		return max
	}
	return v
}

// parseChainID parses ?chain_id= as uint64. Returns 0, true if absent;
// 0, false on parse error.
func parseChainID(raw string) (uint64, bool) {
	if raw == "" {
		return 0, true
	}
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// HandlerDeps groups the dependencies every handler needs.
type HandlerDeps struct {
	Store *store.ReadStore
	Cache *store.ReadCache
	L1    *store.L1Cache
}

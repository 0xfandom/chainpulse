// Package tools holds the concrete MCP tool implementations. Each tool
// wraps the existing api ReadStore + ReadCache so query logic isn't
// duplicated from the REST/gRPC handlers.
package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/0xfandom/chainpulse/internal/api/store"
	"github.com/0xfandom/chainpulse/internal/monitor"
)

// Deps is the dependency bundle every tool registrar takes. Constructed
// once at boot and shared across tools.
type Deps struct {
	Store *store.ReadStore
	Cache *store.ReadCache
	TTL   time.Duration
}

// cacheKey hashes the tool name + sorted arguments map into a stable
// key so two equivalent calls produce the same Redis key regardless of
// argument order.
func cacheKey(tool string, args map[string]any) string {
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	canon := make(map[string]any, len(args))
	for _, k := range keys {
		canon[k] = args[k]
	}
	body, _ := json.Marshal(canon)
	sum := sha256.Sum256(body)
	return "mcp:" + tool + ":" + hex.EncodeToString(sum[:8])
}

// cached runs loader on cache miss and stores the JSON-encoded result
// under the derived key with the configured TTL. On cache failure (Redis
// down) the loader still runs — caching is opportunistic.
func cached[T any](ctx context.Context, deps *Deps, tool string, args map[string]any, loader func() (T, error)) (T, error) {
	return cachedWithTTL(ctx, deps, tool, args, deps.TTL, loader)
}

// cachedWithTTL is the explicit-TTL variant of cached. Tools that need
// per-call freshness (latest-block / whale activity / token transfers)
// pass TTL = 0, which bypasses Redis read+write entirely so callers
// always see ClickHouse-fresh data. Heavier aggregations (positions,
// balances, history, protocol stats) keep the default TTL.
func cachedWithTTL[T any](ctx context.Context, deps *Deps, tool string, args map[string]any, ttl time.Duration, loader func() (T, error)) (T, error) {
	var zero T
	if ttl <= 0 {
		out, err := loader()
		if err != nil {
			return zero, err
		}
		monitor.IncAPICacheHit(monitor.APICacheSourceClickHouse)
		return out, nil
	}
	key := cacheKey(tool, args)
	if deps.Cache != nil {
		raw, hit, err := deps.Cache.GetBytes(ctx, key)
		if err == nil && hit {
			var out T
			if err := json.Unmarshal(raw, &out); err == nil {
				monitor.IncAPICacheHit(monitor.APICacheSourceRedis)
				return out, nil
			}
		}
	}
	out, err := loader()
	if err != nil {
		return zero, err
	}
	if deps.Cache != nil {
		body, mErr := json.Marshal(out)
		if mErr == nil {
			_ = deps.Cache.SetBytes(ctx, key, body, ttl)
		}
	}
	monitor.IncAPICacheHit(monitor.APICacheSourceClickHouse)
	return out, nil
}

// decodeArgs unmarshals raw JSON-RPC arguments into a typed struct. JSON
// Schema has already validated shape upstream; this just hydrates fields.
func decodeArgs[T any](raw json.RawMessage, out *T) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode arguments: %w", err)
	}
	return nil
}

// argsAsMap re-decodes raw arguments into a map[string]any for cacheKey.
// Cheaper than reflecting on the typed struct.
func argsAsMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

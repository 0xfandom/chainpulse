package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/0xfandom/chainpulse/internal/monitor"
)

// Aside runs the canonical cache-aside flow:
//
//  1. GET cache[key]. On hit, decode JSON, bump api_cache_hits_total{redis},
//     return (value, true, nil).
//  2. On miss, call fallback. Bump api_cache_hits_total{clickhouse}.
//  3. Encode the returned value to JSON and SET it with ttl. SET errors
//     are logged via the returned error but the value is still returned;
//     callers may choose to surface or swallow the cache write failure.
//  4. Return (value, false, nil).
//
// Generic over T so handlers don't need to write boilerplate per type.
func Aside[T any](
	ctx context.Context,
	cache *ReadCache,
	key string,
	ttl time.Duration,
	fallback func(ctx context.Context) (T, error),
) (T, bool, error) {
	var zero T

	if cache != nil {
		raw, hit, err := cache.GetBytes(ctx, key)
		if err == nil && hit {
			var v T
			if dErr := json.Unmarshal(raw, &v); dErr == nil {
				monitor.IncAPICacheHit(monitor.APICacheSourceRedis)
				return v, true, nil
			}
			// fall through on decode error — treat as miss
		}
	}

	v, err := fallback(ctx)
	if err != nil {
		return zero, false, err
	}
	monitor.IncAPICacheHit(monitor.APICacheSourceClickHouse)

	if cache != nil {
		payload, mErr := json.Marshal(v)
		if mErr != nil {
			return v, false, fmt.Errorf("cache marshal: %w", mErr)
		}
		if sErr := cache.SetBytes(ctx, key, payload, ttl); sErr != nil {
			return v, false, fmt.Errorf("cache set: %w", sErr)
		}
	}
	return v, false, nil
}

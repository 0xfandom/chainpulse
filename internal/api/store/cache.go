package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/0xfandom/chainpulse/internal/monitor"
)

// AsideRaw is the raw-bytes cache-aside flow used by handlers that want
// to skip the double JSON encode imposed by the generic Aside helper.
//
// Lookup order: in-process L1 -> Redis -> fallback. On L1 hit the bytes
// are returned untouched. On Redis hit the bytes are also written
// through to L1 so subsequent lookups skip the network. On miss the
// fallback is invoked, its result populates both Redis (with ttl) and
// L1 (with the L1's own TTL).
//
// Cache write failures do not surface to callers as fatal errors —
// they are wrapped and returned so handlers can decide whether to log.
// The bytes themselves are always returned when the fallback succeeds.
func AsideRaw(
	ctx context.Context,
	l1 *L1Cache,
	cache *ReadCache,
	key string,
	ttl time.Duration,
	fallback func(ctx context.Context) ([]byte, error),
) ([]byte, bool, error) {
	if ttl <= 0 {
		v, err := fallback(ctx)
		if err != nil {
			return nil, false, err
		}
		monitor.IncAPICacheHit(monitor.APICacheSourceClickHouse)
		return v, false, nil
	}
	start := time.Now()
	if v, ok := l1.Get(key); ok {
		monitor.IncAPICacheHit(monitor.APICacheSourceL1)
		monitor.ObserveAPICacheHit(monitor.APICacheSourceL1, time.Since(start))
		return v, true, nil
	}

	if cache != nil {
		raw, hit, err := cache.GetBytes(ctx, key)
		if err == nil && hit {
			monitor.IncAPICacheHit(monitor.APICacheSourceRedis)
			monitor.ObserveAPICacheHit(monitor.APICacheSourceRedis, time.Since(start))
			l1.Set(key, raw)
			return raw, true, nil
		}
	}

	v, err := fallback(ctx)
	if err != nil {
		return nil, false, err
	}
	monitor.IncAPICacheHit(monitor.APICacheSourceClickHouse)

	if cache != nil {
		if sErr := cache.SetBytes(ctx, key, v, ttl); sErr != nil {
			l1.Set(key, v)
			return v, false, fmt.Errorf("cache set: %w", sErr)
		}
	}
	l1.Set(key, v)
	return v, false, nil
}

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

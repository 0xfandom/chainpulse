package monitor

import (
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HeadLastSeenSeconds is a per-chain Unix-timestamp gauge of the most
// recent head emission. Dashboards compute freshness as
// `time() - chain_listener_head_last_seen_timestamp_seconds`.
var HeadLastSeenSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "chain_listener_head_last_seen_timestamp_seconds",
	Help: "Unix timestamp of the most recent head emitted per chain.",
}, []string{"chain"})

type readinessRegistry struct {
	mu        sync.RWMutex
	chains    map[string]time.Time
	threshold time.Duration
	enabled   bool
}

var ready = &readinessRegistry{
	chains:    map[string]time.Time{},
	threshold: 60 * time.Second,
}

// EnableReadiness opts the current process into per-chain head-staleness
// readiness checks. Indexer-only: other binaries leave readiness disabled
// and the /ready endpoint always reports "ok". Threshold values <= 0 are
// ignored, leaving the default in place.
func EnableReadiness(threshold time.Duration) {
	ready.mu.Lock()
	defer ready.mu.Unlock()
	ready.enabled = true
	if threshold > 0 {
		ready.threshold = threshold
	}
}

// RegisterChain adds chain to the readiness registry with no head seen
// yet. Calling RegisterChain multiple times for the same chain is a
// no-op. Empty names are ignored.
func RegisterChain(chain string) {
	if chain == "" {
		return
	}
	ready.mu.Lock()
	defer ready.mu.Unlock()
	if _, ok := ready.chains[chain]; !ok {
		ready.chains[chain] = time.Time{}
	}
}

// RecordHead stamps the registry with the current time for chain. Lazily
// registers the chain if EnableReadiness has been called but
// RegisterChain has not. Always updates the Prometheus gauge so metrics
// remain accurate even when readiness is disabled.
func RecordHead(chain string) {
	if chain == "" {
		return
	}
	now := time.Now()
	ready.mu.Lock()
	ready.chains[chain] = now
	ready.mu.Unlock()
	HeadLastSeenSeconds.WithLabelValues(chain).Set(float64(now.Unix()))
}

// ChainReadiness reports per-chain head-freshness state for a single
// chain in a readiness snapshot.
type ChainReadiness struct {
	Chain        string  `json:"chain"`
	LastSeenUnix int64   `json:"last_seen_unix"`
	AgeSeconds   float64 `json:"age_seconds"`
	Stale        bool    `json:"stale"`
}

// ReadinessSnapshot is the JSON-serialisable view returned from the
// /ready handler. Status is one of "ok" | "stale".
type ReadinessSnapshot struct {
	Enabled          bool             `json:"enabled"`
	Status           string           `json:"status"`
	ThresholdSeconds float64          `json:"threshold_seconds"`
	Chains           []ChainReadiness `json:"chains"`
}

// Readiness builds a snapshot of the registry as observed at now. A chain
// is stale when its last-seen timestamp is zero (no head ever seen) or
// older than the configured threshold. Disabled readiness always reports
// status "ok"; an empty registry with readiness enabled reports "stale"
// so docker holds traffic until at least one chain reports a head.
func Readiness(now time.Time) ReadinessSnapshot {
	ready.mu.RLock()
	defer ready.mu.RUnlock()

	snap := ReadinessSnapshot{
		Enabled:          ready.enabled,
		ThresholdSeconds: ready.threshold.Seconds(),
		Chains:           make([]ChainReadiness, 0, len(ready.chains)),
	}
	if !ready.enabled {
		snap.Status = "ok"
		return snap
	}

	names := make([]string, 0, len(ready.chains))
	for c := range ready.chains {
		names = append(names, c)
	}
	sort.Strings(names)

	allFresh := len(names) > 0
	for _, c := range names {
		ts := ready.chains[c]
		cr := ChainReadiness{Chain: c}
		if ts.IsZero() {
			cr.Stale = true
			cr.AgeSeconds = -1
			allFresh = false
		} else {
			age := now.Sub(ts)
			cr.LastSeenUnix = ts.Unix()
			cr.AgeSeconds = age.Seconds()
			if age > ready.threshold {
				cr.Stale = true
				allFresh = false
			}
		}
		snap.Chains = append(snap.Chains, cr)
	}
	if allFresh {
		snap.Status = "ok"
	} else {
		snap.Status = "stale"
	}
	return snap
}

// resetReadinessForTest clears the registry. Test-only helper, not part
// of the public API.
func resetReadinessForTest() {
	ready.mu.Lock()
	defer ready.mu.Unlock()
	ready.chains = map[string]time.Time{}
	ready.threshold = 60 * time.Second
	ready.enabled = false
}

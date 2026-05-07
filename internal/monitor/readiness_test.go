package monitor

import (
	"testing"
	"time"
)

func TestReadiness_DisabledReportsOK(t *testing.T) {
	resetReadinessForTest()
	snap := Readiness(time.Now())
	if snap.Enabled {
		t.Fatalf("Enabled = true, want false")
	}
	if snap.Status != "ok" {
		t.Errorf("Status = %q, want ok", snap.Status)
	}
}

func TestReadiness_EnabledNoChainsIsStale(t *testing.T) {
	resetReadinessForTest()
	EnableReadiness(2 * time.Second)
	snap := Readiness(time.Now())
	if !snap.Enabled {
		t.Fatalf("Enabled = false, want true")
	}
	if snap.Status != "stale" {
		t.Errorf("Status = %q, want stale", snap.Status)
	}
	if snap.ThresholdSeconds != 2 {
		t.Errorf("ThresholdSeconds = %v, want 2", snap.ThresholdSeconds)
	}
}

func TestReadiness_RegisteredButNoHeadIsStale(t *testing.T) {
	resetReadinessForTest()
	EnableReadiness(time.Second)
	RegisterChain("ethereum")

	snap := Readiness(time.Now())
	if snap.Status != "stale" {
		t.Errorf("Status = %q, want stale", snap.Status)
	}
	if got := len(snap.Chains); got != 1 {
		t.Fatalf("len(Chains) = %d, want 1", got)
	}
	c := snap.Chains[0]
	if c.Chain != "ethereum" {
		t.Errorf("Chain = %q, want ethereum", c.Chain)
	}
	if !c.Stale {
		t.Errorf("Stale = false, want true (no head yet)")
	}
	if c.AgeSeconds != -1 {
		t.Errorf("AgeSeconds = %v, want -1 sentinel for no-head", c.AgeSeconds)
	}
}

func TestReadiness_FreshHeadIsOK(t *testing.T) {
	resetReadinessForTest()
	EnableReadiness(10 * time.Second)
	RegisterChain("ethereum")
	RecordHead("ethereum")

	snap := Readiness(time.Now())
	if snap.Status != "ok" {
		t.Errorf("Status = %q, want ok", snap.Status)
	}
	if got := len(snap.Chains); got != 1 {
		t.Fatalf("len(Chains) = %d, want 1", got)
	}
	if snap.Chains[0].Stale {
		t.Errorf("Stale = true, want false")
	}
}

func TestReadiness_StaleAfterThreshold(t *testing.T) {
	resetReadinessForTest()
	EnableReadiness(50 * time.Millisecond)
	RegisterChain("ethereum")
	RecordHead("ethereum")

	// Use Readiness's own time argument to deterministically advance
	// past the threshold without sleeping.
	future := time.Now().Add(time.Second)
	snap := Readiness(future)
	if snap.Status != "stale" {
		t.Errorf("Status = %q, want stale", snap.Status)
	}
	if !snap.Chains[0].Stale {
		t.Errorf("Chain stale = false, want true")
	}
}

func TestReadiness_OneStaleChainTaintsSnapshot(t *testing.T) {
	resetReadinessForTest()
	EnableReadiness(time.Second)
	RegisterChain("ethereum")
	RegisterChain("arbitrum")
	RecordHead("ethereum")
	// arbitrum left without a head; should mark snapshot stale even
	// though ethereum is fresh.

	snap := Readiness(time.Now())
	if snap.Status != "stale" {
		t.Errorf("Status = %q, want stale", snap.Status)
	}
	if got := len(snap.Chains); got != 2 {
		t.Fatalf("len(Chains) = %d, want 2", got)
	}
	// Order is alphabetical for stable JSON output.
	if snap.Chains[0].Chain != "arbitrum" || snap.Chains[1].Chain != "ethereum" {
		t.Errorf("Chains = [%q,%q], want [arbitrum,ethereum]", snap.Chains[0].Chain, snap.Chains[1].Chain)
	}
}

func TestRecordHead_LazyRegistersChain(t *testing.T) {
	resetReadinessForTest()
	EnableReadiness(10 * time.Second)
	RecordHead("base")

	snap := Readiness(time.Now())
	if got := len(snap.Chains); got != 1 {
		t.Fatalf("len(Chains) = %d, want 1 (lazy register)", got)
	}
	if snap.Chains[0].Chain != "base" {
		t.Errorf("Chain = %q, want base", snap.Chains[0].Chain)
	}
}

func TestEnableReadiness_DefaultThreshold(t *testing.T) {
	resetReadinessForTest()
	EnableReadiness(0)
	snap := Readiness(time.Now())
	if snap.ThresholdSeconds != 60 {
		t.Errorf("ThresholdSeconds = %v, want 60s default", snap.ThresholdSeconds)
	}
}

package neighborhood

import (
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

// ── NewNeighborhood ───────────────────────────────────────────────────────────

func TestNewNeighborhoodResidentialPersonality(t *testing.T) {
	n := NewNeighborhood("district-residential")
	if n.Personality.Pride != 70 {
		t.Errorf("expected Residential Pride=70, got %.0f", n.Personality.Pride)
	}
}

func TestNewNeighborhoodAbandonedTolerance(t *testing.T) {
	n := NewNeighborhood("district-abandoned")
	if n.Personality.Tolerance != 90 {
		t.Errorf("expected Abandoned Tolerance=90, got %.0f", n.Personality.Tolerance)
	}
}

func TestNewNeighborhoodUnknownUsesDefault(t *testing.T) {
	n := NewNeighborhood("district-unknown")
	if n.Personality.Tolerance != 50 {
		t.Errorf("expected default Tolerance=50, got %.0f", n.Personality.Tolerance)
	}
}

func TestNewNeighborhoodInitialFear(t *testing.T) {
	n := NewNeighborhood("district-commercial")
	if n.Fear != 10 {
		t.Errorf("expected Fear=10, got %.0f", n.Fear)
	}
}

// ── AddFear ───────────────────────────────────────────────────────────────────

func TestAddFearIncreases(t *testing.T) {
	n := NewNeighborhood("district-commercial") // Tolerance=40 → effective ~0.8x
	n.AddFear(20, t0, "k9_deploy")
	if n.Fear <= 10 {
		t.Errorf("fear should increase, got %.1f", n.Fear)
	}
}

func TestAddFearClamped(t *testing.T) {
	n := NewNeighborhood("district-commercial")
	n.AddFear(500, t0, "test")
	if n.Fear > FearMax {
		t.Errorf("fear should not exceed %.0f, got %.1f", FearMax, n.Fear)
	}
}

func TestHighToleranceReducesFearUptake(t *testing.T) {
	nHigh := NewNeighborhood("district-abandoned") // Tolerance=90
	nLow := NewNeighborhood("district-commercial") // Tolerance=40
	nHigh.AddFear(50, t0, "test")
	nLow.AddFear(50, t0, "test")
	if nHigh.Fear >= nLow.Fear {
		t.Errorf("high tolerance should see less fear: high=%.1f low=%.1f", nHigh.Fear, nLow.Fear)
	}
}

// ── Myth seeding ──────────────────────────────────────────────────────────────

func TestMythSeededOnFearSaturation(t *testing.T) {
	n := NewNeighborhood("district-residential") // Cohesion=65 > 60
	n.Fear = 88
	evts := n.AddFear(5, t0, "rogue_swarm")
	var found bool
	for _, e := range evts {
		if e.Verb == "MYTH_SEEDED" {
			found = true
		}
	}
	if !found {
		t.Error("expected MYTH_SEEDED event when fear crosses 90")
	}
	if n.MythCount() == 0 {
		t.Error("myth should be seeded in Myths slice")
	}
}

func TestMythNotSeededWhenCohesionLow(t *testing.T) {
	n := NewNeighborhood("district-abandoned") // Cohesion=30 < 60
	n.Fear = 88
	evts := n.AddFear(5, t0, "test")
	for _, e := range evts {
		if e.Verb == "MYTH_SEEDED" {
			t.Error("should not seed myth when cohesion < 60")
		}
	}
}

func TestMythCapAtMax(t *testing.T) {
	n := NewNeighborhood("district-residential")
	for i := 0; i < MaxMythsPerDistrict+5; i++ {
		n.Fear = 88
		n.AddFear(5, t0.Add(time.Duration(i)*time.Minute), "test")
	}
	if n.MythCount() > MaxMythsPerDistrict {
		t.Errorf("myths should be capped at %d, got %d", MaxMythsPerDistrict, n.MythCount())
	}
}

func TestMythHasText(t *testing.T) {
	n := NewNeighborhood("district-residential")
	n.Fear = 88
	n.AddFear(5, t0, "test")
	if n.MythCount() == 0 {
		t.Skip("no myths seeded")
	}
	if strings.TrimSpace(n.Myths[0].Text) == "" {
		t.Error("myth text should not be empty")
	}
}

// ── Tick ──────────────────────────────────────────────────────────────────────

func TestTickDecaysFear(t *testing.T) {
	n := NewNeighborhood("district-commercial")
	n.Fear = 50
	n.Tick(60*time.Second, 30, t0)
	if n.Fear >= 50 {
		t.Error("fear should decay over 60s")
	}
}

func TestTickFatigueRisesWhenAlertHigh(t *testing.T) {
	n := NewNeighborhood("district-commercial")
	n.Tick(60*time.Second, 80, t0)
	if n.FatigueVal <= 0 {
		t.Error("fatigue should rise when alertness > 70")
	}
}

func TestTickFatigueDecaysWhenAlertLow(t *testing.T) {
	n := NewNeighborhood("district-commercial")
	n.FatigueVal = 30
	n.Tick(60*time.Second, 30, t0)
	if n.FatigueVal >= 30 {
		t.Error("fatigue should decay when alertness < 70")
	}
}

// ── Derived values ─────────────────────────────────────────────────────────────

func TestWatcherVisibilityMultiplier(t *testing.T) {
	n := NewNeighborhood("district-commercial") // Visibility=80
	mult := n.WatcherVisibilityMultiplier()
	if mult <= 0 || mult > 1 {
		t.Errorf("visibility multiplier should be (0,1], got %.3f", mult)
	}
}

func TestWatcherVisibilityReducedByFatigue(t *testing.T) {
	n := NewNeighborhood("district-commercial")
	multFresh := n.WatcherVisibilityMultiplier()
	n.FatigueVal = FatigueMax
	multFatigued := n.WatcherVisibilityMultiplier()
	if multFatigued >= multFresh {
		t.Errorf("fatigued visibility should be lower: fresh=%.3f fatigued=%.3f", multFresh, multFatigued)
	}
}

func TestEffectiveCohesionReducedByFatigue(t *testing.T) {
	n := NewNeighborhood("district-residential")
	cohFresh := n.EffectiveCohesion()
	n.FatigueVal = FatigueMax
	cohFatigued := n.EffectiveCohesion()
	if cohFatigued >= cohFresh {
		t.Errorf("fatigued cohesion should be lower: fresh=%.1f fatigued=%.1f", cohFresh, cohFatigued)
	}
}

func TestMythSeedRateZeroWhenFearLow(t *testing.T) {
	n := NewNeighborhood("district-residential")
	n.Fear = 50
	if n.MythSeedRate() != 0 {
		t.Errorf("myth seed rate should be 0 when fear < 90, got %.3f", n.MythSeedRate())
	}
}

// ── Registry ──────────────────────────────────────────────────────────────────

func TestRegistryGetOrCreate(t *testing.T) {
	r := NewRegistry()
	n1 := r.GetOrCreate("district-residential")
	n2 := r.GetOrCreate("district-residential")
	if n1 != n2 {
		t.Error("GetOrCreate should return same instance")
	}
}

func TestRegistryTotalMythCount(t *testing.T) {
	r := NewRegistry()
	n := r.GetOrCreate("district-residential")
	n.Fear = 88
	n.AddFear(5, t0, "test")
	if r.TotalMythCount() != n.MythCount() {
		t.Errorf("registry myth count mismatch")
	}
}

func TestRegistryTickAll(t *testing.T) {
	r := NewRegistry()
	r.GetOrCreate("district-commercial")
	alertness := map[string]float64{"district-commercial": 80}
	r.TickAll(60*time.Second, alertness, t0)
	// Verify fatigue rose.
	if r.Get("district-commercial").FatigueVal <= 0 {
		t.Error("expected fatigue to rise for commercial district with alertness=80")
	}
}

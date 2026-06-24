package techpressure

import (
	"math"
	"testing"
	"time"
)

var t0 = time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

func newClock() *Clock { return NewClock() }

// ── Basic input ───────────────────────────────────────────────────────────────

func TestAddTierUnlock(t *testing.T) {
	c := newClock()
	c.AddTierUnlock(2, t0)
	expected := TierUnlockSpike * 2
	if math.Abs(c.Pressure-expected) > 0.001 {
		t.Errorf("expected %.0f after tier-2 unlock, got %.4f", expected, c.Pressure)
	}
}

func TestAddDogDeploy(t *testing.T) {
	c := newClock()
	c.AddDogDeploy(t0)
	if math.Abs(c.Pressure-DogDeploySpike) > 0.001 {
		t.Errorf("expected %.0f after dog deploy, got %.4f", DogDeploySpike, c.Pressure)
	}
}

func TestAddSwarmActivity(t *testing.T) {
	c := newClock()
	c.AddSwarmActivity(5, 10*time.Second, t0)
	expected := SwarmActivityRate * 5 * 10
	if math.Abs(c.Pressure-expected) > 0.001 {
		t.Errorf("expected %.2f after swarm activity, got %.4f", expected, c.Pressure)
	}
}

func TestAddCapsAtMax(t *testing.T) {
	c := newClock()
	c.add(PressureMax*10, t0, "overflow")
	if c.Pressure != PressureMax {
		t.Errorf("expected cap at %.0f, got %.4f", PressureMax, c.Pressure)
	}
}

// ── Decay ─────────────────────────────────────────────────────────────────────

func TestTickDecays(t *testing.T) {
	c := newClock()
	c.Pressure = 100.0
	c.Tick(10*time.Second, t0)
	expected := 100.0 - DecayRate*10
	if math.Abs(c.Pressure-expected) > 0.001 {
		t.Errorf("expected %.2f after decay, got %.4f", expected, c.Pressure)
	}
}

func TestTickFloorAtZero(t *testing.T) {
	c := newClock()
	c.Pressure = 0.5
	c.Tick(time.Hour, t0)
	if c.Pressure != 0 {
		t.Errorf("expected 0 floor, got %.4f", c.Pressure)
	}
}

// ── Tier transitions ──────────────────────────────────────────────────────────

func TestT1Activates(t *testing.T) {
	c := newClock()
	evs := c.add(T1LeashFrays, t0, "test")
	found := false
	for _, e := range evs {
		if e.Verb == "TIER_ACTIVATED" && e.Tier == TierLeashFrays {
			found = true
		}
	}
	if !found {
		t.Errorf("expected TIER_ACTIVATED LeashFrays, got %v", evs)
	}
	if !c.IsAtTier(TierLeashFrays) {
		t.Error("IsAtTier(TierLeashFrays) should be true")
	}
}

func TestT3Activates(t *testing.T) {
	c := newClock()
	evs := c.add(T3QuietAudit, t0, "test")
	foundT1, foundT2, foundT3 := false, false, false
	for _, e := range evs {
		if e.Verb == "TIER_ACTIVATED" {
			switch e.Tier {
			case TierLeashFrays:
				foundT1 = true
			case TierProcurement:
				foundT2 = true
			case TierQuietAudit:
				foundT3 = true
			}
		}
	}
	if !foundT1 || !foundT2 || !foundT3 {
		t.Errorf("expected T1+T2+T3 activated, got %v", evs)
	}
}

func TestTierNotFiredTwice(t *testing.T) {
	c := newClock()
	c.add(T1LeashFrays, t0, "first")
	evs := c.add(10, t0, "second") // stay in T1
	for _, e := range evs {
		if e.Verb == "TIER_ACTIVATED" && e.Tier == TierLeashFrays {
			t.Error("TIER_ACTIVATED LeashFrays should not fire twice while already active")
		}
	}
}

func TestTierClearOnDecay(t *testing.T) {
	c := newClock()
	c.add(T1LeashFrays, t0, "raise")
	if !c.IsAtTier(TierLeashFrays) {
		t.Fatal("should be at T1 after raise")
	}
	c.Pressure = T1LeashFrays - 0.5
	c.ActiveTier = TierLeashFrays // simulate being at T1
	evs := c.Tick(time.Second, t0) // decay drops below T1
	found := false
	for _, e := range evs {
		if e.Verb == "TIER_CLEARED" && e.Tier == TierLeashFrays {
			found = true
		}
	}
	if !found {
		t.Errorf("expected TIER_CLEARED for LeashFrays, got %v", evs)
	}
	if c.IsAtTier(TierLeashFrays) {
		t.Error("IsAtTier(TierLeashFrays) should be false after clear")
	}
}

func TestAllFiveTiersActivate(t *testing.T) {
	c := newClock()
	evs := c.add(PressureMax, t0, "max")
	activated := map[Tier]bool{}
	for _, e := range evs {
		if e.Verb == "TIER_ACTIVATED" || e.Verb == "CROWN_PROTOCOL" {
			activated[e.Tier] = true
		}
	}
	for _, tier := range []Tier{TierLeashFrays, TierProcurement, TierQuietAudit, TierPackmind, TierCrownProtocol} {
		if !activated[tier] {
			t.Errorf("tier %s not activated on max pressure", TierName(tier))
		}
	}
}

// ── Crown Protocol ────────────────────────────────────────────────────────────

func TestCrownProtocolFiresOnce(t *testing.T) {
	c := newClock()
	evs := c.add(PressureMax, t0, "first")
	crowns := 0
	for _, e := range evs {
		if e.Verb == "CROWN_PROTOCOL" {
			crowns++
		}
	}
	if crowns != 1 {
		t.Errorf("expected exactly 1 CROWN_PROTOCOL, got %d", crowns)
	}
	if !c.CrownFired {
		t.Error("CrownFired should be true")
	}

	// Second time at T5 should NOT fire another CROWN_PROTOCOL.
	c.Pressure = 0
	c.ActiveTier = TierNone
	evs2 := c.add(PressureMax, t0, "second")
	for _, e := range evs2 {
		if e.Verb == "CROWN_PROTOCOL" {
			t.Error("CROWN_PROTOCOL should not fire a second time")
		}
	}
}

func TestCrownProtocolVerb(t *testing.T) {
	c := newClock()
	evs := c.add(PressureMax, t0, "test")
	for _, e := range evs {
		if e.Tier == TierCrownProtocol && e.Verb == "TIER_ACTIVATED" {
			t.Error("first CrownProtocol event should have Verb=CROWN_PROTOCOL, not TIER_ACTIVATED")
		}
	}
}

// ── Bird Correction ───────────────────────────────────────────────────────────

func TestBirdCorrectionDrops(t *testing.T) {
	c := newClock()
	c.Pressure = 500.0
	c.BirdCorrection(t0)
	expected := 500.0 - BirdCorrectionDrop
	if math.Abs(c.Pressure-expected) > 0.001 {
		t.Errorf("expected %.0f after bird correction, got %.4f", expected, c.Pressure)
	}
}

func TestBirdCorrectionFloorAtZero(t *testing.T) {
	c := newClock()
	c.Pressure = 50.0 // less than BirdCorrectionDrop
	c.BirdCorrection(t0)
	if c.Pressure != 0 {
		t.Errorf("expected 0 after excessive bird correction, got %.4f", c.Pressure)
	}
}

func TestBirdCorrectionClearsTier(t *testing.T) {
	c := newClock()
	c.add(T2ProcurementWar, t0, "raise")
	if !c.IsAtTier(TierProcurement) {
		t.Fatal("should be at T2")
	}
	evs := c.BirdCorrection(t0) // drops 150 → below T2
	found := false
	for _, e := range evs {
		if e.Verb == "TIER_CLEARED" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected TIER_CLEARED from bird correction, got %v", evs)
	}
}

// ── IsAtTier ──────────────────────────────────────────────────────────────────

func TestIsAtTierFalseInitially(t *testing.T) {
	c := newClock()
	for _, tier := range []Tier{TierLeashFrays, TierProcurement, TierQuietAudit, TierPackmind, TierCrownProtocol} {
		if c.IsAtTier(tier) {
			t.Errorf("IsAtTier(%s) should be false initially", TierName(tier))
		}
	}
}

// ── Helper ────────────────────────────────────────────────────────────────────

func TestTierForPressure(t *testing.T) {
	cases := []struct {
		p    float64
		want Tier
	}{
		{0, TierNone},
		{249, TierNone},
		{250, TierLeashFrays},
		{450, TierProcurement},
		{650, TierQuietAudit},
		{850, TierPackmind},
		{1000, TierCrownProtocol},
	}
	for _, tc := range cases {
		got := TierForPressure(tc.p)
		if got != tc.want {
			t.Errorf("TierForPressure(%.0f): expected %s, got %s", tc.p, TierName(tc.want), TierName(got))
		}
	}
}

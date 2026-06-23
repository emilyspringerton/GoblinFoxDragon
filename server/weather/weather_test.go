package weather

import (
	"testing"
	"time"
)

func TestNewEngineStartsClear(t *testing.T) {
	e := New()
	if e.Phase() != PhaseClear {
		t.Errorf("expected Clear, got %v", e.Phase())
	}
}

func TestPhaseEndsInFuture(t *testing.T) {
	e := New()
	if !e.PhaseEnds().After(time.Now()) {
		t.Error("phase should end in the future")
	}
}

func TestTickNoChangeBeforeExpiry(t *testing.T) {
	now := time.Now()
	e := newForTest(42, now)
	changed, old, new := e.Tick(now)
	if changed {
		t.Error("phase should not change before expiry")
	}
	if old != new {
		t.Errorf("old/new should be same: %v vs %v", old, new)
	}
}

func TestTickChangesPhaseAfterExpiry(t *testing.T) {
	now := time.Now()
	e := newForTest(42, now)
	past := e.PhaseEnds().Add(time.Second)
	changed, old, new := e.Tick(past)
	if !changed {
		t.Error("phase should change after expiry")
	}
	if old == new {
		t.Error("old and new should differ after transition")
	}
}

func TestPhaseOrderCycles(t *testing.T) {
	e := newForTest(1, time.Now())
	seen := []Phase{e.Phase()}
	for i := 0; i < 8; i++ {
		// Jump past current phase end.
		future := e.PhaseEnds().Add(time.Second)
		changed, _, newP := e.Tick(future)
		if changed {
			seen = append(seen, newP)
		}
	}
	// Should see at least 4 distinct phases.
	phaseSet := make(map[Phase]bool)
	for _, p := range seen {
		phaseSet[p] = true
	}
	if len(phaseSet) < 4 {
		t.Errorf("expected all 4 phases, got %v", phaseSet)
	}
}

func TestModifiersStorm(t *testing.T) {
	e := newForTest(1, time.Now())
	e.ForcePhase(PhaseStorm, time.Now())
	m := e.Mods()
	if m.MobDamageBonus <= 0 {
		t.Error("Storm should have positive mob damage bonus")
	}
	if m.BSTTameBonus <= 0 {
		t.Error("Storm should have positive BST tame bonus")
	}
}

func TestModifiersClear(t *testing.T) {
	e := newForTest(1, time.Now())
	e.ForcePhase(PhaseClear, time.Now())
	m := e.Mods()
	if m.MobDamageBonus != 0 {
		t.Errorf("Clear should have 0 mob damage bonus, got %d", m.MobDamageBonus)
	}
	if m.BSTTameBonus != 0 {
		t.Errorf("Clear should have 0 BST tame bonus, got %f", m.BSTTameBonus)
	}
}

func TestModifiersRain(t *testing.T) {
	m := ModifiersFor(PhaseRain)
	if m.MobDamageBonus <= 0 {
		t.Error("Rain should have positive mob damage bonus")
	}
}

func TestForcePhase(t *testing.T) {
	e := newForTest(1, time.Now())
	e.ForcePhase(PhaseStorm, time.Now())
	if e.Phase() != PhaseStorm {
		t.Errorf("ForcePhase should set to Storm, got %v", e.Phase())
	}
}

func TestForcePhaseSchedulesNewDuration(t *testing.T) {
	now := time.Now()
	e := newForTest(1, now)
	e.ForcePhase(PhaseStorm, now)
	// Storm duration range is 60–240s.
	dur := e.PhaseEnds().Sub(now)
	if dur < 60*time.Second || dur > 240*time.Second+time.Second {
		t.Errorf("Storm duration out of range: %v", dur)
	}
}

func TestClearDurationRange(t *testing.T) {
	// Clear should be 5–12 min.
	r := phaseDurations[PhaseClear]
	if r[0] != 300 || r[1] != 720 {
		t.Errorf("Clear duration range wrong: %v", r)
	}
}

func TestStormDurationRange(t *testing.T) {
	// Storm should be 1–4 min.
	r := phaseDurations[PhaseStorm]
	if r[0] != 60 || r[1] != 240 {
		t.Errorf("Storm duration range wrong: %v", r)
	}
}

func TestStormBSTBonusValue(t *testing.T) {
	m := ModifiersFor(PhaseStorm)
	if m.BSTTameBonus != 0.10 {
		t.Errorf("expected 0.10 BST bonus for Storm, got %f", m.BSTTameBonus)
	}
}

func TestStormMobDamageBonusValue(t *testing.T) {
	m := ModifiersFor(PhaseStorm)
	if m.MobDamageBonus != 3 {
		t.Errorf("expected 3 mob damage bonus for Storm, got %d", m.MobDamageBonus)
	}
}

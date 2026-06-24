package enforcement

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

// ── NewState ──────────────────────────────────────────────────────────────────

func TestNewStateIsQuiet(t *testing.T) {
	s := NewState("district-residential")
	if s.Level != LevelQuiet {
		t.Errorf("expected LevelQuiet, got %v", s.Level)
	}
}

// ── Escalation ────────────────────────────────────────────────────────────────

func TestEscalatesWhenHot(t *testing.T) {
	s := NewState("d1")
	evts := s.Evaluate(70, -30, t0)
	if s.Level != LevelAlert {
		t.Errorf("expected LevelAlert after escalation, got %v", s.Level)
	}
	if len(evts) == 0 || evts[0].Verb != "ESCALATE" {
		t.Error("expected ESCALATE event")
	}
}

func TestNoEscalationWhenAlertLow(t *testing.T) {
	s := NewState("d1")
	s.Evaluate(40, -30, t0)
	if s.Level != LevelQuiet {
		t.Errorf("expected LevelQuiet (alertness too low), got %v", s.Level)
	}
}

func TestNoEscalationWhenTrustHigh(t *testing.T) {
	s := NewState("d1")
	s.Evaluate(70, 10, t0)
	if s.Level != LevelQuiet {
		t.Errorf("expected LevelQuiet (trust too high), got %v", s.Level)
	}
}

func TestEscalatesThroughLevels(t *testing.T) {
	s := NewState("d1")
	for i := 0; i < 4; i++ {
		s.Evaluate(70, -30, t0)
	}
	if s.Level != LevelLockdown {
		t.Errorf("expected LevelLockdown after 4 escalations, got %v", s.Level)
	}
}

func TestLockdownVerbOnFinalEscalation(t *testing.T) {
	s := NewState("d1")
	s.Level = LevelHeavy
	evts := s.Evaluate(70, -30, t0)
	if len(evts) == 0 || evts[0].Verb != "LOCKDOWN" {
		t.Errorf("expected LOCKDOWN verb, got %v", evts)
	}
}

func TestNoBeyondLockdown(t *testing.T) {
	s := NewState("d1")
	s.Level = LevelLockdown
	s.Evaluate(70, -30, t0)
	if s.Level != LevelLockdown {
		t.Errorf("should not go above LevelLockdown")
	}
}

// ── De-escalation ─────────────────────────────────────────────────────────────

func TestDeescalatesAfterTwoQuietTicks(t *testing.T) {
	s := NewState("d1")
	s.Level = LevelActive
	s.Evaluate(30, 10, t0) // tick 1
	if s.Level != LevelActive {
		t.Error("should not de-escalate on first quiet tick")
	}
	s.Evaluate(30, 10, t0) // tick 2
	if s.Level != LevelAlert {
		t.Errorf("expected LevelAlert after 2 quiet ticks, got %v", s.Level)
	}
}

func TestDeescalateResetsOnHotTick(t *testing.T) {
	s := NewState("d1")
	s.Level = LevelActive
	s.Evaluate(30, 10, t0) // quiet tick 1
	s.Evaluate(70, -30, t0) // hot tick — resets counter + escalates
	// Escalated again so now LevelHeavy
	if s.Level != LevelHeavy {
		t.Errorf("expected LevelHeavy after hot tick, got %v", s.Level)
	}
}

func TestLockdownLiftVerb(t *testing.T) {
	s := NewState("d1")
	s.Level = LevelLockdown
	s.Evaluate(30, 10, t0)
	evts := s.Evaluate(30, 10, t0)
	if len(evts) == 0 || evts[0].Verb != "LOCKDOWN_LIFT" {
		t.Errorf("expected LOCKDOWN_LIFT, got %v", evts)
	}
}

// ── Effects ───────────────────────────────────────────────────────────────────

func TestEffectsQuiet(t *testing.T) {
	e := Effects(LevelQuiet)
	if e.CopDensity != 0 {
		t.Error("LevelQuiet should have 0 cops")
	}
	if e.K9Eligible {
		t.Error("LevelQuiet should not have K9")
	}
}

func TestEffectsActiveCopDensity(t *testing.T) {
	e := Effects(LevelActive)
	if e.CopDensity != 2 {
		t.Errorf("LevelActive should have 2 cops, got %d", e.CopDensity)
	}
	if !e.K9Eligible {
		t.Error("LevelActive should have K9 eligible")
	}
}

func TestEffectsLockdownUnclaim(t *testing.T) {
	e := Effects(LevelLockdown)
	if !e.FOUnclaim {
		t.Error("LevelLockdown should have FOUnclaim=true")
	}
	if !e.CustodyImmediate {
		t.Error("LevelLockdown should have CustodyImmediate=true")
	}
	if e.CopDensity != 8 {
		t.Errorf("LevelLockdown should have 8 cops, got %d", e.CopDensity)
	}
}

// ── LevelName ─────────────────────────────────────────────────────────────────

func TestLevelNameKnown(t *testing.T) {
	if LevelName(LevelHeavy) != "HEAVY" {
		t.Errorf("expected HEAVY, got %q", LevelName(LevelHeavy))
	}
}

func TestLevelNameUnknown(t *testing.T) {
	if LevelName(99) != "UNKNOWN" {
		t.Errorf("expected UNKNOWN for unknown level")
	}
}

// ── Registry ──────────────────────────────────────────────────────────────────

func TestRegistryGetOrCreate(t *testing.T) {
	r := NewRegistry()
	s1 := r.GetOrCreate("district-residential")
	s2 := r.GetOrCreate("district-residential")
	if s1 != s2 {
		t.Error("GetOrCreate should return same instance")
	}
}

func TestRegistryEvaluateAll(t *testing.T) {
	r := NewRegistry()
	r.GetOrCreate("district-commercial")
	r.GetOrCreate("district-residential")
	alertness := map[string]float64{"district-commercial": 70, "district-residential": 30}
	trust := map[string]float64{"district-commercial": -30, "district-residential": 10}
	evts := r.EvaluateAll(alertness, trust, t0)
	if len(evts) == 0 {
		t.Error("expected at least one escalation event")
	}
}

func TestRegistryLockdownDistricts(t *testing.T) {
	r := NewRegistry()
	s := r.GetOrCreate("district-underground")
	s.Level = LevelLockdown
	ld := r.LockdownDistricts()
	if len(ld) != 1 {
		t.Errorf("expected 1 lockdown district, got %d", len(ld))
	}
}

package integrity

import (
	"math"
	"testing"
	"time"
)

var t0 = time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

func newState() *IntegrityState { return NewState("district-1") }

// ── Tick + decay ──────────────────────────────────────────────────────────────

func TestTickPassiveRecovery(t *testing.T) {
	s := newState()
	s.CI = 0.5
	s.Tick(10*time.Second, 0, t0)
	expected := math.Min(0.5+PassiveRecoveryRate*10, CIMax)
	if math.Abs(s.CI-expected) > 0.0001 {
		t.Errorf("expected %.4f after passive recovery, got %.4f", expected, s.CI)
	}
}

func TestTickDogDecay(t *testing.T) {
	s := newState()
	before := s.CI
	s.Tick(10*time.Second, 5, t0)
	if s.CI >= before {
		t.Error("5 dogs for 10s should decay CI below initial value")
	}
}

func TestTickNoDogsNoDecay(t *testing.T) {
	s := newState()
	s.CI = 0.5
	s.Tick(time.Second, 0, t0)
	// With 0 dogs, only passive recovery should occur (CI should go up slightly).
	if s.CI <= 0.5 {
		t.Errorf("no dogs should allow CI to recover, got %.4f", s.CI)
	}
}

func TestTickFloorAtZero(t *testing.T) {
	s := newState()
	s.CI = 0.001
	// Lots of dogs for a long time.
	s.Tick(10*time.Second, 10, t0)
	if s.CI < 0 {
		t.Errorf("CI should not go below 0, got %.4f", s.CI)
	}
}

func TestTickCapsAtOne(t *testing.T) {
	s := newState()
	s.CI = 0.999
	s.Tick(time.Hour, 0, t0) // pure recovery
	if s.CI > CIMax {
		t.Errorf("CI should not exceed 1.0, got %.4f", s.CI)
	}
}

// ── Jammer + flip decay ───────────────────────────────────────────────────────

func TestJammerDecay(t *testing.T) {
	s := newState()
	before := s.CI
	s.RecordJammerUse(t0)
	if math.Abs((before-s.CI)-JammerDecayPerUse) > 0.0001 {
		t.Errorf("jammer should reduce CI by %.3f, got diff %.4f", JammerDecayPerUse, before-s.CI)
	}
}

func TestFlipDecay(t *testing.T) {
	s := newState()
	before := s.CI
	s.RecordFlip(t0)
	if math.Abs((before-s.CI)-FlipDecayPerFlip) > 0.0001 {
		t.Errorf("flip should reduce CI by %.3f, got diff %.4f", FlipDecayPerFlip, before-s.CI)
	}
}

func TestJammerNoDecayDuringRogue(t *testing.T) {
	s := newState()
	s.CI = 0.0
	s.IsRogue = true
	evs := s.RecordJammerUse(t0)
	if len(evs) != 0 {
		t.Error("no events expected during Rogue state")
	}
}

// ── Clean audit + Bird Correction ─────────────────────────────────────────────

func TestCleanAuditBoost(t *testing.T) {
	s := newState()
	s.CI = 0.5
	ev := s.CleanAudit(t0)
	if math.Abs(s.CI-(0.5+AuditBonus)) > 0.0001 {
		t.Errorf("expected CI=%.3f after audit, got %.4f", 0.5+AuditBonus, s.CI)
	}
	if ev.Verb != "CI_RESTORED" {
		t.Errorf("expected CI_RESTORED, got %q", ev.Verb)
	}
}

func TestBirdCorrectionBoost(t *testing.T) {
	s := newState()
	s.CI = 0.5
	ev := s.BirdCorrection(t0)
	if math.Abs(s.CI-(0.5+BirdBonus)) > 0.0001 {
		t.Errorf("expected CI=%.3f after bird, got %.4f", 0.5+BirdBonus, s.CI)
	}
	if ev.Verb != "CI_RESTORED" {
		t.Errorf("expected CI_RESTORED, got %q", ev.Verb)
	}
}

func TestAuditCapsAtOne(t *testing.T) {
	s := newState()
	s.CI = 0.99
	s.CleanAudit(t0)
	if s.CI > CIMax {
		t.Errorf("CI should cap at 1.0, got %.4f", s.CI)
	}
}

// ── Rogue Swarm ───────────────────────────────────────────────────────────────

func TestRogueSwarmTrigger(t *testing.T) {
	s := newState()
	s.CI = RogueThreshold + 0.001
	// Force below threshold via direct assignment then trigger.
	s.CI = RogueThreshold
	evs := s.checkRogue(t0, "test")
	if len(evs) == 0 {
		t.Fatal("expected ROGUE_SWARM event at threshold")
	}
	if evs[0].Verb != "ROGUE_SWARM" {
		t.Errorf("expected ROGUE_SWARM, got %q", evs[0].Verb)
	}
	if !s.IsRogue {
		t.Error("IsRogue should be true after trigger")
	}
}

func TestRogueSwarmNotFiredTwice(t *testing.T) {
	s := newState()
	s.CI = 0.0
	s.IsRogue = true
	evs := s.checkRogue(t0, "second-call")
	if len(evs) != 0 {
		t.Error("ROGUE_SWARM should not fire twice while already rogue")
	}
}

func TestTickNoRogueDuringRogue(t *testing.T) {
	s := newState()
	s.CI = 0.0
	s.IsRogue = true
	evs := s.Tick(10*time.Second, 5, t0)
	if len(evs) != 0 {
		t.Error("no tick events should fire during active Rogue Swarm")
	}
}

func TestRogueSwarmViaJammer(t *testing.T) {
	s := newState()
	s.CI = RogueThreshold + 0.001 // just above threshold
	// One jammer push should drop below.
	evs := s.RecordJammerUse(t0)
	found := false
	for _, e := range evs {
		if e.Verb == "ROGUE_SWARM" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ROGUE_SWARM from jammer, got %v", evs)
	}
}

func TestRogueSwarmViaFlip(t *testing.T) {
	s := newState()
	s.CI = RogueThreshold + 0.001
	evs := s.RecordFlip(t0)
	found := false
	for _, e := range evs {
		if e.Verb == "ROGUE_SWARM" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ROGUE_SWARM from flip, got %v", evs)
	}
}

// ── Containment ───────────────────────────────────────────────────────────────

func TestSetContainmentProgressPartial(t *testing.T) {
	s := newState()
	s.IsRogue = true
	done, evs := s.SetContainmentProgress(2, t0)
	if done {
		t.Error("2/3 objectives should not complete containment")
	}
	if len(evs) != 0 {
		t.Error("no events expected at 2/3 objectives")
	}
}

func TestSetContainmentProgressComplete(t *testing.T) {
	s := newState()
	s.IsRogue = true
	s.RogueCI = 0.05
	done, evs := s.SetContainmentProgress(3, t0)
	if !done {
		t.Error("3/3 objectives should complete containment")
	}
	foundEnd, foundScar := false, false
	for _, e := range evs {
		if e.Verb == "CONTAINMENT_END" {
			foundEnd = true
		}
		if e.Verb == "SCAR_WRITTEN" {
			foundScar = true
		}
	}
	if !foundEnd {
		t.Error("expected CONTAINMENT_END event")
	}
	if !foundScar {
		t.Error("expected SCAR_WRITTEN event")
	}
	if s.IsRogue {
		t.Error("IsRogue should be false after containment ends")
	}
	if len(s.Scars) != 1 {
		t.Errorf("expected 1 scar, got %d", len(s.Scars))
	}
}

func TestScarCountAccumulates(t *testing.T) {
	s := newState()
	for i := 0; i < 3; i++ {
		s.IsRogue = true
		s.RogueCI = 0.1
		s.EndContainment(t0)
	}
	if s.ScarCount() != 3 {
		t.Errorf("expected 3 scars, got %d", s.ScarCount())
	}
}

func TestEndContainmentCIBump(t *testing.T) {
	s := newState()
	s.IsRogue = true
	s.CI = 0.1
	before := s.CI
	s.EndContainment(t0)
	if s.CI <= before {
		t.Errorf("CI should recover on containment end, got %.4f (was %.4f)", s.CI, before)
	}
}

// ── Registry ──────────────────────────────────────────────────────────────────

func TestRegistryGetOrCreate(t *testing.T) {
	reg := NewRegistry()
	s1 := reg.GetOrCreate("d-1")
	s2 := reg.GetOrCreate("d-1")
	if s1 != s2 {
		t.Error("GetOrCreate should return same state for same district")
	}
}

func TestRegistryGetMissing(t *testing.T) {
	reg := NewRegistry()
	if reg.Get("d-missing") != nil {
		t.Error("Get should return nil for unknown district")
	}
}

func TestRegistryRogueDistricts(t *testing.T) {
	reg := NewRegistry()
	d1 := reg.GetOrCreate("d-1")
	d2 := reg.GetOrCreate("d-2")
	d1.IsRogue = true
	_ = d2
	rogues := reg.RogueDistricts()
	if len(rogues) != 1 || rogues[0].DistrictID != "d-1" {
		t.Errorf("expected 1 rogue district d-1, got %v", rogues)
	}
}

func TestRegistryTickAll(t *testing.T) {
	reg := NewRegistry()
	s := reg.GetOrCreate("d-1")
	s.CI = 0.5
	dogs := map[string]int{"d-1": 0}
	reg.TickAll(time.Second, dogs, t0)
	// CI should have ticked (slight recovery with 0 dogs).
	if s.CI <= 0.5 {
		t.Error("expected CI to recover after tick with 0 dogs")
	}
}

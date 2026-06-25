package k9

import (
	"testing"
	"time"
)

func newNow() time.Time {
	return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
}

// ── Initiate ──────────────────────────────────────────────────────────────────

func TestInitiate(t *testing.T) {
	r := NewOpRegistry()
	now := newNow()
	op, err := r.Initiate("fo-1", now)
	if err != nil {
		t.Fatalf("Initiate: %v", err)
	}
	if op.Phase != PhaseEncirclement {
		t.Errorf("phase=%d want PhaseEncirclement", op.Phase)
	}
	if len(op.Events) != 1 {
		t.Errorf("events=%d want 1", len(op.Events))
	}
}

func TestInitiateDuplicate(t *testing.T) {
	r := NewOpRegistry()
	now := newNow()
	r.Initiate("fo-1", now)
	_, err := r.Initiate("fo-1", now)
	if err != ErrOperationAlreadyActive {
		t.Errorf("expected ErrOperationAlreadyActive, got %v", err)
	}
}

// ── AdvancePhase ──────────────────────────────────────────────────────────────

func TestAdvancePhaseTooEarly(t *testing.T) {
	r := NewOpRegistry()
	now := newNow()
	r.Initiate("fo-1", now)
	_, _, err := r.AdvancePhase("fo-1", 3, now.Add(1*time.Minute))
	if err != ErrPhaseNotComplete {
		t.Errorf("expected ErrPhaseNotComplete, got %v", err)
	}
}

func TestAdvancePhaseNotEnoughDogs(t *testing.T) {
	r := NewOpRegistry()
	now := newNow()
	r.Initiate("fo-1", now)
	after := now.Add(PhaseDuration + time.Second)
	_, _, err := r.AdvancePhase("fo-1", 1, after) // only 1 dog
	if err != ErrNotEnoughDogs {
		t.Errorf("expected ErrNotEnoughDogs, got %v", err)
	}
}

func TestAdvancePhaseSuccess(t *testing.T) {
	r := NewOpRegistry()
	now := newNow()
	r.Initiate("fo-1", now)
	after := now.Add(PhaseDuration + time.Second)
	newPhase, ev, err := r.AdvancePhase("fo-1", 3, after)
	if err != nil {
		t.Fatalf("AdvancePhase: %v", err)
	}
	if newPhase != PhaseIsolation {
		t.Errorf("phase=%s want Isolation", PhaseName(newPhase))
	}
	if ev.Verb != "PHASE_ADVANCED" {
		t.Errorf("ev.Verb=%q want PHASE_ADVANCED", ev.Verb)
	}
}

func TestFullPhaseProgression(t *testing.T) {
	r := NewOpRegistry()
	t0 := newNow()
	r.Initiate("fo-x", t0)

	// Phase 1→2→3→4
	phases := []Phase{PhaseIsolation, PhaseCustodyLock, PhaseResolution}
	now := t0
	for i, want := range phases {
		now = now.Add(PhaseDuration + time.Second)
		got, _, err := r.AdvancePhase("fo-x", 3, now)
		if err != nil {
			t.Fatalf("phase step %d: %v", i, err)
		}
		if got != want {
			t.Errorf("step %d: got %s want %s", i, PhaseName(got), PhaseName(want))
		}
	}
}

func TestAdvancePhaseAtResolution(t *testing.T) {
	r := NewOpRegistry()
	t0 := newNow()
	r.Initiate("fo-x", t0)
	now := t0
	for i := 0; i < 3; i++ {
		now = now.Add(PhaseDuration + time.Second)
		r.AdvancePhase("fo-x", 3, now)
	}
	now = now.Add(PhaseDuration + time.Second)
	_, _, err := r.AdvancePhase("fo-x", 3, now)
	if err != ErrAlreadyResolved {
		t.Errorf("expected ErrAlreadyResolved, got %v", err)
	}
}

// ── IsFlipBlocked ─────────────────────────────────────────────────────────────

func TestFlipNotBlockedInPhase1(t *testing.T) {
	r := NewOpRegistry()
	now := newNow()
	r.Initiate("fo-1", now)
	if r.IsFlipBlocked("fo-1", now) {
		t.Error("expected flip allowed in Phase 1 (Encirclement)")
	}
}

func TestFlipBlockedInPhase2(t *testing.T) {
	r := NewOpRegistry()
	t0 := newNow()
	r.Initiate("fo-1", t0)
	now := t0.Add(PhaseDuration + time.Second)
	r.AdvancePhase("fo-1", 3, now)
	if !r.IsFlipBlocked("fo-1", now) {
		t.Error("expected flip blocked in Phase 2 (Isolation)")
	}
}

// ── Counterplay ───────────────────────────────────────────────────────────────

func TestBirdCorrection(t *testing.T) {
	r := NewOpRegistry()
	now := newNow()
	r.Initiate("fo-1", now)
	reduction, ev, err := r.UseBirdCorrection("fo-1", now)
	if err != nil {
		t.Fatalf("BirdCorrection: %v", err)
	}
	if reduction != BirdCorrectionReduction {
		t.Errorf("reduction=%.0f want %.0f", reduction, BirdCorrectionReduction)
	}
	if ev.Verb != "BIRD_CORRECTION" {
		t.Errorf("verb=%q", ev.Verb)
	}
	// Second use must fail.
	_, _, err2 := r.UseBirdCorrection("fo-1", now)
	if err2 != ErrCounterplayUsed {
		t.Errorf("expected ErrCounterplayUsed on second use, got %v", err2)
	}
}

func TestScarBurn(t *testing.T) {
	r := NewOpRegistry()
	now := newNow()
	r.Initiate("fo-1", now)

	scarRemoved := false
	scarRemover := func(districtID string) (string, bool) {
		if districtID == "district-x" {
			scarRemoved = true
			return "RogueSwarm breach", true
		}
		return "", false
	}
	ev, err := r.UseScarBurn("fo-1", "district-x", scarRemover, now)
	if err != nil {
		t.Fatalf("ScarBurn: %v", err)
	}
	if !scarRemoved {
		t.Error("scar remover not called")
	}
	if ev.Verb != "SCAR_BURN" {
		t.Errorf("verb=%q", ev.Verb)
	}
	// Second use must fail.
	_, err2 := r.UseScarBurn("fo-1", "district-x", scarRemover, now)
	if err2 != ErrCounterplayUsed {
		t.Errorf("expected ErrCounterplayUsed, got %v", err2)
	}
}

func TestFlipWindowCounterplay(t *testing.T) {
	r := NewOpRegistry()
	t0 := newNow()
	r.Initiate("fo-1", t0)
	// Advance to Phase 2 (flip blocked).
	now := t0.Add(PhaseDuration + time.Second)
	r.AdvancePhase("fo-1", 3, now)
	if !r.IsFlipBlocked("fo-1", now) {
		t.Fatal("expected flip blocked before FlipWindow")
	}

	// Open a 5-minute flip window.
	_, err := r.UseFlipWindow("fo-1", 5*time.Minute, now)
	if err != nil {
		t.Fatalf("FlipWindow: %v", err)
	}
	if r.IsFlipBlocked("fo-1", now.Add(time.Minute)) {
		t.Error("expected flip allowed within window")
	}
	// After window expires, flip is blocked again.
	if !r.IsFlipBlocked("fo-1", now.Add(10*time.Minute)) {
		t.Error("expected flip blocked after window expires")
	}
	// Second use.
	_, err2 := r.UseFlipWindow("fo-1", 5*time.Minute, now)
	if err2 != ErrCounterplayUsed {
		t.Errorf("expected ErrCounterplayUsed, got %v", err2)
	}
}

// ── Resolve ───────────────────────────────────────────────────────────────────

func TestResolveRequiresPhase4(t *testing.T) {
	r := NewOpRegistry()
	now := newNow()
	r.Initiate("fo-1", now)
	_, err := r.Resolve("fo-1", now)
	if err == nil {
		t.Error("expected error resolving from Phase 1")
	}
}

func TestResolveSuccess(t *testing.T) {
	r := NewOpRegistry()
	t0 := newNow()
	r.Initiate("fo-1", t0)
	now := t0
	for i := 0; i < 3; i++ {
		now = now.Add(PhaseDuration + time.Second)
		r.AdvancePhase("fo-1", 3, now)
	}
	ev, err := r.Resolve("fo-1", now)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if ev.Verb != "RESOLVED" {
		t.Errorf("verb=%q want RESOLVED", ev.Verb)
	}
	// Operation removed from registry.
	if r.Get("fo-1") != nil {
		t.Error("expected op removed after resolve")
	}
}

func TestResolveNonExistent(t *testing.T) {
	r := NewOpRegistry()
	_, err := r.Resolve("no-such-fo", newNow())
	if err != ErrOperationNotActive {
		t.Errorf("expected ErrOperationNotActive, got %v", err)
	}
}

// ── ActiveOps ─────────────────────────────────────────────────────────────────

func TestActiveOps(t *testing.T) {
	r := NewOpRegistry()
	now := newNow()
	r.Initiate("fo-1", now)
	r.Initiate("fo-2", now)
	ops := r.ActiveOps()
	if len(ops) != 2 {
		t.Errorf("ActiveOps=%d want 2", len(ops))
	}
}

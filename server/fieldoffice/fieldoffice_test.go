package fieldoffice

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

// forceOpts returns options with forced-open flip windows for testing.
func forceOpts() *Options {
	return &Options{
		FlipWindowsPerDay: 288, // window every 5 min; always open in tests
		ContestDuration:   5 * time.Minute,
		FlowBaseRate:      1.0,
		PressureRate:      0.1,
	}
}

func newFO(id string) *FieldOffice {
	return New(id, 200, 0, 0, 0, forceOpts())
}

// ── Phase transitions ─────────────────────────────────────────────────────────

func TestClaimUnclaimed(t *testing.T) {
	fo := newFO("fo-1")
	r, err := fo.Claim("crew-a", t0)
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}
	if fo.Phase != PhaseHeld {
		t.Errorf("expected PhaseHeld, got %v", fo.Phase)
	}
	if fo.HolderID != "crew-a" {
		t.Errorf("expected holder crew-a, got %q", fo.HolderID)
	}
	if r.Verb != "CLAIMED" {
		t.Errorf("expected receipt verb CLAIMED, got %q", r.Verb)
	}
}

func TestClaimAlreadyHeld(t *testing.T) {
	fo := newFO("fo-1")
	fo.Claim("crew-a", t0)
	_, err := fo.Claim("crew-b", t0)
	if err != ErrAlreadyClaimed {
		t.Errorf("expected ErrAlreadyClaimed, got %v", err)
	}
}

func TestContestHeld(t *testing.T) {
	fo := newFO("fo-1")
	fo.Claim("crew-a", t0)
	r, err := fo.Contest("crew-b", t0, true)
	if err != nil {
		t.Fatalf("Contest failed: %v", err)
	}
	if fo.Phase != PhaseContested {
		t.Errorf("expected PhaseContested, got %v", fo.Phase)
	}
	if r.Verb != "CONTESTED" {
		t.Errorf("expected CONTESTED, got %q", r.Verb)
	}
	if fo.ContesterID != "crew-b" {
		t.Errorf("expected contester crew-b, got %q", fo.ContesterID)
	}
}

func TestContestSelfFlip(t *testing.T) {
	fo := newFO("fo-1")
	fo.Claim("crew-a", t0)
	_, err := fo.Contest("crew-a", t0, true)
	if err != ErrSelfFlip {
		t.Errorf("expected ErrSelfFlip, got %v", err)
	}
}

func TestContestWindowClosed(t *testing.T) {
	fo := New("fo-1", 200, 0, 0, 0, &Options{
		FlipWindowsPerDay: 1,
		ContestDuration:   1 * time.Second, // window only 1s wide
	})
	fo.Claim("crew-a", t0)
	// Use a time that's not in the 1-second window at the start of a 24h period.
	// The window opens at t0 (unix 0 mod 86400 = 0); 2s later it's closed.
	later := t0.Add(2 * time.Second)
	_, err := fo.Contest("crew-b", later, false)
	if err != ErrFlipWindowClosed {
		t.Errorf("expected ErrFlipWindowClosed, got %v", err)
	}
}

func TestFlip(t *testing.T) {
	fo := newFO("fo-1")
	fo.Claim("crew-a", t0)
	fo.Contest("crew-b", t0, true)
	r, err := fo.Flip(t0.Add(1 * time.Minute))
	if err != nil {
		t.Fatalf("Flip failed: %v", err)
	}
	if fo.Phase != PhaseHeld {
		t.Errorf("expected PhaseHeld after flip, got %v", fo.Phase)
	}
	if fo.HolderID != "crew-b" {
		t.Errorf("expected holder crew-b after flip, got %q", fo.HolderID)
	}
	if r.Verb != "FLIPPED" {
		t.Errorf("expected FLIPPED, got %q", r.Verb)
	}
	if fo.Flow != 0 {
		t.Errorf("Flow should reset to 0 on flip, got %.2f", fo.Flow)
	}
}

func TestDefend(t *testing.T) {
	fo := newFO("fo-1")
	fo.Claim("crew-a", t0)
	fo.Contest("crew-b", t0, true)
	r, err := fo.Defend(t0.Add(6 * time.Minute))
	if err != nil {
		t.Fatalf("Defend failed: %v", err)
	}
	if fo.Phase != PhaseHeld {
		t.Errorf("expected PhaseHeld after defend, got %v", fo.Phase)
	}
	if fo.HolderID != "crew-a" {
		t.Errorf("expected holder crew-a after defend, got %q", fo.HolderID)
	}
	if r.Verb != "DEFENDED" {
		t.Errorf("expected DEFENDED, got %q", r.Verb)
	}
}

func TestAutoDefendOnTickExpiry(t *testing.T) {
	fo := newFO("fo-1")
	fo.Claim("crew-a", t0)
	fo.Contest("crew-b", t0, true)
	// Tick after ContestEnd.
	expiry := fo.ContestEnd.Add(1 * time.Second)
	receipts := fo.Tick(expiry, time.Second)
	if len(receipts) == 0 {
		t.Fatal("expected auto-defend receipt from Tick")
	}
	if receipts[0].Verb != "DEFENDED" {
		t.Errorf("expected DEFENDED from auto-tick, got %q", receipts[0].Verb)
	}
	if fo.Phase != PhaseHeld {
		t.Errorf("expected PhaseHeld after auto-defend tick")
	}
}

// ── Containment ───────────────────────────────────────────────────────────────

func TestContainmentStart(t *testing.T) {
	fo := newFO("fo-1")
	fo.Claim("crew-a", t0)
	r := fo.StartContainment(3, t0)
	if fo.Phase != PhaseContainment {
		t.Errorf("expected PhaseContainment, got %v", fo.Phase)
	}
	if r.Verb != "CONTAINMENT_START" {
		t.Errorf("expected CONTAINMENT_START, got %q", r.Verb)
	}
}

func TestContainmentBlocksClaim(t *testing.T) {
	fo := newFO("fo-1")
	fo.StartContainment(3, t0)
	_, err := fo.Claim("crew-a", t0)
	if err != ErrInContainment {
		t.Errorf("expected ErrInContainment, got %v", err)
	}
}

func TestContainmentBlocksContest(t *testing.T) {
	fo := newFO("fo-1")
	fo.Claim("crew-a", t0)
	fo.StartContainment(3, t0)
	_, err := fo.Contest("crew-b", t0, true)
	if err != ErrInContainment {
		t.Errorf("expected ErrInContainment, got %v", err)
	}
}

func TestContainmentObjectivesResolve(t *testing.T) {
	fo := newFO("fo-1")
	fo.Claim("crew-a", t0)
	fo.StartContainment(3, t0)

	done, _ := fo.CompleteObjective(t0)
	if done {
		t.Error("should not be done after 1/3 objectives")
	}
	done, _ = fo.CompleteObjective(t0)
	if done {
		t.Error("should not be done after 2/3 objectives")
	}
	done, r := fo.CompleteObjective(t0)
	if !done {
		t.Error("should be done after 3/3 objectives")
	}
	if fo.Phase != PhaseHeld {
		t.Errorf("expected PhaseHeld after containment end, got %v", fo.Phase)
	}
	if r.Verb != "CONTAINMENT_END" {
		t.Errorf("expected CONTAINMENT_END, got %q", r.Verb)
	}
}

// ── Flow + Pressure ───────────────────────────────────────────────────────────

func TestTickAccumulatesFlow(t *testing.T) {
	fo := newFO("fo-1")
	fo.Claim("crew-a", t0)
	fo.Tick(t0.Add(10*time.Second), 10*time.Second)
	if fo.Flow <= 0 {
		t.Errorf("expected positive Flow after 10s tick, got %.4f", fo.Flow)
	}
}

func TestTickAccumulatesPressure(t *testing.T) {
	fo := newFO("fo-1")
	fo.Claim("crew-a", t0)
	fo.Tick(t0.Add(10*time.Second), 10*time.Second)
	fo.Tick(t0.Add(20*time.Second), 10*time.Second)
	if fo.Pressure <= 0 {
		t.Errorf("expected positive Pressure after 20s ticks, got %.4f", fo.Pressure)
	}
}

func TestFlowResetOnFlip(t *testing.T) {
	fo := newFO("fo-1")
	fo.Claim("crew-a", t0)
	fo.Tick(t0.Add(60*time.Second), 60*time.Second)
	if fo.Flow == 0 {
		t.Fatal("expected non-zero Flow before flip")
	}
	fo.Contest("crew-b", t0.Add(60*time.Second), true)
	fo.Flip(t0.Add(90*time.Second))
	if fo.Flow != 0 {
		t.Errorf("expected Flow reset to 0 after flip, got %.4f", fo.Flow)
	}
}

// ── Registry ──────────────────────────────────────────────────────────────────

func TestRegistryAddAndGet(t *testing.T) {
	reg := NewRegistry()
	fo := newFO("fo-test")
	reg.Add(fo)
	got, ok := reg.Get("fo-test")
	if !ok || got != fo {
		t.Error("Registry.Get should return the added FO")
	}
}

func TestRegistryAll(t *testing.T) {
	reg := NewRegistry()
	reg.Add(newFO("fo-1"))
	reg.Add(newFO("fo-2"))
	if len(reg.All()) != 2 {
		t.Errorf("expected 2 FOs, got %d", len(reg.All()))
	}
}

func TestRegistryTickAll(t *testing.T) {
	reg := NewRegistry()
	fo1, fo2 := newFO("fo-1"), newFO("fo-2")
	fo1.Claim("crew-a", t0)
	fo2.Claim("crew-b", t0)
	reg.Add(fo1)
	reg.Add(fo2)
	// Contest fo2 so it auto-defends on tick.
	fo2.Contest("crew-c", t0, true)
	expiry := fo2.ContestEnd.Add(time.Second)
	receipts := reg.TickAll(expiry, time.Second)
	// Should have at least the auto-defend receipt from fo2.
	if len(receipts) == 0 {
		t.Error("expected receipts from TickAll")
	}
}

func TestDefaultFieldOffices(t *testing.T) {
	fos := DefaultFieldOffices(nil)
	if len(fos) != 6 {
		t.Errorf("expected 6 default field offices, got %d", len(fos))
	}
	for _, fo := range fos {
		if fo.Phase != PhaseUnclaimed {
			t.Errorf("FO %s should start Unclaimed", fo.ID)
		}
	}
}

func TestSummaryStrings(t *testing.T) {
	fo := newFO("fo-1")
	_ = fo.Summary() // unclaimed

	fo.Claim("crew-a", t0)
	_ = fo.Summary() // held

	fo.Contest("crew-b", t0, true)
	_ = fo.Summary() // contested

	fo.StartContainment(3, t0)
	_ = fo.Summary() // containment
}

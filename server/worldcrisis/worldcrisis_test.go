package worldcrisis

import (
	"testing"
	"time"
)

func now() time.Time { return time.Now() }

func TestNew_IdleState(t *testing.T) {
	c := New()
	if c.Phase != PhaseIdle {
		t.Errorf("phase: got %q, want idle", c.Phase)
	}
	if c.LeyIntegrity != LeyMax {
		t.Errorf("ley: got %d, want %d", c.LeyIntegrity, LeyMax)
	}
}

func TestStart_AdvancesToOmens(t *testing.T) {
	c := New()
	n := now()
	if err := c.Start(n, "ev-001"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if c.Phase != PhaseOmens {
		t.Errorf("phase: got %q, want omens", c.Phase)
	}
	if c.EventID != "ev-001" {
		t.Errorf("EventID: got %q, want ev-001", c.EventID)
	}
}

func TestStart_AlreadyActive(t *testing.T) {
	c := New()
	c.Start(now(), "ev-001")
	if err := c.Start(now(), "ev-002"); err != ErrAlreadyActive {
		t.Errorf("double start: got %v, want ErrAlreadyActive", err)
	}
}

func TestTick_NoAdvanceBeforeDeadline(t *testing.T) {
	c := New()
	n := now()
	c.Start(n, "ev-1")
	changed, _, _ := c.Tick(n.Add(time.Second))
	if changed {
		t.Error("tick before deadline should not change phase")
	}
	if c.Phase != PhaseOmens {
		t.Errorf("phase: got %q, want omens", c.Phase)
	}
}

func TestTick_AdvanceOmensToBurrow(t *testing.T) {
	c := New()
	c.SetPhaseDuration(PhaseOmens, 10*time.Millisecond)
	n := now()
	c.Start(n, "ev-1")
	changed, old, newP := c.Tick(n.Add(20 * time.Millisecond))
	if !changed {
		t.Error("should have advanced")
	}
	if old != PhaseOmens || newP != PhaseBurrow {
		t.Errorf("transition: got %q→%q, want omens→burrow", old, newP)
	}
}

func TestTick_FullSequence(t *testing.T) {
	c := New()
	c.ConcurrentGate = time.Hour
	d := 5 * time.Millisecond
	for _, p := range []Phase{PhaseOmens, PhaseBurrow, PhaseEmergence, PhaseSplitWar, PhaseFinalWindow} {
		c.SetPhaseDuration(p, d)
	}
	n := now()
	c.Start(n, "ev-seq")
	// Complete all 3 objectives before the sequence ends.
	c.CompleteObjective(ObjectiveAnchor, 0, n)
	c.CompleteObjective(ObjectiveRitual, 0, n)
	c.CompleteObjective(ObjectiveIntercept, 0, n)
	// Drive clock forward one tick at a time; each tick >= deadline advances the phase.
	tick := n.Add(2 * time.Millisecond) // start after first phase deadline
	var lastPhase Phase = PhaseOmens
	for i := 0; i < 30; i++ {
		changed, _, np := c.Tick(tick)
		if changed {
			lastPhase = np
		}
		tick = tick.Add(d) // advance by one phase duration each loop
	}
	if lastPhase != PhaseResolution {
		t.Errorf("full sequence: ended at %q, want resolution", lastPhase)
	}
	if c.Outcome != OutcomeVictory {
		t.Errorf("outcome: got %q, want victory", c.Outcome)
	}
}

func TestTick_IdleNoChange(t *testing.T) {
	c := New()
	changed, _, _ := c.Tick(now())
	if changed {
		t.Error("idle tick should not change phase")
	}
}

func TestLeyDecay_DropsToMin(t *testing.T) {
	c := New()
	c.Start(now(), "ev-1")
	c.LeyDecay(200) // way over max
	if c.LeyIntegrity != LeyMin {
		t.Errorf("ley clamp: got %d, want %d", c.LeyIntegrity, LeyMin)
	}
}

func TestLeyDecay_AutoFailOnTick(t *testing.T) {
	c := New()
	c.SetPhaseDuration(PhaseOmens, time.Hour)
	n := now()
	c.Start(n, "ev-1")
	c.LeyDecay(100) // drain to 0
	changed, _, newP := c.Tick(n.Add(time.Millisecond))
	if !changed {
		t.Error("ley=0 should trigger resolution on tick")
	}
	if newP != PhaseResolution {
		t.Errorf("got %q, want resolution", newP)
	}
	if c.Outcome != OutcomeFailure {
		t.Errorf("outcome: got %q, want failure", c.Outcome)
	}
}

func TestCompleteObjective_StabilisesLey(t *testing.T) {
	c := New()
	c.Start(now(), "ev-1")
	c.LeyDecay(30)
	before := c.LeyIntegrity
	c.CompleteObjective(ObjectiveAnchor, 20, now())
	if c.LeyIntegrity != before+20 {
		t.Errorf("stabilise: got %d, want %d", c.LeyIntegrity, before+20)
	}
}

func TestCompleteObjective_CapsAtMax(t *testing.T) {
	c := New()
	c.Start(now(), "ev-1")
	c.CompleteObjective(ObjectiveAnchor, 999, now())
	if c.LeyIntegrity > LeyMax {
		t.Errorf("ley over max: got %d", c.LeyIntegrity)
	}
}

func TestCompleteObjective_NotActive(t *testing.T) {
	c := New()
	if err := c.CompleteObjective(ObjectiveAnchor, 10, now()); err != ErrNotActive {
		t.Errorf("inactive: got %v, want ErrNotActive", err)
	}
}

func TestConcurrentGateMet_AllThreeInWindow(t *testing.T) {
	c := New()
	c.ConcurrentGate = time.Hour
	n := now()
	c.Start(n, "ev-1")
	c.CompleteObjective(ObjectiveAnchor, 0, n)
	c.CompleteObjective(ObjectiveRitual, 0, n.Add(time.Second))
	c.CompleteObjective(ObjectiveIntercept, 0, n.Add(2*time.Second))
	c.mu.RLock()
	met := c.concurrentGateMet(n.Add(3 * time.Second))
	c.mu.RUnlock()
	if !met {
		t.Error("concurrent gate: should be met")
	}
}

func TestConcurrentGateMet_TooSpread(t *testing.T) {
	c := New()
	c.ConcurrentGate = time.Second
	n := now()
	c.Start(n, "ev-1")
	c.CompleteObjective(ObjectiveAnchor, 0, n)
	c.CompleteObjective(ObjectiveRitual, 0, n.Add(10*time.Second))
	c.CompleteObjective(ObjectiveIntercept, 0, n.Add(20*time.Second))
	c.mu.RLock()
	met := c.concurrentGateMet(n.Add(20 * time.Second))
	c.mu.RUnlock()
	if met {
		t.Error("concurrent gate: should NOT be met when too spread")
	}
}

func TestConcurrentGateMet_OnlyTwo(t *testing.T) {
	c := New()
	c.ConcurrentGate = time.Hour
	n := now()
	c.Start(n, "ev-1")
	c.CompleteObjective(ObjectiveAnchor, 0, n)
	c.CompleteObjective(ObjectiveRitual, 0, n)
	c.mu.RLock()
	met := c.concurrentGateMet(n)
	c.mu.RUnlock()
	if met {
		t.Error("only 2 objectives: gate should not be met")
	}
}

func TestOutcome_FailsOnLeyZero(t *testing.T) {
	c := New()
	c.ConcurrentGate = time.Minute
	for _, p := range []Phase{PhaseOmens, PhaseBurrow, PhaseEmergence, PhaseSplitWar, PhaseFinalWindow} {
		c.SetPhaseDuration(p, 5*time.Millisecond)
	}
	n := now()
	c.Start(n, "ev-fail")
	c.LeyDecay(100) // drain
	c.Tick(n.Add(time.Millisecond))
	if c.Outcome != OutcomeFailure {
		t.Errorf("ley 0: outcome=%q, want failure", c.Outcome)
	}
}

func TestStatus_Snapshot(t *testing.T) {
	c := New()
	n := now()
	c.Start(n, "ev-snap")
	s := c.Status()
	if s.Phase != PhaseOmens {
		t.Errorf("status phase: got %q, want omens", s.Phase)
	}
	if s.LeyIntegrity != LeyMax {
		t.Errorf("status ley: got %d, want %d", s.LeyIntegrity, LeyMax)
	}
	if s.EventID != "ev-snap" {
		t.Errorf("status eventID: got %q", s.EventID)
	}
}

func TestReset_ClearsState(t *testing.T) {
	c := New()
	c.Start(now(), "ev-1")
	c.Reset()
	if c.Phase != PhaseIdle {
		t.Errorf("after reset: phase=%q, want idle", c.Phase)
	}
	if c.EventID != "" {
		t.Error("after reset: EventID should be empty")
	}
}

func TestTickDecay_LeyDropsOverTime(t *testing.T) {
	c := New()
	c.SetPhaseDuration(PhaseOmens, time.Hour)
	n := now()
	c.Start(n, "ev-decay")
	// advance 2 decay intervals
	c.Tick(n.Add(2 * DecayInterval))
	if c.LeyIntegrity >= LeyMax {
		t.Error("ley should have decayed after two intervals")
	}
}

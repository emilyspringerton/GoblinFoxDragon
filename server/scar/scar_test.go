package scar

import (
	"strings"
	"testing"
)

func TestAppendAndCount(t *testing.T) {
	r := NewRegistry()
	_, err := r.Append("district-alpha", CauseRogueSwarm, "rogue swarm containment failed")
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	_, err = r.Append("district-alpha", CauseCrownProtocol, "Tier-5 crown event")
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if r.Count("district-alpha") != 2 {
		t.Errorf("Count=%d want 2", r.Count("district-alpha"))
	}
	if r.Count("district-beta") != 0 {
		t.Errorf("Count(beta)=%d want 0", r.Count("district-beta"))
	}
}

func TestVisibilityBonus(t *testing.T) {
	r := NewRegistry()
	if r.VisibilityBonus("x") != 0 {
		t.Error("empty registry: expected 0")
	}
	r.Append("x", CauseMercilessOp, "op complete")
	if r.VisibilityBonus("x") != 0.05 {
		t.Errorf("1 scar: expected 0.05, got %.4f", r.VisibilityBonus("x"))
	}
	r.Append("x", CauseFactionWar, "faction war annexation")
	if r.VisibilityBonus("x") != 0.10 {
		t.Errorf("2 scars: expected 0.10, got %.4f", r.VisibilityBonus("x"))
	}
}

func TestRemoveLast(t *testing.T) {
	r := NewRegistry()
	r.Append("d", CauseRogueSwarm, "first")
	r.Append("d", CauseCrownProtocol, "second")

	removed, ok := r.RemoveLast("d")
	if !ok {
		t.Fatal("expected removal to succeed")
	}
	if removed.Cause != CauseCrownProtocol {
		t.Errorf("removed.Cause=%q want CrownProtocol", removed.Cause)
	}
	if r.Count("d") != 1 {
		t.Errorf("count after remove=%d want 1", r.Count("d"))
	}
	// Global All() should also reflect the removal.
	if len(r.All()) != 1 {
		t.Errorf("All() len=%d want 1", len(r.All()))
	}
}

func TestRemoveLastEmpty(t *testing.T) {
	r := NewRegistry()
	_, ok := r.RemoveLast("nonexistent")
	if ok {
		t.Error("expected false for nonexistent district")
	}
}

func TestForDistrict_Isolated(t *testing.T) {
	r := NewRegistry()
	r.Append("d1", CauseMercilessOp, "")
	r.Append("d2", CauseRogueSwarm, "")
	r.Append("d1", CauseFactionWar, "")

	d1 := r.ForDistrict("d1")
	if len(d1) != 2 {
		t.Errorf("d1 scars=%d want 2", len(d1))
	}
	d2 := r.ForDistrict("d2")
	if len(d2) != 1 {
		t.Errorf("d2 scars=%d want 1", len(d2))
	}
}

func TestMUDScarsCommand(t *testing.T) {
	r := NewRegistry()
	r.Append("d-main", CauseRogueSwarm, "swarm breach event")
	r.Append("d-main", CauseMercilessOp, "K9 op resolution")

	out := r.MUDScarsCommand("d-main")
	if !strings.Contains(out, "2 total") {
		t.Errorf("expected '2 total' in output: %q", out)
	}
	if !strings.Contains(out, "10%") {
		t.Errorf("expected '10%%' visibility bonus in output: %q", out)
	}
	if !strings.Contains(out, "RogueSwarm") || !strings.Contains(out, "MercilessOp") {
		t.Errorf("output missing cause names: %q", out)
	}
}

func TestMUDScarsCommandEmpty(t *testing.T) {
	r := NewRegistry()
	out := r.MUDScarsCommand("empty-district")
	if !strings.Contains(out, "No scars") {
		t.Errorf("expected 'No scars', got %q", out)
	}
}

func TestAppendBadInputs(t *testing.T) {
	r := NewRegistry()
	_, err := r.Append("", CauseRogueSwarm, "")
	if err != ErrEmptyDistrictID {
		t.Errorf("expected ErrEmptyDistrictID, got %v", err)
	}
	_, err = r.Append("d", "NotACause", "")
	if err != ErrUnknownCause {
		t.Errorf("expected ErrUnknownCause, got %v", err)
	}
}

func TestAppendIDUnique(t *testing.T) {
	r := NewRegistry()
	s1, _ := r.Append("d", CauseRogueSwarm, "")
	s2, _ := r.Append("d", CauseMercilessOp, "")
	if s1.ID == s2.ID {
		t.Errorf("duplicate IDs: %s", s1.ID)
	}
}

func TestAllReturnsCopyNotReference(t *testing.T) {
	r := NewRegistry()
	r.Append("d", CauseRogueSwarm, "")
	snapshot := r.All()
	r.Append("d", CauseFactionWar, "")
	// Snapshot should not grow.
	if len(snapshot) != 1 {
		t.Errorf("All() snapshot mutated: len=%d", len(snapshot))
	}
}

func TestConcurrentAppend(t *testing.T) {
	r := NewRegistry()
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			r.Append("d", CauseRogueSwarm, "concurrent")
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	if r.Count("d") != 10 {
		t.Errorf("count=%d want 10", r.Count("d"))
	}
}

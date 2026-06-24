package timeline

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

// ── NewRegistry ───────────────────────────────────────────────────────────────

func TestNewRegistryHasPresentBranch(t *testing.T) {
	r := NewRegistry()
	b := r.Get(DefaultBranch)
	if b == nil {
		t.Fatal("expected present branch to exist")
	}
	if b.ID != DefaultBranch {
		t.Errorf("expected ID %q, got %q", DefaultBranch, b.ID)
	}
}

func TestNewRegistryBranchCount(t *testing.T) {
	r := NewRegistry()
	if r.BranchCount() != 1 {
		t.Errorf("expected 1 branch, got %d", r.BranchCount())
	}
}

// ── Cut ───────────────────────────────────────────────────────────────────────

func TestCutCreatesNewBranch(t *testing.T) {
	r := NewRegistry()
	_, err := r.Cut("vs0-divergence", DefaultBranch, "migration_event", "VS0 branch", t0, nil)
	if err != nil {
		t.Fatalf("Cut failed: %v", err)
	}
	if r.BranchCount() != 2 {
		t.Errorf("expected 2 branches, got %d", r.BranchCount())
	}
}

func TestCutStoresDistricts(t *testing.T) {
	r := NewRegistry()
	input := []SnapshotInput{
		{DistrictID: "district-residential", ScarCount: 3, Alertness: 72},
	}
	b, err := r.Cut("vs0-divergence", DefaultBranch, "migration_event", "VS0", t0, input)
	if err != nil {
		t.Fatalf("Cut failed: %v", err)
	}
	snap, ok := b.Districts["district-residential"]
	if !ok {
		t.Fatal("expected district-residential in snapshot")
	}
	if snap.ScarCount != 3 {
		t.Errorf("expected ScarCount=3, got %d", snap.ScarCount)
	}
	if snap.AlertnessAtBirth != 72 {
		t.Errorf("expected AlertnessAtBirth=72, got %.1f", snap.AlertnessAtBirth)
	}
}

func TestCutDuplicateIDReturnsError(t *testing.T) {
	r := NewRegistry()
	r.Cut("vs0-divergence", DefaultBranch, "rogue_swarm", "first", t0, nil)
	_, err := r.Cut("vs0-divergence", DefaultBranch, "rogue_swarm", "duplicate", t0, nil)
	if err == nil {
		t.Error("expected error on duplicate branch ID")
	}
}

func TestCutUnknownParentReturnsError(t *testing.T) {
	r := NewRegistry()
	_, err := r.Cut("orphan", "nonexistent-parent", "manual", "", t0, nil)
	if err == nil {
		t.Error("expected error for unknown parent")
	}
}

// ── TotalScars ────────────────────────────────────────────────────────────────

func TestTotalScars(t *testing.T) {
	r := NewRegistry()
	b, _ := r.Cut("b1", DefaultBranch, "test", "", t0, []SnapshotInput{
		{DistrictID: "district-residential", ScarCount: 3},
		{DistrictID: "district-commercial", ScarCount: 2},
	})
	if b.TotalScars() != 5 {
		t.Errorf("expected 5 total scars, got %d", b.TotalScars())
	}
}

func TestScarDelta(t *testing.T) {
	r := NewRegistry()
	base := r.Get(DefaultBranch)
	newB, _ := r.Cut("heavy", DefaultBranch, "rogue_swarm", "", t0, []SnapshotInput{
		{DistrictID: "district-abandoned", ScarCount: 5},
	})
	if newB.ScarDelta(base) != 5 {
		t.Errorf("expected delta=5, got %d", newB.ScarDelta(base))
	}
}

// ── SummaryLines ──────────────────────────────────────────────────────────────

func TestSummaryLinesContainsBranches(t *testing.T) {
	r := NewRegistry()
	r.Cut("branch-a", DefaultBranch, "migration_event", "Branch A", t0, nil)
	lines := r.SummaryLines()
	if len(lines) < 3 { // header + present + branch-a
		t.Errorf("expected at least 3 lines, got %d: %v", len(lines), lines)
	}
}

// ── DistrictHasConflict ───────────────────────────────────────────────────────

func TestDistrictHasConflictWhenScarsDiffer(t *testing.T) {
	r := NewRegistry()
	r.Cut("branch-low", DefaultBranch, "rogue_swarm", "", t0, []SnapshotInput{
		{DistrictID: "district-residential", ScarCount: 1},
	})
	r.Cut("branch-high", DefaultBranch, "rogue_swarm", "", t0.Add(time.Second), []SnapshotInput{
		{DistrictID: "district-residential", ScarCount: 5},
	})
	if !r.DistrictHasConflict("district-residential", "branch-low", "branch-high") {
		t.Error("expected conflict when scar counts differ")
	}
}

func TestDistrictNoConflictWhenScarsEqual(t *testing.T) {
	r := NewRegistry()
	r.Cut("a", DefaultBranch, "manual", "", t0, []SnapshotInput{
		{DistrictID: "district-residential", ScarCount: 3},
	})
	r.Cut("b", DefaultBranch, "manual", "", t0, []SnapshotInput{
		{DistrictID: "district-residential", ScarCount: 3},
	})
	if r.DistrictHasConflict("district-residential", "a", "b") {
		t.Error("no conflict expected when scar counts are equal")
	}
}

func TestDistrictNoConflictForUnknownBranch(t *testing.T) {
	r := NewRegistry()
	if r.DistrictHasConflict("district-residential", "nonexistent", DefaultBranch) {
		t.Error("no conflict expected when branch doesn't exist")
	}
}

// ── All ───────────────────────────────────────────────────────────────────────

func TestAllReturnsInInsertionOrder(t *testing.T) {
	r := NewRegistry()
	r.Cut("first", DefaultBranch, "manual", "", t0, nil)
	r.Cut("second", DefaultBranch, "manual", "", t0, nil)
	all := r.All()
	if len(all) != 3 {
		t.Fatalf("expected 3 branches, got %d", len(all))
	}
	if all[0].ID != DefaultBranch {
		t.Errorf("first should be %q, got %q", DefaultBranch, all[0].ID)
	}
	if all[1].ID != "first" {
		t.Errorf("second should be 'first', got %q", all[1].ID)
	}
}

package enmity

import (
	"sort"
	"testing"
)

// ── Ensure + Add ──────────────────────────────────────────────────────────────

func TestAdd_RegistersAndAccumulates(t *testing.T) {
	tb := NewTable()
	tb.Add("alice", 100)
	tb.Add("alice", 50)
	s, _ := tb.Score("alice")
	if s != 150 {
		t.Errorf("Add: got %d, want 150", s)
	}
}

func TestAdd_CapAt10000(t *testing.T) {
	tb := NewTable()
	tb.Add("alice", 9999)
	tb.Add("alice", 999) // would exceed cap
	s, _ := tb.Score("alice")
	if s != EnmityCap {
		t.Errorf("cap: got %d, want %d", s, EnmityCap)
	}
}

func TestAdd_MultipleSlots(t *testing.T) {
	tb := NewTable()
	tb.Add("alice", 200)
	tb.Add("bob", 100)
	if tb.Len() != 2 {
		t.Errorf("Len: got %d, want 2", tb.Len())
	}
}

// ── AddCure ───────────────────────────────────────────────────────────────────

func TestAddCure_SingleTarget(t *testing.T) {
	tb := NewTable()
	tb.AddCure("healer", 300, false)
	s, _ := tb.Score("healer")
	if s != 300 {
		t.Errorf("single cure CE: got %d, want 300", s)
	}
}

func TestAddCure_AoEHalved(t *testing.T) {
	tb := NewTable()
	tb.AddCure("healer", 300, true)
	s, _ := tb.Score("healer")
	if s != 150 {
		t.Errorf("AoE cure CE halved: got %d, want 150", s)
	}
}

func TestAddCure_StacksWithDamage(t *testing.T) {
	tb := NewTable()
	tb.Add("healer", 100)
	tb.AddCure("healer", 200, false)
	s, _ := tb.Score("healer")
	if s != 300 {
		t.Errorf("cure+damage: got %d, want 300", s)
	}
}

// ── Reduce ────────────────────────────────────────────────────────────────────

func TestReduce_ClampAtZero(t *testing.T) {
	tb := NewTable()
	tb.Add("alice", 50)
	tb.Reduce("alice", 200)
	s, _ := tb.Score("alice")
	if s != 0 {
		t.Errorf("reduce below 0: got %d, want 0", s)
	}
}

func TestReduce_UnknownSlotNoOp(t *testing.T) {
	tb := NewTable()
	tb.Reduce("ghost", 100) // should not panic
	if tb.Len() != 0 {
		t.Error("Reduce unknown slot should not add it")
	}
}

// ── Top ───────────────────────────────────────────────────────────────────────

func TestTop_HighestHateTargeted(t *testing.T) {
	tb := NewTable()
	tb.Add("alice", 100)
	tb.Add("bob", 500)
	tb.Add("carol", 200)
	top, err := tb.Top()
	if err != nil {
		t.Fatalf("Top: %v", err)
	}
	if top != "bob" {
		t.Errorf("top hate: got %q, want bob", top)
	}
}

func TestTop_TieBrokenAlphabetically(t *testing.T) {
	tb := NewTable()
	tb.Add("bob", 100)
	tb.Add("alice", 100)
	top, _ := tb.Top()
	if top != "alice" {
		t.Errorf("tie: got %q, want alice (alphabetically first)", top)
	}
}

func TestTop_EmptyTableError(t *testing.T) {
	tb := NewTable()
	_, err := tb.Top()
	if err != ErrNoPlayers {
		t.Errorf("empty table: got %v, want ErrNoPlayers", err)
	}
}

// ── Score ─────────────────────────────────────────────────────────────────────

func TestScore_UnknownSlotError(t *testing.T) {
	tb := NewTable()
	_, err := tb.Score("ghost")
	if err != ErrUnknownPlayer {
		t.Errorf("unknown slot: got %v, want ErrUnknownPlayer", err)
	}
}

// ── Clear ─────────────────────────────────────────────────────────────────────

func TestClear_RemovesAll(t *testing.T) {
	tb := NewTable()
	tb.Add("alice", 100)
	tb.Add("bob", 200)
	tb.Clear()
	if tb.Len() != 0 {
		t.Errorf("after Clear Len=%d, want 0", tb.Len())
	}
}

func TestClear_TopReturnsError(t *testing.T) {
	tb := NewTable()
	tb.Add("alice", 100)
	tb.Clear()
	_, err := tb.Top()
	if err != ErrNoPlayers {
		t.Errorf("clear then Top: got %v, want ErrNoPlayers", err)
	}
}

// ── Remove ────────────────────────────────────────────────────────────────────

func TestRemove_SpecificSlot(t *testing.T) {
	tb := NewTable()
	tb.Add("alice", 100)
	tb.Add("bob", 200)
	tb.Remove("alice")
	if tb.Len() != 1 {
		t.Errorf("after Remove Len=%d, want 1", tb.Len())
	}
	if _, err := tb.Score("alice"); err != ErrUnknownPlayer {
		t.Error("removed slot should return ErrUnknownPlayer")
	}
}

// ── Slots ─────────────────────────────────────────────────────────────────────

func TestSlots_ReturnsAll(t *testing.T) {
	tb := NewTable()
	tb.Add("alice", 10)
	tb.Add("bob", 20)
	slots := tb.Slots()
	sort.Strings(slots)
	if len(slots) != 2 || slots[0] != "alice" || slots[1] != "bob" {
		t.Errorf("Slots: got %v, want [alice bob]", slots)
	}
}

// ── Overaggro scenario ────────────────────────────────────────────────────────

func TestOveraggro_TankVsDPS(t *testing.T) {
	tb := NewTable()
	// Tank deals modest damage but healer cures tank heavily
	tb.Add("tank", 300)
	tb.Add("dps", 800)
	// Healer cures tank for 500 → 500 CE on healer
	tb.AddCure("healer", 500, false)
	// DPS should still be top with 800
	top, _ := tb.Top()
	if top != "dps" {
		t.Errorf("overaggro: want dps on top, got %q", top)
	}
	// Now tank uses provoke-like mechanic (manual add)
	tb.Add("tank", 600)
	top, _ = tb.Top()
	if top != "tank" {
		t.Errorf("after tank provoke: want tank on top, got %q", top)
	}
}

package loot

import "testing"

func twoPlayerPool() *Pool {
	return NewPool("mob-001",
		[]string{"alice", "bob"},
		[]Item{{ID: "sword", Name: "Iron Sword"}, {ID: "potion", Name: "Potion"}},
		12345,
	)
}

// ── Lot ───────────────────────────────────────────────────────────────────────

func TestLot_InRange(t *testing.T) {
	p := twoPlayerPool()
	roll, err := p.Lot("alice", "sword")
	if err != nil {
		t.Fatalf("Lot: %v", err)
	}
	if roll < LotMin || roll > LotMax {
		t.Errorf("roll %d out of range [%d,%d]", roll, LotMin, LotMax)
	}
}

func TestLot_NotEligible(t *testing.T) {
	p := twoPlayerPool()
	if _, err := p.Lot("stranger", "sword"); err != ErrNotEligible {
		t.Errorf("non-eligible lot: got %v, want ErrNotEligible", err)
	}
}

func TestLot_ItemNotFound(t *testing.T) {
	p := twoPlayerPool()
	if _, err := p.Lot("alice", "nonexistent"); err != ErrItemNotFound {
		t.Errorf("bad item: got %v, want ErrItemNotFound", err)
	}
}

func TestLot_DuplicateReturnsError(t *testing.T) {
	p := twoPlayerPool()
	_, _ = p.Lot("alice", "sword")
	if _, err := p.Lot("alice", "sword"); err != ErrAlreadyActed {
		t.Errorf("duplicate lot: got %v, want ErrAlreadyActed", err)
	}
}

// ── Pass ──────────────────────────────────────────────────────────────────────

func TestPass_RecordsZeroRoll(t *testing.T) {
	p := twoPlayerPool()
	if err := p.Pass("alice", "sword"); err != nil {
		t.Fatalf("Pass: %v", err)
	}
	status, _ := p.LotStatus("sword")
	for _, lr := range status {
		if lr.Slot == "alice" && lr.Roll != 0 {
			t.Errorf("pass should record roll=0: got %d", lr.Roll)
		}
	}
}

func TestPass_NotEligible(t *testing.T) {
	p := twoPlayerPool()
	if err := p.Pass("stranger", "sword"); err != ErrNotEligible {
		t.Errorf("non-eligible pass: got %v, want ErrNotEligible", err)
	}
}

func TestPass_AfterLotReturnsError(t *testing.T) {
	p := twoPlayerPool()
	_, _ = p.Lot("alice", "sword")
	if err := p.Pass("alice", "sword"); err != ErrAlreadyActed {
		t.Errorf("pass after lot: got %v, want ErrAlreadyActed", err)
	}
}

// ── Ready ─────────────────────────────────────────────────────────────────────

func TestReady_FalseWhenPending(t *testing.T) {
	p := twoPlayerPool()
	_, _ = p.Lot("alice", "sword")
	// bob hasn't acted yet on sword; nobody acted on potion
	if p.Ready() {
		t.Error("should not be ready with pending actions")
	}
}

func TestReady_TrueWhenAllActed(t *testing.T) {
	p := twoPlayerPool()
	_, _ = p.Lot("alice", "sword")
	_ = p.Pass("bob", "sword")
	_, _ = p.Lot("alice", "potion")
	_, _ = p.Lot("bob", "potion")
	if !p.Ready() {
		t.Error("should be ready when all players have acted on all items")
	}
}

// ── Resolve ───────────────────────────────────────────────────────────────────

func TestResolve_HighestLotWins(t *testing.T) {
	// Use a pool where we can control rolls deterministically by seeding.
	p := NewPool("mob", []string{"alice", "bob"}, []Item{{ID: "it", Name: "Item"}}, 99999)
	roll1, _ := p.Lot("alice", "it")
	roll2, _ := p.Lot("bob", "it")
	awards, err := p.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(awards) != 1 {
		t.Fatalf("awards: got %d, want 1", len(awards))
	}
	expected := "alice"
	if roll2 > roll1 {
		expected = "bob"
	}
	if awards[0].Slot != expected {
		t.Errorf("winner: got %q, expected highest roller (%q)", awards[0].Slot, expected)
	}
}

func TestResolve_AllPassNoWinner(t *testing.T) {
	p := twoPlayerPool()
	_ = p.Pass("alice", "sword")
	_ = p.Pass("bob", "sword")
	_ = p.Pass("alice", "potion")
	_ = p.Pass("bob", "potion")
	awards, _ := p.Resolve()
	for _, a := range awards {
		if a.Slot != "" {
			t.Errorf("all-pass item %s should have no winner: got %q", a.ItemID, a.Slot)
		}
	}
}

func TestResolve_DoubleCallReturnsError(t *testing.T) {
	p := twoPlayerPool()
	p.Resolve()
	if _, err := p.Resolve(); err != ErrPoolResolved {
		t.Errorf("double resolve: got %v, want ErrPoolResolved", err)
	}
}

func TestResolve_LotAfterResolveError(t *testing.T) {
	p := twoPlayerPool()
	p.Resolve()
	if _, err := p.Lot("alice", "sword"); err != ErrPoolResolved {
		t.Errorf("lot after resolve: got %v, want ErrPoolResolved", err)
	}
}

func TestResolve_TieBrokenBySlotOrder(t *testing.T) {
	// Manually inject identical rolls to test tie-break.
	p := NewPool("mob", []string{"alice", "bob"}, []Item{{ID: "it", Name: "x"}}, 1)
	// Force same roll by passing custom results via the public API isn't possible —
	// instead verify the property: with random rolls we never get a panic or empty winner
	// when both lot.
	_, _ = p.Lot("alice", "it")
	_, _ = p.Lot("bob", "it")
	awards, _ := p.Resolve()
	if awards[0].Slot == "" {
		t.Error("tie should still produce a winner via slot tie-break")
	}
}

func TestResolve_PartialActionsResolves(t *testing.T) {
	// bob hasn't acted — treated as pass.
	p := twoPlayerPool()
	roll, _ := p.Lot("alice", "sword")
	_ = p.Pass("alice", "potion")
	awards, err := p.Resolve()
	if err != nil {
		t.Fatalf("partial resolve: %v", err)
	}
	// alice should win sword (only lot)
	var swordAward Award
	for _, a := range awards {
		if a.ItemID == "sword" {
			swordAward = a
		}
	}
	if swordAward.Slot != "alice" {
		t.Errorf("alice should win sword: got %q (roll=%d)", swordAward.Slot, roll)
	}
}

// ── LotStatus ─────────────────────────────────────────────────────────────────

func TestLotStatus_ReturnsActed(t *testing.T) {
	p := twoPlayerPool()
	_, _ = p.Lot("alice", "sword")
	_ = p.Pass("bob", "sword")
	status, ok := p.LotStatus("sword")
	if !ok {
		t.Fatal("LotStatus returned false for valid item")
	}
	if len(status) != 2 {
		t.Fatalf("status len: got %d, want 2", len(status))
	}
}

func TestLotStatus_UnknownItemFalse(t *testing.T) {
	p := twoPlayerPool()
	_, ok := p.LotStatus("nonexistent")
	if ok {
		t.Error("unknown item should return false")
	}
}

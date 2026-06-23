package combat

import "testing"

// ── TakeDamage ────────────────────────────────────────────────────────────────

func TestTakeDamage_ReducesHP(t *testing.T) {
	h := NewHPState(100)
	ko, err := h.TakeDamage(30)
	if err != nil || ko {
		t.Errorf("TakeDamage: ko=%v err=%v", ko, err)
	}
	if h.Current != 70 {
		t.Errorf("HP: got %d, want 70", h.Current)
	}
}

func TestTakeDamage_TriggersKO(t *testing.T) {
	h := NewHPState(100)
	ko, err := h.TakeDamage(100)
	if err != nil || !ko {
		t.Errorf("lethal damage: ko=%v err=%v, want ko=true", ko, err)
	}
	if !h.IsKO || h.Current != 0 {
		t.Errorf("KO state: IsKO=%v HP=%d", h.IsKO, h.Current)
	}
}

func TestTakeDamage_WhenKO(t *testing.T) {
	h := NewHPState(100)
	h.TakeKO()
	_, err := h.TakeDamage(10)
	if err != ErrAlreadyKO {
		t.Errorf("damage while KO: got %v, want ErrAlreadyKO", err)
	}
}

func TestTakeDamage_OverkillClampsToZero(t *testing.T) {
	h := NewHPState(100)
	h.TakeDamage(1000) // overkill
	if h.Current != 0 {
		t.Errorf("overkill HP: got %d, want 0", h.Current)
	}
}

// ── Heal ─────────────────────────────────────────────────────────────────────

func TestHeal_RestoresHP(t *testing.T) {
	h := NewHPState(100)
	h.TakeDamage(40)
	h.Heal(20)
	if h.Current != 80 {
		t.Errorf("Heal: got %d, want 80", h.Current)
	}
}

func TestHeal_ClampsAtMax(t *testing.T) {
	h := NewHPState(100)
	h.TakeDamage(10)
	h.Heal(999)
	if h.Current != 100 {
		t.Errorf("over-heal clamp: got %d, want 100", h.Current)
	}
}

func TestHeal_NoOpWhenKO(t *testing.T) {
	h := NewHPState(100)
	h.TakeKO()
	h.Heal(50)
	if h.Current != 0 {
		t.Errorf("heal while KO should no-op: got %d, want 0", h.Current)
	}
}

// ── TakeKO ───────────────────────────────────────────────────────────────────

func TestTakeKO_SetsState(t *testing.T) {
	h := NewHPState(100)
	if err := h.TakeKO(); err != nil {
		t.Fatalf("TakeKO: %v", err)
	}
	if !h.IsKO || h.Current != 0 {
		t.Errorf("KO state: IsKO=%v HP=%d", h.IsKO, h.Current)
	}
}

func TestTakeKO_AlreadyKO(t *testing.T) {
	h := NewHPState(100)
	h.TakeKO()
	if err := h.TakeKO(); err != ErrAlreadyKO {
		t.Errorf("double KO: got %v, want ErrAlreadyKO", err)
	}
}

// ── Raise ─────────────────────────────────────────────────────────────────────

func TestRaise_RestoresHPTo1(t *testing.T) {
	h := NewHPState(100)
	h.TakeKO()
	_, err := h.RaiseDefault(1000)
	if err != nil {
		t.Fatalf("Raise: %v", err)
	}
	if h.Current != RaiseHPRestore {
		t.Errorf("HP after raise: got %d, want %d", h.Current, RaiseHPRestore)
	}
	if h.IsKO {
		t.Error("IsKO should be false after Raise")
	}
}

func TestRaise_XPPenalty(t *testing.T) {
	h := NewHPState(100)
	h.TakeKO()
	penalty, err := h.RaiseDefault(1000) // 10% of 1000 = 100
	if err != nil {
		t.Fatalf("Raise: %v", err)
	}
	if penalty != 100 {
		t.Errorf("XP penalty: got %d, want 100", penalty)
	}
}

func TestRaise_NotKOError(t *testing.T) {
	h := NewHPState(100)
	_, err := h.RaiseDefault(500)
	if err != ErrNotKO {
		t.Errorf("raise alive: got %v, want ErrNotKO", err)
	}
}

func TestRaise_ZeroXPNoPanic(t *testing.T) {
	h := NewHPState(100)
	h.TakeKO()
	penalty, err := h.RaiseDefault(0)
	if err != nil {
		t.Fatalf("Raise 0xp: %v", err)
	}
	if penalty != 0 {
		t.Errorf("0xp penalty: got %d, want 0", penalty)
	}
}

func TestRaise_CustomPenaltyPct(t *testing.T) {
	h := NewHPState(100)
	h.TakeKO()
	penalty, _ := h.Raise(500, 20.0) // 20% of 500 = 100
	if penalty != 100 {
		t.Errorf("custom 20%%: got %d, want 100", penalty)
	}
}

// ── HPPct ─────────────────────────────────────────────────────────────────────

func TestHPPct_Full(t *testing.T) {
	h := NewHPState(100)
	if h.HPPct() != 100 {
		t.Errorf("full HP%%: got %d, want 100", h.HPPct())
	}
}

func TestHPPct_Half(t *testing.T) {
	h := NewHPState(100)
	h.TakeDamage(50)
	if h.HPPct() != 50 {
		t.Errorf("50%% HP: got %d, want 50", h.HPPct())
	}
}

func TestHPPct_KO(t *testing.T) {
	h := NewHPState(100)
	h.TakeKO()
	if h.HPPct() != 0 {
		t.Errorf("KO HP%%: got %d, want 0", h.HPPct())
	}
}

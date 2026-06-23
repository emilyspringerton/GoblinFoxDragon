package merit

import "testing"

// ── Earn ──────────────────────────────────────────────────────────────────────

func TestEarn_ConversionRate(t *testing.T) {
	b := NewMeritBank()
	got, err := b.Earn(3000)
	if err != nil {
		t.Fatalf("Earn: %v", err)
	}
	if got != 3 || b.Points != 3 {
		t.Errorf("3000 XP: earned=%d pts=%d, want earned=3 pts=3", got, b.Points)
	}
}

func TestEarn_PartialXPDiscarded(t *testing.T) {
	b := NewMeritBank()
	got, _ := b.Earn(1500)
	if got != 1 {
		t.Errorf("1500 XP: got %d merits, want 1", got)
	}
}

func TestEarn_CapEnforced(t *testing.T) {
	b := NewMeritBank()
	b.Earn(MeritCap * XPPerMerit * 2) // way over cap
	if b.Points > MeritCap {
		t.Errorf("points exceeded cap: %d > %d", b.Points, MeritCap)
	}
}

func TestEarn_AtCapError(t *testing.T) {
	b := NewMeritBank()
	b.Points = MeritCap
	_, err := b.Earn(1000)
	if err != ErrMeritCapReached {
		t.Errorf("at cap: got %v, want ErrMeritCapReached", err)
	}
}

func TestEarn_ZeroXPZeroMerits(t *testing.T) {
	b := NewMeritBank()
	got, err := b.Earn(0)
	if err != nil || got != 0 {
		t.Errorf("0 xp: got=%d err=%v", got, err)
	}
}

// ── Spend ─────────────────────────────────────────────────────────────────────

func TestSpend_AdvancesTier(t *testing.T) {
	b := NewMeritBank()
	b.Points = 5
	if err := b.Spend(CatSTR); err != nil {
		t.Fatalf("Spend: %v", err)
	}
	if b.TierOf(CatSTR) != 1 {
		t.Errorf("STR tier: got %d, want 1", b.TierOf(CatSTR))
	}
}

func TestSpend_TierCostScales(t *testing.T) {
	b := NewMeritBank()
	b.Points = 15
	// T0→T1 costs 1, T1→T2 costs 2, T2→T3 costs 3
	b.Spend(CatSTR) // 1 point, now at T1, 14 remaining
	b.Spend(CatSTR) // 2 points, now at T2, 12 remaining
	b.Spend(CatSTR) // 3 points, now at T3, 9 remaining
	if b.TierOf(CatSTR) != 3 {
		t.Errorf("STR tier: got %d, want 3", b.TierOf(CatSTR))
	}
	if b.Points != 9 {
		t.Errorf("remaining points: got %d, want 9", b.Points)
	}
}

func TestSpend_MaxTierCapped(t *testing.T) {
	b := NewMeritBank()
	b.Points = MeritCap
	// Spend all 5 tiers (1+2+3+4+5 = 15 merits)
	for i := 0; i < 5; i++ {
		if err := b.Spend(CatSTR); err != nil {
			t.Fatalf("Spend tier %d: %v", i+1, err)
		}
	}
	if err := b.Spend(CatSTR); err != ErrTierCapped {
		t.Errorf("over cap: got %v, want ErrTierCapped", err)
	}
}

func TestSpend_InsufficientMerits(t *testing.T) {
	b := NewMeritBank()
	b.Points = 0
	if err := b.Spend(CatSTR); err != ErrInsufficientMerits {
		t.Errorf("no merits: got %v, want ErrInsufficientMerits", err)
	}
}

func TestSpend_InvalidCategory(t *testing.T) {
	b := NewMeritBank()
	b.Points = 10
	if err := b.Spend("unknown-cat"); err != ErrInvalidCategory {
		t.Errorf("unknown category: got %v, want ErrInvalidCategory", err)
	}
}

// ── TotalSpent ────────────────────────────────────────────────────────────────

func TestTotalSpent_Accurate(t *testing.T) {
	b := NewMeritBank()
	b.Points = 10
	b.Spend(CatSTR) // cost 1
	b.Spend(CatSTR) // cost 2
	b.Spend(CatDEX) // cost 1
	spent := b.TotalSpent()
	if spent != 4 {
		t.Errorf("TotalSpent: got %d, want 4", spent)
	}
}

package food

import (
	"testing"
	"time"
)

var now = time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)

// ── Registry ──────────────────────────────────────────────────────────────────

func TestRegistryGetKnown(t *testing.T) {
	r := DefaultRegistry()
	f, err := r.Get(IDMeatMithkabob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.STRBonus != 5 {
		t.Errorf("expected STRBonus=5, got %d", f.STRBonus)
	}
}

func TestRegistryGetUnknown(t *testing.T) {
	r := DefaultRegistry()
	_, err := r.Get("food-invalid")
	if err != ErrUnknownFood {
		t.Errorf("expected ErrUnknownFood, got %v", err)
	}
}

func TestRegistryAllCount(t *testing.T) {
	r := DefaultRegistry()
	all := r.All()
	if len(all) != len(StandardFoods()) {
		t.Errorf("expected %d foods, got %d", len(StandardFoods()), len(all))
	}
}

func TestNewRegistryWithCustomFoods(t *testing.T) {
	custom := []Food{{ID: "food-test", Name: "Test Bread", STRBonus: 1, Duration: time.Minute}}
	r := NewRegistry(custom)
	f, err := r.Get("food-test")
	if err != nil || f.Name != "Test Bread" {
		t.Errorf("custom food not found: %v", err)
	}
}

// ── Eat ───────────────────────────────────────────────────────────────────────

func TestEatReturnsActiveEffect(t *testing.T) {
	r := DefaultRegistry()
	effect, err := r.Eat(IDMeatMithkabob, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !effect.IsActive(now) {
		t.Error("effect should be active immediately after eating")
	}
}

func TestEatUnknownReturnsError(t *testing.T) {
	r := DefaultRegistry()
	_, err := r.Eat("food-unknown", now)
	if err != ErrUnknownFood {
		t.Errorf("expected ErrUnknownFood, got %v", err)
	}
}

func TestEatReplacesOldEffect(t *testing.T) {
	r := DefaultRegistry()
	old, _ := r.Eat(IDMeatMithkabob, now)
	new_, _ := r.Eat(IDTavnazianSalad, now)
	if old.Food.ID == new_.Food.ID {
		t.Error("eating a second food should produce a different effect")
	}
	// Caller is responsible for replacing the old effect pointer.
	if new_.Food.STRBonus != 0 {
		t.Errorf("Tavnazian Salad should have STRBonus=0, got %d", new_.Food.STRBonus)
	}
	if new_.Food.DEXBonus != 5 {
		t.Errorf("Tavnazian Salad should have DEXBonus=5, got %d", new_.Food.DEXBonus)
	}
}

// ── FoodEffect.IsActive / Remaining ──────────────────────────────────────────

func TestFoodEffectIsActive(t *testing.T) {
	r := DefaultRegistry()
	effect, _ := r.Eat(IDSelbinaMilk, now)
	if !effect.IsActive(now) {
		t.Error("effect should be active at time of consumption")
	}
	expired := now.Add(effect.Food.Duration + time.Second)
	if effect.IsActive(expired) {
		t.Error("effect should not be active after duration")
	}
}

func TestFoodEffectRemaining(t *testing.T) {
	r := DefaultRegistry()
	effect, _ := r.Eat(IDRolanberryPie, now)
	rem := effect.Remaining(now.Add(5 * time.Minute))
	expected := 15 * time.Minute
	if rem < expected-time.Second || rem > expected+time.Second {
		t.Errorf("expected ~15m remaining, got %v", rem)
	}
}

func TestFoodEffectRemainingZeroWhenExpired(t *testing.T) {
	r := DefaultRegistry()
	effect, _ := r.Eat(IDPear, now)
	rem := effect.Remaining(now.Add(1 * time.Hour))
	if rem != 0 {
		t.Errorf("remaining should be 0 after expiry, got %v", rem)
	}
}

func TestNilEffectNotActive(t *testing.T) {
	var e *FoodEffect
	if e.IsActive(now) {
		t.Error("nil effect should not be active")
	}
	if e.Remaining(now) != 0 {
		t.Error("nil effect remaining should be 0")
	}
}

// ── Standard food stat correctness ───────────────────────────────────────────

func TestCrabSoupStats(t *testing.T) {
	r := DefaultRegistry()
	f, _ := r.Get(IDCrabSoup)
	if f.VITBonus != 4 || f.MNDBonus != 2 || f.HPBonus != 40 {
		t.Errorf("CrabSoup stats wrong: VIT=%d MND=%d HP=%d", f.VITBonus, f.MNDBonus, f.HPBonus)
	}
}

func TestSwampHerbalTeaStats(t *testing.T) {
	r := DefaultRegistry()
	f, _ := r.Get(IDSwampHerbalTea)
	if f.INTBonus != 3 || f.MNDBonus != 3 || f.MPBonus != 30 {
		t.Errorf("SwampHerbalTea stats wrong: INT=%d MND=%d MP=%d", f.INTBonus, f.MNDBonus, f.MPBonus)
	}
}

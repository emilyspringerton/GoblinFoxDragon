package pet

import (
	"errors"
	"testing"
	"time"
)

// ── AllKinds / Stats ──────────────────────────────────────────────────────────

func TestAllKindsCount(t *testing.T) {
	if len(AllKinds) != 8 {
		t.Fatalf("expected 8 pet kinds, got %d", len(AllKinds))
	}
}

func TestAllKindsHaveStats(t *testing.T) {
	for _, k := range AllKinds {
		if _, ok := Stats[k]; !ok {
			t.Errorf("kind %q missing from Stats", k)
		}
	}
}

func TestStatsPositive(t *testing.T) {
	for k, s := range Stats {
		if s.BaseHP <= 0 || s.HPPerLevel <= 0 || s.Damage <= 0 || s.SwingDelay <= 0 {
			t.Errorf("kind %q has non-positive stat", k)
		}
	}
}

// ── JugPet ────────────────────────────────────────────────────────────────────

func TestJugPetBasic(t *testing.T) {
	sl := NewSlot()
	p, err := sl.JugPet(KindWolf, "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Kind != KindWolf {
		t.Errorf("expected wolf got %v", p.Kind)
	}
	if p.HP <= 0 {
		t.Errorf("expected HP>0 got %d", p.HP)
	}
	if p.HP != p.MaxHP {
		t.Errorf("fresh pet HP should equal MaxHP")
	}
}

func TestJugPetOwnerSlot(t *testing.T) {
	sl := NewSlot()
	p, _ := sl.JugPet(KindBear, "alice")
	if p.OwnerSlot != "alice" {
		t.Errorf("expected owner alice got %q", p.OwnerSlot)
	}
}

func TestJugPetUnknownKind(t *testing.T) {
	sl := NewSlot()
	_, err := sl.JugPet("dragon", "p1")
	if !errors.Is(err, ErrUnknownKind) {
		t.Errorf("expected ErrUnknownKind got %v", err)
	}
}

func TestJugPetAlreadyActive(t *testing.T) {
	sl := NewSlot()
	sl.JugPet(KindWolf, "p1")
	_, err := sl.JugPet(KindBird, "p1")
	if !errors.Is(err, ErrPetAlive) {
		t.Errorf("expected ErrPetAlive got %v", err)
	}
}

func TestJugPetCooldown(t *testing.T) {
	sl := NewSlot()
	sl.JugPet(KindWolf, "p1")
	sl.TakeDamage(9999) // kill pet
	_, err := sl.JugPet(KindBird, "p1")
	if !errors.Is(err, ErrCooldown) {
		t.Errorf("expected ErrCooldown immediately after death, got %v", err)
	}
}

// ── Tame ─────────────────────────────────────────────────────────────────────

func TestTameHighHP(t *testing.T) {
	sl := NewSlot()
	_, err := sl.Tame("wolf", 0.80, 20, "p1")
	if !errors.Is(err, ErrTameHP) {
		t.Errorf("expected ErrTameHP got %v", err)
	}
}

func TestTameUnknownKind(t *testing.T) {
	sl := NewSlot()
	_, err := sl.Tame("phoenix", 0.30, 20, "p1")
	if !errors.Is(err, ErrUnknownKind) {
		t.Errorf("expected ErrUnknownKind got %v", err)
	}
}

func TestTameSuccessHighLevel(t *testing.T) {
	// At level 99 chance is capped at 95%; over many rolls we should succeed most of the time.
	sl := NewSlot()
	successes := 0
	for i := 0; i < 20; i++ {
		sl.Pet = nil
		sl.diedAt = time.Time{}
		if p, err := sl.Tame("wolf", 0.30, 99, "p1"); err == nil && p != nil {
			successes++
		}
	}
	if successes < 10 {
		t.Errorf("expected at least 10/20 tame successes at level 99, got %d", successes)
	}
}

func TestTameAlreadyActive(t *testing.T) {
	sl := NewSlot()
	sl.JugPet(KindWolf, "p1")
	_, err := sl.Tame("bird", 0.50, 20, "p1")
	if !errors.Is(err, ErrPetAlive) {
		t.Errorf("expected ErrPetAlive got %v", err)
	}
}

// ── Release ───────────────────────────────────────────────────────────────────

func TestRelease(t *testing.T) {
	sl := NewSlot()
	sl.JugPet(KindBird, "p1")
	if err := sl.Release(); err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}
	if sl.IsAlive() {
		t.Error("pet should not be alive after release")
	}
}

func TestReleaseNoPet(t *testing.T) {
	sl := NewSlot()
	if err := sl.Release(); !errors.Is(err, ErrNoPet) {
		t.Errorf("expected ErrNoPet got %v", err)
	}
}

// ── TakeDamage / death ────────────────────────────────────────────────────────

func TestTakeDamagePartial(t *testing.T) {
	sl := NewSlot()
	sl.JugPet(KindBear, "p1")
	died := sl.TakeDamage(10)
	if died {
		t.Error("pet should not die from 10 damage")
	}
	if !sl.IsAlive() {
		t.Error("pet should still be alive")
	}
	if sl.Pet.HP >= sl.Pet.MaxHP {
		t.Error("HP should have decreased")
	}
}

func TestTakeDamageKillsPet(t *testing.T) {
	sl := NewSlot()
	sl.JugPet(KindBird, "p1")
	died := sl.TakeDamage(9999)
	if !died {
		t.Error("expected pet to die")
	}
	if sl.IsAlive() {
		t.Error("pet should not be alive after death")
	}
	if sl.diedAt.IsZero() {
		t.Error("diedAt should be set after pet dies")
	}
}

func TestTakeDamageNoPet(t *testing.T) {
	sl := NewSlot()
	died := sl.TakeDamage(100)
	if died {
		t.Error("TakeDamage with no pet should return false")
	}
}

// ── Heal ──────────────────────────────────────────────────────────────────────

func TestHeal(t *testing.T) {
	sl := NewSlot()
	sl.JugPet(KindCrab, "p1")
	sl.TakeDamage(30)
	healed, err := sl.Heal(10)
	if err != nil {
		t.Fatalf("unexpected heal error: %v", err)
	}
	if healed != 10 {
		t.Errorf("expected 10 healed, got %d", healed)
	}
}

func TestHealCapsAtMaxHP(t *testing.T) {
	sl := NewSlot()
	sl.JugPet(KindCrab, "p1")
	_, err := sl.Heal(9999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sl.Pet.HP != sl.Pet.MaxHP {
		t.Errorf("HP should be capped at MaxHP, got %d/%d", sl.Pet.HP, sl.Pet.MaxHP)
	}
}

func TestHealNoPet(t *testing.T) {
	sl := NewSlot()
	_, err := sl.Heal(100)
	if !errors.Is(err, ErrNoPet) {
		t.Errorf("expected ErrNoPet got %v", err)
	}
}

// ── Tick (combat) ─────────────────────────────────────────────────────────────

func TestTickNoSwingYet(t *testing.T) {
	sl := NewSlot()
	sl.JugPet(KindWolf, "p1")
	// First tick at t=0 should swing (lastSwing is zero).
	now := time.Now()
	evt := sl.Tick("mob-1", now)
	if evt == nil {
		t.Fatal("expected first tick to produce a combat event")
	}
	// Second immediate tick should NOT swing.
	evt2 := sl.Tick("mob-1", now)
	if evt2 != nil {
		t.Error("second immediate tick should not produce an event")
	}
}

func TestTickAfterDelay(t *testing.T) {
	sl := NewSlot()
	sl.JugPet(KindWolf, "p1")
	now := time.Now()
	sl.Tick("mob-1", now)
	// Advance past swing delay.
	future := now.Add(sl.Pet.SwingDelay + time.Millisecond)
	evt := sl.Tick("mob-1", future)
	if evt == nil {
		t.Fatal("expected swing after delay elapsed")
	}
	if evt.Damage <= 0 {
		t.Errorf("damage should be positive, got %d", evt.Damage)
	}
	if evt.TargetSlot != "mob-1" {
		t.Errorf("expected TargetSlot mob-1 got %q", evt.TargetSlot)
	}
}

func TestTickNoPet(t *testing.T) {
	sl := NewSlot()
	evt := sl.Tick("mob-1", time.Now())
	if evt != nil {
		t.Error("Tick with no pet should return nil")
	}
}

// ── Status ────────────────────────────────────────────────────────────────────

func TestStatusNoPet(t *testing.T) {
	sl := NewSlot()
	s := sl.Status()
	if s != "(no pet)" {
		t.Errorf("unexpected status: %q", s)
	}
}

func TestStatusWithPet(t *testing.T) {
	sl := NewSlot()
	sl.JugPet(KindWolf, "p1")
	s := sl.Status()
	if len(s) == 0 || s == "(no pet)" {
		t.Errorf("unexpected status with pet: %q", s)
	}
}

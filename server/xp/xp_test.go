package xp

import "testing"

// ── XPToLevel ─────────────────────────────────────────────────────────────────

func TestXPToLevel_L1IsZero(t *testing.T) {
	cost, err := XPToLevel(1)
	if err != nil || cost != 0 {
		t.Errorf("L1 cost: got (%d, %v), want (0, nil)", cost, err)
	}
}

func TestXPToLevel_FloorsAt75(t *testing.T) {
	// L2: 100 * 1^1.8 = 100 → ≥75 ✓
	// L1 = 0, L2 should be ≥75
	cost, _ := XPToLevel(2)
	if cost < xpFloor {
		t.Errorf("L2 cost below floor: got %d, want ≥%d", cost, xpFloor)
	}
}

func TestXPToLevel_Increases(t *testing.T) {
	prev, _ := XPToLevel(1)
	for lvl := 2; lvl <= MaxLevel; lvl++ {
		cur, err := XPToLevel(lvl)
		if err != nil {
			t.Fatalf("XPToLevel(%d): %v", lvl, err)
		}
		if cur < prev {
			t.Errorf("XP not increasing at L%d: cost=%d, prev=%d", lvl, cur, prev)
		}
		prev = cur
	}
}

func TestXPToLevel_InvalidLevel(t *testing.T) {
	if _, err := XPToLevel(0); err != ErrInvalidLevel {
		t.Errorf("L0: got %v, want ErrInvalidLevel", err)
	}
	if _, err := XPToLevel(100); err != ErrInvalidLevel {
		t.Errorf("L100: got %v, want ErrInvalidLevel", err)
	}
}

func TestXPToLevel_L99Reasonable(t *testing.T) {
	cost, _ := XPToLevel(99)
	// 100 * 98^1.8 ≈ 448,000 — should be in a reasonable range
	if cost < 100_000 || cost > 10_000_000 {
		t.Errorf("L99 cost out of expected range: %d", cost)
	}
}

// ── CharXP ────────────────────────────────────────────────────────────────────

func TestNewCharXP_Level1(t *testing.T) {
	c := NewCharXP()
	if c.Level != MinLevel || c.CurrentXP != 0 {
		t.Errorf("NewCharXP: Level=%d XP=%d", c.Level, c.CurrentXP)
	}
}

func TestAddXP_AccumulatesXP(t *testing.T) {
	c := NewCharXP()
	leveled, err := c.AddXP(50)
	if err != nil {
		t.Fatalf("AddXP: %v", err)
	}
	if leveled || c.CurrentXP != 50 {
		t.Errorf("no level-up yet: leveled=%v XP=%d", leveled, c.CurrentXP)
	}
}

func TestAddXP_LevelUp(t *testing.T) {
	c := NewCharXP()
	cost, _ := XPToLevel(2)
	leveled, _ := c.AddXP(cost)
	if !leveled {
		t.Error("should have leveled up to 2")
	}
	if c.Level != 2 {
		t.Errorf("Level: got %d, want 2", c.Level)
	}
	if c.CurrentXP != 0 {
		t.Errorf("CurrentXP after clean level: got %d, want 0", c.CurrentXP)
	}
}

func TestAddXP_OverflowToNextLevel(t *testing.T) {
	c := NewCharXP()
	cost2, _ := XPToLevel(2)
	cost3, _ := XPToLevel(3)
	// Give enough for L2 and half of L3
	overflow := cost2 + cost3/2
	leveled, _ := c.AddXP(overflow)
	if !leveled {
		t.Error("should have leveled up")
	}
	if c.Level != 2 {
		t.Errorf("Level: got %d, want 2", c.Level)
	}
	if c.CurrentXP != cost3/2 {
		t.Errorf("CurrentXP overflow: got %d, want %d", c.CurrentXP, cost3/2)
	}
}

func TestAddXP_NonTaggerGetsZeroXP(t *testing.T) {
	// Non-tagger simply does not call AddXP; test 0 gain is a no-op
	c := NewCharXP()
	leveled, err := c.AddXP(0)
	if err != nil || leveled {
		t.Errorf("0 XP: leveled=%v err=%v", leveled, err)
	}
	if c.Level != 1 || c.CurrentXP != 0 {
		t.Errorf("0 XP should not change state: Level=%d XP=%d", c.Level, c.CurrentXP)
	}
}

func TestAddXP_AtCap(t *testing.T) {
	c := &CharXP{Level: MaxLevel}
	_, err := c.AddXP(100)
	if err != ErrAtCap {
		t.Errorf("at cap: got %v, want ErrAtCap", err)
	}
}

func TestXPToNextLevel_AtCap(t *testing.T) {
	c := &CharXP{Level: MaxLevel}
	if n := c.XPToNextLevel(); n != 0 {
		t.Errorf("cap XPToNextLevel: got %d, want 0", n)
	}
}

func TestProgress_ZeroXP(t *testing.T) {
	c := NewCharXP()
	if c.Progress() != 0.0 {
		t.Errorf("0xp progress: got %f, want 0.0", c.Progress())
	}
}

func TestProgress_AtCap(t *testing.T) {
	c := &CharXP{Level: MaxLevel}
	if c.Progress() != 1.0 {
		t.Errorf("cap progress: got %f, want 1.0", c.Progress())
	}
}

func TestTotalXP_Tracks(t *testing.T) {
	c := NewCharXP()
	c.AddXP(100)
	c.AddXP(200)
	if c.TotalXP != 300 {
		t.Errorf("TotalXP: got %d, want 300", c.TotalXP)
	}
}

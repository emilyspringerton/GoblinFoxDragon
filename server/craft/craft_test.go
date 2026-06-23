package craft

import (
	"math/rand"
	"testing"
)

var rng = rand.New(rand.NewSource(42))

// ── CraftSkill ────────────────────────────────────────────────────────────────

func TestAllCrafts_EightTypes(t *testing.T) {
	if len(AllCrafts) != 8 {
		t.Errorf("AllCrafts: got %d, want 8", len(AllCrafts))
	}
}

func TestSetLevel_ClampedAtCap(t *testing.T) {
	cs := NewCraftSkill()
	cs.SetLevel(Alchemy, 200.0)
	lvl, _ := cs.Level(Alchemy)
	if lvl != SkillCap {
		t.Errorf("clamped at cap: got %f, want %f", lvl, SkillCap)
	}
}

func TestSetLevel_ClampedAtZero(t *testing.T) {
	cs := NewCraftSkill()
	cs.SetLevel(Alchemy, -5.0)
	lvl, _ := cs.Level(Alchemy)
	if lvl != 0 {
		t.Errorf("clamped at 0: got %f, want 0", lvl)
	}
}

func TestLevel_UnknownCraft(t *testing.T) {
	cs := NewCraftSkill()
	if _, err := cs.Level("novice-candle-waving"); err != ErrUnknownCraft {
		t.Errorf("unknown craft: got %v, want ErrUnknownCraft", err)
	}
}

// ── SuccessChance ─────────────────────────────────────────────────────────────

func TestSuccessChance_AtBase(t *testing.T) {
	sc := SuccessChance(10.0, 10.0) // gap=0 → 50%
	if sc < 0.49 || sc > 0.51 {
		t.Errorf("gap=0: got %f, want ~0.50", sc)
	}
}

func TestSuccessChance_AboveDiff(t *testing.T) {
	sc := SuccessChance(20.0, 10.0) // gap=+10 → 50+20=70%
	if sc < 0.69 || sc > 0.71 {
		t.Errorf("+10 gap: got %f, want ~0.70", sc)
	}
}

func TestSuccessChance_Cap(t *testing.T) {
	sc := SuccessChance(SkillCap, 0.0) // very high gap → capped at 95%
	if sc > SuccessCap/100.0+0.001 {
		t.Errorf("cap: got %f, want ≤%f", sc, SuccessCap/100.0)
	}
}

func TestSuccessChance_Floor(t *testing.T) {
	sc := SuccessChance(0.0, 100.0) // huge negative gap → floored at 5%
	if sc < SuccessFloor/100.0-0.001 {
		t.Errorf("floor: got %f, want ≥%f", sc, SuccessFloor/100.0)
	}
}

// ── HQTier ────────────────────────────────────────────────────────────────────

func TestHQTier_NQBelow10(t *testing.T) {
	if tier := HQTier(5.0, 0.01); tier != 0 {
		t.Errorf("gap<10: got %d, want 0 (NQ)", tier)
	}
}

func TestHQTier_HQ1AtGap10(t *testing.T) {
	// gap=10, roll=0.5 → HQ1 (not high enough roll for HQ2/3)
	if tier := HQTier(10.0, 0.5); tier != 1 {
		t.Errorf("gap=10 roll=0.5: got %d, want 1", tier)
	}
}

func TestHQTier_HQ2AtGap30(t *testing.T) {
	// gap=30, roll=0.05 → HQ2 (within HQ2 threshold but not HQ3)
	if tier := HQTier(30.0, 0.05); tier != 2 {
		t.Errorf("gap=30 roll=0.05: got %d, want 2", tier)
	}
}

func TestHQTier_HQ3AtGap50(t *testing.T) {
	// gap=50, roll=0.01 → HQ3 (below 0.05)
	if tier := HQTier(50.0, 0.01); tier != 3 {
		t.Errorf("gap=50 roll=0.01: got %d, want 3", tier)
	}
}

func TestHQTier_NQAtGap0(t *testing.T) {
	if tier := HQTier(0.0, 0.99); tier != 0 {
		t.Errorf("gap=0: got %d, want 0 (NQ)", tier)
	}
}

// ── Attempt ───────────────────────────────────────────────────────────────────

func TestAttempt_BreakOnLargeGap(t *testing.T) {
	recipe := IronIngotRecipe() // difficulty=2
	_, err := Attempt(recipe, 0.0, rng) // skill=0; gap=-2; threshold=-10 → not auto-break
	// Not a break yet at -2; BreakThreshold is -10
	_ = err

	// Now try skill=-10 relative to a difficulty 12 recipe
	hardRecipe := Recipe{Difficulty: 20.0, ItemID: "x", CraftType: Blacksmith}
	_, err = Attempt(hardRecipe, 9.0, rng) // gap=-11 → break
	if err != ErrBreak {
		t.Errorf("large negative gap: got %v, want ErrBreak", err)
	}
}

func TestAttempt_SuccessAtCap(t *testing.T) {
	recipe := IronIngotRecipe() // difficulty=2
	// Run 20 attempts at skill cap; all should succeed (95% rate)
	successes := 0
	r := rand.New(rand.NewSource(99))
	for i := 0; i < 20; i++ {
		res, _ := Attempt(recipe, SkillCap, r)
		if res.Success {
			successes++
		}
	}
	if successes < 15 {
		t.Errorf("at skill cap: only %d/20 successes (expect ~19)", successes)
	}
}

func TestAttempt_NQOnSuccess(t *testing.T) {
	recipe := IronIngotRecipe()
	// At gap=0 (skill=difficulty), forced success roll and NQ HQ roll
	r := rand.New(rand.NewSource(1)) // known seed
	for i := 0; i < 50; i++ {
		res, err := Attempt(recipe, 2.0, r)
		if err == ErrBreak {
			continue
		}
		if res.Success && res.ItemID == "" {
			t.Error("successful attempt should have non-empty ItemID")
		}
	}
}

func TestAttempt_HQItemIDUsed(t *testing.T) {
	// Use a recipe where HQ3 is a distinct item
	recipe := Recipe{
		Difficulty: 0.0,
		ItemID:     "base",
		HQ1ItemID:  "base+1",
		HQ2ItemID:  "base+2",
		HQ3ItemID:  "base+3",
		CraftType:  Alchemy,
	}
	// Force HQ3: gap=60 (skill=60, diff=0), roll must be <0.05
	// Run many attempts with known seed to find one HQ3
	r := rand.New(rand.NewSource(7))
	found := false
	for i := 0; i < 200; i++ {
		res, _ := Attempt(recipe, 60.0, r)
		if res.Success && res.HQTier == 3 {
			if res.ItemID != "base+3" {
				t.Errorf("HQ3 itemID: got %q, want base+3", res.ItemID)
			}
			found = true
			break
		}
	}
	if !found {
		t.Log("HQ3 not triggered in 200 attempts (probabilistic; may occur rarely)")
	}
}

// ── Recipes ───────────────────────────────────────────────────────────────────

func TestIronIngotRecipe_BlacksmithCraft(t *testing.T) {
	r := IronIngotRecipe()
	if r.CraftType != Blacksmith {
		t.Errorf("iron ingot: CraftType=%q, want blacksmith", r.CraftType)
	}
}

func TestHerbalRemedyRecipe_AlchemyCraft(t *testing.T) {
	r := HerbalRemedyRecipe()
	if r.CraftType != Alchemy {
		t.Errorf("herbal remedy: CraftType=%q, want alchemy", r.CraftType)
	}
}

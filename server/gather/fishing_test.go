package gather

import (
	"testing"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func fishPoint(diff float64, maxAttempts int) *FishingPoint {
	return NewFishingPoint("test-pond", "Test Pond", 0, diff, maxAttempts, []FishLootEntry{
		{ItemID: FishCarp, ItemName: "Carp", Weight: 100},
	}, 42)
}

func fishPointHQ(diff float64) *FishingPoint {
	return NewFishingPoint("test-hq", "HQ Pond", 0, diff, 20, []FishLootEntry{
		{ItemID: FishCarp, ItemName: "Carp", Weight: 100, HQItemID: FishTrophyCarp},
	}, 77)
}

// ── FishName ──────────────────────────────────────────────────────────────────

func TestFishName(t *testing.T) {
	if FishName(FishCarp) != "Carp" {
		t.Errorf("expected Carp, got %q", FishName(FishCarp))
	}
	if FishName("unknown-fish") != "unknown-fish" {
		t.Errorf("expected passthrough for unknown id")
	}
}

// ── Attempt success/failure ───────────────────────────────────────────────────

func TestFishAttemptHighSkillSucceeds(t *testing.T) {
	pt := fishPoint(0, 100)
	skill := FishSkill{Level: 110}
	successes := 0
	for i := 0; i < 50; i++ {
		y, err := pt.Attempt(skill)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if y.ItemID != "" {
			successes++
		}
	}
	if successes < 40 {
		t.Errorf("high skill should land most casts, got %d/50", successes)
	}
}

func TestFishAttemptLowSkillHighDiffHasSomeSuccess(t *testing.T) {
	pt := fishPoint(110, 100)
	skill := FishSkill{Level: 0}
	successes := 0
	for i := 0; i < 100; i++ {
		y, err := pt.Attempt(skill)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if y.ItemID != "" {
			successes++
		}
	}
	// Floor is 5%, so expect ~5 successes in 100 tries (±noise with seed)
	if successes == 0 {
		t.Error("even bad skill should occasionally succeed (floor 5%)")
	}
}

// ── Depletion ─────────────────────────────────────────────────────────────────

func TestFishAttemptDepleted(t *testing.T) {
	pt := fishPoint(0, 1)
	skill := FishSkill{Level: 110}
	// Drain the one remaining yield (may take a few misses due to rng — but skill 110 diff 0 is capped at 95%).
	for i := 0; i < 30; i++ {
		y, err := pt.Attempt(skill)
		if err == ErrFishingPointDepleted {
			return // pass
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if y.ItemID != "" && pt.Depleted() {
			break
		}
	}
	_, err := pt.Attempt(skill)
	if err != ErrFishingPointDepleted {
		t.Errorf("expected ErrFishingPointDepleted, got %v", err)
	}
}

func TestFishRemainingDecrementsOnSuccess(t *testing.T) {
	pt := fishPoint(0, 10)
	skill := FishSkill{Level: 110}
	initial := pt.Remaining()
	for i := 0; i < 20; i++ {
		y, _ := pt.Attempt(skill)
		if y.ItemID != "" {
			break
		}
	}
	if pt.Remaining() >= initial {
		t.Error("remaining should decrease after a successful catch")
	}
}

func TestFishRespawn(t *testing.T) {
	pt := fishPoint(0, 2)
	skill := FishSkill{Level: 110}
	// Exhaust the point.
	for i := 0; i < 50 && !pt.Depleted(); i++ {
		pt.Attempt(skill)
	}
	if !pt.Depleted() {
		t.Skip("could not exhaust point in 50 attempts")
	}
	pt.Respawn()
	if pt.Depleted() {
		t.Error("Respawn should restore the point")
	}
	if pt.Remaining() != 2 {
		t.Errorf("expected remaining=2 after Respawn, got %d", pt.Remaining())
	}
}

// ── Errors ────────────────────────────────────────────────────────────────────

func TestFishAttemptNoLoot(t *testing.T) {
	pt := NewFishingPoint("empty", "Empty", 0, 0, 10, nil, 1)
	_, err := pt.Attempt(FishSkill{Level: 50})
	if err != ErrNoFishLoot {
		t.Errorf("expected ErrNoFishLoot, got %v", err)
	}
}

// ── SuccessChance ─────────────────────────────────────────────────────────────

func TestFishSuccessChanceClamped(t *testing.T) {
	// High skill → cap.
	ptEasy := fishPoint(0, 100)
	if ptEasy.SuccessChance(FishSkill{Level: 200}) != SuccessCap {
		t.Error("success chance should be capped at SuccessCap")
	}
	// Skill=0 at max difficulty → floor.
	ptHard := fishPoint(110, 100)
	if ptHard.SuccessChance(FishSkill{Level: 0}) != SuccessFloor {
		t.Errorf("success chance should floor at SuccessFloor, got %v", ptHard.SuccessChance(FishSkill{Level: 0}))
	}
}

func TestFishSuccessChanceMidSkill(t *testing.T) {
	pt := fishPoint(20, 100)
	c := pt.SuccessChance(FishSkill{Level: 20})
	if c != SuccessBase {
		t.Errorf("equal skill/diff should give SuccessBase (%v), got %v", SuccessBase, c)
	}
}

// ── Trophy (HQ) ───────────────────────────────────────────────────────────────

func TestFishTrophyVariantOnHQ(t *testing.T) {
	pt := fishPointHQ(0) // diff=0, so skill 110 is +110 over threshold → guaranteed trophy
	skill := FishSkill{Level: 110}
	found := false
	for i := 0; i < 100; i++ {
		y, _ := pt.Attempt(skill)
		if y.Trophy && y.ItemID == FishTrophyCarp {
			found = true
			break
		}
	}
	if !found {
		t.Error("should get trophy carp (HQ item) with skill 110 at diff 0")
	}
}

// ── Presets ───────────────────────────────────────────────────────────────────

func TestMeadowFishingPointsNotEmpty(t *testing.T) {
	pts := MeadowFishingPoints()
	if len(pts) == 0 {
		t.Error("MeadowFishingPoints should return at least one point")
	}
	for _, pt := range pts {
		if len(pt.Loot) == 0 {
			t.Errorf("point %s has empty loot table", pt.ID)
		}
	}
}

func TestSwampFishingPointsHaveHigherDifficulty(t *testing.T) {
	meadow := MeadowFishingPoints()
	swamp := SwampFishingPoints()
	if len(swamp) == 0 {
		t.Fatal("SwampFishingPoints should not be empty")
	}
	maxMeadow := 0.0
	for _, pt := range meadow {
		if pt.Difficulty > maxMeadow {
			maxMeadow = pt.Difficulty
		}
	}
	minSwamp := swamp[0].Difficulty
	for _, pt := range swamp {
		if pt.Difficulty < minSwamp {
			minSwamp = pt.Difficulty
		}
	}
	if minSwamp <= maxMeadow {
		t.Errorf("swamp difficulty (%v) should exceed meadow max (%v)", minSwamp, maxMeadow)
	}
}

func TestFishingPointSceneID(t *testing.T) {
	pts := SwampFishingPoints()
	for _, pt := range pts {
		if pt.SceneID != 3 {
			t.Errorf("swamp fishing point %s should have SceneID=3, got %d", pt.ID, pt.SceneID)
		}
	}
}

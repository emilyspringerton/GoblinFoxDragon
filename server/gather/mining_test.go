package gather

import (
	"testing"
)

// helpers

func ironPoint(seed int64) *MiningPoint {
	return NewPoint("test-iron", "Iron Rock", 20, 5, []LootEntry{
		lootEntry(OreIron, 100),
	}, seed)
}

func multiLootPoint(seed int64) *MiningPoint {
	return NewPoint("test-multi", "Multi Rock", 10, 20, []LootEntry{
		lootEntry(OreCopper, 60),
		lootEntry(OreIron, 30),
		lootEntry(CrystalEarth, 10),
	}, seed)
}

// --- OreName ---

func TestOreName_KnownIDs(t *testing.T) {
	cases := map[string]string{
		OreIron:      "Iron Ore",
		OreCopper:    "Copper Ore",
		OreSwampite:  "Swampite",
		MatClay:      "Clay",
		MatPeat:      "Peat",
		CrystalEarth: "Earth Crystal",
		CrystalWater: "Water Crystal",
	}
	for id, want := range cases {
		if got := OreName(id); got != want {
			t.Errorf("OreName(%q) = %q, want %q", id, got, want)
		}
	}
}

func TestOreName_UnknownReturnsID(t *testing.T) {
	if OreName("unknown-ore-xyz") != "unknown-ore-xyz" {
		t.Error("unknown ore ID should be returned as-is")
	}
}

// --- NewPoint ---

func TestNewPoint_RemainingStartsAtMax(t *testing.T) {
	p := ironPoint(42)
	if p.Remaining() != 5 {
		t.Errorf("Remaining: got %d, want 5", p.Remaining())
	}
}

func TestNewPoint_NotDepleted(t *testing.T) {
	p := ironPoint(42)
	if p.Depleted() {
		t.Error("new point should not be depleted")
	}
}

func TestNewPoint_DifficultyClamped(t *testing.T) {
	p := NewPoint("x", "x", 200, 5, []LootEntry{lootEntry(OreIron, 1)}, 1)
	if p.Difficulty != SkillCap {
		t.Errorf("Difficulty clamped: got %f", p.Difficulty)
	}
	p2 := NewPoint("x", "x", -10, 5, []LootEntry{lootEntry(OreIron, 1)}, 1)
	if p2.Difficulty != SkillMin {
		t.Errorf("Difficulty clamped low: got %f", p2.Difficulty)
	}
}

// --- Mine: depletion ---

func TestMine_DepletsAfterMaxAttempts(t *testing.T) {
	p := ironPoint(1)
	successes := 0
	for i := 0; i < 100 && !p.Depleted(); i++ {
		y, err := p.Mine(SkillCap) // max skill → near 95% success
		if err != nil {
			t.Fatalf("Mine: %v", err)
		}
		if y.ItemID != "" {
			successes++
		}
	}
	if !p.Depleted() {
		t.Error("point should be depleted after enough successful mines")
	}
	if successes != 5 {
		t.Errorf("expected exactly 5 successful yields before depletion, got %d", successes)
	}
}

func TestMine_DepletedReturnsError(t *testing.T) {
	p := ironPoint(2)
	p.remaining = 0
	_, err := p.Mine(50)
	if err != ErrPointDepleted {
		t.Errorf("depleted point: got %v, want ErrPointDepleted", err)
	}
}

func TestMine_EmptyLootReturnsError(t *testing.T) {
	p := NewPoint("x", "x", 0, 5, nil, 1)
	_, err := p.Mine(50)
	if err != ErrNoLoot {
		t.Errorf("no loot: got %v, want ErrNoLoot", err)
	}
}

// --- Respawn ---

func TestRespawn_ResetsRemaining(t *testing.T) {
	p := ironPoint(3)
	p.remaining = 0
	p.Respawn()
	if p.Remaining() != 5 {
		t.Errorf("Remaining after respawn: got %d, want 5", p.Remaining())
	}
	if p.Depleted() {
		t.Error("should not be depleted after respawn")
	}
}

// --- Success rate ---

func TestSuccessChance_AtDifficulty(t *testing.T) {
	p := ironPoint(1)
	got := p.SuccessChance(20) // skill == difficulty
	if got != SuccessBase {
		t.Errorf("skill==difficulty: got %.1f%%, want %.1f%%", got, SuccessBase)
	}
}

func TestSuccessChance_AboveDifficulty(t *testing.T) {
	p := ironPoint(1)
	got := p.SuccessChance(30) // 10 levels above → +20%
	want := SuccessBase + 10*SuccessPerLevel
	if got != want {
		t.Errorf("10 above difficulty: got %.1f%%, want %.1f%%", got, want)
	}
}

func TestSuccessChance_FloorsAtMin(t *testing.T) {
	// diff=35, skill=0 → 60 + (0-35)*2 = -10 → clamped to SuccessFloor
	p := NewPoint("x", "x", 35, 5, []LootEntry{lootEntry(OreIron, 1)}, 1)
	got := p.SuccessChance(0)
	if got != SuccessFloor {
		t.Errorf("well below difficulty: got %.1f%%, want %.1f%%", got, SuccessFloor)
	}
}

func TestSuccessChance_CapsAtMax(t *testing.T) {
	p := ironPoint(1)
	got := p.SuccessChance(SkillCap)
	if got != SuccessCap {
		t.Errorf("max skill: got %.1f%%, want %.1f%%", got, SuccessCap)
	}
}

// --- HQ rate ---

func TestHQChance_ZeroBelowThreshold(t *testing.T) {
	p := ironPoint(1)
	// skill=20, diff=20, threshold=10 → skill must be diff+threshold=30 for any HQ
	got := p.HQChance(29)
	if got != 0 {
		t.Errorf("below HQ threshold: got %.1f%%", got)
	}
}

func TestHQChance_PositiveAboveThreshold(t *testing.T) {
	p := ironPoint(1) // diff=20, threshold=10 → HQ starts at skill=30
	got := p.HQChance(35) // 5 levels above threshold → 5%
	want := 5 * HQPerLevel
	if got != want {
		t.Errorf("HQChance: got %.1f%%, want %.1f%%", got, want)
	}
}

func TestHQChance_CapsAtMax(t *testing.T) {
	p := ironPoint(1)
	got := p.HQChance(SkillCap)
	if got != HQCap {
		t.Errorf("max HQ: got %.1f%%, want %.1f%%", got, HQCap)
	}
}

// --- Mine: loot distribution ---

func TestMine_YieldsCorrectItem(t *testing.T) {
	p := ironPoint(99)
	// Force at max skill for reliable success.
	for i := 0; i < 10; i++ {
		y, err := p.Mine(SkillCap)
		if err == ErrPointDepleted {
			break
		}
		if err != nil {
			t.Fatalf("Mine error: %v", err)
		}
		if y.ItemID == "" {
			continue // failed roll
		}
		if y.ItemID != OreIron && !(y.HQ) {
			// non-HQ yield must be iron
			t.Errorf("expected OreIron, got %q", y.ItemID)
		}
	}
}

func TestMine_MultiLootAllItemsSeen(t *testing.T) {
	p := multiLootPoint(7777)
	// Run many attempts to check all loot entries can appear.
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		p.Respawn()
		y, _ := p.Mine(SkillCap)
		if y.ItemID != "" {
			seen[y.ItemID] = true
		}
	}
	for _, want := range []string{OreCopper, OreIron, CrystalEarth} {
		if !seen[want] {
			t.Errorf("loot item %q never seen over many runs", want)
		}
	}
}

func TestMine_HighSkillProducesHQ(t *testing.T) {
	// diff=0, HQThreshold=10 → HQ starts at skill=10; at skill=110 HQ=25%
	p := NewPoint("hq-test", "HQ Rock", 0, 1000, []LootEntry{
		lootEntry(OreCopper, 100),
	}, 55555)
	hqCount := 0
	total := 0
	for i := 0; i < 500; i++ {
		if p.Depleted() {
			p.Respawn()
		}
		y, _ := p.Mine(SkillCap)
		if y.ItemID != "" {
			total++
			if y.HQ {
				hqCount++
			}
		}
	}
	if total == 0 {
		t.Fatal("no yields in 500 attempts at max skill")
	}
	hqRate := float64(hqCount) / float64(total) * 100
	if hqRate < 5 {
		t.Errorf("HQ rate at max skill too low: %.1f%%", hqRate)
	}
}

func TestMine_HQWithAlternateItem(t *testing.T) {
	// Point with HQ alternate item specified.
	p := NewPoint("hq-alt", "Alt Rock", 0, 1000, []LootEntry{
		lootEntryHQ(OreCopper, OreIron, 100),
	}, 11111)
	foundHQAlt := false
	for i := 0; i < 500; i++ {
		if p.Depleted() {
			p.Respawn()
		}
		y, _ := p.Mine(SkillCap)
		if y.HQ && y.ItemID == OreIron {
			foundHQAlt = true
			break
		}
	}
	if !foundHQAlt {
		t.Error("HQ alternate item (iron) never returned from copper-HQ-iron point in 500 attempts")
	}
}

func TestMine_HQWithNoAlternateDoubleQty(t *testing.T) {
	p := NewPoint("hq-dbl", "Double Rock", 0, 1000, []LootEntry{
		lootEntry(OreCopper, 100),
	}, 22222)
	foundDouble := false
	for i := 0; i < 500; i++ {
		if p.Depleted() {
			p.Respawn()
		}
		y, _ := p.Mine(SkillCap)
		if y.HQ && y.Qty == 2 {
			foundDouble = true
			break
		}
	}
	if !foundDouble {
		t.Error("HQ double-qty yield never seen in 500 attempts at max skill")
	}
}

// --- Preset points ---

func TestMeadowMiningPoints_Count(t *testing.T) {
	pts := MeadowMiningPoints()
	if len(pts) != 3 {
		t.Errorf("MeadowMiningPoints: want 3, got %d", len(pts))
	}
}

func TestMeadowMiningPoints_LowDifficulty(t *testing.T) {
	for _, p := range MeadowMiningPoints() {
		if p.Difficulty > 15 {
			t.Errorf("meadow point %q difficulty %v should be ≤15 (starter zone)", p.Name, p.Difficulty)
		}
	}
}

func TestSwampMiningPoints_Count(t *testing.T) {
	pts := SwampMiningPoints()
	if len(pts) != 3 {
		t.Errorf("SwampMiningPoints: want 3, got %d", len(pts))
	}
}

func TestSwampMiningPoints_ContainSwampite(t *testing.T) {
	found := false
	for _, p := range SwampMiningPoints() {
		for _, l := range p.Loot {
			if l.ItemID == OreSwampite {
				found = true
			}
		}
	}
	if !found {
		t.Error("swamp mining points should contain swampite ore")
	}
}

func TestSwampMiningPoints_HigherDifficultyThanMeadow(t *testing.T) {
	meadowMax := 0.0
	for _, p := range MeadowMiningPoints() {
		if p.Difficulty > meadowMax {
			meadowMax = p.Difficulty
		}
	}
	swampMin := SkillCap
	for _, p := range SwampMiningPoints() {
		if p.Difficulty < swampMin {
			swampMin = p.Difficulty
		}
	}
	if swampMin <= meadowMax {
		t.Errorf("swamp points should be harder than meadow: swamp min=%.0f meadow max=%.0f", swampMin, meadowMax)
	}
}

func TestMine_LowSkillRarelySucceedsHardPoint(t *testing.T) {
	p := SwampMiningPoints()[2] // Deep Mire Node, diff=40
	// Skill 0 → 60 + (0-40)*2 = -20 → floors at SuccessFloor (5%)
	chance := p.SuccessChance(0)
	if chance != SuccessFloor {
		t.Errorf("skill 0 on hard point: chance=%.1f%%, want %.1f%%", chance, SuccessFloor)
	}
}

// Package craft implements the DragonsNShit synthesis / crafting guild system.
//
// # Crafting model (FFXI-parity)
//
// Eight craft types map to eight guilds. Each character has a skill level (0–110)
// per craft. Synthesis attempts produce NQ (normal quality) on success, or break
// (lose ingredients) if the skill gap is too wide.
//
// # Success model
//   Success% = Base + (playerSkill - recipeDifficulty) * SuccessPerPoint
//   Clamped to [SuccessFloor, SuccessCap]
//   Break occurs below BreakThreshold skill gap.
//
// # HQ (S84-02) — 3 tiers based on skill overshoot:
//   HQ1: overshoot ≥  10  (small surplus)
//   HQ2: overshoot ≥  30
//   HQ3: overshoot ≥  50  (large surplus)
//   Within each tier, a secondary roll determines which HQ tier actually produces.
package craft

import (
	"errors"
	"math/rand"
)

// CraftType identifies a crafting guild / skill.
type CraftType = string

const (
	Alchemy      CraftType = "alchemy"
	Blacksmith   CraftType = "blacksmith"
	Bonecraft    CraftType = "bonecraft"
	Clothcraft   CraftType = "clothcraft"
	Cooking      CraftType = "cooking"
	Goldsmithing CraftType = "goldsmithing"
	Leathercraft CraftType = "leathercraft"
	Woodworking  CraftType = "woodworking"
)

// AllCrafts lists all 8 crafting guild types.
var AllCrafts = []CraftType{Alchemy, Blacksmith, Bonecraft, Clothcraft, Cooking, Goldsmithing, Leathercraft, Woodworking}

const (
	SkillMin       = 0.0
	SkillCap       = 110.0
	SuccessBase    = 50.0  // base success % at 0 skill gap
	SuccessPerPt   = 2.0   // +2% per point of skill over difficulty
	SuccessFloor   = 5.0   // minimum success % regardless of gap
	SuccessCap     = 95.0  // maximum success %
	BreakThreshold = -10.0 // skill gap below which synth breaks 100%

	// HQ thresholds (skill overshoot above recipe difficulty)
	HQ1Threshold = 10.0
	HQ2Threshold = 30.0
	HQ3Threshold = 50.0
)

var (
	ErrUnknownCraft = errors.New("unknown craft type")
	ErrBreak        = errors.New("synthesis broke — ingredients lost")
	ErrNoLoot       = errors.New("synthesis produced no items (should not happen)")
)

// Result is the outcome of a synthesis attempt.
type Result struct {
	Success bool
	HQTier  int    // 0 = NQ, 1/2/3 = HQ tier
	ItemID  string // produced item (same as recipe.ItemID unless HQ override)
	Break   bool   // ingredients destroyed on failure
}

// Recipe defines a synthesis recipe.
type Recipe struct {
	ID          string
	CraftType   CraftType
	Difficulty  float64  // reference skill level for this recipe
	ItemID      string   // NQ result
	HQ1ItemID   string   // HQ1 result (may equal ItemID)
	HQ2ItemID   string   // HQ2 result
	HQ3ItemID   string   // HQ3 result
}

// CraftSkill holds a character's skill level per craft.
type CraftSkill struct {
	levels map[CraftType]float64
}

// NewCraftSkill creates a zero-skill character.
func NewCraftSkill() *CraftSkill {
	m := make(map[CraftType]float64, len(AllCrafts))
	for _, c := range AllCrafts {
		m[c] = 0
	}
	return &CraftSkill{levels: m}
}

// Level returns skill level for the given craft (0.0–110.0).
func (cs *CraftSkill) Level(craft CraftType) (float64, error) {
	if l, ok := cs.levels[craft]; ok {
		return l, nil
	}
	return 0, ErrUnknownCraft
}

// SetLevel sets skill level, clamped to [SkillMin, SkillCap].
func (cs *CraftSkill) SetLevel(craft CraftType, lvl float64) error {
	if _, ok := cs.levels[craft]; !ok {
		return ErrUnknownCraft
	}
	if lvl < SkillMin {
		lvl = SkillMin
	}
	if lvl > SkillCap {
		lvl = SkillCap
	}
	cs.levels[craft] = lvl
	return nil
}

// ── Synthesis ─────────────────────────────────────────────────────────────────

// SuccessChance returns the synthesis success probability (0.0–1.0).
func SuccessChance(playerSkill, recipeDifficulty float64) float64 {
	gap := playerSkill - recipeDifficulty
	pct := SuccessBase + gap*SuccessPerPt
	if pct < SuccessFloor {
		pct = SuccessFloor
	}
	if pct > SuccessCap {
		pct = SuccessCap
	}
	return pct / 100.0
}

// HQTier returns the HQ tier based on skill overshoot (0=NQ, 1/2/3).
// Uses a secondary roll to determine which HQ tier triggers.
func HQTier(skillGap float64, roll float64) int {
	if skillGap < HQ1Threshold {
		return 0
	}
	// HQ3 at gap ≥50 with any roll <0.05 (5% of HQ1+ attempts → HQ3)
	if skillGap >= HQ3Threshold && roll < 0.05 {
		return 3
	}
	if skillGap >= HQ2Threshold && roll < 0.15 {
		return 2
	}
	return 1
}

// Attempt performs a synthesis attempt.
// playerSkill is the character's skill in the recipe's CraftType.
// rng is used for the success and HQ rolls.
func Attempt(recipe Recipe, playerSkill float64, rng *rand.Rand) (Result, error) {
	gap := playerSkill - recipe.Difficulty

	// Automatic break if skill far below difficulty
	if gap <= BreakThreshold {
		return Result{Break: true}, ErrBreak
	}

	successProb := SuccessChance(playerSkill, recipe.Difficulty)
	if rng.Float64() >= successProb {
		// Failure — ingredients not broken (gap is OK, just failed RNG)
		return Result{Success: false}, nil
	}

	// Success — determine HQ tier
	hqRoll := rng.Float64()
	tier := HQTier(gap, hqRoll)

	itemID := recipe.ItemID
	switch tier {
	case 1:
		if recipe.HQ1ItemID != "" {
			itemID = recipe.HQ1ItemID
		}
	case 2:
		if recipe.HQ2ItemID != "" {
			itemID = recipe.HQ2ItemID
		}
	case 3:
		if recipe.HQ3ItemID != "" {
			itemID = recipe.HQ3ItemID
		}
	}

	return Result{Success: true, HQTier: tier, ItemID: itemID}, nil
}

// ── Sample recipes ────────────────────────────────────────────────────────────

// IronIngotRecipe is a sample Blacksmith recipe.
func IronIngotRecipe() Recipe {
	return Recipe{
		ID:         "iron-ingot",
		CraftType:  Blacksmith,
		Difficulty: 2.0,
		ItemID:     "iron-ingot",
		HQ1ItemID:  "iron-ingot+1",
		HQ2ItemID:  "iron-ingot+2",
		HQ3ItemID:  "iron-ingot+3",
	}
}

// HerbalRemedy is a sample Alchemy recipe.
func HerbalRemedyRecipe() Recipe {
	return Recipe{
		ID:         "herbal-remedy",
		CraftType:  Alchemy,
		Difficulty: 15.0,
		ItemID:     "herbal-remedy",
		HQ1ItemID:  "herbal-remedy+1",
		HQ2ItemID:  "herbal-remedy+2",
		HQ3ItemID:  "herbal-remedy+3",
	}
}

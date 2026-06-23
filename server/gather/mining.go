// Package gather implements server-authoritative resource gathering for DragonsNShit.
//
// # Mining model (FFXI-parity)
//
// Players use a pickaxe at a MiningPoint. Each attempt:
//   - Rolls success against (skill - point.Difficulty): 60% base + 2% per level over difficulty.
//   - On success: pulls a Yield from the point's loot table (weighted random).
//   - HQ yield: additional roll when skill significantly exceeds difficulty.
//   - Points have finite attempts (MaxAttempts); ErrPointDepleted when exhausted.
//
// Skill range: 0–110 (matching FFXI crafting/gathering skill cap).
// Difficulty range: 0–110; attempting a point 10+ levels above skill is still possible
// but success drops toward the floor (MinSuccessChance).
//
// # Loot table
//
// Each MiningPoint carries a []LootEntry. On a successful attempt the server rolls
// against cumulative Weight values to pick the specific ore returned.
// HQ flag is set when the HQ roll also passes; the caller maps HQ→alternate item
// (e.g., "Iron Ore" HQ → "Dark Steel Ore") if the game design calls for it.
package gather

import (
	"errors"
	"math/rand"
)

// Skill constants (match FFXI gathering skill scale).
const (
	SkillMin float64 = 0
	SkillCap float64 = 110

	// SuccessBase is the success rate when skill equals point difficulty (%).
	SuccessBase float64 = 60
	// SuccessPerLevel is the % gain per skill level over difficulty.
	SuccessPerLevel float64 = 2
	// SuccessFloor is the minimum success rate regardless of skill deficit.
	SuccessFloor float64 = 5
	// SuccessCap is the maximum success rate.
	SuccessCap float64 = 95

	// HQThreshold: skill must exceed difficulty by at least this much for HQ rolls.
	HQThreshold float64 = 10
	// HQPerLevel is the % HQ chance per skill level above HQThreshold.
	HQPerLevel float64 = 1
	// HQCap is the maximum HQ rate.
	HQCap float64 = 25
)

// Ore IDs — canonical material identifiers returned by mining points.
const (
	OreIron     = "ore-iron"
	OreCopper   = "ore-copper"
	OreTin      = "ore-tin"
	OreSilver   = "ore-silver"
	OreGold     = "ore-gold"
	OreSwampite = "ore-swampite"  // Swampville-specific dark ore
	MatClay     = "mat-clay"
	MatPeat     = "mat-peat"      // Swampville biomass fuel material
	CrystalEarth = "crystal-earth"
	CrystalWater = "crystal-water"
)

var oreNames = map[string]string{
	OreIron:      "Iron Ore",
	OreCopper:    "Copper Ore",
	OreTin:       "Tin Ore",
	OreSilver:    "Silver Ore",
	OreGold:      "Gold Ore",
	OreSwampite:  "Swampite",
	MatClay:      "Clay",
	MatPeat:      "Peat",
	CrystalEarth: "Earth Crystal",
	CrystalWater: "Water Crystal",
}

// OreName returns the display name for an ore ID.
func OreName(id string) string {
	if n, ok := oreNames[id]; ok {
		return n
	}
	return id
}

// LootEntry is one item in a MiningPoint's loot table.
type LootEntry struct {
	ItemID   string
	ItemName string
	Weight   int // relative probability weight; higher = more common
	HQItemID string // item returned if the HQ roll also succeeds; "" = same item, qty*2
}

// Yield is the result of a successful mining attempt.
type Yield struct {
	ItemID   string
	ItemName string
	Qty      int
	HQ       bool // true if the HQ roll passed
}

// MiningPoint is a resource node in the world.
// Not goroutine-safe; callers synchronise via the server tick loop.
type MiningPoint struct {
	ID          string
	Name        string
	Difficulty  float64  // 0–110; governs success/HQ rates
	MaxAttempts int      // how many successful pulls before depletion
	Loot        []LootEntry
	remaining   int      // attempts remaining; set to MaxAttempts on first Mine
	rng         *rand.Rand
}

// Errors
var (
	ErrPointDepleted = errors.New("mining point depleted")
	ErrNoLoot        = errors.New("mining point has no loot table")
)

// NewPoint creates a MiningPoint ready for mining.
func NewPoint(id, name string, difficulty float64, maxAttempts int, loot []LootEntry, seed int64) *MiningPoint {
	return &MiningPoint{
		ID:          id,
		Name:        name,
		Difficulty:  clampSkill(difficulty),
		MaxAttempts: maxAttempts,
		Loot:        loot,
		remaining:   maxAttempts,
		rng:         rand.New(rand.NewSource(seed)),
	}
}

// Remaining returns how many successful pulls are left before this point depletes.
func (p *MiningPoint) Remaining() int { return p.remaining }

// Depleted reports whether this point is exhausted.
func (p *MiningPoint) Depleted() bool { return p.remaining <= 0 }

// Respawn resets the point to full (server tick loop calls this after respawn timer).
func (p *MiningPoint) Respawn() { p.remaining = p.MaxAttempts }

// Mine attempts one mining action by a player with the given skill level.
// Returns ErrPointDepleted if no yields remain, ErrNoLoot if loot table is empty.
// A failed roll returns (Yield{}, nil) — no yield but not an error.
func (p *MiningPoint) Mine(playerSkill float64) (Yield, error) {
	if len(p.Loot) == 0 {
		return Yield{}, ErrNoLoot
	}
	if p.remaining <= 0 {
		return Yield{}, ErrPointDepleted
	}

	skill := clampSkill(playerSkill)

	// Success roll.
	successChance := SuccessBase + (skill-p.Difficulty)*SuccessPerLevel
	if successChance < SuccessFloor {
		successChance = SuccessFloor
	}
	if successChance > SuccessCap {
		successChance = SuccessCap
	}
	if p.rng.Float64()*100 >= successChance {
		return Yield{}, nil // failed attempt — no yield, point attempt NOT consumed
	}

	// Pick loot.
	entry := p.rollLoot()
	p.remaining--

	// HQ roll.
	hqChance := (skill - p.Difficulty - HQThreshold) * HQPerLevel
	if hqChance < 0 {
		hqChance = 0
	}
	if hqChance > HQCap {
		hqChance = HQCap
	}
	hq := hqChance > 0 && p.rng.Float64()*100 < hqChance

	itemID := entry.ItemID
	itemName := entry.ItemName
	qty := 1
	if hq && entry.HQItemID != "" {
		itemID = entry.HQItemID
		itemName = OreName(itemID)
	} else if hq {
		qty = 2 // HQ without alternate item = double quantity
	}

	return Yield{ItemID: itemID, ItemName: itemName, Qty: qty, HQ: hq}, nil
}

// SuccessChance returns the computed success % for a given skill (for UI display).
func (p *MiningPoint) SuccessChance(playerSkill float64) float64 {
	skill := clampSkill(playerSkill)
	c := SuccessBase + (skill-p.Difficulty)*SuccessPerLevel
	if c < SuccessFloor {
		return SuccessFloor
	}
	if c > SuccessCap {
		return SuccessCap
	}
	return c
}

// HQChance returns the computed HQ % for a given skill (for UI display).
func (p *MiningPoint) HQChance(playerSkill float64) float64 {
	skill := clampSkill(playerSkill)
	c := (skill - p.Difficulty - HQThreshold) * HQPerLevel
	if c < 0 {
		return 0
	}
	if c > HQCap {
		return HQCap
	}
	return c
}

func (p *MiningPoint) rollLoot() LootEntry {
	total := 0
	for _, e := range p.Loot {
		total += e.Weight
	}
	roll := p.rng.Intn(total)
	cum := 0
	for _, e := range p.Loot {
		cum += e.Weight
		if roll < cum {
			return e
		}
	}
	return p.Loot[len(p.Loot)-1]
}

func clampSkill(s float64) float64 {
	if s < SkillMin {
		return SkillMin
	}
	if s > SkillCap {
		return SkillCap
	}
	return s
}

// --- Preset mining points ---

// loot helpers

func lootEntry(itemID string, weight int) LootEntry {
	return LootEntry{ItemID: itemID, ItemName: OreName(itemID), Weight: weight}
}

func lootEntryHQ(itemID, hqID string, weight int) LootEntry {
	return LootEntry{ItemID: itemID, ItemName: OreName(itemID), Weight: weight, HQItemID: hqID}
}

// MeadowMiningPoints returns the default mining point set for the Meadow (zone 0).
// Copper/iron/tin ore + earth crystals; suitable for skill 0–30.
func MeadowMiningPoints() []*MiningPoint {
	return []*MiningPoint{
		NewPoint("mine-meadow-0", "Meadow Rock", 0, 8, []LootEntry{
			lootEntry(OreCopper, 50),
			lootEntry(OreTin, 30),
			lootEntry(MatClay, 15),
			lootEntry(CrystalEarth, 5),
		}, 1001),
		NewPoint("mine-meadow-1", "Meadow Outcrop", 5, 8, []LootEntry{
			lootEntryHQ(OreCopper, OreIron, 40),
			lootEntry(OreTin, 35),
			lootEntry(OreIron, 20),
			lootEntry(CrystalEarth, 5),
		}, 1002),
		NewPoint("mine-meadow-2", "Old Stone Wall", 10, 6, []LootEntry{
			lootEntry(OreIron, 50),
			lootEntry(OreCopper, 30),
			lootEntry(CrystalEarth, 20),
		}, 1003),
	}
}

// SwampMiningPoints returns the default mining point set for Swampville (zone 3).
// Clay/peat/swampite + water crystals; suitable for skill 10–50.
func SwampMiningPoints() []*MiningPoint {
	return []*MiningPoint{
		NewPoint("mine-swamp-0", "Bog Deposit", 15, 10, []LootEntry{
			lootEntry(MatClay, 45),
			lootEntry(MatPeat, 35),
			lootEntry(CrystalWater, 15),
			lootEntry(OreSwampite, 5),
		}, 2001),
		NewPoint("mine-swamp-1", "Murk Vein", 25, 8, []LootEntry{
			lootEntryHQ(OreSwampite, OreSilver, 40),
			lootEntry(MatClay, 30),
			lootEntry(CrystalWater, 20),
			lootEntry(MatPeat, 10),
		}, 2002),
		NewPoint("mine-swamp-2", "Deep Mire Node", 40, 6, []LootEntry{
			lootEntry(OreSwampite, 45),
			lootEntryHQ(OreSilver, OreGold, 30),
			lootEntry(CrystalWater, 25),
		}, 2003),
	}
}

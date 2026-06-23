package gather

import (
	"errors"
	"math/rand"
)

// Fish IDs — canonical item identifiers for fish species.
const (
	FishCarp           = "fish-carp"
	FishBass           = "fish-bass"
	FishTrout          = "fish-trout"
	FishEel            = "fish-eel"
	FishCrab           = "fish-crab"
	FishSardine        = "fish-fullmoon-sardine"
	FishTrophyCarp     = "fish-trophy-carp"
	FishTrophyBass     = "fish-trophy-bass"
	FishGiantSquid     = "fish-giant-squid"
	FishMudCrabSwamp   = "fish-mud-crab"
	FishBogEel         = "fish-bog-eel"
	FishCrimsonTrout   = "fish-crimson-trout"
)

var fishNames = map[string]string{
	FishCarp:         "Carp",
	FishBass:         "Bass",
	FishTrout:        "Trout",
	FishEel:          "Eel",
	FishCrab:         "Crab",
	FishSardine:      "Fullmoon Sardine",
	FishTrophyCarp:   "Trophy Carp",
	FishTrophyBass:   "Trophy Bass",
	FishGiantSquid:   "Giant Squid",
	FishMudCrabSwamp: "Mud Crab",
	FishBogEel:       "Bog Eel",
	FishCrimsonTrout: "Crimson Trout",
}

// FishName returns the display name for a fish item ID.
func FishName(id string) string {
	if n, ok := fishNames[id]; ok {
		return n
	}
	return id
}

// FishingErrors.
var (
	ErrFishingPointDepleted = errors.New("fishing point depleted — no bites here now")
	ErrNoFishLoot           = errors.New("fishing point has no fish table")
)

// FishLootEntry is one fish species in a FishingPoint's loot table.
type FishLootEntry struct {
	ItemID    string
	ItemName  string
	Weight    int
	HQItemID  string // trophy variant; "" = same item qty*2
}

// FishYield is the result of a successful fishing attempt.
type FishYield struct {
	ItemID   string
	ItemName string
	Qty      int
	Trophy   bool // true if HQ trophy fish
}

// FishingPoint is a body of water in the world where players can fish.
// Not goroutine-safe; callers synchronise via the server tick loop.
type FishingPoint struct {
	ID          string
	Name        string
	SceneID     int
	Difficulty  float64    // 0–110
	MaxAttempts int        // pulls before depletion
	Loot        []FishLootEntry
	remaining   int
	rng         *rand.Rand
}

// FishSkill tracks a player's fishing skill (0–110) and wraps attempt logic.
type FishSkill struct {
	Level float64
}

// NewFishingPoint creates a FishingPoint ready for use.
func NewFishingPoint(id, name string, sceneID int, difficulty float64, maxAttempts int, loot []FishLootEntry, seed int64) *FishingPoint {
	return &FishingPoint{
		ID:          id,
		Name:        name,
		SceneID:     sceneID,
		Difficulty:  clampSkill(difficulty),
		MaxAttempts: maxAttempts,
		Loot:        loot,
		remaining:   maxAttempts,
		rng:         rand.New(rand.NewSource(seed)),
	}
}

// Remaining returns pulls left before depletion.
func (fp *FishingPoint) Remaining() int { return fp.remaining }

// Depleted reports whether the point is exhausted.
func (fp *FishingPoint) Depleted() bool { return fp.remaining <= 0 }

// Respawn resets the point to full.
func (fp *FishingPoint) Respawn() { fp.remaining = fp.MaxAttempts }

// Attempt performs one fishing cast at the point using the player's FishSkill.
// Returns ErrFishingPointDepleted or ErrNoFishLoot on hard errors.
// A missed cast (bad luck) returns (FishYield{}, nil) — no yield, no error.
func (fp *FishingPoint) Attempt(skill FishSkill) (FishYield, error) {
	if len(fp.Loot) == 0 {
		return FishYield{}, ErrNoFishLoot
	}
	if fp.remaining <= 0 {
		return FishYield{}, ErrFishingPointDepleted
	}

	s := clampSkill(skill.Level)

	successChance := SuccessBase + (s-fp.Difficulty)*SuccessPerLevel
	if successChance < SuccessFloor {
		successChance = SuccessFloor
	}
	if successChance > SuccessCap {
		successChance = SuccessCap
	}
	if fp.rng.Float64()*100 >= successChance {
		return FishYield{}, nil // missed bite
	}

	entry := fp.rollFish()
	fp.remaining--

	// Trophy roll (same thresholds as mining HQ).
	hqChance := (s - fp.Difficulty - HQThreshold) * HQPerLevel
	if hqChance < 0 {
		hqChance = 0
	}
	if hqChance > HQCap {
		hqChance = HQCap
	}
	trophy := hqChance > 0 && fp.rng.Float64()*100 < hqChance

	itemID := entry.ItemID
	itemName := entry.ItemName
	qty := 1
	if trophy && entry.HQItemID != "" {
		itemID = entry.HQItemID
		itemName = FishName(itemID)
	} else if trophy {
		qty = 2
	}

	return FishYield{ItemID: itemID, ItemName: itemName, Qty: qty, Trophy: trophy}, nil
}

// SuccessChance returns the computed success % for a given FishSkill.
func (fp *FishingPoint) SuccessChance(skill FishSkill) float64 {
	s := clampSkill(skill.Level)
	c := SuccessBase + (s-fp.Difficulty)*SuccessPerLevel
	if c < SuccessFloor {
		return SuccessFloor
	}
	if c > SuccessCap {
		return SuccessCap
	}
	return c
}

func (fp *FishingPoint) rollFish() FishLootEntry {
	total := 0
	for _, e := range fp.Loot {
		total += e.Weight
	}
	roll := fp.rng.Intn(total)
	cum := 0
	for _, e := range fp.Loot {
		cum += e.Weight
		if roll < cum {
			return e
		}
	}
	return fp.Loot[len(fp.Loot)-1]
}

// --- loot helpers ---

func fishEntry(itemID string, weight int) FishLootEntry {
	return FishLootEntry{ItemID: itemID, ItemName: FishName(itemID), Weight: weight}
}

func fishEntryHQ(itemID, hqID string, weight int) FishLootEntry {
	return FishLootEntry{ItemID: itemID, ItemName: FishName(itemID), Weight: weight, HQItemID: hqID}
}

// --- Preset fishing points ---

// MeadowFishingPoints returns the default fishing spots for the Meadow (scene 0).
// Common freshwater fish; suitable for fishing skill 0–20.
func MeadowFishingPoints() []*FishingPoint {
	return []*FishingPoint{
		NewFishingPoint("fish-meadow-0", "Meadow Stream", 0, 0, 10, []FishLootEntry{
			fishEntryHQ(FishCarp, FishTrophyCarp, 50),
			fishEntry(FishBass, 30),
			fishEntry(FishSardine, 20),
		}, 3001),
		NewFishingPoint("fish-meadow-1", "Still Pond", 0, 10, 8, []FishLootEntry{
			fishEntry(FishTrout, 45),
			fishEntryHQ(FishBass, FishTrophyBass, 35),
			fishEntry(FishCrab, 20),
		}, 3002),
	}
}

// SwampFishingPoints returns the default fishing spots for Swampville (scene 3).
// Swamp species with greater difficulty; suitable for fishing skill 20–50.
func SwampFishingPoints() []*FishingPoint {
	return []*FishingPoint{
		NewFishingPoint("fish-swamp-0", "Bog Shallows", 3, 20, 10, []FishLootEntry{
			fishEntry(FishMudCrabSwamp, 45),
			fishEntry(FishEel, 35),
			fishEntry(FishBogEel, 20),
		}, 4001),
		NewFishingPoint("fish-swamp-1", "Deep Mire", 3, 35, 8, []FishLootEntry{
			fishEntryHQ(FishBogEel, FishGiantSquid, 40),
			fishEntry(FishMudCrabSwamp, 30),
			fishEntryHQ(FishCrimsonTrout, FishGiantSquid, 30),
		}, 4002),
	}
}

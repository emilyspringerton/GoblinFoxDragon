// Package merit implements the DragonsNShit merit point system.
//
// # Merit points (FFXI-parity, simplified)
//
// At level cap (L99), kill XP converts to merit points at a fixed rate.
// A character may accumulate up to MeritCap total merit points.
// Merit points are spent on job enhancements (categories with tier caps).
// Unspent points accumulate; total banked is capped at MeritCap.
//
// FFXI original: 10,000 XP = 1 merit. We use a simpler 1000:1 ratio.
package merit

import "errors"

const (
	MeritCap          = 30    // maximum banked merit points
	XPPerMerit        = 1000  // XP needed to earn 1 merit point
	MaxTierPerCategory = 5    // maximum tier within any single category
)

var (
	ErrMeritCapReached    = errors.New("merit point cap already reached")
	ErrInsufficientMerits = errors.New("insufficient merit points")
	ErrInvalidCategory    = errors.New("unknown merit category")
	ErrTierCapped         = errors.New("category is already at maximum tier")
)

// Category is a string key identifying a merit enhancement category.
type Category = string

// Standard merit categories (FFXI-parity subset).
const (
	CatHPMP    Category = "hp-mp"     // HP/MP increase
	CatSTR     Category = "str"       // STR enhancement
	CatDEX     Category = "dex"
	CatVIT     Category = "vit"
	CatAGI     Category = "agi"
	CatINT     Category = "int"
	CatMND     Category = "mnd"
	CatCHR     Category = "chr"
	CatAttack  Category = "attack"    // melee attack bonus
	CatMagic   Category = "magic"     // magic attack bonus
)

// AllCategories lists all merit categories.
var AllCategories = []Category{CatHPMP, CatSTR, CatDEX, CatVIT, CatAGI, CatINT, CatMND, CatCHR, CatAttack, CatMagic}

// MeritBank holds a character's merit points and spending history.
// Not goroutine-safe; callers synchronise via server tick.
type MeritBank struct {
	Points    int              // current banked merit points
	Tiers     map[Category]int // tiers spent per category
	TotalEarned int            // lifetime total merits earned
}

// NewMeritBank creates a bank with 0 points and 0 tiers in all categories.
func NewMeritBank() *MeritBank {
	tiers := make(map[Category]int, len(AllCategories))
	for _, cat := range AllCategories {
		tiers[cat] = 0
	}
	return &MeritBank{Tiers: tiers}
}

// Earn converts xp amount into merit points (1 merit per XPPerMerit XP).
// Excess XP below one merit is discarded.
// Returns the number of merits earned (may be 0 if xp < XPPerMerit).
// Returns ErrMeritCapReached if already at cap.
func (b *MeritBank) Earn(xp int) (int, error) {
	if b.Points >= MeritCap {
		return 0, ErrMeritCapReached
	}
	gained := xp / XPPerMerit
	if gained == 0 {
		return 0, nil
	}
	b.Points += gained
	b.TotalEarned += gained
	if b.Points > MeritCap {
		b.Points = MeritCap
	}
	return gained, nil
}

// Spend spends 1 merit point to advance category to the next tier.
// Tier cost scales: each tier costs (tier+1) merits, i.e. T1=1, T2=2, T3=3, T4=4, T5=5.
// Returns ErrInvalidCategory, ErrTierCapped, or ErrInsufficientMerits on failure.
func (b *MeritBank) Spend(category Category) error {
	cur, ok := b.Tiers[category]
	if !ok {
		return ErrInvalidCategory
	}
	if cur >= MaxTierPerCategory {
		return ErrTierCapped
	}
	cost := cur + 1 // T0→T1 costs 1, T1→T2 costs 2, etc.
	if b.Points < cost {
		return ErrInsufficientMerits
	}
	b.Points -= cost
	b.Tiers[category]++
	return nil
}

// TierOf returns the current tier for a category. 0 if unspent.
func (b *MeritBank) TierOf(category Category) int { return b.Tiers[category] }

// TotalSpent returns how many merits have been spent across all categories.
func (b *MeritBank) TotalSpent() int {
	total := 0
	for _, tier := range b.Tiers {
		for t := 0; t < tier; t++ {
			total += t + 1
		}
	}
	return total
}

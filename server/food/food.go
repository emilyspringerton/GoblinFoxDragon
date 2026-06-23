// Package food implements server-authoritative food consumption and stat buff effects.
//
// Eating a Food item grants a FoodEffect that lasts for Duration seconds.
// Only one food effect may be active at a time; eating a second food replaces the first.
//
// Stat bonuses are additive with base stats and applied by the game loop when computing
// effective stats for combat/gathering checks.
package food

import (
	"errors"
	"time"
)

// Food describes a consumable food item and its stat buffs.
type Food struct {
	ID        string
	Name      string
	STRBonus  int
	DEXBonus  int
	VITBonus  int
	INTBonus  int
	MNDBonus  int
	HPBonus   int
	MPBonus   int
	Duration  time.Duration
}

// FoodEffect tracks an active food buff on a character.
type FoodEffect struct {
	Food     Food
	ExpiresAt time.Time
}

// IsActive reports whether the food buff has not yet expired.
func (e *FoodEffect) IsActive(now time.Time) bool {
	return e != nil && now.Before(e.ExpiresAt)
}

// Remaining returns the time left on the buff, or 0 if expired/nil.
func (e *FoodEffect) Remaining(now time.Time) time.Duration {
	if !e.IsActive(now) {
		return 0
	}
	return e.ExpiresAt.Sub(now)
}

// Registry is the server-side food item catalog.
type Registry struct {
	items map[string]Food
}

var ErrUnknownFood = errors.New("unknown food item")

// NewRegistry creates a Registry pre-loaded with the given items.
func NewRegistry(items []Food) *Registry {
	r := &Registry{items: make(map[string]Food, len(items))}
	for _, f := range items {
		r.items[f.ID] = f
	}
	return r
}

// Get returns the Food for the given item ID, or ErrUnknownFood.
func (r *Registry) Get(id string) (Food, error) {
	f, ok := r.items[id]
	if !ok {
		return Food{}, ErrUnknownFood
	}
	return f, nil
}

// All returns all registered food items in undefined order.
func (r *Registry) All() []Food {
	out := make([]Food, 0, len(r.items))
	for _, f := range r.items {
		out = append(out, f)
	}
	return out
}

// Eat consumes a food item, replacing any prior active effect.
// Returns ErrUnknownFood if the item ID is not in the registry.
func (r *Registry) Eat(id string, now time.Time) (*FoodEffect, error) {
	f, err := r.Get(id)
	if err != nil {
		return nil, err
	}
	return &FoodEffect{
		Food:      f,
		ExpiresAt: now.Add(f.Duration),
	}, nil
}

// --- Standard food items (FFXI-inspired) ---

const (
	IDMeatMithkabob    = "food-meat-mithkabob"
	IDTavnazianSalad   = "food-tavnazian-salad"
	IDRolanberryPie    = "food-rolanberry-pie"
	IDSelbinaMilk      = "food-selbina-milk"
	IDPear             = "food-pear"
	IDFishAndChips     = "food-fish-and-chips"
	IDCrabSoup         = "food-crab-soup"
	IDSwampHerbalTea   = "food-swamp-herbal-tea"
)

// StandardFoods returns the default food registry for DragonsNShit.
func StandardFoods() []Food {
	return []Food{
		{
			ID:       IDMeatMithkabob,
			Name:     "Meat Mithkabob",
			STRBonus: 5,
			Duration: 30 * time.Minute,
		},
		{
			ID:       IDTavnazianSalad,
			Name:     "Tavnazian Salad",
			DEXBonus: 5,
			Duration: 30 * time.Minute,
		},
		{
			ID:       IDRolanberryPie,
			Name:     "Rolanberry Pie",
			MPBonus:  50,
			Duration: 20 * time.Minute,
		},
		{
			ID:       IDSelbinaMilk,
			Name:     "Selbina Milk",
			VITBonus: 3,
			HPBonus:  30,
			Duration: 10 * time.Minute,
		},
		{
			ID:      IDPear,
			Name:    "Pear",
			HPBonus: 10,
			MPBonus: 10,
			Duration: 5 * time.Minute,
		},
		{
			ID:       IDFishAndChips,
			Name:     "Fish and Chips",
			STRBonus: 2,
			VITBonus: 2,
			HPBonus:  20,
			Duration: 20 * time.Minute,
		},
		{
			ID:       IDCrabSoup,
			Name:     "Crab Soup",
			VITBonus: 4,
			MNDBonus: 2,
			HPBonus:  40,
			Duration: 15 * time.Minute,
		},
		{
			ID:       IDSwampHerbalTea,
			Name:     "Swamp Herbal Tea",
			INTBonus: 3,
			MNDBonus: 3,
			MPBonus:  30,
			Duration: 20 * time.Minute,
		},
	}
}

// DefaultRegistry returns a Registry loaded with StandardFoods.
func DefaultRegistry() *Registry {
	return NewRegistry(StandardFoods())
}

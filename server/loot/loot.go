// Package loot implements the DragonsNShit treasure pool system.
//
// # Treasure pool model (FFXI-parity)
//
// When a mob dies its loot drops into a shared Pool visible to all party members
// who have claim (tagged the mob). Each item in the pool must be resolved:
//   - Lot: roll 1–999 randomly. Each player may lot each item once.
//   - Pass: explicitly decline the item.
//
// Resolution happens when every eligible player has either lotted or passed on
// every item. The player with the highest lot wins; ties are resolved by slot
// string order (deterministic). Passers receive nothing.
package loot

import (
	"errors"
	"math/rand"
	"sort"
)

const (
	LotMin = 1
	LotMax = 999
)

var (
	ErrAlreadyActed  = errors.New("player already lotted or passed this item")
	ErrNotEligible   = errors.New("player is not eligible for this pool")
	ErrItemNotFound  = errors.New("item not found in pool")
	ErrPoolResolved  = errors.New("pool is already resolved")
)

// Item is a single drop in the pool.
type Item struct {
	ID   string
	Name string
}

// LotResult is a player's lot on a specific item.
type LotResult struct {
	Slot string
	Roll int // 0 = passed
}

// Award records the winner of one item after pool resolution.
type Award struct {
	ItemID string
	Slot   string // slot that won; "" = no winner (all passed)
	Roll   int
}

// Pool is the shared treasure pool for a mob kill.
// Not goroutine-safe; callers synchronise via the server tick loop.
type Pool struct {
	MobID    string
	Eligible []string        // slots allowed to lot (tagged party members)
	Items    []Item
	lots     map[string]map[string]LotResult // itemID → slot → result
	resolved bool
	rng      *rand.Rand
}

// NewPool creates a treasure pool.
func NewPool(mobID string, eligible []string, items []Item, seed int64) *Pool {
	p := &Pool{
		MobID:    mobID,
		Eligible: eligible,
		Items:    items,
		lots:     make(map[string]map[string]LotResult),
		rng:      rand.New(rand.NewSource(seed)),
	}
	for _, it := range items {
		p.lots[it.ID] = make(map[string]LotResult)
	}
	return p
}

// isEligible reports whether slot is in the eligible list.
func (p *Pool) isEligible(slot string) bool {
	for _, s := range p.Eligible {
		if s == slot {
			return true
		}
	}
	return false
}

// Lot records a random lot roll (LotMin–LotMax) for slot on itemID.
func (p *Pool) Lot(slot, itemID string) (int, error) {
	if p.resolved {
		return 0, ErrPoolResolved
	}
	if !p.isEligible(slot) {
		return 0, ErrNotEligible
	}
	itemLots, ok := p.lots[itemID]
	if !ok {
		return 0, ErrItemNotFound
	}
	if _, acted := itemLots[slot]; acted {
		return 0, ErrAlreadyActed
	}
	roll := p.rng.Intn(LotMax-LotMin+1) + LotMin
	itemLots[slot] = LotResult{Slot: slot, Roll: roll}
	return roll, nil
}

// Pass records a pass (roll=0) for slot on itemID.
func (p *Pool) Pass(slot, itemID string) error {
	if p.resolved {
		return ErrPoolResolved
	}
	if !p.isEligible(slot) {
		return ErrNotEligible
	}
	itemLots, ok := p.lots[itemID]
	if !ok {
		return ErrItemNotFound
	}
	if _, acted := itemLots[slot]; acted {
		return ErrAlreadyActed
	}
	itemLots[slot] = LotResult{Slot: slot, Roll: 0}
	return nil
}

// Ready reports whether every eligible player has acted on every item.
func (p *Pool) Ready() bool {
	for _, it := range p.Items {
		itemLots := p.lots[it.ID]
		for _, slot := range p.Eligible {
			if _, acted := itemLots[slot]; !acted {
				return false
			}
		}
	}
	return true
}

// Resolve determines winners for all items and marks the pool resolved.
// Returns ErrPoolResolved if already resolved.
// Partial resolution (not all players have acted) is allowed — pending slots
// are treated as having passed.
func (p *Pool) Resolve() ([]Award, error) {
	if p.resolved {
		return nil, ErrPoolResolved
	}
	p.resolved = true

	awards := make([]Award, 0, len(p.Items))
	for _, it := range p.Items {
		itemLots := p.lots[it.ID]

		// Collect all actual lots (roll > 0).
		type candidate struct {
			slot string
			roll int
		}
		var candidates []candidate
		for slot, lr := range itemLots {
			if lr.Roll > 0 {
				candidates = append(candidates, candidate{slot, lr.Roll})
			}
		}

		if len(candidates) == 0 {
			awards = append(awards, Award{ItemID: it.ID, Slot: "", Roll: 0})
			continue
		}

		// Sort by roll desc, then slot asc for deterministic tie-break.
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].roll != candidates[j].roll {
				return candidates[i].roll > candidates[j].roll
			}
			return candidates[i].slot < candidates[j].slot
		})
		w := candidates[0]
		awards = append(awards, Award{ItemID: it.ID, Slot: w.slot, Roll: w.roll})
	}
	return awards, nil
}

// Resolved reports whether Resolve has been called.
func (p *Pool) Resolved() bool { return p.resolved }

// LotStatus returns a snapshot of all lot results for a given itemID.
func (p *Pool) LotStatus(itemID string) ([]LotResult, bool) {
	m, ok := p.lots[itemID]
	if !ok {
		return nil, false
	}
	out := make([]LotResult, 0, len(m))
	for _, lr := range m {
		out = append(out, lr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slot < out[j].Slot })
	return out, true
}

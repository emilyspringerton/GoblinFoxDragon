// Package mobdrop implements the DragonsNShit mob loot-table registry.
//
// A DropTable describes what a mob of a given Kind drops on death — the same
// real role server/itemdef plays for equippable items, and the same real
// "data-driven, loaded once at startup from JSON" shape. This replaced an
// earlier hardcoded Go switch statement (apps2/mud's own dropsForMob) so the
// GFD-MD-001 admin GUI can manage drop tables without a code change + redeploy.
//
// # Real, honest scope limitation
//
// Drop tables key on mob Kind only, not on (Kind, zone). GFD's own mob spawn
// code (server/mob/hills.go, caves.go, swamp.go, etc.) does not track which
// zone a given Kind spawns in as data either — a mob's zone is purely a
// runtime fact of which zone's registry it was spawned into, not a property
// of its Kind. So "different drops for the same mob Kind in different zones"
// is not representable today; this mirrors the exact same real limitation
// already named for the NPC vendor catalog (S251-06). Every mob of a given
// Kind, in any zone, drops from the same table.
package mobdrop

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
)

// Item is one entry in a drop table — deliberately the same shape as
// server/loot.Item so a DropTable's Items slice can be passed straight into
// loot.NewPool with no conversion.
//
// DropChance (S412-09, founder real-time: "the gfd-mob-drops admin interface needs to be able to
// tune drop rates for each item") is real, new, and additive -- every item in every existing
// data/mob_drops.json table drops unconditionally today (checked directly: dropsForMob/DropsFor
// return the WHOLE table on every single kill, no roll of any kind exists anywhere in this
// codebase), which is exactly why the admin page could never expose a rate control -- there was
// no rate concept in the data model to expose. A pointer (not a plain float64) so JSON can tell
// "this key was never in the file" (nil -- an existing entry authored before this field existed)
// apart from "an operator explicitly configured 0%" (a real, deliberate "never drops right now"
// setting, e.g. to temporarily disable one item without deleting it from the table) -- a plain
// float64's own zero value can't distinguish those two real, different intents, and a real
// tuning tool needs the full 0-100% range including a genuine zero, not just "unset defaults to
// always."
type Item struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	DropChance *float64 `json:"drop_chance,omitempty"` // nil = unset (always, pre-existing behavior); 0.0-1.0 otherwise
}

// EffectiveDropChance returns i's real roll probability -- nil (the key was never in the JSON at
// all) normalizes to 1.0 (always drops, the exact real, pre-existing behavior every table had
// before this field existed), so every pre-existing drop table's real, current behavior is
// completely unchanged unless an operator explicitly sets a rate. An explicit value outside
// [0, 1] is clamped rather than trusted verbatim (a real, malformed admin-page edit -- e.g. "150"
// typed into a percent-shaped field by mistake -- should degrade to a sane bound, not silently
// guarantee or forbid a drop in a way the operator didn't actually intend); an explicit 0 is
// respected as a real, deliberate "never drops."
func (i Item) EffectiveDropChance() float64 {
	if i.DropChance == nil {
		return 1.0
	}
	v := *i.DropChance
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1.0
	}
	return v
}

// DropTable is the full set of items a mob of a given Kind can drop.
type DropTable struct {
	Kind  string `json:"kind"` // mob.Kind, case-insensitive (e.g. "worm", "King Worm")
	Items []Item `json:"items"`
}

// DefaultDrop is what any mob Kind with no registered table drops — matches
// the old dropsForMob switch statement's own "default:" branch exactly.
var DefaultDrop = Item{ID: "flow-drop", Name: "100 Flow"}

// Registry is the server-authoritative mob drop-table store.
// Safe for concurrent reads after construction.
type Registry struct {
	mu     sync.RWMutex
	byKind map[string]*DropTable // lowercase kind
}

// NewRegistry constructs an empty registry.
func NewRegistry() *Registry {
	return &Registry{byKind: make(map[string]*DropTable)}
}

// LoadFile loads drop tables from a JSON file (array of DropTable).
func (r *Registry) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("mobdrop: load %s: %w", path, err)
	}
	return r.LoadJSON(data)
}

// LoadJSON parses a JSON byte slice of []DropTable and registers all tables.
func (r *Registry) LoadJSON(data []byte) error {
	var tables []DropTable
	if err := json.Unmarshal(data, &tables); err != nil {
		return fmt.Errorf("mobdrop: parse: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range tables {
		t := &tables[i]
		r.byKind[strings.ToLower(t.Kind)] = t
	}
	return nil
}

// DropsFor returns the FULL configured drop table for the given kind, unaffected by
// DropChance/randomness -- the real, raw "what's in the table" view the admin API's own listing
// endpoint needs (an operator editing rates wants to see and change every entry, not a randomly
// filtered subset of them). A kind with no registered table falls back to DefaultDrop, matching
// the old hardcoded switch statement's own default branch. Real gameplay drop resolution goes
// through RollDropsFor below, not this function directly.
func (r *Registry) DropsFor(kind string) []Item {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if t, ok := r.byKind[strings.ToLower(kind)]; ok {
		out := make([]Item, len(t.Items))
		copy(out, t.Items)
		return out
	}
	return []Item{DefaultDrop}
}

// RollDropsFor is the real, per-item chance roll (S412-09) -- each configured item independently
// rolls against its own EffectiveDropChance, so a kill can drop anywhere from zero to every item
// in the table, not always the whole thing. DefaultDrop (the no-table fallback) is never rolled
// against -- a mob with no real, configured table keeps its existing, unconditional 100% flow
// drop rather than gaining a new, unintended chance to drop nothing at all.
func (r *Registry) RollDropsFor(kind string, rng *rand.Rand) []Item {
	all := r.DropsFor(kind)
	r.mu.RLock()
	_, hasRealTable := r.byKind[strings.ToLower(kind)]
	r.mu.RUnlock()
	if !hasRealTable {
		return all // DefaultDrop -- always drops, unaffected by rate rolling
	}
	out := make([]Item, 0, len(all))
	for _, item := range all {
		if rng.Float64() < item.EffectiveDropChance() {
			out = append(out, item)
		}
	}
	return out
}

// All returns every registered drop table.
func (r *Registry) All() []*DropTable {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*DropTable, 0, len(r.byKind))
	for _, t := range r.byKind {
		out = append(out, t)
	}
	return out
}

// Package mobvariant implements the DragonsNShit mob difficulty-tier variant registry.
//
// Phase 4 of GoblinFoxDragon/docs2/MOB_SPAWN_NORTHSTAR.md (kanban GFD-MOBSPAWN-001, founder
// real-time: "ffxi builds lots of different difficulties of mobs from the same models with
// repaints and sometimes without even repaints" / "they like to have some way harder mobs
// pretty close to lower level mobs with the same model and texture just a different name").
//
// A Variant is a real, named difficulty tier layered on top of an existing base mob Kind --
// same model/texture, a display name override, and a stat multiplier applied to HP and melee
// damage (generalizing server/mob/dungeon.go's own real DungeonEliteHPMul/DungeonBossHPMul
// precedent, which already does exactly this for named dungeon bosses/elites). A variant
// deliberately keeps the underlying mob.Mob.Kind unchanged -- only mob.Mob.DisplayName is set --
// so loot (dropsForMob), quest tracking, and AI dispatch all key off the real base Kind
// automatically. This is a real, decided design choice, not an oversight: the founder chose
// "a variant inherits its base Kind's drop table" over "every variant needs its own table" when
// asked directly (2026-09-04).
package mobvariant

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Variant is one row of data/mob_variants.json.
type Variant struct {
	BaseKind    string  `json:"base_kind"`
	DisplayName string  `json:"display_name"`
	PowerMul    float64 `json:"power_mul"` // scales HP and melee damage; 1.0 = no change
}

// Registry is the server-authoritative variant store, keyed by DisplayName (the real, unique
// identifier a spawn rule or admin GUI names a variant by). Safe for concurrent reads after
// construction.
type Registry struct {
	mu   sync.RWMutex
	byID map[string]*Variant // lowercase display name
}

// NewRegistry constructs an empty registry.
func NewRegistry() *Registry {
	return &Registry{byID: make(map[string]*Variant)}
}

// LoadFile loads variants from a JSON file (array of Variant).
func (r *Registry) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("mobvariant: load %s: %w", path, err)
	}
	return r.LoadJSON(data)
}

// LoadJSON parses a JSON byte slice of []Variant and registers all of them.
func (r *Registry) LoadJSON(data []byte) error {
	var variants []Variant
	if err := json.Unmarshal(data, &variants); err != nil {
		return fmt.Errorf("mobvariant: parse: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range variants {
		v := &variants[i]
		if v.PowerMul <= 0 {
			v.PowerMul = 1.0 // real, honest default -- a zero/negative multiplier would zero out or invert HP/damage, never intentional
		}
		r.byID[strings.ToLower(v.DisplayName)] = v
	}
	return nil
}

// All returns every registered variant.
func (r *Registry) All() []*Variant {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Variant, 0, len(r.byID))
	for _, v := range r.byID {
		out = append(out, v)
	}
	return out
}

// ForBaseKind returns every variant registered against a given base Kind (case-insensitive).
func (r *Registry) ForBaseKind(baseKind string) []*Variant {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*Variant
	for _, v := range r.byID {
		if strings.EqualFold(v.BaseKind, baseKind) {
			out = append(out, v)
		}
	}
	return out
}

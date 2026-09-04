// Package spawn implements the DragonsNShit mob spawn-toggle registry.
//
// Phase 2 of GoblinFoxDragon/docs2/MOB_SPAWN_NORTHSTAR.md (kanban GFD-MOBSPAWN-001, founder
// real-time: "we should be able to control if we want to turn bunnies on or off in the
// meadow etc"). Deliberately NOT a full rewrite of server/mob's own *Spawns() functions into
// generic position data -- those functions already carry real, individually-tuned per-Kind
// stat blocks (HP/damage/aggro range/etc via each NewX constructor) that a flat JSON position
// list would either have to duplicate or throw away. Instead, this registry answers one real,
// narrow question data-driven per (zone, kind): is this kind currently allowed to spawn in this
// zone at all? The *Spawns() functions themselves stay the real source of truth for WHERE and
// WHAT STATS; this is the on/off switch layered on top, exactly the "turn bunnies off" ask.
package spawn

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// NMRule (GFD-NM-123, "the mob spawn interface can optionally specify a notorious monster
// spawn for one of the base mobs") is an optional, real config for server/nm.NMSpawn, carried
// on a Rule so the JSON-driven spawn interface can declare an NM the same way
// MeadowNMs()/HillsNMs()/etc already do in hardcoded Go -- this doesn't replace those presets
// (registered separately at world init, left untouched to avoid re-registering/renaming
// already-live NM IDs), it's the same real mechanism made available data-driven for any OTHER
// base mob a zone's own designer wants to give an NM to, without writing a new Go function.
type NMRule struct {
	ID             string  `json:"id"`               // NM mob ID
	SpawnChance    float64 `json:"spawn_chance"`     // 0.0-1.0, per nm.NMSpawn.SpawnChance
	WindowOpenSec  int     `json:"window_open_sec"`  // seconds after placeholder kill the window opens
	WindowCloseSec int     `json:"window_close_sec"` // seconds after placeholder kill the window closes
	RespawnMinutes int     `json:"respawn_minutes"`  // 0 = no respawn, per nm.NMSpawn.RespawnMinutes
}

// Rule is one row of data/mob_spawns.json: whether a given mob Kind is allowed to spawn in a
// given zone, plus an optional NM association (GFD-NM-123).
type Rule struct {
	ZoneID  int     `json:"zone_id"`
	Kind    string  `json:"kind"`
	Enabled bool    `json:"enabled"`
	NM      *NMRule `json:"nm,omitempty"`
}

type ruleKey struct {
	zoneID int
	kind   string
}

// Registry is the server-authoritative spawn-toggle store. Safe for concurrent reads after
// construction.
type Registry struct {
	mu      sync.RWMutex
	rules   map[ruleKey]bool
	nmRules map[ruleKey]NMRule // only populated for rules that declared an "nm" block
}

// NewRegistry constructs an empty registry. With no rules loaded, Enabled defaults every
// (zone, kind) pair to true -- fail-open, matching the real pre-existing behavior of every
// *Spawns() function before this registry existed (an unreachable data file must never silently
// empty out a zone).
func NewRegistry() *Registry {
	return &Registry{rules: make(map[ruleKey]bool), nmRules: make(map[ruleKey]NMRule)}
}

// LoadFile loads spawn rules from a JSON file (array of Rule).
func (r *Registry) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("spawn: load %s: %w", path, err)
	}
	return r.LoadJSON(data)
}

// LoadJSON parses a JSON byte slice of []Rule and registers all rules.
func (r *Registry) LoadJSON(data []byte) error {
	var rules []Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("spawn: parse: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rule := range rules {
		key := ruleKey{zoneID: rule.ZoneID, kind: strings.ToLower(rule.Kind)}
		r.rules[key] = rule.Enabled
		if rule.NM != nil {
			r.nmRules[key] = *rule.NM
		}
	}
	return nil
}

// Enabled reports whether a mob of the given kind is currently allowed to spawn in the given
// zone. A (zone, kind) pair with no registered rule defaults to true.
func (r *Registry) Enabled(zoneID int, kind string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	enabled, ok := r.rules[ruleKey{zoneID: zoneID, kind: strings.ToLower(kind)}]
	if !ok {
		return true
	}
	return enabled
}

// NMFor returns the NM config declared for the given (zone, kind), or nil if that rule has no
// "nm" block (the common case -- most base mobs have no NM). GFD-NM-123.
func (r *Registry) NMFor(zoneID int, kind string) *NMRule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	nmRule, ok := r.nmRules[ruleKey{zoneID: zoneID, kind: strings.ToLower(kind)}]
	if !ok {
		return nil
	}
	out := nmRule
	return &out
}

// All returns every registered rule, sorted by zone then kind is left to the caller -- this is
// a real, small registry (dozens of rows at most), not worth a sort here when every real caller
// (the admin GUI) already needs to group/sort for display anyway.
func (r *Registry) All() []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Rule, 0, len(r.rules))
	for k, enabled := range r.rules {
		rule := Rule{ZoneID: k.zoneID, Kind: k.kind, Enabled: enabled}
		if nmRule, ok := r.nmRules[k]; ok {
			nmCopy := nmRule
			rule.NM = &nmCopy
		}
		out = append(out, rule)
	}
	return out
}

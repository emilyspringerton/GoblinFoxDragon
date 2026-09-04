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

// Rule is one row of data/mob_spawns.json: whether a given mob Kind is allowed to spawn in a
// given zone.
type Rule struct {
	ZoneID  int    `json:"zone_id"`
	Kind    string `json:"kind"`
	Enabled bool   `json:"enabled"`
}

type ruleKey struct {
	zoneID int
	kind   string
}

// Registry is the server-authoritative spawn-toggle store. Safe for concurrent reads after
// construction.
type Registry struct {
	mu    sync.RWMutex
	rules map[ruleKey]bool
}

// NewRegistry constructs an empty registry. With no rules loaded, Enabled defaults every
// (zone, kind) pair to true -- fail-open, matching the real pre-existing behavior of every
// *Spawns() function before this registry existed (an unreachable data file must never silently
// empty out a zone).
func NewRegistry() *Registry {
	return &Registry{rules: make(map[ruleKey]bool)}
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
		r.rules[ruleKey{zoneID: rule.ZoneID, kind: strings.ToLower(rule.Kind)}] = rule.Enabled
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

// All returns every registered rule, sorted by zone then kind is left to the caller -- this is
// a real, small registry (dozens of rows at most), not worth a sort here when every real caller
// (the admin GUI) already needs to group/sort for display anyway.
func (r *Registry) All() []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Rule, 0, len(r.rules))
	for k, enabled := range r.rules {
		out = append(out, Rule{ZoneID: k.zoneID, Kind: k.kind, Enabled: enabled})
	}
	return out
}

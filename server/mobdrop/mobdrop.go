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
	"os"
	"strings"
	"sync"
)

// Item is one entry in a drop table — deliberately the same shape as
// server/loot.Item so a DropTable's Items slice can be passed straight into
// loot.NewPool with no conversion.
type Item struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

// DropsFor returns the loot items a mob of the given kind drops on death.
// A kind with no registered table falls back to DefaultDrop, matching the
// old hardcoded switch statement's own default branch.
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

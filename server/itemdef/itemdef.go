// Package itemdef implements the DragonsNShit item definition registry.
//
// An ItemDef describes what an item *is* — its stats, restrictions, category,
// and visual properties. ItemDefs are server-authoritative and loaded once from
// data/items.json at startup. They never mutate at runtime.
//
// Actual item instances (with UUID + provenance) live in the items DB table.
// The def_id column on that table joins to an ItemDef.
//
// # Job bitmask
//
// JobMask is a uint32 with one bit per job in canonical FFXI order:
//
//	bit 0  WAR    bit 7  BST    bit 14 SMN
//	bit 1  MNK    bit 8  BRD    bit 15 BLU
//	bit 2  WHM    bit 9  RNG    bit 16 COR
//	bit 3  BLM    bit 10 SAM    bit 17 PUP
//	bit 4  RDM    bit 11 NIN    bit 18 DNC
//	bit 5  THF    bit 12 DRG    bit 19 SCH
//	bit 6  PLD    bit 13 DRK
//
// AllJobs = (1<<20) - 1 (all 20 bits set). 0 means no job restrictions.
//
// # Stack sizes
//
// StackSize 1 = not stackable (all weapons, armor, accessories).
// StackSize 12 = crystals. StackSize 12 or 99 = consumables/materials.
package itemdef

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Category classifies an item's broad type.
type Category string

const (
	CatWeapon     Category = "weapon"
	CatArmor      Category = "armor"
	CatAccessory  Category = "accessory"
	CatConsumable Category = "consumable"
	CatMaterial   Category = "material"
	CatCrystal    Category = "crystal"
	CatKeyItem    Category = "key_item"
	CatTemporary  Category = "temporary"
)

// ItemFlags are bit flags on an item definition.
type ItemFlags uint32

const (
	FlagRare      ItemFlags = 1 << 0 // only one per character across all bags
	FlagEx        ItemFlags = 1 << 1 // cannot be traded, dropped, or bazaared
	FlagTemporary ItemFlags = 1 << 2 // auto-cleared on zone exit or death
	FlagNoSave    ItemFlags = 1 << 3 // not persisted to DB
)

func (f ItemFlags) Has(flag ItemFlags) bool { return f&flag != 0 }

// JobMask is a bitmask of job restrictions. AllJobs = all 20 jobs.
type JobMask uint32

const AllJobs JobMask = (1 << 20) - 1

// job index lookup
var jobIndex = map[string]int{
	"WAR": 0, "MNK": 1, "WHM": 2, "BLM": 3, "RDM": 4,
	"THF": 5, "PLD": 6, "DRK": 7, "BST": 8, "BRD": 9,
	"RNG": 10, "SAM": 11, "NIN": 12, "DRG": 13, "SMN": 14,
	"BLU": 15, "COR": 16, "PUP": 17, "DNC": 18, "SCH": 19,
}

// AllJobNames returns the canonical list of job abbreviations.
func AllJobNames() []string {
	out := make([]string, 20)
	for name, idx := range jobIndex {
		out[idx] = name
	}
	return out
}

// JobMaskFor builds a JobMask from a slice of job abbreviations.
// Unknown job names are silently ignored.
func JobMaskFor(jobs []string) JobMask {
	var m JobMask
	for _, j := range jobs {
		if idx, ok := jobIndex[strings.ToUpper(j)]; ok {
			m |= 1 << idx
		}
	}
	return m
}

// CanEquipJob reports whether a job can equip an item with this mask.
// A mask of 0 is treated as AllJobs.
func (m JobMask) CanEquipJob(job string) bool {
	if m == 0 {
		return true
	}
	idx, ok := jobIndex[strings.ToUpper(job)]
	if !ok {
		return false
	}
	return m&(1<<idx) != 0
}

var (
	ErrNotFound       = errors.New("itemdef: item not found")
	ErrLevelTooLow    = errors.New("itemdef: character level too low")
	ErrJobRestricted  = errors.New("itemdef: job cannot equip this item")
	ErrSlotMismatch   = errors.New("itemdef: item cannot be equipped in that slot")
	ErrNotEquipment   = errors.New("itemdef: item is not equipment")
)

// ItemDef describes what an item is.
// JSON tags match data/items.json schema.
type ItemDef struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Category    Category          `json:"category"`
	EquipSlots  []string          `json:"equip_slots,omitempty"` // subset of gear.AllSlots
	Jobs        []string          `json:"jobs,omitempty"`        // [] = all jobs; listed = restriction
	Level       int               `json:"level,omitempty"`
	Stats       map[string]int    `json:"stats,omitempty"`
	StackSize   int               `json:"stack_size"`
	FlagsRaw    []string          `json:"flags,omitempty"` // ["rare","ex",...]
	ModelID     string            `json:"model_id,omitempty"`
	IconID      int               `json:"icon_id,omitempty"`

	// Computed at load time — not in JSON.
	jobMask  JobMask
	flags    ItemFlags
}

// JobMask returns the computed job restriction mask.
func (d *ItemDef) JobMask() JobMask { return d.jobMask }

// Flags returns the computed item flags.
func (d *ItemDef) Flags() ItemFlags { return d.flags }

// IsEquipment reports whether this item can be equipped in any slot.
func (d *ItemDef) IsEquipment() bool { return len(d.EquipSlots) > 0 }

// CanEquip checks whether a character with the given job and level can equip
// this item in the given slot.
func (d *ItemDef) CanEquip(slot, job string, level int) error {
	if !d.IsEquipment() {
		return ErrNotEquipment
	}
	found := false
	for _, s := range d.EquipSlots {
		if s == slot {
			found = true
			break
		}
	}
	if !found {
		return ErrSlotMismatch
	}
	if level < d.Level {
		return fmt.Errorf("%w: need %d, have %d", ErrLevelTooLow, d.Level, level)
	}
	if len(d.Jobs) > 0 && !d.jobMask.CanEquipJob(job) {
		return fmt.Errorf("%w: %s cannot equip %q", ErrJobRestricted, job, d.Name)
	}
	return nil
}

// Registry is the server-authoritative item definition store.
// Safe for concurrent reads after construction.
type Registry struct {
	mu    sync.RWMutex
	byID  map[int]*ItemDef
	byName map[string]*ItemDef // lowercase name
}

// NewRegistry constructs an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		byID:   make(map[int]*ItemDef),
		byName: make(map[string]*ItemDef),
	}
}

// LoadFile loads item definitions from a JSON file (array of ItemDef).
// Can be called multiple times to merge additional files.
func (r *Registry) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("itemdef: load %s: %w", path, err)
	}
	return r.LoadJSON(data)
}

// LoadJSON parses a JSON byte slice of []ItemDef and registers all items.
func (r *Registry) LoadJSON(data []byte) error {
	var defs []ItemDef
	if err := json.Unmarshal(data, &defs); err != nil {
		return fmt.Errorf("itemdef: parse: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range defs {
		d := &defs[i]
		// Compute derived fields.
		d.jobMask = JobMaskFor(d.Jobs)
		d.flags = parseFlags(d.FlagsRaw)
		if d.StackSize == 0 {
			d.StackSize = 1
		}
		r.byID[d.ID] = d
		r.byName[strings.ToLower(d.Name)] = d
	}
	return nil
}

// ByID returns the ItemDef for the given numeric ID.
func (r *Registry) ByID(id int) (*ItemDef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.byID[id]
	return d, ok
}

// ByName returns the ItemDef for the given name (case-insensitive).
func (r *Registry) ByName(name string) (*ItemDef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.byName[strings.ToLower(name)]
	return d, ok
}

// All returns all registered item definitions.
func (r *Registry) All() []*ItemDef {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ItemDef, 0, len(r.byID))
	for _, d := range r.byID {
		out = append(out, d)
	}
	return out
}

// Len returns the number of registered definitions.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byID)
}

func parseFlags(raw []string) ItemFlags {
	var f ItemFlags
	for _, s := range raw {
		switch strings.ToLower(s) {
		case "rare":
			f |= FlagRare
		case "ex":
			f |= FlagEx
		case "temporary":
			f |= FlagTemporary
		case "nosave":
			f |= FlagNoSave
		}
	}
	return f
}

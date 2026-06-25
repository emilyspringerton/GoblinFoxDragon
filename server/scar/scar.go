// Package scar implements the TRAPX Scar system.
//
// A Scar is a permanent trauma record appended to a district when a
// catastrophic event occurs. Scars are append-only: they accumulate over
// a district's lifetime and increase Watcher visibility (+5% per scar).
//
// # Causes
//
//   - RogueSwarm        — containment failure of a rogue swarm operation
//   - CrownProtocol     — Tier-5 Crown Protocol fire in this district
//   - MercilessOp       — K9 Merciless Operation completed (Phase 4 resolution)
//   - FactionWar        — faction war ended with district annexation
//
// # Visibility effect
//
//   WatcherVisibilityBonus = 0.05 * len(Registry.ByDistrict[districtID])
//
// This stacks with the neighborhood base WatcherVisibilityMultiplier; the
// combat server adds the two together when computing final visibility.
//
// # MUD command
//
//   scars [district-id]   — list all scars for a district
package scar

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Cause is the category of event that wrote the scar.
type Cause string

const (
	CauseRogueSwarm    Cause = "RogueSwarm"
	CauseCrownProtocol Cause = "CrownProtocol"
	CauseMercilessOp   Cause = "MercilessOp"
	CauseFactionWar    Cause = "FactionWar"
)

// VisibilityBonusPerScar is the Watcher visibility multiplier bonus each scar adds.
const VisibilityBonusPerScar = 0.05

var (
	ErrUnknownCause    = errors.New("scar: unknown cause")
	ErrEmptyDistrictID = errors.New("scar: district ID must not be empty")
)

// validCauses is the set of accepted cause values.
var validCauses = map[Cause]bool{
	CauseRogueSwarm:    true,
	CauseCrownProtocol: true,
	CauseMercilessOp:   true,
	CauseFactionWar:    true,
}

// Scar is a single trauma record for a district.
type Scar struct {
	ID         string
	DistrictID string
	Cause      Cause
	At         time.Time
	Detail     string // optional human-readable event description
}

// String returns a one-line summary for MUD display.
func (s Scar) String() string {
	return fmt.Sprintf("[%s] %s — %s — %s", s.At.Format("2006-01-02"), s.DistrictID, s.Cause, s.Detail)
}

// Registry is an append-only store of all scars.
// Safe for concurrent reads; mutations hold the write lock.
type Registry struct {
	mu         sync.RWMutex
	scars      []Scar
	byDistrict map[string][]Scar
	seq        int64
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{byDistrict: make(map[string][]Scar)}
}

// Append records a new scar. Returns the appended scar.
func (r *Registry) Append(districtID string, cause Cause, detail string) (Scar, error) {
	if strings.TrimSpace(districtID) == "" {
		return Scar{}, ErrEmptyDistrictID
	}
	if !validCauses[cause] {
		return Scar{}, ErrUnknownCause
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	s := Scar{
		ID:         fmt.Sprintf("scar-%d", r.seq),
		DistrictID: districtID,
		Cause:      cause,
		At:         time.Now().UTC(),
		Detail:     detail,
	}
	r.scars = append(r.scars, s)
	r.byDistrict[districtID] = append(r.byDistrict[districtID], s)
	return s, nil
}

// ForDistrict returns all scars for the given district, oldest first.
func (r *Registry) ForDistrict(districtID string) []Scar {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Scar, len(r.byDistrict[districtID]))
	copy(out, r.byDistrict[districtID])
	return out
}

// All returns all scars across all districts, oldest first.
func (r *Registry) All() []Scar {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Scar, len(r.scars))
	copy(out, r.scars)
	return out
}

// VisibilityBonus returns the cumulative Watcher visibility bonus for a district.
// Caller adds this to the neighborhood base multiplier.
func (r *Registry) VisibilityBonus(districtID string) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return float64(len(r.byDistrict[districtID])) * VisibilityBonusPerScar
}

// Count returns the total scar count for a district.
func (r *Registry) Count(districtID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byDistrict[districtID])
}

// MUDScarsCommand renders the scars list for MUD output.
// If districtID is empty, returns all scars.
func (r *Registry) MUDScarsCommand(districtID string) string {
	var scars []Scar
	if districtID == "" {
		scars = r.All()
	} else {
		scars = r.ForDistrict(districtID)
	}
	if len(scars) == 0 {
		if districtID != "" {
			return fmt.Sprintf("No scars recorded for district %s.", districtID)
		}
		return "No scars recorded."
	}
	var sb strings.Builder
	if districtID != "" {
		sb.WriteString(fmt.Sprintf("Scars for %s (%d total, visibility bonus +%.0f%%):\n",
			districtID, len(scars), r.VisibilityBonus(districtID)*100))
	} else {
		sb.WriteString(fmt.Sprintf("All scars (%d total):\n", len(scars)))
	}
	for _, s := range scars {
		sb.WriteString("  " + s.String() + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// RemoveLast removes the most recently appended scar for a district.
// Used by ScarBurn counterplay in K9 Merciless Operation.
// Returns the removed scar and true if one existed; returns zero value and false otherwise.
func (r *Registry) RemoveLast(districtID string) (Scar, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	district := r.byDistrict[districtID]
	if len(district) == 0 {
		return Scar{}, false
	}
	removed := district[len(district)-1]
	r.byDistrict[districtID] = district[:len(district)-1]

	// Remove from the global slice too.
	for i := len(r.scars) - 1; i >= 0; i-- {
		if r.scars[i].ID == removed.ID {
			r.scars = append(r.scars[:i], r.scars[i+1:]...)
			break
		}
	}
	return removed, true
}

// Package timeline implements the TRAPX multi-timeline branch system (S123-03).
//
// City state is keyed by branch_id. The default branch is "present".
// Migration Events (Rogue Swarms fired by Dragon ACT) create new branches.
//
// Branches hold snapshots of:
//   - Per-district Scar counts (myth counts from neighborhood)
//   - Per-district Watcher alertness delta from branch creation
//   - Branch creation trigger and episode timestamp
//
// The MUD 'timeline' command lists all branches.
// Portal travel lets a player select a branch (time travel between episode timestamps).
// Scars in branch A don't appear in branch B until a merge is filed.
package timeline

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

const DefaultBranch = "present"

// DistrictSnapshot captures one district's social state at a moment in time.
type DistrictSnapshot struct {
	DistrictID       string
	ScarCount        int
	AlertnessAtBirth float64 // watcher alertness when branch was cut
}

// Branch is one timeline snapshot.
type Branch struct {
	ID          string
	Parent      string // parent branch ID ("" for root)
	CreatedAt   time.Time
	Trigger     string // what event created this branch: "migration_event", "rogue_swarm", "manual"
	Description string
	Districts   map[string]DistrictSnapshot
}

func (b *Branch) String() string {
	return fmt.Sprintf("[%s] %-22s parent=%-10s trigger=%-20s scars=%d districts=%d  (%s)",
		b.CreatedAt.Format("15:04:05"), b.ID, b.Parent, b.Trigger, b.TotalScars(), len(b.Districts),
		b.Description)
}

// TotalScars returns the total scar count across all districts in this branch.
func (b *Branch) TotalScars() int {
	total := 0
	for _, d := range b.Districts {
		total += d.ScarCount
	}
	return total
}

// ScarDelta returns the scar count difference between this branch and a reference branch.
func (b *Branch) ScarDelta(ref *Branch) int {
	return b.TotalScars() - ref.TotalScars()
}

// Registry holds all branches.
type Registry struct {
	mu       sync.RWMutex
	branches map[string]*Branch
	order    []string // insertion order
}

// NewRegistry creates a Registry with the default "present" branch.
func NewRegistry() *Registry {
	r := &Registry{
		branches: make(map[string]*Branch),
	}
	r.branches[DefaultBranch] = &Branch{
		ID:          DefaultBranch,
		Parent:      "",
		CreatedAt:   time.Now(),
		Trigger:     "genesis",
		Description: "The city as it is now.",
		Districts:   make(map[string]DistrictSnapshot),
	}
	r.order = []string{DefaultBranch}
	return r
}

// SnapshotInput is passed by the caller to capture current city state when cutting a branch.
type SnapshotInput struct {
	DistrictID  string
	ScarCount   int
	Alertness   float64
}

// Cut creates a new branch from a parent, capturing a snapshot of district state.
func (r *Registry) Cut(id, parent, trigger, description string, now time.Time, districts []SnapshotInput) (*Branch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.branches[id]; exists {
		return nil, fmt.Errorf("branch %q already exists", id)
	}
	if _, ok := r.branches[parent]; !ok {
		return nil, fmt.Errorf("parent branch %q not found", parent)
	}
	b := &Branch{
		ID:          id,
		Parent:      parent,
		CreatedAt:   now,
		Trigger:     trigger,
		Description: description,
		Districts:   make(map[string]DistrictSnapshot, len(districts)),
	}
	for _, d := range districts {
		b.Districts[d.DistrictID] = DistrictSnapshot{
			DistrictID:       d.DistrictID,
			ScarCount:        d.ScarCount,
			AlertnessAtBirth: d.Alertness,
		}
	}
	r.branches[id] = b
	r.order = append(r.order, id)
	return b, nil
}

// Get returns a branch by ID, or nil if not found.
func (r *Registry) Get(id string) *Branch {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.branches[id]
}

// All returns all branches in insertion order.
func (r *Registry) All() []*Branch {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Branch, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.branches[id])
	}
	return out
}

// BranchCount returns the total number of branches.
func (r *Registry) BranchCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.branches)
}

// SummaryLines returns human-readable lines for the 'timeline' MUD command.
func (r *Registry) SummaryLines() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	present := r.branches[DefaultBranch]
	branches := make([]*Branch, 0, len(r.order))
	for _, id := range r.order {
		branches = append(branches, r.branches[id])
	}
	sort.Slice(branches, func(i, j int) bool {
		return branches[i].CreatedAt.Before(branches[j].CreatedAt)
	})
	out := make([]string, 0, len(branches)+1)
	out = append(out, fmt.Sprintf("[TIMELINE] %d branch(es) on record.", len(branches)))
	for _, b := range branches {
		scarDelta := ""
		if b.ID != DefaultBranch && present != nil {
			delta := b.TotalScars() - present.TotalScars()
			if delta > 0 {
				scarDelta = fmt.Sprintf(" Δscars=+%d", delta)
			} else if delta < 0 {
				scarDelta = fmt.Sprintf(" Δscars=%d", delta)
			}
		}
		marker := "  "
		if b.ID == DefaultBranch {
			marker = "► "
		}
		out = append(out, fmt.Sprintf("%s%s  %s%s", marker, b.CreatedAt.Format("2006-01-02 15:04"), b.String(), scarDelta))
	}
	return out
}

// DistrictHasConflict returns true if a district's scar count differs between two branches.
// Used to trigger dual-state voxel rendering at portal transitions.
func (r *Registry) DistrictHasConflict(districtID, branchA, branchB string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ba, ok := r.branches[branchA]
	if !ok {
		return false
	}
	bb, ok2 := r.branches[branchB]
	if !ok2 {
		return false
	}
	da, aOK := ba.Districts[districtID]
	db, bOK := bb.Districts[districtID]
	if !aOK || !bOK {
		return false
	}
	return da.ScarCount != db.ScarCount
}

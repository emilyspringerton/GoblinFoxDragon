// Package factionwar implements TRAPX scheduled district conflict events.
//
// Every 72h a district enters FactionConflict state. During conflict:
//   - Enforcement alertness multiplied ×1.5
//   - K9 battery drain doubled
//   - FieldOffice FlipWindows forced open
//
// Conflict resolves when one faction controls 3+ FOs in the district,
// or after 24h with the dominant faction declared winner.
package factionwar

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Phase is the lifecycle state of a faction war.
type Phase int

const (
	PhaseIdle    Phase = iota // no war active
	PhaseActive               // war in progress
	PhaseResolve              // grace window — winner computing
)

// War represents a single active faction conflict over a district.
type War struct {
	ID         string
	DistrictID string
	StartedAt  time.Time
	EndsAt     time.Time // conflict auto-expires after WarDuration
	Phase      Phase
	FOsHeld    map[string]int // faction name → FO count in district
	Winner     string         // set on resolve
}

const (
	WarDuration        = 24 * time.Hour
	WarCyclePeriod     = 72 * time.Hour
	AlertnessMultiply  = 1.5
	K9DrainMultiply    = 2.0
	MinFOsToWin        = 3
)

// Engine manages faction wars across districts.
type Engine struct {
	mu        sync.RWMutex
	wars      map[string]*War // districtID → active war (nil if idle)
	lastCycle map[string]time.Time
	districts []string
	rng       *rand.Rand
	clock     func() time.Time
}

// NewEngine creates a faction war engine managing the given district IDs.
func NewEngine(districtIDs []string) *Engine {
	return &Engine{
		wars:      make(map[string]*War),
		lastCycle: make(map[string]time.Time),
		districts: districtIDs,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
		clock:     func() time.Time { return time.Now().UTC() },
	}
}

// Tick advances the war engine. Call once per game tick (1 Hz).
// Returns newly triggered wars and newly resolved wars for broadcast.
func (e *Engine) Tick() (triggered []War, resolved []War) {
	now := e.clock()
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, distID := range e.districts {
		war, active := e.wars[distID]
		if !active || war == nil {
			// Check if 72h cycle has elapsed since last war in this district.
			last := e.lastCycle[distID]
			if now.Sub(last) >= WarCyclePeriod {
				// Randomly pick one district per cycle to activate (33% chance per district per check).
				if e.rng.Float64() < 0.33 {
					w := e.startWar(distID, now)
					e.wars[distID] = w
					triggered = append(triggered, *w)
				}
			}
			continue
		}

		// Check for resolution: time expired or dominant faction has 3+ FOs.
		if now.After(war.EndsAt) || e.checkWinCondition(war) {
			resolved = append(resolved, *war)
			war.Phase = PhaseResolve
			e.lastCycle[distID] = now
			delete(e.wars, distID)
		}
	}
	return
}

func (e *Engine) startWar(districtID string, now time.Time) *War {
	return &War{
		ID:         fmt.Sprintf("war-%s-%d", districtID, now.Unix()),
		DistrictID: districtID,
		StartedAt:  now,
		EndsAt:     now.Add(WarDuration),
		Phase:      PhaseActive,
		FOsHeld:    map[string]int{"The Frequency": 0, "The Bloc": 0, "Procurement Houses": 0},
	}
}

func (e *Engine) checkWinCondition(w *War) bool {
	for _, count := range w.FOsHeld {
		if count >= MinFOsToWin {
			return true
		}
	}
	return false
}

// UpdateFOCount updates the FO count held by a faction in a district's active war.
// Called by the MUD when a FieldOffice changes hands.
func (e *Engine) UpdateFOCount(districtID, faction string, count int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if w, ok := e.wars[districtID]; ok && w != nil {
		w.FOsHeld[faction] = count
	}
}

// ActiveWar returns the active war for a district, or nil if none.
func (e *Engine) ActiveWar(districtID string) *War {
	e.mu.RLock()
	defer e.mu.RUnlock()
	w := e.wars[districtID]
	if w == nil {
		return nil
	}
	cp := *w
	return &cp
}

// IsAtWar returns true if the district is currently in a faction conflict.
func (e *Engine) IsAtWar(districtID string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	w := e.wars[districtID]
	return w != nil && w.Phase == PhaseActive
}

// AlertnessMultiplier returns the enforcement alertness multiplier for a district.
// Returns AlertnessMultiply during an active war, 1.0 otherwise.
func (e *Engine) AlertnessMultiplier(districtID string) float64 {
	if e.IsAtWar(districtID) {
		return AlertnessMultiply
	}
	return 1.0
}

// K9DrainMultiplier returns the K9 battery drain multiplier for a district.
// Returns K9DrainMultiply during an active war, 1.0 otherwise.
func (e *Engine) K9DrainMultiplier(districtID string) float64 {
	if e.IsAtWar(districtID) {
		return K9DrainMultiply
	}
	return 1.0
}

// AllWars returns a snapshot of all currently active wars.
func (e *Engine) AllWars() []War {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]War, 0, len(e.wars))
	for _, w := range e.wars {
		if w != nil {
			out = append(out, *w)
		}
	}
	return out
}

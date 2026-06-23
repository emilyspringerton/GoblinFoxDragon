// Package worldcrisis implements the DragonsNShit World Crisis server-authoritative phase machine.
//
// See GoblinFoxDragon/docs2/specs/WORLD_CRISIS_VS0.md for full acceptance criteria.
//
// # Phase sequence
//
//	OMENS → BURROW → EMERGENCE → SPLIT_WAR → FINAL_WINDOW → RESOLUTION
//
// # LEY_INTEGRITY
//
// A global meter (0–100) that decays automatically during active phases.
// Players stabilize it via objectives. If it reaches 0, the event fails.
package worldcrisis

import (
	"errors"
	"sync"
	"time"
)

// Phase is a named stage of the World Crisis.
type Phase string

const (
	PhaseIdle        Phase = "idle"
	PhaseOmens       Phase = "omens"        // 60s — telegraphing / pre-warn
	PhaseBurrow      Phase = "burrow"        // 120s — passive threat escalation
	PhaseEmergence   Phase = "emergence"     // 180s — combat window opens
	PhaseSplitWar    Phase = "split_war"     // 300s — two sub-boss zones
	PhaseFinalWindow Phase = "final_window"  // 300s — requires concurrent objectives
	PhaseResolution  Phase = "resolution"    // terminal — victory or failure
)

// Outcome records the final state after resolution.
type Outcome string

const (
	OutcomeNone    Outcome = ""
	OutcomeVictory Outcome = "victory"
	OutcomeFailure Outcome = "failure"
)

// DefaultPhaseDurations defines how long each phase lasts (server-configurable).
var DefaultPhaseDurations = map[Phase]time.Duration{
	PhaseOmens:       60 * time.Second,
	PhaseBurrow:      120 * time.Second,
	PhaseEmergence:   180 * time.Second,
	PhaseSplitWar:    300 * time.Second,
	PhaseFinalWindow: 300 * time.Second,
}

const (
	LeyMax         = 100
	LeyMin         = 0
	LeyDecayPerTick = 1    // per decay interval
	DecayInterval  = 5 * time.Second

	// Concurrent objectives required to pass the Final Window gate.
	RequiredConcurrentObjectives = 3
)

var (
	ErrAlreadyActive  = errors.New("worldcrisis: event already active")
	ErrNotActive      = errors.New("worldcrisis: event not active")
	ErrAlreadyResolved = errors.New("worldcrisis: event already resolved")
	ErrInvalidPhase   = errors.New("worldcrisis: invalid phase")
)

// ObjectiveType categorises objective contributions (B1 multi-objective gate).
type ObjectiveType string

const (
	ObjectiveAnchor   ObjectiveType = "anchor"   // keep pylons alive
	ObjectiveRitual   ObjectiveType = "ritual"   // stabilize nodes
	ObjectiveIntercept ObjectiveType = "intercept" // stop add wave
)

// Crisis is the server-side world crisis state machine.
// It is safe for concurrent use.
type Crisis struct {
	mu sync.RWMutex

	Phase          Phase
	LeyIntegrity   int
	Outcome        Outcome
	PhaseDeadline  time.Time // when current phase auto-advances
	PhaseStartedAt time.Time
	lastDecay      time.Time

	// Objective completion state for Final Window gate.
	// Key = ObjectiveType, value = completion time.
	objectiveTimes map[ObjectiveType]time.Time

	// Concurrent objective window for the final gate (5 min).
	ConcurrentGate time.Duration

	// Config overrides (zero = use DefaultPhaseDurations)
	durations map[Phase]time.Duration

	// EventID is the IDUNA world_event ID, set on Start.
	EventID string
}

// New creates a new idle Crisis with default config.
func New() *Crisis {
	return &Crisis{
		Phase:          PhaseIdle,
		LeyIntegrity:   LeyMax,
		ConcurrentGate: 5 * time.Minute,
		durations:      map[Phase]time.Duration{},
	}
}

// Start activates the crisis, setting phase to Omens.
// eventID is the IDUNA world_event ID created by the caller.
func (c *Crisis) Start(now time.Time, eventID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Phase != PhaseIdle {
		return ErrAlreadyActive
	}
	c.EventID = eventID
	c.LeyIntegrity = LeyMax
	c.Outcome = OutcomeNone
	c.objectiveTimes = map[ObjectiveType]time.Time{}
	c.lastDecay = now
	c.setPhase(PhaseOmens, now)
	return nil
}

// phaseDuration returns the configured or default duration for a phase.
func (c *Crisis) phaseDuration(p Phase) time.Duration {
	if d, ok := c.durations[p]; ok {
		return d
	}
	if d, ok := DefaultPhaseDurations[p]; ok {
		return d
	}
	return time.Hour // fallback for unmapped phases
}

// setPhase transitions to p, updating PhaseDeadline. Must hold mu.Lock.
func (c *Crisis) setPhase(p Phase, now time.Time) {
	c.Phase = p
	c.PhaseStartedAt = now
	if d, ok := DefaultPhaseDurations[p]; ok {
		if d2, ok2 := c.durations[p]; ok2 {
			d = d2
		}
		c.PhaseDeadline = now.Add(d)
	}
}

// Tick advances the state machine based on current time.
// Returns (phaseChanged, oldPhase, newPhase).
// Should be called periodically (e.g., every 250ms) by the server goroutine.
func (c *Crisis) Tick(now time.Time) (phaseChanged bool, old, new Phase) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Phase == PhaseIdle || c.Phase == PhaseResolution {
		return false, c.Phase, c.Phase
	}

	// Decay LEY_INTEGRITY at fixed interval.
	if now.Sub(c.lastDecay) >= DecayInterval {
		c.LeyIntegrity -= LeyDecayPerTick
		if c.LeyIntegrity < LeyMin {
			c.LeyIntegrity = LeyMin
		}
		c.lastDecay = now
	}

	// If LEY_INTEGRITY hits 0 at any active phase, fail the event.
	if c.LeyIntegrity <= 0 && c.Phase != PhaseResolution {
		old := c.Phase
		c.Outcome = OutcomeFailure
		c.Phase = PhaseResolution
		c.PhaseStartedAt = now
		return true, old, PhaseResolution
	}

	// Auto-advance on timer.
	if !c.PhaseDeadline.IsZero() && now.After(c.PhaseDeadline) {
		old := c.Phase
		next := c.nextPhase()
		if next == PhaseResolution {
			c.Outcome = c.computeOutcome()
		}
		c.setPhase(next, now)
		return true, old, next
	}

	return false, c.Phase, c.Phase
}

// nextPhase returns the phase that follows the current one.
func (c *Crisis) nextPhase() Phase {
	switch c.Phase {
	case PhaseOmens:
		return PhaseBurrow
	case PhaseBurrow:
		return PhaseEmergence
	case PhaseEmergence:
		return PhaseSplitWar
	case PhaseSplitWar:
		return PhaseFinalWindow
	case PhaseFinalWindow:
		return PhaseResolution
	default:
		return PhaseResolution
	}
}

// computeOutcome decides victory/failure at the end of FinalWindow.
// Victory if LEY_INTEGRITY > 0 AND concurrent objectives gate was met.
func (c *Crisis) computeOutcome() Outcome {
	if c.LeyIntegrity <= 0 {
		return OutcomeFailure
	}
	if !c.concurrentGateMet(time.Now()) {
		return OutcomeFailure
	}
	return OutcomeVictory
}

// concurrentGateMet returns true if ≥3 distinct objective types were completed
// within a ConcurrentGate window of each other.
func (c *Crisis) concurrentGateMet(now time.Time) bool {
	if len(c.objectiveTimes) < RequiredConcurrentObjectives {
		return false
	}
	// All completion times must fall within the ConcurrentGate window.
	var earliest, latest time.Time
	for _, t := range c.objectiveTimes {
		if earliest.IsZero() || t.Before(earliest) {
			earliest = t
		}
		if latest.IsZero() || t.After(latest) {
			latest = t
		}
	}
	return latest.Sub(earliest) <= c.ConcurrentGate
}

// CompleteObjective records an objective completion, stabilising LEY_INTEGRITY
// and advancing the concurrent gate tracking.
func (c *Crisis) CompleteObjective(obj ObjectiveType, stabilizeAmount int, now time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Phase == PhaseIdle || c.Phase == PhaseResolution {
		return ErrNotActive
	}
	c.objectiveTimes[obj] = now
	c.LeyIntegrity += stabilizeAmount
	if c.LeyIntegrity > LeyMax {
		c.LeyIntegrity = LeyMax
	}
	return nil
}

// LeyDecay manually decays LEY_INTEGRITY (e.g., when objectives fail).
// No-op if already at minimum.
func (c *Crisis) LeyDecay(amount int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LeyIntegrity -= amount
	if c.LeyIntegrity < LeyMin {
		c.LeyIntegrity = LeyMin
	}
}

// Status returns a snapshot of the current crisis state.
type Status struct {
	Phase         Phase
	LeyIntegrity  int
	Outcome       Outcome
	PhaseDeadline time.Time
	EventID       string
	Objectives    map[ObjectiveType]time.Time
}

// Status returns a point-in-time snapshot of the crisis.
func (c *Crisis) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	obj := make(map[ObjectiveType]time.Time, len(c.objectiveTimes))
	for k, v := range c.objectiveTimes {
		obj[k] = v
	}
	return Status{
		Phase:         c.Phase,
		LeyIntegrity:  c.LeyIntegrity,
		Outcome:       c.Outcome,
		PhaseDeadline: c.PhaseDeadline,
		EventID:       c.EventID,
		Objectives:    obj,
	}
}

// Reset returns the crisis to idle, clearing all state.
func (c *Crisis) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Phase = PhaseIdle
	c.LeyIntegrity = LeyMax
	c.Outcome = OutcomeNone
	c.EventID = ""
	c.objectiveTimes = map[ObjectiveType]time.Time{}
}

// SetPhaseDuration overrides the duration for a phase (for testing / config).
func (c *Crisis) SetPhaseDuration(p Phase, d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.durations[p] = d
}

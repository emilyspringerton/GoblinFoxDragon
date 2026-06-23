// Package weather implements the DragonsNShit MUD weather system.
//
// Weather phases cycle over time. Each phase has a duration range and
// gameplay modifiers (mob damage, BST tame bonus).
package weather

import (
	"math/rand"
	"time"
)

// Phase is a weather state.
type Phase string

const (
	PhaseNone     Phase = "None" // initial before first Tick
	PhaseClear    Phase = "Clear"
	PhaseOvercast Phase = "Overcast"
	PhaseRain     Phase = "Rain"
	PhaseStorm    Phase = "Storm"
)

// Modifiers returns the gameplay modifiers for a weather phase.
type Modifiers struct {
	MobDamageBonus  int     // flat bonus to mob MeleeDamage
	BSTTameBonus    float64 // added to tame success chance (0.0–1.0)
}

// ModifiersFor returns the Modifiers for a given phase.
func ModifiersFor(p Phase) Modifiers {
	switch p {
	case PhaseStorm:
		return Modifiers{MobDamageBonus: 3, BSTTameBonus: 0.10}
	case PhaseRain:
		return Modifiers{MobDamageBonus: 1, BSTTameBonus: 0.05}
	default:
		return Modifiers{}
	}
}

// duration ranges by phase (min, max seconds).
var phaseDurations = map[Phase][2]int{
	PhaseClear:    {300, 720}, // 5–12 min
	PhaseOvercast: {180, 480}, // 3–8 min
	PhaseRain:     {120, 360}, // 2–6 min
	PhaseStorm:    {60, 240},  // 1–4 min
}

// phaseOrder defines the cycle order.
var phaseOrder = []Phase{PhaseClear, PhaseOvercast, PhaseRain, PhaseStorm}

// Engine drives weather phase transitions.
type Engine struct {
	phase      Phase
	phaseEnds  time.Time
	rng        *rand.Rand
	phaseIndex int // current index in phaseOrder
}

// New creates a new Engine starting in Clear weather.
func New() *Engine {
	e := &Engine{
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
		phaseIndex: 0,
		phase:      PhaseClear,
	}
	e.scheduleDuration(time.Now())
	return e
}

// newForTest creates an Engine with a seeded rng and explicit start time (for tests).
func newForTest(seed int64, now time.Time) *Engine {
	e := &Engine{
		rng:        rand.New(rand.NewSource(seed)),
		phaseIndex: 0,
		phase:      PhaseClear,
	}
	e.scheduleDuration(now)
	return e
}

func (e *Engine) scheduleDuration(now time.Time) {
	r := phaseDurations[e.phase]
	secs := r[0] + e.rng.Intn(r[1]-r[0]+1)
	e.phaseEnds = now.Add(time.Duration(secs) * time.Second)
}

// Phase returns the current weather phase.
func (e *Engine) Phase() Phase {
	return e.phase
}

// PhaseEnds returns when the current phase will expire.
func (e *Engine) PhaseEnds() time.Time {
	return e.phaseEnds
}

// Mods returns the current phase's gameplay modifiers.
func (e *Engine) Mods() Modifiers {
	return ModifiersFor(e.phase)
}

// Tick advances the weather engine. Returns (changed, oldPhase, newPhase).
// If the current phase has not expired, returns (false, current, current).
func (e *Engine) Tick(now time.Time) (changed bool, old, new Phase) {
	if now.Before(e.phaseEnds) {
		return false, e.phase, e.phase
	}
	old = e.phase
	e.phaseIndex = (e.phaseIndex + 1) % len(phaseOrder)
	e.phase = phaseOrder[e.phaseIndex]
	e.scheduleDuration(now)
	return true, old, e.phase
}

// ForcePhase sets the phase immediately (for tests / GM commands).
func (e *Engine) ForcePhase(p Phase, now time.Time) {
	e.phase = p
	for i, ph := range phaseOrder {
		if ph == p {
			e.phaseIndex = i
			break
		}
	}
	e.scheduleDuration(now)
}

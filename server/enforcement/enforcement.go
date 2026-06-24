// Package enforcement implements the TRAPX Enforcement Level state machine.
//
// Enforcement is the city's institutional response to high watcher alertness and
// low community trust. It runs as a 5-level state machine (0=Quiet to 4=Lockdown).
// Each level adds cop NPC density, FO defense difficulty, and attention decay resistance.
//
// Threshold evaluation (fed by watcher.WatcherState):
//   - alertness ≥ 65 + trust ≤ -20  → escalate
//   - alertness < 50 + trust > 0     → deescalate (slow)
//
// Enforcement level affects gameplay:
//   - Level 0 (Quiet):    no cops, FO easy to flip
//   - Level 1 (Alert):    1 cop NPC, FO defense +10%
//   - Level 2 (Active):   2 cop NPCs, FO defense +25%, K9 units eligible
//   - Level 3 (Heavy):    4 cop NPCs, FO defense +50%, K9 always deployed
//   - Level 4 (Lockdown): 8 cop NPCs, FO unclaim-able, Custody Lock immediate
package enforcement

import (
	"fmt"
	"time"
)

// Level is the Enforcement state for a district.
type Level int

const (
	LevelQuiet    Level = 0
	LevelAlert    Level = 1
	LevelActive   Level = 2
	LevelHeavy    Level = 3
	LevelLockdown Level = 4
)

var levelNames = map[Level]string{
	LevelQuiet:    "QUIET",
	LevelAlert:    "ALERT",
	LevelActive:   "ACTIVE",
	LevelHeavy:    "HEAVY",
	LevelLockdown: "LOCKDOWN",
}

// LevelName returns the display name for an Enforcement Level.
func LevelName(l Level) string {
	if s, ok := levelNames[l]; ok {
		return s
	}
	return "UNKNOWN"
}

const (
	// Thresholds for escalation (both must be met).
	EscalateAlertnessThreshold = 65.0
	EscalateTrustThreshold     = -20.0

	// Thresholds for de-escalation (both must be met for 2 consecutive ticks).
	DeescalateAlertnessThreshold = 50.0
	DeescalateTrustThreshold     = 0.0

	// CopDensity returns the number of cop NPCs spawnable at this level.
	// Used by mob spawner: 0, 1, 2, 4, 8.
)

// LevelEffects describes what changes at each enforcement level.
type LevelEffects struct {
	CopDensity    int     // number of cop NPCs to spawn
	FODefenseBonus float64 // multiplier on FO defense difficulty [1.0, 2.0]
	K9Eligible    bool    // whether K9 units can be deployed
	FOUnclaim     bool    // if true, FOs cannot be unclaimed by crews
	CustodyImmediate bool // if true, K9 latches trigger immediate custody lock
}

var levelEffects = map[Level]LevelEffects{
	LevelQuiet:    {CopDensity: 0, FODefenseBonus: 1.00, K9Eligible: false, FOUnclaim: false, CustodyImmediate: false},
	LevelAlert:    {CopDensity: 1, FODefenseBonus: 1.10, K9Eligible: false, FOUnclaim: false, CustodyImmediate: false},
	LevelActive:   {CopDensity: 2, FODefenseBonus: 1.25, K9Eligible: true, FOUnclaim: false, CustodyImmediate: false},
	LevelHeavy:    {CopDensity: 4, FODefenseBonus: 1.50, K9Eligible: true, FOUnclaim: false, CustodyImmediate: false},
	LevelLockdown: {CopDensity: 8, FODefenseBonus: 2.00, K9Eligible: true, FOUnclaim: true, CustodyImmediate: true},
}

// Effects returns the gameplay effects for a given Level.
func Effects(l Level) LevelEffects {
	if e, ok := levelEffects[l]; ok {
		return e
	}
	return levelEffects[LevelQuiet]
}

// Event is a single Enforcement state change.
type Event struct {
	At         time.Time
	DistrictID string
	Verb       string // ESCALATE, DEESCALATE, LOCKDOWN, LOCKDOWN_LIFT
	PrevLevel  Level
	NewLevel   Level
	Detail     string
}

func (e Event) String() string {
	return fmt.Sprintf("[%s] enf/%s %s %s→%s %s",
		e.At.Format("15:04:05"), e.DistrictID, e.Verb,
		LevelName(e.PrevLevel), LevelName(e.NewLevel), e.Detail)
}

// State is the per-district Enforcement state machine.
type State struct {
	DistrictID     string
	Level          Level
	deescTicks     int // consecutive ticks below de-escalation threshold
}

// NewState creates a new Enforcement State at LevelQuiet.
func NewState(districtID string) *State {
	return &State{DistrictID: districtID, Level: LevelQuiet}
}

// Evaluate processes current watcher signals and returns any state-change events.
// alertness and trust come from watcher.WatcherState.
func (s *State) Evaluate(alertness, trust float64, now time.Time) []Event {
	var events []Event

	shouldEscalate := alertness >= EscalateAlertnessThreshold && trust <= EscalateTrustThreshold
	shouldDeescalate := alertness < DeescalateAlertnessThreshold && trust > DeescalateTrustThreshold

	if shouldEscalate && s.Level < LevelLockdown {
		prev := s.Level
		s.Level++
		s.deescTicks = 0
		verb := "ESCALATE"
		if s.Level == LevelLockdown {
			verb = "LOCKDOWN"
		}
		events = append(events, Event{
			At: now, DistrictID: s.DistrictID, Verb: verb,
			PrevLevel: prev, NewLevel: s.Level,
			Detail: fmt.Sprintf("alertness=%.0f trust=%.0f", alertness, trust),
		})
	} else if shouldDeescalate && s.Level > LevelQuiet {
		s.deescTicks++
		if s.deescTicks >= 2 {
			prev := s.Level
			s.Level--
			s.deescTicks = 0
			verb := "DEESCALATE"
			if prev == LevelLockdown {
				verb = "LOCKDOWN_LIFT"
			}
			events = append(events, Event{
				At: now, DistrictID: s.DistrictID, Verb: verb,
				PrevLevel: prev, NewLevel: s.Level,
				Detail: fmt.Sprintf("alertness=%.0f trust=%.0f", alertness, trust),
			})
		}
	} else {
		s.deescTicks = 0
	}

	return events
}

// Effects returns the current level's gameplay effects.
func (s *State) Effects() LevelEffects {
	return Effects(s.Level)
}

// Summary returns a one-line status for the MUD district command.
func (s *State) Summary() string {
	e := s.Effects()
	return fmt.Sprintf("%s enf=%s cops=%d fo_defense=x%.2f k9=%v lockdown=%v",
		s.DistrictID, LevelName(s.Level), e.CopDensity, e.FODefenseBonus, e.K9Eligible, e.FOUnclaim)
}

// Registry holds all district Enforcement States.
type Registry struct {
	states map[string]*State
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{states: make(map[string]*State)}
}

// GetOrCreate returns the Enforcement State for a district, creating it if absent.
func (r *Registry) GetOrCreate(districtID string) *State {
	if s, ok := r.states[districtID]; ok {
		return s
	}
	s := NewState(districtID)
	r.states[districtID] = s
	return s
}

// Get returns the State for a district, or nil.
func (r *Registry) Get(districtID string) *State {
	return r.states[districtID]
}

// EvaluateAll runs Evaluate on all districts using provided alertness/trust maps.
func (r *Registry) EvaluateAll(alertness, trust map[string]float64, now time.Time) []Event {
	var out []Event
	for id, s := range r.states {
		a := alertness[id]
		tr := trust[id]
		out = append(out, s.Evaluate(a, tr, now)...)
	}
	return out
}

// DistrictPressure returns the vendor price multiplier for a district based on
// its current Enforcement Level: Held=1.0, Alert=1.10, Active=1.20, Heavy=1.40, Lockdown=1.60.
// Returns 1.0 for unknown districts (no enforcement state yet).
func (r *Registry) DistrictPressure(districtID string) float64 {
	s, ok := r.states[districtID]
	if !ok {
		return 1.0
	}
	switch s.Level {
	case LevelAlert:
		return 1.10
	case LevelActive:
		return 1.20
	case LevelHeavy:
		return 1.40
	case LevelLockdown:
		return 1.60
	default:
		return 1.0
	}
}

// LockdownDistricts returns all districts at LevelLockdown.
func (r *Registry) LockdownDistricts() []*State {
	var out []*State
	for _, s := range r.states {
		if s.Level == LevelLockdown {
			out = append(out, s)
		}
	}
	return out
}

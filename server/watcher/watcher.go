// Package watcher implements the TRAPX Watcher district social simulation.
//
// Watchers are neighbourhood eyes-and-ears — the community surveillance layer.
// They are not cops. They are residents, shop owners, and corner-post observers
// who generate informational pressure on the city. High Watcher Alertness means
// field operations become visible; Bias determines whether they push toward the
// Frequency, the Bloc, or Procurement Houses; Trust measures the district's
// relationship with field crews.
//
// The Watcher system feeds into enforcement.Enforcement thresholds:
// when alertness ≥ 65 and trust ≤ -20, enforcement escalates.
package watcher

import (
	"fmt"
	"math"
	"time"
)

const (
	AlertnessMax  = 100.0
	AlertnessMin  = 0.0
	TrustMax      = 100.0
	TrustMin      = -100.0
	BiasMax       = 100.0
	BiasMin       = -100.0

	// Passive alertness decay: 1 unit/minute when quiet.
	PassiveDecayRate = 1.0 / 60.0

	// Alertness thresholds that matter to enforcement.
	AlertEnforcementTrigger = 65.0
	AlertHighViz            = 80.0
	AlertSaturation         = 95.0

	// Trust thresholds.
	TrustEnforcementTrigger = -20.0
	TrustFriendly           = 40.0
)

// Bias represents which faction the district's Watcher network currently leans toward.
type Bias int

const (
	BiasNeutral    Bias = iota
	BiasFrequency       // The Frequency — information + code access
	BiasBloc            // The Bloc — social cohesion + enforcement resistance
	BiasProcurement     // Procurement Houses — economic leverage
)

var biasNames = map[Bias]string{
	BiasNeutral:    "Neutral",
	BiasFrequency:  "Frequency",
	BiasBloc:       "Bloc",
	BiasProcurement: "Procurement",
}

// BiastName returns the display name for a Bias.
func BiasName(b Bias) string {
	if s, ok := biasNames[b]; ok {
		return s
	}
	return "Unknown"
}

// Event is a single Watcher state change.
type Event struct {
	At         time.Time
	DistrictID string
	Verb       string // ALERT_SPIKE, ALERT_DECAY, TRUST_SHIFT, BIAS_SHIFT, SATURATION, ENFORCEMENT_TRIGGER
	Delta      float64
	Value      float64
	Detail     string
}

func (e Event) String() string {
	return fmt.Sprintf("[%s] watcher/%s %s delta=%.1f value=%.1f %s",
		e.At.Format("15:04:05"), e.DistrictID, e.Verb, e.Delta, e.Value, e.Detail)
}

// WatcherState is the district-level social surveillance state.
type WatcherState struct {
	DistrictID string
	Alertness  float64 // [0, 100] — how visible field activity is
	Trust      float64 // [-100, 100] — crew relationship with residents
	Bias       Bias    // current factional lean of the Watcher network
	BiasScore  float64 // [-100, 100] — magnitude of factional pull

	EnforcementTriggered bool // true when alertness ≥ 65 AND trust ≤ -20
}

// NewState creates a fresh WatcherState for a district.
func NewState(districtID string) *WatcherState {
	return &WatcherState{
		DistrictID: districtID,
		Alertness:  20.0,
		Trust:      0.0,
		Bias:       BiasNeutral,
		BiasScore:  0.0,
	}
}

// AddAlertness raises alertness by delta (clamped to [0,100]).
// Returns emitted events (ALERT_SPIKE, SATURATION, ENFORCEMENT_TRIGGER if thresholds crossed).
func (w *WatcherState) AddAlertness(delta float64, now time.Time, reason string) []Event {
	prev := w.Alertness
	w.Alertness = clamp(w.Alertness+delta, AlertnessMin, AlertnessMax)
	var events []Event
	events = append(events, Event{
		At: now, DistrictID: w.DistrictID, Verb: "ALERT_SPIKE",
		Delta: delta, Value: w.Alertness, Detail: reason,
	})
	if prev < AlertSaturation && w.Alertness >= AlertSaturation {
		events = append(events, Event{
			At: now, DistrictID: w.DistrictID, Verb: "SATURATION",
			Delta: 0, Value: w.Alertness, Detail: "watcher network saturated — all field activity visible",
		})
	}
	events = append(events, w.checkEnforcementTrigger(now)...)
	return events
}

// DecayAlertness reduces alertness passively.
func (w *WatcherState) DecayAlertness(dt time.Duration, now time.Time) []Event {
	if w.Alertness <= AlertnessMin {
		return nil
	}
	decayDelta := PassiveDecayRate * dt.Seconds()
	prev := w.Alertness
	w.Alertness = clamp(w.Alertness-decayDelta, AlertnessMin, AlertnessMax)
	if prev >= AlertEnforcementTrigger && w.Alertness < AlertEnforcementTrigger {
		w.EnforcementTriggered = false
		return []Event{{
			At: now, DistrictID: w.DistrictID, Verb: "ALERT_DECAY",
			Delta: -decayDelta, Value: w.Alertness, Detail: "alertness below enforcement threshold",
		}}
	}
	return nil
}

// ShiftTrust modifies trust by delta (clamped to [-100, 100]).
// Positive = crew did something good for the district.
// Negative = extraction/enforcement called in/property damage.
func (w *WatcherState) ShiftTrust(delta float64, now time.Time, reason string) []Event {
	w.Trust = clamp(w.Trust+delta, TrustMin, TrustMax)
	events := []Event{{
		At: now, DistrictID: w.DistrictID, Verb: "TRUST_SHIFT",
		Delta: delta, Value: w.Trust, Detail: reason,
	}}
	events = append(events, w.checkEnforcementTrigger(now)...)
	return events
}

// ShiftBias nudges the district's factional lean by delta (positive = toward biasTarget, negative = away).
func (w *WatcherState) ShiftBias(biasTarget Bias, delta float64, now time.Time, reason string) []Event {
	if biasTarget == w.Bias {
		w.BiasScore = clamp(w.BiasScore+delta, BiasMin, BiasMax)
	} else {
		w.BiasScore = clamp(w.BiasScore-delta, BiasMin, BiasMax)
		if w.BiasScore < 0 {
			w.Bias = biasTarget
			w.BiasScore = -w.BiasScore
		}
	}
	return []Event{{
		At: now, DistrictID: w.DistrictID, Verb: "BIAS_SHIFT",
		Delta: delta, Value: w.BiasScore,
		Detail: fmt.Sprintf("lean=%s reason=%s", BiasName(w.Bias), reason),
	}}
}

// Tick runs one time step. Decays alertness passively.
func (w *WatcherState) Tick(dt time.Duration, now time.Time) []Event {
	return w.DecayAlertness(dt, now)
}

// IsEnforcementHot returns true when both trigger conditions are met.
func (w *WatcherState) IsEnforcementHot() bool {
	return w.Alertness >= AlertEnforcementTrigger && w.Trust <= TrustEnforcementTrigger
}

// Summary returns a one-line district status for the MUD.
func (w *WatcherState) Summary() string {
	enfFlag := ""
	if w.EnforcementTriggered {
		enfFlag = " [ENF HOT]"
	}
	return fmt.Sprintf("%s | alert=%.0f trust=%.0f bias=%s%s",
		w.DistrictID, w.Alertness, w.Trust, BiasName(w.Bias), enfFlag)
}

func (w *WatcherState) checkEnforcementTrigger(now time.Time) []Event {
	hot := w.IsEnforcementHot()
	if hot && !w.EnforcementTriggered {
		w.EnforcementTriggered = true
		return []Event{{
			At: now, DistrictID: w.DistrictID, Verb: "ENFORCEMENT_TRIGGER",
			Delta: 0, Value: w.Alertness,
			Detail: fmt.Sprintf("alertness=%.0f trust=%.0f — enforcement escalation imminent", w.Alertness, w.Trust),
		}}
	}
	if !hot && w.EnforcementTriggered {
		w.EnforcementTriggered = false
	}
	return nil
}

// Registry holds all district WatcherStates.
type Registry struct {
	states map[string]*WatcherState
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{states: make(map[string]*WatcherState)}
}

// GetOrCreate returns the WatcherState for a district, creating it if absent.
func (r *Registry) GetOrCreate(districtID string) *WatcherState {
	if s, ok := r.states[districtID]; ok {
		return s
	}
	s := NewState(districtID)
	r.states[districtID] = s
	return s
}

// Get returns the WatcherState for a district, or nil if absent.
func (r *Registry) Get(districtID string) *WatcherState {
	return r.states[districtID]
}

// TickAll advances all districts by dt.
func (r *Registry) TickAll(dt time.Duration, now time.Time) []Event {
	var out []Event
	for _, s := range r.states {
		out = append(out, s.Tick(dt, now)...)
	}
	return out
}

// HotDistricts returns all districts where enforcement is triggered.
func (r *Registry) HotDistricts() []*WatcherState {
	var out []*WatcherState
	for _, s := range r.states {
		if s.EnforcementTriggered {
			out = append(out, s)
		}
	}
	return out
}

// AlertnessByDistrict returns a map of districtID → alertness for batch reads (e.g. trapxapi).
func (r *Registry) AlertnessByDistrict() map[string]float64 {
	out := make(map[string]float64, len(r.states))
	for id, s := range r.states {
		out[id] = s.Alertness
	}
	return out
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

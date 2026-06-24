// Package integrity implements the TRAPX Control Integrity + Rogue Swarm system.
//
// Control Integrity (CI) is a per-district metric (0.0–1.0) that represents the
// cohesion of K9 operator control. High dog counts, jammer use, and rapid FO flips
// all erode CI. Quiet periods, clean audits, and Bird Corrections restore it.
//
// # Rogue Swarm
//
// When CI falls to or below RogueThreshold, a RogueSwarmEvent fires. Dogs in the
// district detach from their operators and begin acting autonomously. Field Offices
// enter Containment Mode. The district remains Rogue until three containment
// objectives are completed (tracked by the server/fieldoffice package).
//
// Once containment ends, a Scar is written — a permanent district memory artifact.
// Scars persist in SessionMemory across game sessions.
//
// # Recovery
//
// Recovery is passive (PassiveRecoveryRate/s) and can be boosted by:
//
//   - CleanAudit: +AuditBonus CI (granted by the server when an Oversight Sect
//     audit completes with no violations)
//   - BirdCorrection: +BirdBonus CI (granted by an in-world Bird Correction event,
//     which is issued by the Federal Oversight layer)
package integrity

import (
	"fmt"
	"math"
	"time"
)

const (
	CIMax               = 1.0
	CIMin               = 0.0
	RogueThreshold      = 0.15 // CI at or below this triggers a Rogue Swarm
	PassiveRecoveryRate = 0.001 // CI/s gained at rest (very slow)
	DogCountDecayBase   = 0.002 // CI/s lost per dog (linear baseline)
	DogCountDecayPower  = 1.5   // superlinear factor: dogs^power
	JammerDecayPerUse   = 0.05  // CI lost per jammer activation
	FlipDecayPerFlip    = 0.08  // CI lost per FO flip in this district
	AuditBonus          = 0.10  // CI gained from a clean audit
	BirdBonus           = 0.15  // CI gained from a Bird Correction
)

// RogueSwarmEvent is emitted when Control Integrity collapses below RogueThreshold.
type RogueSwarmEvent struct {
	At         time.Time
	DistrictID string
	CI         float64 // CI value at the moment of collapse
	Detail     string
}

func (r RogueSwarmEvent) String() string {
	return fmt.Sprintf("[%s] ROGUE_SWARM district=%s CI=%.3f %s",
		r.At.Format("15:04:05"), r.DistrictID, r.CI, r.Detail)
}

// Scar is a permanent artifact written when a Rogue Swarm is contained.
type Scar struct {
	At         time.Time
	DistrictID string
	RogueCI    float64 // CI at the moment the swarm triggered
	Detail     string
}

func (s Scar) String() string {
	return fmt.Sprintf("SCAR district=%s roguedAt=%.3f written=%s %s",
		s.DistrictID, s.RogueCI, s.At.Format("2006-01-02 15:04:05"), s.Detail)
}

// StateEvent is a generic event emitted by IntegrityState transitions.
type StateEvent struct {
	At         time.Time
	DistrictID string
	Verb       string  // CI_TICK, ROGUE_SWARM, CONTAINMENT_END, SCAR_WRITTEN, CI_RESTORED
	CI         float64
	Detail     string
}

func (e StateEvent) String() string {
	return fmt.Sprintf("[%s] %s district=%s CI=%.3f %s",
		e.At.Format("15:04:05"), e.Verb, e.DistrictID, e.CI, e.Detail)
}

// IntegrityState tracks Control Integrity for a single district.
// Not goroutine-safe; callers synchronise via the server tick loop.
type IntegrityState struct {
	DistrictID string
	CI         float64 // current Control Integrity [0.0, 1.0]

	IsRogue    bool    // true while a Rogue Swarm is active
	RogueCI    float64 // CI value at time of Rogue trigger
	Scars      []Scar  // permanent scar history for this district

	// containment progress (set by fieldoffice package via SetContainmentProgress)
	ContainmentObjectives int
	ContainmentRequired   int
}

// NewState creates a district with full Control Integrity.
func NewState(districtID string) *IntegrityState {
	return &IntegrityState{
		DistrictID:          districtID,
		CI:                  CIMax,
		ContainmentRequired: 3,
	}
}

// Tick advances the district by dt.
// nDogs is the count of active dogs currently in the district.
// Returns all events generated (may include ROGUE_SWARM).
func (s *IntegrityState) Tick(dt time.Duration, nDogs int, now time.Time) []StateEvent {
	if s.IsRogue {
		return nil // no further CI changes during active Rogue Swarm
	}

	secs := dt.Seconds()

	// Passive recovery.
	s.CI = math.Min(s.CI+PassiveRecoveryRate*secs, CIMax)

	// Dog-count decay (superlinear).
	if nDogs > 0 {
		decay := DogCountDecayBase * math.Pow(float64(nDogs), DogCountDecayPower) * secs
		s.CI = math.Max(s.CI-decay, CIMin)
	}

	return s.checkRogue(now, fmt.Sprintf("dogs=%d", nDogs))
}

// RecordJammerUse applies CI loss from a jammer activation.
func (s *IntegrityState) RecordJammerUse(now time.Time) []StateEvent {
	if s.IsRogue {
		return nil
	}
	s.CI = math.Max(s.CI-JammerDecayPerUse, CIMin)
	return s.checkRogue(now, "jammer")
}

// RecordFlip applies CI loss from an FO flip in this district.
func (s *IntegrityState) RecordFlip(now time.Time) []StateEvent {
	if s.IsRogue {
		return nil
	}
	s.CI = math.Max(s.CI-FlipDecayPerFlip, CIMin)
	return s.checkRogue(now, "fo-flip")
}

// CleanAudit boosts CI by AuditBonus.
func (s *IntegrityState) CleanAudit(now time.Time) StateEvent {
	s.CI = math.Min(s.CI+AuditBonus, CIMax)
	return StateEvent{At: now, DistrictID: s.DistrictID, Verb: "CI_RESTORED",
		CI: s.CI, Detail: fmt.Sprintf("clean-audit +%.2f", AuditBonus)}
}

// BirdCorrection boosts CI by BirdBonus.
func (s *IntegrityState) BirdCorrection(now time.Time) StateEvent {
	s.CI = math.Min(s.CI+BirdBonus, CIMax)
	return StateEvent{At: now, DistrictID: s.DistrictID, Verb: "CI_RESTORED",
		CI: s.CI, Detail: fmt.Sprintf("bird-correction +%.2f", BirdBonus)}
}

// checkRogue fires a ROGUE_SWARM event if CI just dropped at or below RogueThreshold.
func (s *IntegrityState) checkRogue(now time.Time, cause string) []StateEvent {
	if !s.IsRogue && s.CI <= RogueThreshold {
		s.IsRogue = true
		s.RogueCI = s.CI
		return []StateEvent{{
			At: now, DistrictID: s.DistrictID, Verb: "ROGUE_SWARM",
			CI:     s.CI,
			Detail: fmt.Sprintf("cause=%s; dogs detach from operators; FOs enter Containment", cause),
		}}
	}
	return nil
}

// SetContainmentProgress updates the containment objective count.
// When objectives >= required, calls EndContainment.
// Returns (true, events) if containment ended.
func (s *IntegrityState) SetContainmentProgress(completed int, now time.Time) (bool, []StateEvent) {
	s.ContainmentObjectives = completed
	if s.IsRogue && completed >= s.ContainmentRequired {
		return true, s.EndContainment(now)
	}
	return false, nil
}

// EndContainment resolves the Rogue Swarm, writes a Scar, and restores CI.
// Returns events: CONTAINMENT_END and SCAR_WRITTEN.
func (s *IntegrityState) EndContainment(now time.Time) []StateEvent {
	scar := Scar{
		At: now, DistrictID: s.DistrictID, RogueCI: s.RogueCI,
		Detail: fmt.Sprintf("objectives=%d/%d", s.ContainmentObjectives, s.ContainmentRequired),
	}
	s.Scars = append(s.Scars, scar)
	s.IsRogue = false
	s.CI = math.Min(s.CI+BirdBonus, CIMax) // recovery bump on containment end
	s.ContainmentObjectives = 0

	return []StateEvent{
		{At: now, DistrictID: s.DistrictID, Verb: "CONTAINMENT_END",
			CI:     s.CI,
			Detail: fmt.Sprintf("rogue_ci=%.3f scars=%d", scar.RogueCI, len(s.Scars))},
		{At: now, DistrictID: s.DistrictID, Verb: "SCAR_WRITTEN",
			CI:     s.CI,
			Detail: scar.String()},
	}
}

// ScarCount returns the number of scars this district has accumulated.
func (s *IntegrityState) ScarCount() int { return len(s.Scars) }

// Registry manages Control Integrity states for all districts in the city.
type Registry struct {
	districts map[string]*IntegrityState
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{districts: make(map[string]*IntegrityState)}
}

// GetOrCreate returns the IntegrityState for districtID, creating one if needed.
func (r *Registry) GetOrCreate(districtID string) *IntegrityState {
	if s, ok := r.districts[districtID]; ok {
		return s
	}
	s := NewState(districtID)
	r.districts[districtID] = s
	return s
}

// Get returns the IntegrityState for districtID, or nil.
func (r *Registry) Get(districtID string) *IntegrityState {
	return r.districts[districtID]
}

// RogueDistricts returns all districts currently in Rogue Swarm state.
func (r *Registry) RogueDistricts() []*IntegrityState {
	var out []*IntegrityState
	for _, s := range r.districts {
		if s.IsRogue {
			out = append(out, s)
		}
	}
	return out
}

// TickAll advances all districts. nDogsPerDistrict maps districtID → active dog count.
func (r *Registry) TickAll(dt time.Duration, nDogsPerDistrict map[string]int, now time.Time) []StateEvent {
	var all []StateEvent
	for id, s := range r.districts {
		n := nDogsPerDistrict[id]
		all = append(all, s.Tick(dt, n, now)...)
	}
	return all
}

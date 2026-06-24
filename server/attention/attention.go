// Package attention implements the TRAPX per-Field-Office Attention meter.
//
// Attention is a 0–1000 scale that rises with K9 dog activity (superlinear)
// and decays slowly over time. High Attention summons hostile actors:
//
//   - AuditThreshold (700):  OversightSect audit events begin firing
//   - VendorThreshold (900): ShadowOperator events begin firing (Procurement/Sabotage)
//
// Attention also feeds three ecosystem multipliers that the game server reads
// each tick to tune world behavior:
//
//   - MigrationWeight  — how strongly citizens flee the FO district
//   - AHTaxMultiplier  — Auction House tax increase (cost of doing business under scrutiny)
//   - ContestScaler    — increases Contest frequency proportional to Attention
package attention

import (
	"fmt"
	"math"
	"time"
)

const (
	AttentionMax      = 1000.0
	AttentionMin      = 0.0
	AuditThreshold    = 700.0
	VendorThreshold   = 900.0
	AttentionExponent = 1.3   // superlinear scaling for dog-count gain
	BaseDecayRate     = 1.0   // Attention units lost per second at rest
)

// SummonType identifies what category of actor a high-Attention event produces.
type SummonType string

const (
	SummonOversightSect  SummonType = "OVERSIGHT_SECT"
	SummonShadowOperator SummonType = "SHADOW_OPERATOR"
)

// Event records an Attention-driven world event emitted by the meter.
type Event struct {
	At      time.Time
	FOID    string
	Verb    string // ATTENTION_TICK, AUDIT_THRESHOLD, VENDOR_THRESHOLD, AUDIT_CLEAR, VENDOR_CLEAR
	Summon  SummonType // non-empty when Verb == AUDIT_THRESHOLD or VENDOR_THRESHOLD
	Current float64
	Detail  string
}

func (e Event) String() string {
	return fmt.Sprintf("[%s] FO:%s — %s att=%.0f %s", e.At.Format("15:04:05"),
		e.FOID, e.Verb, e.Current, e.Detail)
}

// Meter tracks Attention for a single Field Office.
// Not goroutine-safe; callers synchronise via the server tick loop.
type Meter struct {
	FOID      string
	Value     float64 // current Attention [0, AttentionMax]
	DecayRate float64 // Attention/s decay when no dogs are active (default BaseDecayRate)

	auditActive  bool // true when Attention >= AuditThreshold
	vendorActive bool // true when Attention >= VendorThreshold
}

// NewMeter creates a zero Attention meter for the given FO.
func NewMeter(foID string) *Meter {
	return &Meter{FOID: foID, DecayRate: BaseDecayRate}
}

// AddDogActivity raises Attention based on the number of active dogs.
// Gain is superlinear: perDog * n^AttentionExponent.
// Called by the game server each time dog activity occurs on this FO.
func (m *Meter) AddDogActivity(nDogs int, perDog float64, now time.Time) []Event {
	if nDogs <= 0 {
		return nil
	}
	gain := perDog * math.Pow(float64(nDogs), AttentionExponent)
	return m.add(gain, now, fmt.Sprintf("dogs=%d gain=%.2f", nDogs, gain))
}

// Add raises Attention by delta directly (from arbitrary sources).
func (m *Meter) Add(delta float64, now time.Time, reason string) []Event {
	return m.add(delta, now, reason)
}

func (m *Meter) add(delta float64, now time.Time, detail string) []Event {
	prev := m.Value
	m.Value = math.Min(m.Value+delta, AttentionMax)
	return m.checkThresholds(prev, now, detail)
}

// Tick decays Attention over dt. Returns threshold-clear events if applicable.
func (m *Meter) Tick(dt time.Duration, now time.Time) []Event {
	prev := m.Value
	m.Value = math.Max(m.Value-m.DecayRate*dt.Seconds(), AttentionMin)
	return m.checkThresholds(prev, now, "decay")
}

// checkThresholds emits events when Attention crosses AuditThreshold or VendorThreshold
// in either direction.
func (m *Meter) checkThresholds(prev float64, now time.Time, detail string) []Event {
	var events []Event

	// Vendor threshold (higher — check first)
	if !m.vendorActive && m.Value >= VendorThreshold {
		m.vendorActive = true
		events = append(events, Event{
			At: now, FOID: m.FOID, Verb: "VENDOR_THRESHOLD",
			Summon: SummonShadowOperator, Current: m.Value,
			Detail: fmt.Sprintf("ShadowOperator summoned (%s)", detail),
		})
	} else if m.vendorActive && m.Value < VendorThreshold {
		m.vendorActive = false
		events = append(events, Event{
			At: now, FOID: m.FOID, Verb: "VENDOR_CLEAR",
			Current: m.Value, Detail: "vendor pressure lifted",
		})
	}

	// Audit threshold
	if !m.auditActive && m.Value >= AuditThreshold {
		m.auditActive = true
		events = append(events, Event{
			At: now, FOID: m.FOID, Verb: "AUDIT_THRESHOLD",
			Summon: SummonOversightSect, Current: m.Value,
			Detail: fmt.Sprintf("OversightSect summoned (%s)", detail),
		})
	} else if m.auditActive && m.Value < AuditThreshold {
		m.auditActive = false
		events = append(events, Event{
			At: now, FOID: m.FOID, Verb: "AUDIT_CLEAR",
			Current: m.Value, Detail: "audit pressure lifted",
		})
	}

	_ = prev
	return events
}

// IsUnderAudit returns true when Attention is at or above AuditThreshold.
func (m *Meter) IsUnderAudit() bool { return m.auditActive }

// IsUnderVendorPressure returns true when Attention is at or above VendorThreshold.
func (m *Meter) IsUnderVendorPressure() bool { return m.vendorActive }

// EcosystemEffects returns the three multipliers driven by current Attention.
// These are read by the game server each tick to tune world behavior.
func (m *Meter) EcosystemEffects() (migrationWeight, ahTaxMultiplier, contestScaler float64) {
	frac := m.Value / AttentionMax // 0.0–1.0
	migrationWeight = frac * 2.0          // 0.0→2.0 (citizens flee at high attention)
	ahTaxMultiplier = 1.0 + frac*0.5      // 1.0→1.5 (AH tax rises 50% at max attention)
	contestScaler   = 1.0 + frac*3.0      // 1.0→4.0 (contest frequency quadruples at max)
	return
}

// Registry manages Attention meters for all Field Offices in a district.
type Registry struct {
	meters map[string]*Meter
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{meters: make(map[string]*Meter)}
}

// GetOrCreate returns the Meter for foID, creating one if it does not exist.
func (r *Registry) GetOrCreate(foID string) *Meter {
	if m, ok := r.meters[foID]; ok {
		return m
	}
	m := NewMeter(foID)
	r.meters[foID] = m
	return m
}

// Get returns the Meter for foID, or nil if not found.
func (r *Registry) Get(foID string) *Meter {
	return r.meters[foID]
}

// TickAll advances all meters by dt and returns all generated events.
func (r *Registry) TickAll(dt time.Duration, now time.Time) []Event {
	var all []Event
	for _, m := range r.meters {
		all = append(all, m.Tick(dt, now)...)
	}
	return all
}

// TotalAttention returns the sum of all Attention values across all FOs.
func (r *Registry) TotalAttention() float64 {
	total := 0.0
	for _, m := range r.meters {
		total += m.Value
	}
	return total
}

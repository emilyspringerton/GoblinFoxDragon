// Package neighborhood implements the TRAPX Neighborhood personality system.
//
// Each city district has a Personality (stable axes) and a Mood (drifting state).
// Personality shapes the ceiling and floor of what the district can feel.
// Mood drifts toward rest state gradually — not instant.
//
// Personality axes (0–100):
//   - Tolerance:  how much unusual activity the block accepts before mobilising
//   - Pride:      how strongly residents resist external control
//   - Cohesion:   how unified the block acts in response to events
//   - Visibility: how much the block surfaces to the media / Watchers
//
// Mood axes (drifting):
//   - Fear:    risk perception (rises with dog presence, enforcement; decays slowly)
//   - Trust:   crew relationship (shifts with Trust in watcher.WatcherState)
//   - Fatigue: exhaustion from sustained high-alertness (rises over time; reduces Cohesion)
//
// Myth seeding: when Fear saturates (>90) + Cohesion > 60, a Myth is written.
// Myths persist and give the Dragon city lore artifacts to reference.
package neighborhood

import (
	"fmt"
	"math"
	"time"
)

const (
	// Personality axes range [0, 100].
	AxisMax = 100.0
	AxisMin = 0.0

	// Mood ranges: Fear and Fatigue [0,100]; Trust [-100,100].
	FearMax    = 100.0
	FatigueMax = 100.0

	// Mood drift rates (per second toward rest).
	FearDecayRate    = 0.5 / 60.0  // 0.5 units/min
	FatigueDecayRate = 0.2 / 60.0  // 0.2 units/min

	// Fatigue rise rate: +1 per second of high alertness (>70).
	FatigueRiseRate = 1.0 / 60.0

	// Myth seeding thresholds.
	MythFearThreshold     = 90.0
	MythCohesionThreshold = 60.0
	MaxMythsPerDistrict   = 10
)

// Personality is the stable character of a neighborhood.
type Personality struct {
	Tolerance  float64 // [0,100] — how much they accept before reacting
	Pride      float64 // [0,100] — resistance to external control
	Cohesion   float64 // [0,100] — unity of response
	Visibility float64 // [0,100] — how much they surface to media/Watchers
}

// DefaultPersonality returns a neutral personality.
func DefaultPersonality() Personality {
	return Personality{Tolerance: 50, Pride: 50, Cohesion: 50, Visibility: 50}
}

// DistrictPersonalities seeds canon personalities for each TRAPX district.
var DistrictPersonalities = map[string]Personality{
	"district-residential": {Tolerance: 60, Pride: 70, Cohesion: 65, Visibility: 45},
	"district-commercial":  {Tolerance: 40, Pride: 40, Cohesion: 35, Visibility: 80},
	"district-industrial":  {Tolerance: 70, Pride: 55, Cohesion: 45, Visibility: 30},
	"district-underground": {Tolerance: 80, Pride: 60, Cohesion: 75, Visibility: 20},
	"district-abandoned":   {Tolerance: 90, Pride: 85, Cohesion: 30, Visibility: 15},
}

// Myth is an in-world narrative fragment seeded when fear saturates + cohesion is high.
type Myth struct {
	At     time.Time
	Origin string // district
	Text   string // one-sentence mythic fragment
	FearAt float64
}

func (m Myth) String() string {
	return fmt.Sprintf("[%s] %s: %q (fear=%.0f)", m.At.Format("15:04:05"), m.Origin, m.Text, m.FearAt)
}

// Event is a single neighborhood state change.
type Event struct {
	At         time.Time
	DistrictID string
	Verb       string // FEAR_SPIKE, FEAR_DECAY, FATIGUE_RISE, TRUST_SHIFT, MYTH_SEEDED, MOOD_REST
	Delta      float64
	Value      float64
	Detail     string
}

func (e Event) String() string {
	return fmt.Sprintf("[%s] nbhd/%s %s delta=%.1f value=%.1f %s",
		e.At.Format("15:04:05"), e.DistrictID, e.Verb, e.Delta, e.Value, e.Detail)
}

// Neighborhood is the living state of a city district's social layer.
type Neighborhood struct {
	DistrictID  string
	Personality Personality
	Fear        float64 // [0, 100]
	FatigueVal  float64 // [0, 100] — "Fatigue" (FatigueVal to avoid shadowing)
	Myths       []Myth
}

// NewNeighborhood creates a Neighborhood for a district with canon personality or default.
func NewNeighborhood(districtID string) *Neighborhood {
	p, ok := DistrictPersonalities[districtID]
	if !ok {
		p = DefaultPersonality()
	}
	return &Neighborhood{
		DistrictID:  districtID,
		Personality: p,
		Fear:        10.0,
		FatigueVal:  0.0,
	}
}

// AddFear increases Fear (clamped to [0, FearMax]).
func (n *Neighborhood) AddFear(delta float64, now time.Time, reason string) []Event {
	// Tolerance reduces fear uptake: high tolerance = slower fear rise.
	effective := delta * (1.0 - n.Personality.Tolerance/200.0)
	prev := n.Fear
	n.Fear = clamp(n.Fear+effective, 0, FearMax)
	events := []Event{{
		At: now, DistrictID: n.DistrictID, Verb: "FEAR_SPIKE",
		Delta: effective, Value: n.Fear, Detail: reason,
	}}
	events = append(events, n.checkMythSeed(prev, now)...)
	return events
}

// Tick advances mood state by dt. Decays Fear and Fatigue toward rest.
// If alertness is high (caller provides), Fatigue rises instead of decaying.
func (n *Neighborhood) Tick(dt time.Duration, alertness float64, now time.Time) []Event {
	var events []Event
	secs := dt.Seconds()

	// Fear decay.
	if n.Fear > 0 {
		decay := FearDecayRate * secs
		n.Fear = clamp(n.Fear-decay, 0, FearMax)
		if n.Fear == 0 {
			events = append(events, Event{
				At: now, DistrictID: n.DistrictID, Verb: "MOOD_REST",
				Delta: -decay, Value: 0, Detail: "fear at rest",
			})
		}
	}

	// Fatigue: rises when alertness > 70, decays otherwise.
	if alertness > 70 {
		rise := FatigueRiseRate * secs * (alertness / 100.0)
		n.FatigueVal = clamp(n.FatigueVal+rise, 0, FatigueMax)
		if n.FatigueVal >= FatigueMax {
			events = append(events, Event{
				At: now, DistrictID: n.DistrictID, Verb: "FATIGUE_RISE",
				Delta: rise, Value: n.FatigueVal, Detail: "district exhausted — cohesion degraded",
			})
		}
	} else {
		decay := FatigueDecayRate * secs
		n.FatigueVal = clamp(n.FatigueVal-decay, 0, FatigueMax)
	}

	return events
}

// WatcherVisibilityMultiplier returns the Watcher Visibility multiplier for this district.
// High Visibility personality + low Fatigue = watchers see more.
func (n *Neighborhood) WatcherVisibilityMultiplier() float64 {
	// Fatigue reduces visibility (exhausted residents stop watching).
	fatiguePenalty := n.FatigueVal / FatigueMax * 0.5
	return (n.Personality.Visibility / 100.0) * (1.0 - fatiguePenalty)
}

// EffectiveCohesion returns cohesion reduced by fatigue.
func (n *Neighborhood) EffectiveCohesion() float64 {
	fatiguePenalty := n.FatigueVal / FatigueMax * 0.4
	return n.Personality.Cohesion * (1.0 - fatiguePenalty)
}

// MythSeedRate returns myths seeded per hour at current fear + cohesion.
// Used to drive narrative content generation by Emily Prime.
func (n *Neighborhood) MythSeedRate() float64 {
	if n.Fear < MythFearThreshold {
		return 0
	}
	cohesion := n.EffectiveCohesion()
	if cohesion < MythCohesionThreshold {
		return 0
	}
	return (n.Fear - MythFearThreshold) / 10.0 * cohesion / 100.0
}

// ScarCount returns the number of myths (lore scars) seeded in this district.
func (n *Neighborhood) MythCount() int { return len(n.Myths) }

// Summary returns a one-line status for the MUD district command.
func (n *Neighborhood) Summary() string {
	return fmt.Sprintf("%s | fear=%.0f fatigue=%.0f myths=%d vis_mult=%.2f cohesion=%.0f",
		n.DistrictID, n.Fear, n.FatigueVal, len(n.Myths), n.WatcherVisibilityMultiplier(), n.EffectiveCohesion())
}

func (n *Neighborhood) checkMythSeed(prevFear float64, now time.Time) []Event {
	if prevFear < MythFearThreshold && n.Fear >= MythFearThreshold {
		if n.EffectiveCohesion() >= MythCohesionThreshold && len(n.Myths) < MaxMythsPerDistrict {
			myth := n.seedMyth(now)
			return []Event{{
				At: now, DistrictID: n.DistrictID, Verb: "MYTH_SEEDED",
				Delta: 0, Value: n.Fear, Detail: myth.Text,
			}}
		}
	}
	return nil
}

func (n *Neighborhood) seedMyth(now time.Time) Myth {
	templates := []string{
		"They say the dogs here remember faces.",
		"The Field Office that burned three seasons ago still smells like ozone when the rain comes.",
		"Channel 11 showed this block on a night nobody can remember clearly.",
		"The woman in the window doesn't blink. She's been there since the last Rogue Swarm.",
		"Baphomet walked this street before the abandoned zone had a name.",
		"The CAST terminal at the corner only answers questions it already knows you'll ask.",
		"Everyone who claimed that FO last winter left the city within 30 days.",
		"The Dragon moved through here once. The streetlights changed color and stayed that way.",
		"Forty-seven receipts. Nobody knows who filed them. They're all signed by the same actor.",
		"The mini bikes stop running at exactly 3:17 AM. Every night.",
	}
	idx := len(n.Myths) % len(templates)
	m := Myth{At: now, Origin: n.DistrictID, Text: templates[idx], FearAt: n.Fear}
	n.Myths = append(n.Myths, m)
	return m
}

// Registry holds all district Neighborhoods.
type Registry struct {
	hoods map[string]*Neighborhood
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{hoods: make(map[string]*Neighborhood)}
}

// GetOrCreate returns the Neighborhood for a district, creating it if absent.
func (r *Registry) GetOrCreate(districtID string) *Neighborhood {
	if h, ok := r.hoods[districtID]; ok {
		return h
	}
	h := NewNeighborhood(districtID)
	r.hoods[districtID] = h
	return h
}

// Get returns the Neighborhood for a district, or nil if absent.
func (r *Registry) Get(districtID string) *Neighborhood {
	return r.hoods[districtID]
}

// TickAll advances all districts by dt.
func (r *Registry) TickAll(dt time.Duration, alertnessByDistrict map[string]float64, now time.Time) []Event {
	var out []Event
	for id, h := range r.hoods {
		alertness := alertnessByDistrict[id]
		out = append(out, h.Tick(dt, alertness, now)...)
	}
	return out
}

// TotalMythCount returns the total number of myths seeded across all districts.
func (r *Registry) TotalMythCount() int {
	total := 0
	for _, h := range r.hoods {
		total += len(h.Myths)
	}
	return total
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

// Package techpressure implements the TRAPX Tech Pressure doom clock.
//
// Tech Pressure is a global metric that rises as players unlock tech tiers,
// deploy K9 swarms, and push the custody system hard. It decays slowly.
// Five thresholds define an escalating threat ladder, each wired into the
// WorldCrisis phase machine via an event channel.
//
// # Threat tiers
//
//   - T1 (250): LeashFrays       — K9 batteries drain faster; minor anomalies
//   - T2 (450): ProcurementWar   — module scarcity; Coverage market opens
//   - T3 (650): QuietAudit       — Federal Oversight activates; Shadow Operators
//   - T4 (850): Packmind         — dogs begin ignoring human commands (micro-rogue)
//   - T5 (1000): CrownProtocol   — super-node unification attempt; map rewrite risk
//
// CrownProtocol (T5) triggers an attempt to unify all FOs into one super-node.
// If successful: map rewrite + Bird Correction. This is the existential climax.
//
// # Inputs that raise Tech Pressure
//
//   - Tier unlock (AddTierUnlock): large spike per tier level
//   - Dog deployment (AddDogDeploy): small spike per dog
//   - Swarm activity per tick (AddSwarmActivity): continuous trickle
//
// # Decay
//
// Passive decay: DecayRate units/s.
// Bird Correction reduces Tech Pressure by BirdCorrectionDrop.
package techpressure

import (
	"fmt"
	"math"
	"time"
)

const (
	PressureMax  = 1000.0
	PressureMin  = 0.0
	DecayRate    = 0.5 // units/s

	TierUnlockSpike    = 80.0  // added per tech tier unlock
	DogDeploySpike     = 5.0   // added per dog deployed
	SwarmActivityRate  = 0.2   // added per active dog per second
	BirdCorrectionDrop = 150.0 // subtracted by a Bird Correction event

	// Thresholds
	T1LeashFrays      = 250.0
	T2ProcurementWar  = 450.0
	T3QuietAudit      = 650.0
	T4Packmind        = 850.0
	T5CrownProtocol   = 1000.0
)

// Tier represents one of the five Tech Pressure threat levels.
type Tier int

const (
	TierNone          Tier = 0
	TierLeashFrays    Tier = 1
	TierProcurement   Tier = 2
	TierQuietAudit    Tier = 3
	TierPackmind      Tier = 4
	TierCrownProtocol Tier = 5
)

var tierNames = map[Tier]string{
	TierNone:          "None",
	TierLeashFrays:    "LeashFrays",
	TierProcurement:   "ProcurementWar",
	TierQuietAudit:    "QuietAudit",
	TierPackmind:      "Packmind",
	TierCrownProtocol: "CrownProtocol",
}

// TierName returns the display name for a threat tier.
func TierName(t Tier) string {
	if s, ok := tierNames[t]; ok {
		return s
	}
	return "Unknown"
}

// PressureThresholdForTier returns the Tech Pressure value at which the given tier activates.
func PressureThresholdForTier(t Tier) float64 {
	switch t {
	case TierLeashFrays:
		return T1LeashFrays
	case TierProcurement:
		return T2ProcurementWar
	case TierQuietAudit:
		return T3QuietAudit
	case TierPackmind:
		return T4Packmind
	case TierCrownProtocol:
		return T5CrownProtocol
	}
	return 0
}

// TierForPressure returns the highest active Tier for the given pressure value.
func TierForPressure(p float64) Tier {
	switch {
	case p >= T5CrownProtocol:
		return TierCrownProtocol
	case p >= T4Packmind:
		return TierPackmind
	case p >= T3QuietAudit:
		return TierQuietAudit
	case p >= T2ProcurementWar:
		return TierProcurement
	case p >= T1LeashFrays:
		return TierLeashFrays
	}
	return TierNone
}

// Event is emitted on every meaningful Tech Pressure state change.
type Event struct {
	At       time.Time
	Verb     string // PRESSURE_TICK, TIER_ACTIVATED, TIER_CLEARED, CROWN_PROTOCOL, BIRD_CORRECTION
	Tier     Tier
	Pressure float64
	Detail   string
}

func (e Event) String() string {
	return fmt.Sprintf("[%s] %s tier=%s pressure=%.1f %s",
		e.At.Format("15:04:05"), e.Verb, TierName(e.Tier), e.Pressure, e.Detail)
}

// Clock is the global Tech Pressure doom clock.
// Not goroutine-safe; callers synchronise via the server tick loop.
type Clock struct {
	Pressure    float64 // current Tech Pressure [0, 1000]
	ActiveTier  Tier    // highest tier currently active
	CrownFired  bool    // true after CrownProtocol event fires for the first time
}

// NewClock creates a zero Tech Pressure clock.
func NewClock() *Clock { return &Clock{} }

// AddTierUnlock adds TierUnlockSpike * level to Tech Pressure.
func (c *Clock) AddTierUnlock(level int, now time.Time) []Event {
	spike := TierUnlockSpike * float64(level)
	return c.add(spike, now, fmt.Sprintf("tier-unlock level=%d", level))
}

// AddDogDeploy adds DogDeploySpike to Tech Pressure.
func (c *Clock) AddDogDeploy(now time.Time) []Event {
	return c.add(DogDeploySpike, now, "dog-deploy")
}

// AddSwarmActivity adds SwarmActivityRate * nDogs * dt.Seconds() to Tech Pressure.
func (c *Clock) AddSwarmActivity(nDogs int, dt time.Duration, now time.Time) []Event {
	delta := SwarmActivityRate * float64(nDogs) * dt.Seconds()
	return c.add(delta, now, fmt.Sprintf("swarm-activity dogs=%d", nDogs))
}

// BirdCorrection subtracts BirdCorrectionDrop from Tech Pressure.
func (c *Clock) BirdCorrection(now time.Time) []Event {
	prev := c.Pressure
	c.Pressure = math.Max(c.Pressure-BirdCorrectionDrop, PressureMin)
	evs := []Event{{
		At: now, Verb: "BIRD_CORRECTION",
		Pressure: c.Pressure,
		Detail:   fmt.Sprintf("drop=%.0f (was %.0f)", prev-c.Pressure, prev),
	}}
	evs = append(evs, c.checkTiers(now, "bird-correction")...)
	return evs
}

// Tick decays Tech Pressure by DecayRate * dt.Seconds().
func (c *Clock) Tick(dt time.Duration, now time.Time) []Event {
	prev := c.Pressure
	c.Pressure = math.Max(c.Pressure-DecayRate*dt.Seconds(), PressureMin)
	_ = prev
	return c.checkTiers(now, "decay")
}

func (c *Clock) add(delta float64, now time.Time, detail string) []Event {
	c.Pressure = math.Min(c.Pressure+delta, PressureMax)
	return c.checkTiers(now, detail)
}

// checkTiers compares current Pressure against all five thresholds and emits
// TIER_ACTIVATED or TIER_CLEARED events as tiers are crossed.
func (c *Clock) checkTiers(now time.Time, detail string) []Event {
	var events []Event
	current := TierForPressure(c.Pressure)

	// Tier rose.
	if current > c.ActiveTier {
		// Emit TIER_ACTIVATED for each tier crossed.
		for t := c.ActiveTier + 1; t <= current; t++ {
			ev := Event{
				At: now, Verb: "TIER_ACTIVATED", Tier: t,
				Pressure: c.Pressure,
				Detail:   fmt.Sprintf("%s (%s)", TierName(t), detail),
			}
			// CrownProtocol is a special one-time climactic event.
			if t == TierCrownProtocol && !c.CrownFired {
				c.CrownFired = true
				ev.Verb = "CROWN_PROTOCOL"
				ev.Detail = "super-node unification attempt — map rewrite risk"
			}
			events = append(events, ev)
		}
		c.ActiveTier = current
	}

	// Tier fell.
	if current < c.ActiveTier {
		for t := c.ActiveTier; t > current; t-- {
			events = append(events, Event{
				At: now, Verb: "TIER_CLEARED", Tier: t,
				Pressure: c.Pressure,
				Detail:   detail,
			})
		}
		c.ActiveTier = current
	}

	return events
}

// IsAtTier returns true if the clock is currently at or above the given tier.
func (c *Clock) IsAtTier(t Tier) bool { return c.ActiveTier >= t }

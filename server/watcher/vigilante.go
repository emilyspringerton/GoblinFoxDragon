package watcher

import (
	"fmt"
	"time"
)

// Vigilante anomaly system.
//
// When Watcher alertness stays above AlertHighViz (80) for an extended period,
// a disruption debt accumulates. Once the debt crosses SpawnDebtThreshold,
// there is a probabilistic chance on each tick that the district generates a
// street vigilante anomaly — a powerful chaotic-neutral entity that operates
// outside all faction structures.
//
// Chaotic neutral: the vigilante targets the highest-disruption entity in the
// district regardless of faction. It may attack the player, cops, or rival crews.
// Player Trust adjusts targeting — high Trust reduces the player's apparent threat
// score; low Trust elevates it. The vigilante is not recruitable or bribeable.
//
// Power tiers: Strong (1) < Dangerous (2) < Anomaly (3).
// Tier 3 spawns only at AlertSaturation (≥ 95) with maximum debt.

const (
	// Disruption debt accumulates when alertness ≥ AlertHighViz.
	// Rate: (alertness - AlertHighViz) / 100 per second.
	DisruptionDebtDecayRate = 0.5 / 60.0 // 0.5 per minute when alertness drops below threshold

	// Debt thresholds.
	SpawnDebtThreshold = 10.0  // minimum debt to roll for a spawn
	SpawnDebtMax       = 100.0 // debt is clamped here; prevents runaway probability

	// Spawn probability: baseProbability + debtBonus * excess debt.
	// Caps at SpawnProbMax.
	SpawnProbBase = 0.05
	SpawnProbBonus = 0.02
	SpawnProbMax  = 0.60

	// Power tier 3 (Anomaly) requires this debt minimum.
	AnomalyDebtThreshold = 60.0
)

// VigilanteArchetype identifies a vigilante's operational style.
type VigilanteArchetype int

const (
	ArchetypeFounder     VigilanteArchetype = iota // old guard; district memory; disrupts the most visible actor
	ArchetypeChemist                               // area denial; elevates Heat on all factions
	ArchetypeApparition                            // invisible to radar; targets highest recent violence
	ArchetypeRiotBreaker                           // saturation-only; suppresses all organized activity
)

var archetypeNames = map[VigilanteArchetype]string{
	ArchetypeFounder:    "Founder",
	ArchetypeChemist:    "Chemist",
	ArchetypeApparition: "Apparition",
	ArchetypeRiotBreaker: "Riot Breaker",
}

// ArchetypeName returns the display name for an archetype.
func ArchetypeName(a VigilanteArchetype) string {
	if s, ok := archetypeNames[a]; ok {
		return s
	}
	return "Unknown"
}

// PowerTier is the vigilante's relative strength level.
type PowerTier int

const (
	TierStrong    PowerTier = 1 // harder than district NMs; manageable with preparation
	TierDangerous PowerTier = 2 // district-disrupting; retreat is a valid option
	TierAnomaly   PowerTier = 3 // near-unstoppable; requires coordinated response or avoidance
)

// VigilanteAnomaly is a spawned street vigilante entity.
// It is not a Mob in the server/mob sense — it is a Watcher-system event
// that the server translates into an NM-class entity with special behaviors.
type VigilanteAnomaly struct {
	Name       string
	Archetype  VigilanteArchetype
	Tier       PowerTier
	DistrictID string
	SpawnedAt  time.Time

	// Combat stats.
	HP        int
	Damage    int
	SpeedMult float64 // relative to base mob speed (1.0 = normal)

	// Chaotic neutral behavior.
	Alignment      string // always "chaotic_neutral"
	TargetBehavior string // how it selects its current target
	Special        string // special ability description
}

// String returns a one-line summary for logging.
func (v *VigilanteAnomaly) String() string {
	return fmt.Sprintf("[vigilante] %s (%s tier=%d) district=%s hp=%d dmg=%d",
		v.Name, ArchetypeName(v.Archetype), v.Tier, v.DistrictID, v.HP, v.Damage)
}

// RandSource is satisfied by *math/rand.Rand or any test double.
type RandSource interface {
	Float64() float64
}

// vigilanteProfiles defines the named vigilante entities per archetype and tier.
// Each entry: Name, HP, Damage, SpeedMult, TargetBehavior, Special.
var vigilanteProfiles = map[VigilanteArchetype]map[PowerTier]struct {
	name          string
	hp, damage    int
	speedMult     float64
	targetBehavior string
	special       string
}{
	ArchetypeFounder: {
		TierStrong: {
			name: "The Shepherd", hp: 480, damage: 55, speedMult: 1.1,
			targetBehavior: "targets the entity with the highest current Heat in district",
			special:        "District Memory: ignores NPC corroboration — hits without warning",
		},
		TierDangerous: {
			name: "The Patriarch", hp: 720, damage: 80, speedMult: 1.2,
			targetBehavior: "targets highest Heat; switches to cops if enforcement level ≥ 3",
			special:        "Crowd Call: civilians briefly shield the Patriarch from ranged attacks",
		},
		TierAnomaly: {
			name: "The Origin", hp: 1100, damage: 120, speedMult: 1.3,
			targetBehavior: "targets the single entity responsible for most recent escalation",
			special:        "Deep Root: immune to Heat-reduction attempts; cannot be bribed or flagged",
		},
	},
	ArchetypeChemist: {
		TierStrong: {
			name: "The Burner", hp: 380, damage: 65, speedMult: 0.9,
			targetBehavior: "moves to center of district; applies Heat spike to all within radius",
			special:        "Smoke Compound: 45s visibility reduction; disables cop radio in area",
		},
		TierDangerous: {
			name: "The Alchemist", hp: 560, damage: 90, speedMult: 1.0,
			targetBehavior: "targets armored entities (cops/heavy crew); ranged throwables",
			special:        "Contagion Payload: on hit, spreads 15 Heat to all entities within 8m",
		},
		TierAnomaly: {
			name: "The Architect of Ruin", hp: 880, damage: 130, speedMult: 1.1,
			targetBehavior: "creates impassable zone; forces all entities to reroute",
			special:        "District Burn: raises all faction Heat +25 simultaneously; persists for 3 minutes",
		},
	},
	ArchetypeApparition: {
		TierStrong: {
			name: "The Shadow", hp: 320, damage: 85, speedMult: 1.6,
			targetBehavior: "invisible until within 4m; targets the most recent act of violence",
			special:        "Ghost Walk: does not appear on radar or minimap until first attack",
		},
		TierDangerous: {
			name: "The Wraith", hp: 480, damage: 110, speedMult: 1.8,
			targetBehavior: "invisible; targets the entity with highest accumulated violence score",
			special:        "Phase Strike: first attack ignores armor; 30s cooldown; returns to invisible",
		},
		TierAnomaly: {
			name: "The Erasure", hp: 700, damage: 160, speedMult: 2.0,
			targetBehavior: "invisible; reads violence debt across 3 sessions; multi-target",
			special:        "Memory Wipe: on kill, removes one Scar from the district permanently",
		},
	},
	ArchetypeRiotBreaker: {
		TierStrong: {
			name: "The Pacifier", hp: 620, damage: 70, speedMult: 0.8,
			targetBehavior: "targets any entity currently in combat; attacks both sides equally",
			special:        "Crowd Scatter: AoE stagger; breaks any active combat engagement in radius",
		},
		TierDangerous: {
			name: "The Equalizer", hp: 900, damage: 100, speedMult: 0.9,
			targetBehavior: "targets the faction with the highest active Heat in district",
			special:        "Suppression Field: 60s window where all entities take 25% bonus damage",
		},
		TierAnomaly: {
			name: "The Leveler", hp: 1400, damage: 145, speedMult: 1.0,
			targetBehavior: "attacks all organized entities simultaneously; neutral parties are safe",
			special:        "Zero Sum: on spawn, resets all faction territorial claims in district to 0",
		},
	},
}

// selectArchetype picks an archetype based on current district state.
// RiotBreaker only spawns at AlertSaturation. Otherwise weighted by Trust and Bias.
func selectArchetype(w *WatcherState, rng RandSource) VigilanteArchetype {
	// Saturation-only: Riot Breaker.
	if w.Alertness >= AlertSaturation {
		return ArchetypeRiotBreaker
	}
	// Low trust → Apparition (violence-focused district).
	if w.Trust <= -50 {
		if rng.Float64() < 0.5 {
			return ArchetypeApparition
		}
	}
	// Bias toward Procurement → Chemist (economic disruption).
	if w.Bias == BiasProcurement && w.BiasScore > 40 {
		return ArchetypeChemist
	}
	// Default weighted roll: Founder 40%, Chemist 25%, Apparition 25%, RiotBreaker 10%.
	r := rng.Float64()
	switch {
	case r < 0.40:
		return ArchetypeFounder
	case r < 0.65:
		return ArchetypeChemist
	case r < 0.90:
		return ArchetypeApparition
	default:
		return ArchetypeRiotBreaker
	}
}

// selectTier picks a power tier based on disruption debt.
func selectTier(debt float64) PowerTier {
	switch {
	case debt >= AnomalyDebtThreshold:
		return TierAnomaly
	case debt >= SpawnDebtThreshold*2.5:
		return TierDangerous
	default:
		return TierStrong
	}
}

// spawnProbability computes the probability of a vigilante spawn this tick.
func spawnProbability(debt float64) float64 {
	if debt < SpawnDebtThreshold {
		return 0
	}
	p := SpawnProbBase + SpawnProbBonus*(debt-SpawnDebtThreshold)
	if p > SpawnProbMax {
		p = SpawnProbMax
	}
	return p
}

// AccumulateDisruptionDebt is called each tick by WatcherState.Tick.
// When alertness ≥ AlertHighViz, debt grows proportional to how far above the
// threshold alertness is. When below, debt decays slowly.
func (w *WatcherState) AccumulateDisruptionDebt(dt time.Duration, now time.Time) {
	if w.Alertness >= AlertHighViz {
		rate := (w.Alertness - AlertHighViz) / 100.0 // [0, 0.20] per second at max
		w.DisruptionDebt = clamp(w.DisruptionDebt+rate*dt.Seconds(), 0, SpawnDebtMax)
	} else {
		decay := DisruptionDebtDecayRate * dt.Seconds()
		w.DisruptionDebt = clamp(w.DisruptionDebt-decay, 0, SpawnDebtMax)
	}
}

// CheckVigilanteSpawn tests whether a vigilante anomaly spawns this tick.
// Returns nil when no spawn occurs. On a spawn, DisruptionDebt resets to zero
// and a VIGILANTE_SPAWN event is emitted via the returned anomaly.
//
// Call once per server tick (after AccumulateDisruptionDebt).
func (w *WatcherState) CheckVigilanteSpawn(rng RandSource, now time.Time) (*VigilanteAnomaly, []Event) {
	p := spawnProbability(w.DisruptionDebt)
	if p <= 0 || rng.Float64() > p {
		return nil, nil
	}

	archetype := selectArchetype(w, rng)
	tier := selectTier(w.DisruptionDebt)
	profile := vigilanteProfiles[archetype][tier]

	v := &VigilanteAnomaly{
		Name:           profile.name,
		Archetype:      archetype,
		Tier:           tier,
		DistrictID:     w.DistrictID,
		SpawnedAt:      now,
		HP:             profile.hp,
		Damage:         profile.damage,
		SpeedMult:      profile.speedMult,
		Alignment:      "chaotic_neutral",
		TargetBehavior: profile.targetBehavior,
		Special:        profile.special,
	}

	debt := w.DisruptionDebt
	w.DisruptionDebt = 0

	events := []Event{{
		At:         now,
		DistrictID: w.DistrictID,
		Verb:       "VIGILANTE_SPAWN",
		Delta:      -debt,
		Value:      0,
		Detail: fmt.Sprintf("name=%q archetype=%s tier=%d hp=%d — %s",
			v.Name, ArchetypeName(archetype), tier, v.HP, v.Special),
	}}

	return v, events
}

// VigilanteTargetPriority computes an entity's targeting priority for a vigilante.
// Higher = more likely to be targeted. playerTrust is the district trust score for
// the player entity; entities without trust scores pass 0.
//
// Used by the server game loop to determine who the vigilante attacks each tick.
func VigilanteTargetPriority(heatScore, violenceDebt float64, playerTrust float64) float64 {
	// Base score: heat + violence.
	base := heatScore + violenceDebt*0.5

	// Trust adjustment: positive trust reduces threat score (vigilante deprioritizes
	// a crew that has built community goodwill). Negative trust increases it.
	trustAdj := -playerTrust * 0.3 // trust=100 → -30; trust=-100 → +30
	return base + trustAdj
}

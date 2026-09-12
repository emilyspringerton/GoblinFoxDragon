package main

// JOB_SPELL_SYSTEM_NORTHSTAR.md §5 -- real combat damage formula, replacing the previous flat
// `playerDamage = 30`/per-mob-`MeleeDamage` deterministic hits on both sides with STR-scaled
// random damage, DEX-based crit, one shared accuracy/evasion roll, and VIT-based reduction.
// Founder direction: "combat loop is too deterministic i always hit for 30 no matter what class
// there should be randomness in the damage... STR should increase auto attack damage mean...
// attacks can critical (dex based) attacks can miss - same with the incoming damage VIT reduces
// the actual damage coming in agi increases evasion which effectively decreases the enemy
// accuracy." Also: "Combat damage formula overhaul also needs to use marble bag RNG with pity."

import (
	"math/rand"
	"time"

	"dragonsnshit/server/job"
	"dragonsnshit/server/rng"
)

const (
	combatBaseHitChance = 90 // percent, before DEX/AGI adjustment (player's own attacks)
	combatMinHitChance  = 50
	combatMaxHitChance  = 99

	combatBaseCritChance = 5 // percent, before DEX adjustment
	combatMaxCritChance  = 50
	combatCritMultiplier = 1.5

	combatDamageVarianceLow  = 0.8 // -20%
	combatDamageVarianceHigh = 1.2 // +20%

	// combatMobBaseHitChance/CritChance (real, named v0 answer to JOB_SPELL_SYSTEM_NORTHSTAR.md
	// §5's own open question: mobs get a simple flat baseline here, not a full STR/DEX/VIT/AGI
	// stat block -- every mob spawn site in this file would need real numbers for the latter,
	// a real, separate, larger change than this pass. A real per-mob stat block is a real,
	// named future refinement, not a silently-abandoned gap.
	combatMobBaseHitChance  = 85
	combatMobBaseCritChance = 5
)

// combatRNG is this file's own random source for every combat roll below -- a real
// *rand.Rand, not the deprecated top-level rand.Seed/rand.Intn, so it can be swapped for a
// seeded one in tests that need deterministic output.
var combatRNG = rand.New(rand.NewSource(time.Now().UnixNano()))

// playerCombatStats returns p's real, current job (+ sub-job) stats -- the same
// CombinedStats()-or-fall-back-to-StatsFor(jobID) pattern cmdCastBlackMagic's own INT-bonus
// lookup already established, reused here rather than duplicated with slightly different
// fallback behavior.
func playerCombatStats(p *player) job.Stats {
	if p.charJob != nil {
		if s, err := p.charJob.CombinedStats(); err == nil {
			return s
		}
	}
	if s, err := job.StatsFor(p.jobID); err == nil {
		return s
	}
	return job.Stats{}
}

func clampPercent(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// pityRoll performs one binary weighted pick via server/rng's shared marble-bag+pity mechanism,
// always framed from the PLAYER's own perspective regardless of which side of the roll they're
// on: outcome 0 is always the player's real desired result (landing their own hit/crit, or
// avoiding an incoming hit/crit), weighted [desiredChancePercent, 100-desiredChancePercent].
// Pity always favors the player, whether attacking or defending -- a real, deliberate design
// choice (this mechanism's own real purpose is bad-luck protection for the person playing, not
// for mobs), not something either FFXI or the founder's own wording specifies either way.
func pityRoll(desiredChancePercent int, pity *[2]int) bool {
	desiredChancePercent = clampPercent(desiredChancePercent, 0, 100)
	picked := rng.MarbleBagPick([]int{desiredChancePercent, 100 - desiredChancePercent}, pity[:], combatRNG)
	return picked == 0
}

// rollDamageVariance returns mean scaled by a real, uniform ±20% roll -- a plain roll, not a
// marble-bag pick: §5 is explicit that marble-bag+pity is for a small, fixed set of discrete
// outcomes (does it land, does it crit), not a continuous quantity like the exact damage number.
func rollDamageVariance(mean int) int {
	if mean <= 0 {
		return 0
	}
	lo := int(float64(mean) * combatDamageVarianceLow)
	hi := int(float64(mean) * combatDamageVarianceHigh)
	if hi <= lo {
		return mean
	}
	return lo + combatRNG.Intn(hi-lo+1)
}

// reduceIncomingDamage applies VIT-based flat reduction (§5): always shaves a real, meaningful
// amount off incoming damage, but never reduces a landed hit below 1 -- a hit that connects
// always does *something*.
func reduceIncomingDamage(incoming, defenderVIT int) int {
	reduced := incoming - defenderVIT/2
	if reduced < 1 {
		reduced = 1
	}
	return reduced
}

// resolvePlayerAutoAttackDamage computes one real player auto-attack swing against a mob: a
// pity-boosted accuracy roll (mob defender AGI is 0 -- no comparable mob stat block exists yet,
// same real v0 simplification combatMobBaseHitChance/CritChance name above), STR-scaled damage
// with real ±20% variance, and a pity-boosted DEX-based crit. Returns (hit, crit, damage) --
// damage is 0 and crit is false whenever hit is false.
func resolvePlayerAutoAttackDamage(p *player, baseDamage int) (hit bool, crit bool, damage int) {
	stats := playerCombatStats(p)
	hitChance := clampPercent(combatBaseHitChance+stats.DEX/2, combatMinHitChance, combatMaxHitChance)
	if !pityRoll(hitChance, &p.pityOwnHit) {
		return false, false, 0
	}

	mean := baseDamage
	if stats.STR > 0 {
		mean = baseDamage * stats.STR / 10
	}
	dmg := rollDamageVariance(mean)

	critChance := clampPercent(combatBaseCritChance+stats.DEX/10, 0, combatMaxCritChance)
	isCrit := pityRoll(critChance, &p.pityOwnCrit)
	if isCrit {
		dmg = int(float64(dmg) * combatCritMultiplier)
	}
	if dmg < 1 {
		dmg = 1
	}
	return true, isCrit, dmg
}

// resolveMobAutoAttackDamage computes one real mob attack landing on a player: accuracy/crit use
// combatMobBaseHitChance/CritChance's own flat baseline (the mob "attacker" side of §5's still-
// open stat-block question), the player's own real AGI reduces the mob's effective hit chance
// (the same accuracy roll doubling as evasion, per §5's own "one roll, not two" framing), and the
// player's own real VIT reduces whatever damage lands. Returns (hit, crit, damage).
func resolveMobAutoAttackDamage(p *player, mobMeleeDamage int) (hit bool, crit bool, damage int) {
	stats := playerCombatStats(p)
	mobHitChance := clampPercent(combatMobBaseHitChance-stats.AGI/2, combatMinHitChance, combatMaxHitChance)
	// Player's desired outcome is a miss -> desired chance is the complement of the mob landing.
	if pityRoll(100-mobHitChance, &p.pityAvoidHit) {
		return false, false, 0
	}

	// Player's desired outcome for the crit roll is NOT being crit.
	isCrit := !pityRoll(100-combatMobBaseCritChance, &p.pityAvoidCrit)
	dmg := mobMeleeDamage
	if isCrit {
		dmg = int(float64(dmg) * combatCritMultiplier)
	}
	dmg = reduceIncomingDamage(dmg, stats.VIT)
	return true, isCrit, dmg
}

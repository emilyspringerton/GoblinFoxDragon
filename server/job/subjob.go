// subjob.go: Sub-job system (S82-02) and job ability recast tracker (S82-03).
//
// # Sub-job model (FFXI-parity)
//
// A character has a main job at full level and an optional sub job at half level.
// Sub job grants a subset of the sub job's skills/abilities up to SubLvl/2 (floor).
// Combined stats = main stats + sub stats at half level.
//
// # Job ability recast model
//
// Each job ability has a recast timer. RecastTracker records the last use time
// and blocks re-use until the timer expires. Trait passives have no recast.
package job

import (
	"errors"
	"time"
)

// ── Sub-job ───────────────────────────────────────────────────────────────────

var ErrSameJob = errors.New("main and sub job must differ")

// CharJob holds a character's main and sub job pairing.
type CharJob struct {
	Main    JobID
	Sub     JobID // "" = no sub job
	MainLvl int
	SubLvl  int
}

// NewCharJob creates a new job pairing. Sub may be "".
// Returns ErrSameJob if main == sub (and sub != "").
func NewCharJob(main, sub JobID, mainLvl, subLvl int) (*CharJob, error) {
	if sub != "" && main == sub {
		return nil, ErrSameJob
	}
	return &CharJob{Main: main, Sub: sub, MainLvl: mainLvl, SubLvl: subLvl}, nil
}

// EffectiveSubLevel returns the effective level of the sub job (SubLvl / 2, floored).
// Returns 0 if no sub job is set.
func (c *CharJob) EffectiveSubLevel() int {
	if c.Sub == "" {
		return 0
	}
	return c.SubLvl / 2
}

// CombinedStats returns the combined Stats for the character.
// Main job stats are at MainLvl; sub job stats are at EffectiveSubLevel.
// Sub stats use the same linear scaling as HPAtLevel.
// Returns ErrUnknownJob if either job is invalid.
func (c *CharJob) CombinedStats() (Stats, error) {
	ms, err := StatsFor(c.Main)
	if err != nil {
		return Stats{}, err
	}
	if c.Sub == "" {
		return ms, nil
	}
	ss, err := StatsFor(c.Sub)
	if err != nil {
		return Stats{}, err
	}
	subLvl := c.EffectiveSubLevel()
	// Combine: main full stats + sub stats at half level
	// Primary stats: simple addition (FFXI adds sub primary stats at half level)
	combined := Stats{
		BaseHP:     ms.BaseHP + ss.BaseHP/2,
		HPPerLevel: ms.HPPerLevel, // only main job HP/level applies
		BaseMP:     ms.BaseMP + ss.BaseMP/2,
		MPPerLevel: ms.MPPerLevel,
		STR:        ms.STR + ss.STR*subLvl/(c.SubLvl+1), // approximate
		DEX:        ms.DEX + ss.DEX*subLvl/(c.SubLvl+1),
		VIT:        ms.VIT + ss.VIT*subLvl/(c.SubLvl+1),
		AGI:        ms.AGI + ss.AGI*subLvl/(c.SubLvl+1),
		INT:        ms.INT + ss.INT*subLvl/(c.SubLvl+1),
		MND:        ms.MND + ss.MND*subLvl/(c.SubLvl+1),
		CHR:        ms.CHR + ss.CHR*subLvl/(c.SubLvl+1),
	}
	return combined, nil
}

// ── Job abilities + recast (S82-03) ──────────────────────────────────────────

var (
	ErrAbilityOnRecast   = errors.New("ability is still on recast")
	ErrAbilityUnknown    = errors.New("unknown ability ID")
	ErrAbilityLevelGated = errors.New("character level too low for this ability")
)

// Ability defines a job ability with recast time and minimum level.
type Ability struct {
	ID       string
	Job      JobID
	Recast   time.Duration
	MinLevel int
}

// Trait is a passive job trait (no recast).
type Trait struct {
	ID       string
	Job      JobID
	MinLevel int
}

// RecastTracker tracks last-used times for job abilities.
type RecastTracker struct {
	lastUsed  map[string]time.Time
	abilities map[string]Ability
}

// NewRecastTracker creates a tracker for a given ability set.
func NewRecastTracker(abilities []Ability) *RecastTracker {
	m := make(map[string]Ability, len(abilities))
	for _, a := range abilities {
		m[a.ID] = a
	}
	return &RecastTracker{lastUsed: make(map[string]time.Time), abilities: m}
}

// Ready reports whether abilityID can be used at now (not on recast).
// Returns ErrAbilityUnknown if the ability is not registered.
func (r *RecastTracker) Ready(abilityID string, now time.Time) (bool, error) {
	a, ok := r.abilities[abilityID]
	if !ok {
		return false, ErrAbilityUnknown
	}
	last, used := r.lastUsed[abilityID]
	if !used {
		return true, nil
	}
	return now.Sub(last) >= a.Recast, nil
}

// Use marks abilityID as used at now.
// Returns ErrAbilityOnRecast if still on cooldown, ErrAbilityLevelGated if level too low.
func (r *RecastTracker) Use(abilityID string, now time.Time, charLevel int) error {
	a, ok := r.abilities[abilityID]
	if !ok {
		return ErrAbilityUnknown
	}
	if charLevel < a.MinLevel {
		return ErrAbilityLevelGated
	}
	ready, _ := r.Ready(abilityID, now)
	if !ready {
		return ErrAbilityOnRecast
	}
	r.lastUsed[abilityID] = now
	return nil
}

// RecastRemaining returns how long until an ability is ready again.
// Returns 0 if already ready.
func (r *RecastTracker) RecastRemaining(abilityID string, now time.Time) (time.Duration, error) {
	a, ok := r.abilities[abilityID]
	if !ok {
		return 0, ErrAbilityUnknown
	}
	last, used := r.lastUsed[abilityID]
	if !used {
		return 0, nil
	}
	elapsed := now.Sub(last)
	if elapsed >= a.Recast {
		return 0, nil
	}
	return a.Recast - elapsed, nil
}

// ── Sample abilities ──────────────────────────────────────────────────────────

// WarriorAbilities returns the sample ability set for WAR.
func WarriorAbilities() []Ability {
	return []Ability{
		{ID: "provoke", Job: WAR, Recast: 30 * time.Second, MinLevel: 5},
		{ID: "berserk", Job: WAR, Recast: 3 * time.Minute, MinLevel: 10},
		{ID: "warcry", Job: WAR, Recast: 5 * time.Minute, MinLevel: 20},
		{ID: "aggressor", Job: WAR, Recast: 5 * time.Minute, MinLevel: 25},
		{ID: "defenders", Job: WAR, Recast: 5 * time.Minute, MinLevel: 30},
		{ID: "mighty_strikes", Job: WAR, Recast: 60 * time.Minute, MinLevel: 15},
	}
}

// SummonerAbilities returns the sample ability set for SMN (2026-07-31, founder real-time:
// "zagan beleth vassago as summoner avatars GFD"). Real FFXI Summoners fight through Avatars
// they call forth rather than their own weapon -- the reverse direction from
// GoblinFoxDragon/docs2/REDGARDEN_GUI_NORTHSTAR.md's own Milestones 1-2 (which ported a
// DragonsNShit job's content INTO REDGARDEN as Battlegrounds ability content). Here, three of
// REDGARDEN's own already-built TYLER-lore heroes become SMN's real Avatar pool instead of
// invented-from-scratch summons: Zagan (armor-shred/stun/mirror, docs/HEROES_VS0.md), Beleth
// (burn-DoT/silence/delayed-burst), Vassago (damage+silence/cast-refund/silence-zone). See
// apps2/mud's own cmdJA case for what each summon actually does mechanically -- a real,
// honestly-scoped simplification (this MUD's `status` package has no Stun kind and no
// armor-shred-shaped debuff, and mob targets have no status stack at all yet), not the full
// REDGARDEN kit ported 1:1.
func SummonerAbilities() []Ability {
	return []Ability{
		{ID: "summon_zagan", Job: SMN, Recast: 3 * time.Minute, MinLevel: 5},
		{ID: "summon_beleth", Job: SMN, Recast: 3 * time.Minute, MinLevel: 10},
		{ID: "summon_vassago", Job: SMN, Recast: 3 * time.Minute, MinLevel: 15},
	}
}

// WhiteMageAbilities returns the sample ability set for WHM.
func WhiteMageAbilities() []Ability {
	return []Ability{
		{ID: "benediction", Job: WHM, Recast: 60 * time.Minute, MinLevel: 50},
		{ID: "holy_water", Job: WHM, Recast: 10 * time.Minute, MinLevel: 15},
	}
}

// Starter kits (2026-08-04, founder, live: "and then add a couple starter kits 2 abilities per
// job to the basic jobs") -- abilitiesForJob (apps2/mud/main.go) only had real sets for
// WAR/WHM/SMN before this; MNK/BLM/RDM/THF, the rest of FFXI's original 6 starting jobs, returned
// nil. Real FFXI level-1 ability parity where it exists (MonkAbilities' Chakra/Boost, ThiefAbilities'
// Sneak Attack/Trick Attack, RedMageAbilities' Convert are all real Lv1 FFXI job abilities);
// BlackMageAbilities takes the same honest liberty this file's own SummonerAbilities doc comment
// already takes ("a real, honestly-scoped simplification") since BLM's own first real FFXI job
// ability (Elemental Seal) doesn't unlock until Lv11 -- too high to serve as a starter kit here.

// MonkAbilities returns the starter ability set for MNK.
func MonkAbilities() []Ability {
	return []Ability{
		{ID: "chakra", Job: MNK, Recast: 5 * time.Minute, MinLevel: 1},
		{ID: "boost", Job: MNK, Recast: 30 * time.Second, MinLevel: 1},
	}
}

// BlackMageAbilities returns the starter ability set for BLM.
func BlackMageAbilities() []Ability {
	return []Ability{
		{ID: "clear_mind", Job: BLM, Recast: 3 * time.Minute, MinLevel: 1},
		{ID: "elemental_seal", Job: BLM, Recast: 5 * time.Minute, MinLevel: 5},
	}
}

// RedMageAbilities returns the starter ability set for RDM.
func RedMageAbilities() []Ability {
	return []Ability{
		{ID: "convert", Job: RDM, Recast: 5 * time.Minute, MinLevel: 1},
		{ID: "chainspell", Job: RDM, Recast: 20 * time.Minute, MinLevel: 15},
	}
}

// ThiefAbilities returns the starter ability set for THF.
func ThiefAbilities() []Ability {
	return []Ability{
		{ID: "sneak_attack", Job: THF, Recast: 1 * time.Minute, MinLevel: 1},
		{ID: "trick_attack", Job: THF, Recast: 1 * time.Minute, MinLevel: 1},
	}
}

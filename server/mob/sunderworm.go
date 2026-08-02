package mob

import "time"

// Sunderworm — the boss threat behind "World Crisis: The Sunderworm"
// (docs2/specs/WORLD_CRISIS_VS0.md). Founder direction, 2026-08-02: "use some of the arena
// heroes as dungeon mobs and bosses" (for the separate dungeon system) and, for this crisis
// specifically, "worm as the northstar" -- the boss is a scaled-up version of the exact mob
// already living in the starter zone (worm.go's KindWorm), not a new, unrelated creature. Its
// burrow/surface cycle reuses StateBurrowed unchanged (targetable() already returns false while
// burrowed) rather than inventing a new invulnerability mechanic -- the spec's own phase names,
// BURROW and EMERGENCE, already describe exactly that cycle. BurrowInterval/BurrowDuration are
// deliberately left at 0 here (unlike NewWorm): the crisis phase handler in apps2/mud/main.go
// drives State directly (burrowed during Phase Burrow, surfaced at Phase Emergence) rather than
// letting the mob's own internal timer decide when to go invulnerable -- the crisis phase
// machine is the authority on that, not an autonomous per-mob clock.
const (
	KindSunderworm     = "sunderworm"
	KindSunderwormHead = "sunderworm-head"

	SunderwormHP         = 15000
	SunderwormDamage     = 120
	SunderwormMoveSpeed  = 2.0 // slow, but not stationary -- it has to be able to chase
	SunderwormAggroRange = 25.0
	SunderwormLeashRange = 60.0
	SunderwormMeleeRange = 4.0 // a boss-scale hitbox, not a worm's
	SunderwormSwingDelay = 3 * time.Second

	// Split War sub-bosses (WORLD_CRISIS_VS0.md D2: "at least two sub-bosses/heads with
	// different mechanics spawn in different locations"). Real HP, real threat, but a fraction
	// of the main body's -- meant to be killable within the Split War window, not a second
	// full boss fight.
	SunderwormHeadHP         = 4000
	SunderwormHeadDamage     = 70
	SunderwormHeadMoveSpeed  = 3.0
	SunderwormHeadAggroRange = 20.0
	SunderwormHeadLeashRange = 40.0
	SunderwormHeadMeleeRange = 3.0
	SunderwormHeadSwingDelay = 2500 * time.Millisecond
)

// NewSunderworm returns the Sunderworm boss Mob, spawned already burrowed (invulnerable) --
// the crisis phase handler surfaces it explicitly when Phase Emergence begins. SceneID 4
// (New Handington / the starter zone) per founder direction ("build it on top of our starter
// zone") -- the same zone worm.go's TownSquareWormSpawns already populates with the small,
// mundane version of this exact creature.
func NewSunderworm(id string, pos Pos) Mob {
	return Mob{
		ID:          id,
		Kind:        KindSunderworm,
		SceneID:     4,
		Pos:         pos,
		HomePos:     pos,
		HP:          SunderwormHP,
		MaxHP:       SunderwormHP,
		State:       StateBurrowed, // invulnerable until the crisis handler surfaces it
		AggroRange:  SunderwormAggroRange,
		LeashRange:  SunderwormLeashRange,
		MeleeRange:  SunderwormMeleeRange,
		MoveSpeed:   SunderwormMoveSpeed,
		SwingDelay:  SunderwormSwingDelay,
		MeleeDamage: SunderwormDamage,
		// BurrowInterval/BurrowDuration intentionally 0 -- see this file's own doc comment.
	}
}

// NewSunderwormHead returns one Split War sub-boss, spawned already surfaced (Split War is a
// combat window, unlike the Sunderworm's own Burrow-phase entry). Positions are the caller's
// own responsibility -- WORLD_CRISIS_VS0.md B3 requires geo-separation between the two, which is
// a spawn-site decision, not something this constructor can enforce on its own.
func NewSunderwormHead(id string, pos Pos) Mob {
	return Mob{
		ID:          id,
		Kind:        KindSunderwormHead,
		SceneID:     4,
		Pos:         pos,
		HomePos:     pos,
		HP:          SunderwormHeadHP,
		MaxHP:       SunderwormHeadHP,
		State:       StateIdle,
		AggroRange:  SunderwormHeadAggroRange,
		LeashRange:  SunderwormHeadLeashRange,
		MeleeRange:  SunderwormHeadMeleeRange,
		MoveSpeed:   SunderwormHeadMoveSpeed,
		SwingDelay:  SunderwormHeadSwingDelay,
		MeleeDamage: SunderwormHeadDamage,
	}
}

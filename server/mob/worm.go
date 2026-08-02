package mob

import "time"

// Worm mob constants — FFXI-inspired earth-element starting-zone mob.
const (
	KindWorm = "worm"

	WormHP           = 90
	WormDamage       = 8
	WormMoveSpeed    = 1.5 // worms are slow
	WormAggroRange   = 8.0 // short sight; mostly passive
	WormLeashRange   = 18.0
	WormMeleeRange   = 2.0
	WormSwingDelay   = 4 * time.Second // slow attacker
	WormBurrowEvery  = 30 * time.Second // burrow every 30 s
	WormBurrowFor    = 4 * time.Second  // stays underground for 4 s
)

// NewWorm returns a Mob configured as a worm with the given ID and spawn position.
// Zone 0 (Meadow) is the intended scene; callers may override SceneID.
func NewWorm(id string, pos Pos) Mob {
	return Mob{
		ID:             id,
		Kind:           KindWorm,
		SceneID:        0, // Meadow
		Pos:            pos,
		HomePos:        pos,
		HP:             WormHP,
		MaxHP:          WormHP,
		State:          StateIdle,
		AggroRange:     WormAggroRange,
		LeashRange:     WormLeashRange,
		MeleeRange:     WormMeleeRange,
		MoveSpeed:      WormMoveSpeed,
		SwingDelay:     WormSwingDelay,
		MeleeDamage:    WormDamage,
		BurrowInterval: WormBurrowEvery,
		BurrowDuration: WormBurrowFor,
	}
}

// MeadowWormSpawns returns the default worm spawn set for the Meadow starting zone
// (zone 0). Worms are placed in a ring around the perimeter, away from the town
// centre at (0, 2, 0).
func MeadowWormSpawns() []Mob {
	positions := []Pos{
		{X: 35, Y: 2, Z: 0},
		{X: -35, Y: 2, Z: 0},
		{X: 0, Y: 2, Z: 35},
		{X: 0, Y: 2, Z: -35},
		{X: 25, Y: 2, Z: 25},
		{X: -25, Y: 2, Z: 25},
		{X: 25, Y: 2, Z: -25},
		{X: -25, Y: 2, Z: -25},
	}
	worms := make([]Mob, len(positions))
	for i, p := range positions {
		worms[i] = NewWorm(wormID(i), p)
	}
	return worms
}

func wormID(i int) string {
	return "worm-meadow-" + itoa(i)
}

// TownSquareWormSpawns returns the starter mobs for zone 4 (New Handington, 2026-08-02 --
// GoblinFoxDragon's Battlegrounds-GUI Town scene, founder: "implement the starter area worm" ->
// "where's my starter zone outside of town with the worms?" -- a single worm didn't read as "a
// starter zone," so this became a 4-worm ring around the origin -> then the real hand-drawn
// town-map.jpeg arrived ("i want the town layout to match town map pretty much exactly"), which
// names this exact spot "Worm Hut" at a specific position in the town, not centered on spawn --
// repositioned to match. Clustered tightly (small deltas around the hut), not spread in a wide
// ring, since it's meant to read as "a hut with worms in it," not an open field. NewWorm
// defaults SceneID to 0 (Meadow); overridden to 4 here since this is the one caller that isn't
// spawning into Meadow itself.
func TownSquareWormSpawns() []Mob {
	const hutX, hutZ = 5.0, 15.0 // "Worm Hut" position, see apps2/battlegrounds_gui's TOWN_BUILDINGS
	positions := []Pos{
		{X: hutX + 1.5, Y: 2, Z: hutZ},
		{X: hutX - 1.5, Y: 2, Z: hutZ},
		{X: hutX, Y: 2, Z: hutZ + 1.5},
		{X: hutX, Y: 2, Z: hutZ - 1.5},
	}
	worms := make([]Mob, len(positions))
	for i, p := range positions {
		w := NewWorm("worm-town-"+itoa(i), p)
		w.SceneID = 4
		worms[i] = w
	}
	return worms
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := [20]byte{}
	pos := len(b)
	for n > 0 {
		pos--
		b[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(b[pos:])
}

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

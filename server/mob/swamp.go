package mob

import "time"

// Swampville mob constants — three starting-zone mob types for scene/zone 3.
//
// Leeches: aggressive, bleed-flavour (high swing frequency, low damage), medium HP.
// Slimes:  passive until struck, slow, large HP, moderate damage (earth parity).
// Lizards: aggressive, fast, glass-cannon — punishing for new players.
const (
	// Leech
	KindLeech         = "leech"
	LeechHP           = 75
	LeechDamage       = 7
	LeechMoveSpeed    = 2.0
	LeechAggroRange   = 14.0 // aggressive — notices players at medium range
	LeechLeashRange   = 25.0
	LeechMeleeRange   = 2.0
	LeechSwingDelay   = 2500 * time.Millisecond // fast nibbler

	// Slime
	KindSlime         = "slime"
	SlimeHP           = 140
	SlimeDamage       = 14
	SlimeMoveSpeed    = 0.9  // very slow
	SlimeAggroRange   = 6.0  // passive; only aggroes on close approach
	SlimeLeashRange   = 15.0
	SlimeMeleeRange   = 2.5  // slightly larger hitbox
	SlimeSwingDelay   = 4 * time.Second

	// Lizard
	KindLizard        = "lizard"
	LizardHP          = 105
	LizardDamage      = 18
	LizardMoveSpeed   = 4.5  // fast chaser
	LizardAggroRange  = 18.0 // aggressive, long sight
	LizardLeashRange  = 35.0
	LizardMeleeRange  = 2.0
	LizardSwingDelay  = 3 * time.Second
)

// NewLeech returns a Mob configured as a leech in Swampville (zone/scene 3).
func NewLeech(id string, pos Pos) Mob {
	return Mob{
		ID:          id,
		Kind:        KindLeech,
		SceneID:     3,
		Pos:         pos,
		HomePos:     pos,
		HP:          LeechHP,
		MaxHP:       LeechHP,
		State:       StateIdle,
		AggroRange:  LeechAggroRange,
		LeashRange:  LeechLeashRange,
		MeleeRange:  LeechMeleeRange,
		MoveSpeed:   LeechMoveSpeed,
		SwingDelay:  LeechSwingDelay,
		MeleeDamage: LeechDamage,
	}
}

// NewSlime returns a Mob configured as a slime in Swampville (zone/scene 3).
func NewSlime(id string, pos Pos) Mob {
	return Mob{
		ID:          id,
		Kind:        KindSlime,
		SceneID:     3,
		Pos:         pos,
		HomePos:     pos,
		HP:          SlimeHP,
		MaxHP:       SlimeHP,
		State:       StateIdle,
		AggroRange:  SlimeAggroRange,
		LeashRange:  SlimeLeashRange,
		MeleeRange:  SlimeMeleeRange,
		MoveSpeed:   SlimeMoveSpeed,
		SwingDelay:  SlimeSwingDelay,
		MeleeDamage: SlimeDamage,
	}
}

// NewLizard returns a Mob configured as a lizard in Swampville (zone/scene 3).
func NewLizard(id string, pos Pos) Mob {
	return Mob{
		ID:          id,
		Kind:        KindLizard,
		SceneID:     3,
		Pos:         pos,
		HomePos:     pos,
		HP:          LizardHP,
		MaxHP:       LizardHP,
		State:       StateIdle,
		AggroRange:  LizardAggroRange,
		LeashRange:  LizardLeashRange,
		MeleeRange:  LizardMeleeRange,
		MoveSpeed:   LizardMoveSpeed,
		SwingDelay:  LizardSwingDelay,
		MeleeDamage: LizardDamage,
	}
}

// SwampvilleSpawns returns the default mob population for Swampville (zone 3).
//
// Layout (top-down, town at 0,1,0):
//   - 6 Leeches in a ring at r≈20 (aggressive; players must fight through them)
//   - 4 Slimes in an outer ring at r≈32 (passive tanks; further from spawn)
//   - 4 Lizards at the zone perimeter r≈42 (for higher-level play)
func SwampvilleSpawns() []Mob {
	var out []Mob

	// Leeches — inner ring ~20u out
	leechPositions := []Pos{
		{X: 20, Y: 1, Z: 0},
		{X: -20, Y: 1, Z: 0},
		{X: 0, Y: 1, Z: 20},
		{X: 0, Y: 1, Z: -20},
		{X: 14, Y: 1, Z: 14},
		{X: -14, Y: 1, Z: -14},
	}
	for i, p := range leechPositions {
		out = append(out, NewLeech("leech-swamp-"+itoa(i), p))
	}

	// Slimes — mid ring ~32u out
	slimePositions := []Pos{
		{X: 32, Y: 1, Z: 0},
		{X: -32, Y: 1, Z: 0},
		{X: 0, Y: 1, Z: 32},
		{X: 0, Y: 1, Z: -32},
	}
	for i, p := range slimePositions {
		out = append(out, NewSlime("slime-swamp-"+itoa(i), p))
	}

	// Lizards — outer perimeter ~42u out
	lizardPositions := []Pos{
		{X: 30, Y: 1, Z: 30},
		{X: -30, Y: 1, Z: 30},
		{X: 30, Y: 1, Z: -30},
		{X: -30, Y: 1, Z: -30},
	}
	for i, p := range lizardPositions {
		out = append(out, NewLizard("lizard-swamp-"+itoa(i), p))
	}

	return out
}

package mob

import "time"

// Caves mob constants — zone 2 (Caves). Underground zone; three archetypes.
// Cave Bat: sound-aggro (represented via wide aggro radius), fast, fragile.
// Cave Spider: passive until approached, inflicts a slow debuff (signalled via events).
// Skeleton: undead, aggressive to all living within range; high magic resistance.
const (
	// Cave Bat — sound-aggro, fast, low HP, low damage; teaching mob for aggro mechanics.
	KindCaveBat        = "cave-bat"
	CaveBatHP          = 60
	CaveBatDamage      = 9
	CaveBatMoveSpeed   = 6.0 // fastest mob in caves; hard to escape
	CaveBatAggroRange  = 20.0 // wide — simulates sound sensitivity
	CaveBatLeashRange  = 30.0
	CaveBatMeleeRange  = 2.0
	CaveBatSwingDelay  = 2200 * time.Millisecond // rapid bite

	// Cave Spider — passive, slow, ensnaring. Ground boss of mid-caves.
	KindCaveSpider        = "cave-spider"
	CaveSpiderHP          = 120
	CaveSpiderDamage      = 15
	CaveSpiderMoveSpeed   = 1.8
	CaveSpiderAggroRange  = 7.0  // passive; closes only when very near
	CaveSpiderLeashRange  = 20.0
	CaveSpiderMeleeRange  = 2.5
	CaveSpiderSwingDelay  = 4500 * time.Millisecond // slow but hits for decent

	// Skeleton — undead, always aggressive; hardest mobs in the base cave zones.
	KindSkeleton        = "skeleton"
	SkeletonHP          = 150
	SkeletonDamage      = 22
	SkeletonMoveSpeed   = 3.0
	SkeletonAggroRange  = 25.0 // wide undead aggro
	SkeletonLeashRange  = 45.0
	SkeletonMeleeRange  = 2.2
	SkeletonSwingDelay  = 3 * time.Second
)

// NewCaveBat returns a Mob configured as a cave bat in Caves (zone/scene 2).
func NewCaveBat(id string, pos Pos) Mob {
	return Mob{
		ID:          id,
		Kind:        KindCaveBat,
		SceneID:     2,
		Pos:         pos,
		HomePos:     pos,
		HP:          CaveBatHP,
		MaxHP:       CaveBatHP,
		State:       StateIdle,
		AggroRange:  CaveBatAggroRange,
		LeashRange:  CaveBatLeashRange,
		MeleeRange:  CaveBatMeleeRange,
		MoveSpeed:   CaveBatMoveSpeed,
		SwingDelay:  CaveBatSwingDelay,
		MeleeDamage: CaveBatDamage,
	}
}

// NewCaveSpider returns a Mob configured as a cave spider in Caves (zone/scene 2).
func NewCaveSpider(id string, pos Pos) Mob {
	return Mob{
		ID:          id,
		Kind:        KindCaveSpider,
		SceneID:     2,
		Pos:         pos,
		HomePos:     pos,
		HP:          CaveSpiderHP,
		MaxHP:       CaveSpiderHP,
		State:       StateIdle,
		AggroRange:  CaveSpiderAggroRange,
		LeashRange:  CaveSpiderLeashRange,
		MeleeRange:  CaveSpiderMeleeRange,
		MoveSpeed:   CaveSpiderMoveSpeed,
		SwingDelay:  CaveSpiderSwingDelay,
		MeleeDamage: CaveSpiderDamage,
	}
}

// NewSkeleton returns a Mob configured as a skeleton in Caves (zone/scene 2).
func NewSkeleton(id string, pos Pos) Mob {
	return Mob{
		ID:          id,
		Kind:        KindSkeleton,
		SceneID:     2,
		Pos:         pos,
		HomePos:     pos,
		HP:          SkeletonHP,
		MaxHP:       SkeletonHP,
		State:       StateIdle,
		AggroRange:  SkeletonAggroRange,
		LeashRange:  SkeletonLeashRange,
		MeleeRange:  SkeletonMeleeRange,
		MoveSpeed:   SkeletonMoveSpeed,
		SwingDelay:  SkeletonSwingDelay,
		MeleeDamage: SkeletonDamage,
	}
}

// CavesSpawns returns the default mob population for Caves (zone 2).
//
// Layout (entry near origin; cave narrows with distance):
//   - 6 Cave Bats clustered near the ceiling-height ceiling sections
//   - 5 Cave Spiders spread through the mid-section
//   - 4 Skeletons deep in the cave, highest threat layer
func CavesSpawns() []Mob {
	var out []Mob

	// Bats — near-entry ceiling clusters; sound-aggro lesson for new arrivals
	batPositions := []Pos{
		{X: 8, Y: 4, Z: 5},
		{X: -8, Y: 4, Z: 5},
		{X: 12, Y: 5, Z: -10},
		{X: -12, Y: 5, Z: -10},
		{X: 5, Y: 6, Z: 18},
		{X: -5, Y: 6, Z: 18},
	}
	for i, p := range batPositions {
		out = append(out, NewCaveBat("bat-caves-"+itoa(i), p))
	}

	// Spiders — mid-cave, passive unless approached
	spiderPositions := []Pos{
		{X: 18, Y: 4, Z: -18},
		{X: -18, Y: 4, Z: -18},
		{X: 0, Y: 4, Z: -25},
		{X: 14, Y: 4, Z: -30},
		{X: -14, Y: 4, Z: -30},
	}
	for i, p := range spiderPositions {
		out = append(out, NewCaveSpider("spider-caves-"+itoa(i), p))
	}

	// Skeletons — deep cave; undead aggro to all
	skeletonPositions := []Pos{
		{X: 10, Y: 4, Z: -42},
		{X: -10, Y: 4, Z: -42},
		{X: 0, Y: 4, Z: -50},
		{X: 20, Y: 4, Z: -48},
	}
	for i, p := range skeletonPositions {
		out = append(out, NewSkeleton("skeleton-caves-"+itoa(i), p))
	}

	return out
}

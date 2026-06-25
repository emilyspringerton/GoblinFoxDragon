package mob

import "time"

// Hills mob constants — zone 1 (Hills). Three archetypes spanning beginner to
// intermediate difficulty. Sound-aggro beetles occupy the mid-ring; rabbits
// are the true beginner target; wolves are the danger layer.
const (
	// Rabbit — passive, low-difficulty first kill for new players.
	KindRabbit        = "rabbit"
	RabbitHP          = 45
	RabbitDamage      = 4
	RabbitMoveSpeed   = 3.2 // fast runner but easy to catch
	RabbitAggroRange  = 0.0 // passive; only aggroes if attacked
	RabbitLeashRange  = 20.0
	RabbitMeleeRange  = 1.8
	RabbitSwingDelay  = 3 * time.Second

	// Beetle — aggressive, sound-aggro (represented as a shorter aggro range
	// that checks all nearby players regardless of line-of-sight facing).
	KindBeetle        = "beetle"
	BeetleHP          = 110
	BeetleDamage      = 12
	BeetleMoveSpeed   = 2.8
	BeetleAggroRange  = 12.0 // moderate aggro
	BeetleLeashRange  = 28.0
	BeetleMeleeRange  = 2.0
	BeetleSwingDelay  = 3500 * time.Millisecond

	// Hills Wolf — fast, high damage, dangerous for new players at the zone edge.
	KindHillsWolf        = "hills-wolf"
	HillsWolfHP          = 135
	HillsWolfDamage      = 20
	HillsWolfMoveSpeed   = 5.5
	HillsWolfAggroRange  = 22.0
	HillsWolfLeashRange  = 40.0
	HillsWolfMeleeRange  = 2.2
	HillsWolfSwingDelay  = 2800 * time.Millisecond
)

// NewRabbit returns a Mob configured as a rabbit in Hills (zone/scene 1).
func NewRabbit(id string, pos Pos) Mob {
	return Mob{
		ID:          id,
		Kind:        KindRabbit,
		SceneID:     1,
		Pos:         pos,
		HomePos:     pos,
		HP:          RabbitHP,
		MaxHP:       RabbitHP,
		State:       StateIdle,
		AggroRange:  RabbitAggroRange,
		LeashRange:  RabbitLeashRange,
		MeleeRange:  RabbitMeleeRange,
		MoveSpeed:   RabbitMoveSpeed,
		SwingDelay:  RabbitSwingDelay,
		MeleeDamage: RabbitDamage,
	}
}

// NewBeetle returns a Mob configured as a beetle in Hills (zone/scene 1).
func NewBeetle(id string, pos Pos) Mob {
	return Mob{
		ID:          id,
		Kind:        KindBeetle,
		SceneID:     1,
		Pos:         pos,
		HomePos:     pos,
		HP:          BeetleHP,
		MaxHP:       BeetleHP,
		State:       StateIdle,
		AggroRange:  BeetleAggroRange,
		LeashRange:  BeetleLeashRange,
		MeleeRange:  BeetleMeleeRange,
		MoveSpeed:   BeetleMoveSpeed,
		SwingDelay:  BeetleSwingDelay,
		MeleeDamage: BeetleDamage,
	}
}

// NewHillsWolf returns a Mob configured as a wolf at the Hills zone edge (zone/scene 1).
func NewHillsWolf(id string, pos Pos) Mob {
	return Mob{
		ID:          id,
		Kind:        KindHillsWolf,
		SceneID:     1,
		Pos:         pos,
		HomePos:     pos,
		HP:          HillsWolfHP,
		MaxHP:       HillsWolfHP,
		State:       StateIdle,
		AggroRange:  HillsWolfAggroRange,
		LeashRange:  HillsWolfLeashRange,
		MeleeRange:  HillsWolfMeleeRange,
		MoveSpeed:   HillsWolfMoveSpeed,
		SwingDelay:  HillsWolfSwingDelay,
		MeleeDamage: HillsWolfDamage,
	}
}

// HillsSpawns returns the default mob population for Hills (zone 1).
//
// Layout (town near origin, hills rising away):
//   - 8 Rabbits scattered in grazing clusters near the midfield
//   - 6 Beetles in a ring at mid-distance (aggressive; punish careless approach)
//   - 4 Wolves at the zone perimeter (high-level content / dangerous)
func HillsSpawns() []Mob {
	var out []Mob

	// Rabbits — scattered grazing groups, ~15–22u out
	rabbitPositions := []Pos{
		{X: 15, Y: 8, Z: 5},
		{X: 18, Y: 8, Z: -8},
		{X: -16, Y: 8, Z: 10},
		{X: -12, Y: 8, Z: -14},
		{X: 22, Y: 9, Z: 12},
		{X: -20, Y: 9, Z: -6},
		{X: 8, Y: 8, Z: 18},
		{X: -5, Y: 8, Z: -20},
	}
	for i, p := range rabbitPositions {
		out = append(out, NewRabbit("rabbit-hills-"+itoa(i), p))
	}

	// Beetles — mid ring ~28u out
	beetlePositions := []Pos{
		{X: 28, Y: 9, Z: 0},
		{X: -28, Y: 9, Z: 0},
		{X: 0, Y: 10, Z: 28},
		{X: 0, Y: 10, Z: -28},
		{X: 20, Y: 9, Z: 20},
		{X: -20, Y: 9, Z: -20},
	}
	for i, p := range beetlePositions {
		out = append(out, NewBeetle("beetle-hills-"+itoa(i), p))
	}

	// Wolves — outer perimeter ~42u out
	wolfPositions := []Pos{
		{X: 42, Y: 11, Z: 0},
		{X: -42, Y: 11, Z: 0},
		{X: 0, Y: 11, Z: 42},
		{X: 0, Y: 11, Z: -42},
	}
	for i, p := range wolfPositions {
		out = append(out, NewHillsWolf("wolf-hills-"+itoa(i), p))
	}

	return out
}

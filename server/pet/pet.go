// Package pet implements the Beastmaster (BST) pet companion system for DragonsNShit.
//
// BST players can Charm or Jug a pet that fights autonomously alongside them.
// Pets auto-attack the owner's current target on a 1.5s swing timer.
// When a pet dies it is gone; the owner must Jug a new pet (5s respawn cooldown).
// Only BST (main or sub) players can use the Tame command.
package pet

import (
	"errors"
	"fmt"
	"math/rand"
	"time"
)

// Kind is one of the 8 tameable mob types.
type Kind string

const (
	KindWolf   Kind = "wolf"
	KindBird   Kind = "bird"
	KindLizard Kind = "lizard"
	KindCrab   Kind = "crab"
	KindLeech  Kind = "leech"
	KindSlime  Kind = "slime"
	KindWorm   Kind = "worm"
	KindBear   Kind = "bear"
)

// AllKinds enumerates the 8 tameable types.
var AllKinds = []Kind{KindWolf, KindBird, KindLizard, KindCrab, KindLeech, KindSlime, KindWorm, KindBear}

// Stats defines the base stats for each pet kind at level 1.
var Stats = map[Kind]struct {
	BaseHP     int
	HPPerLevel int
	Damage     int
	SwingDelay time.Duration
}{
	KindWolf:   {80, 8, 12, 2 * time.Second},
	KindBird:   {60, 6, 10, 1500 * time.Millisecond},
	KindLizard: {90, 9, 14, 2 * time.Second},
	KindCrab:   {120, 12, 8, 2500 * time.Millisecond},
	KindLeech:  {70, 7, 11, 1800 * time.Millisecond},
	KindSlime:  {100, 10, 9, 2200 * time.Millisecond},
	KindWorm:   {85, 8, 13, 1900 * time.Millisecond},
	KindBear:   {140, 14, 16, 3 * time.Second},
}

// Pet is a BST companion.
type Pet struct {
	Kind      Kind
	Level     int
	HP        int
	MaxHP     int
	Damage    int
	SwingDelay time.Duration
	OwnerSlot string // slot key of the BST owner
	lastSwing time.Time
	alive     bool
}

// newPet creates a Pet of the given kind at the given level.
func newPet(k Kind, level int, owner string) *Pet {
	st, ok := Stats[k]
	if !ok {
		st = Stats[KindWolf]
	}
	hp := st.BaseHP + st.HPPerLevel*level
	return &Pet{
		Kind:       k,
		Level:      level,
		HP:         hp,
		MaxHP:      hp,
		Damage:     st.Damage + level,
		SwingDelay: st.SwingDelay,
		OwnerSlot:  owner,
		alive:      true,
	}
}

// Errors returned by pet operations.
var (
	ErrNoPet          = errors.New("no pet")
	ErrPetDead        = errors.New("pet is dead")
	ErrPetAlive       = errors.New("pet already active")
	ErrCooldown       = errors.New("jug pet on cooldown")
	ErrTameFailed     = errors.New("tame failed")
	ErrTameHP         = errors.New("mob HP too high to tame (must be ≤70%)")
	ErrUnknownKind    = errors.New("unknown pet kind")
)

// JugCooldown is the time after a pet dies before a new jug pet can be called.
const JugCooldown = 5 * time.Second

// TameSuccessBase is the base tame success rate (40%).
const TameSuccessBase = 0.40

// TameSuccessPerLevel is added per BST level (2%/lvl).
const TameSuccessPerLevel = 0.02

// MaxTameSuccessChance caps the tame success rate.
const MaxTameSuccessChance = 0.95

// Slot tracks one BST player's pet state.
type Slot struct {
	Pet       *Pet
	diedAt    time.Time
	rng       *rand.Rand
}

// NewSlot creates a Slot for one BST player.
func NewSlot() *Slot {
	return &Slot{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

// Tame attempts to charm a mob into a pet.
// mobKind: the mob's Kind string (must map to a known pet Kind).
// mobHPPct: mob's current HP as a fraction of max HP (0.0–1.0).
// bstLevel: owner's BST job level.
// ownerSlot: caller's slot key.
func (s *Slot) Tame(mobKind string, mobHPPct float64, bstLevel int, ownerSlot string) (*Pet, error) {
	if s.Pet != nil && s.Pet.alive {
		return nil, ErrPetAlive
	}
	k := Kind(mobKind)
	if _, ok := Stats[k]; !ok {
		return nil, ErrUnknownKind
	}
	if mobHPPct > 0.70 {
		return nil, ErrTameHP
	}
	chance := TameSuccessBase + TameSuccessPerLevel*float64(bstLevel)
	if chance > MaxTameSuccessChance {
		chance = MaxTameSuccessChance
	}
	if s.rng.Float64() >= chance {
		return nil, ErrTameFailed
	}
	lvl := bstLevel
	if lvl <= 0 {
		lvl = 1
	}
	s.Pet = newPet(k, lvl, ownerSlot)
	return s.Pet, nil
}

// JugPet summons a fresh pet of the given kind (no tame roll; BST job ability).
// Returns ErrCooldown if the cooldown since last pet death has not elapsed.
func (s *Slot) JugPet(k Kind, ownerSlot string) (*Pet, error) {
	if s.Pet != nil && s.Pet.alive {
		return nil, ErrPetAlive
	}
	if _, ok := Stats[k]; !ok {
		return nil, ErrUnknownKind
	}
	if !s.diedAt.IsZero() && time.Since(s.diedAt) < JugCooldown {
		return nil, fmt.Errorf("%w (%.0fs remaining)", ErrCooldown, (JugCooldown - time.Since(s.diedAt)).Seconds())
	}
	s.Pet = newPet(k, 1, ownerSlot)
	s.diedAt = time.Time{} // reset cooldown
	return s.Pet, nil
}

// Release dismisses the active pet.
func (s *Slot) Release() error {
	if s.Pet == nil || !s.Pet.alive {
		return ErrNoPet
	}
	s.Pet.alive = false
	s.Pet = nil
	return nil
}

// IsAlive reports whether a pet is active and alive.
func (s *Slot) IsAlive() bool {
	return s.Pet != nil && s.Pet.alive
}

// Status returns a human-readable pet status string.
func (s *Slot) Status() string {
	if !s.IsAlive() {
		if !s.diedAt.IsZero() {
			remain := JugCooldown - time.Since(s.diedAt)
			if remain > 0 {
				return fmt.Sprintf("(no pet — jug cooldown %.0fs)", remain.Seconds())
			}
		}
		return "(no pet)"
	}
	return fmt.Sprintf("%s Lv%d HP:%d/%d", s.Pet.Kind, s.Pet.Level, s.Pet.HP, s.Pet.MaxHP)
}

// CombatEvent is returned by Tick when the pet deals damage.
type CombatEvent struct {
	Damage    int
	TargetSlot string
}

// Tick advances the pet's swing timer. If a target is set and the pet is alive,
// it may deal damage. Returns nil if no swing occurred.
// targetSlot: the slot key of the mob/player being attacked.
// now: current server time.
func (s *Slot) Tick(targetSlot string, now time.Time) *CombatEvent {
	if !s.IsAlive() || targetSlot == "" {
		return nil
	}
	if now.Sub(s.Pet.lastSwing) < s.Pet.SwingDelay {
		return nil
	}
	s.Pet.lastSwing = now
	dmg := s.Pet.Damage
	return &CombatEvent{Damage: dmg, TargetSlot: targetSlot}
}

// TakeDamage applies damage to the pet. If HP reaches 0, the pet dies.
// Returns true if the pet died from this damage.
func (s *Slot) TakeDamage(dmg int) bool {
	if !s.IsAlive() {
		return false
	}
	s.Pet.HP -= dmg
	if s.Pet.HP <= 0 {
		s.Pet.HP = 0
		s.Pet.alive = false
		s.diedAt = time.Now()
		s.Pet = nil
		return true
	}
	return false
}

// Heal restores HP to the pet (capped at MaxHP). Returns ErrNoPet if no pet.
func (s *Slot) Heal(amount int) (int, error) {
	if !s.IsAlive() {
		return 0, ErrNoPet
	}
	before := s.Pet.HP
	s.Pet.HP += amount
	if s.Pet.HP > s.Pet.MaxHP {
		s.Pet.HP = s.Pet.MaxHP
	}
	return s.Pet.HP - before, nil
}

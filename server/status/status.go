// Package status implements server-authoritative status effects for DragonsNShit.
//
// # Effect model (FFXI-inspired)
//
// Every effect has a Kind (what it is), Potency (how strong), and optional
// expiry time.  A Stack holds at most one active effect per Kind.
//
// # Apply rules
//
//   - Weaker effect (lower Potency) → rejected; existing effect is unchanged.
//   - Same Potency → duration refreshed to the new ExpiresAt.
//   - Stronger effect (higher Potency) → overwrites existing effect.
//   - Permanent effects (ExpiresAt.IsZero()) never expire through Tick.
//
// # Tick
//
// Call Tick(now) on every server tick (e.g., every 3 seconds in FFXI).  It:
//   - removes effects whose ExpiresAt has passed, returning them as Expired
//   - generates TickEvents for DoT/HoT effects (Poison, Regen, Refresh)
//
// # Combat integration
//
// Stack exposes helpers for the combat loop:
//
//	NetHastePct()     int  — Haste.Potency - Slow.Potency (may be negative)
//	PhysDefBonus()    int  — Protect.Potency
//	MagicDefBonus()   int  — Shell.Potency
//	IsParalyzed()     bool — Paralyze is active
//	IsSilenced()      bool — Silence is active
//	IsBound()         bool — Bind is active
package status

import (
	"fmt"
	"time"
)

// Kind identifies a specific status effect.
type Kind int

const (
	// Debuffs
	Poison   Kind = 1 // HP loss per tick
	Paralyze Kind = 2 // chance to fail actions (checked per action by caller)
	Slow     Kind = 3 // negative haste%; reduces attack speed
	Silence  Kind = 4 // prevents spellcasting
	Bind     Kind = 5 // prevents movement

	// Buffs
	Haste   Kind = 10 // positive haste%; increases attack speed
	Regen   Kind = 11 // HP recovery per tick
	Refresh Kind = 12 // MP recovery per tick
	Protect Kind = 13 // physical defense bonus (flat)
	Shell   Kind = 14 // magic defense bonus (flat)
)

func (k Kind) String() string {
	switch k {
	case Poison:
		return "Poison"
	case Paralyze:
		return "Paralyze"
	case Slow:
		return "Slow"
	case Silence:
		return "Silence"
	case Bind:
		return "Bind"
	case Haste:
		return "Haste"
	case Regen:
		return "Regen"
	case Refresh:
		return "Refresh"
	case Protect:
		return "Protect"
	case Shell:
		return "Shell"
	default:
		return fmt.Sprintf("Kind(%d)", int(k))
	}
}

// Category classifies an effect as a buff or debuff (used for targeted dispel).
type Category int

const (
	CategoryBuff   Category = 1
	CategoryDebuff Category = 2
)

var kindCategory = map[Kind]Category{
	Poison:   CategoryDebuff,
	Paralyze: CategoryDebuff,
	Slow:     CategoryDebuff,
	Silence:  CategoryDebuff,
	Bind:     CategoryDebuff,
	Haste:    CategoryBuff,
	Regen:    CategoryBuff,
	Refresh:  CategoryBuff,
	Protect:  CategoryBuff,
	Shell:    CategoryBuff,
}

// Cat returns the category of a Kind, or 0 if unknown.
func Cat(k Kind) Category { return kindCategory[k] }

// TickTarget identifies what resource a tick event affects.
type TickTarget int

const (
	TargetHP TickTarget = 1
	TargetMP TickTarget = 2
)

// TickEvent is generated each server tick for DoT/HoT effects.
// Value is negative for damage (Poison) and positive for healing (Regen, Refresh).
type TickEvent struct {
	Kind   Kind
	Target TickTarget
	Value  int
}

// ApplyResult reports what happened when Apply was called.
type ApplyResult int

const (
	ApplyAdded    ApplyResult = 1 // new effect, did not exist before
	ApplyRefreshed ApplyResult = 2 // same potency: duration reset
	ApplyUpgraded ApplyResult = 3 // higher potency: overwrote existing
	ApplyRejected ApplyResult = 4 // lower potency: existing stronger, rejected
)

// Effect is a single active status effect on a character.
type Effect struct {
	Kind      Kind
	Potency   int       // strength; semantics vary by Kind (see package doc)
	AppliedAt time.Time
	ExpiresAt time.Time // zero = permanent (never expires via Tick)
}

// Active returns true if the effect has not yet expired relative to now.
func (e Effect) Active(now time.Time) bool {
	return e.ExpiresAt.IsZero() || now.Before(e.ExpiresAt)
}

// TickResult is returned by Stack.Tick.
type TickResult struct {
	Expired []Effect     // effects that expired this tick (already removed)
	Events  []TickEvent  // DoT/HoT events that fired this tick
}

// Stack holds the active status effects on one character.
// It is not safe for concurrent access; callers synchronise via the server loop.
type Stack struct {
	effects map[Kind]*Effect
}

// New returns an empty Stack.
func New() *Stack { return &Stack{effects: make(map[Kind]*Effect)} }

// Apply adds or updates an effect according to the overwrite rules.
// Returns ApplyRejected if an existing stronger effect blocks the new one.
func (s *Stack) Apply(e Effect) ApplyResult {
	existing, ok := s.effects[e.Kind]
	if ok {
		switch {
		case e.Potency < existing.Potency:
			return ApplyRejected
		case e.Potency == existing.Potency:
			existing.ExpiresAt = e.ExpiresAt // refresh duration
			existing.AppliedAt = e.AppliedAt
			return ApplyRefreshed
		default: // e.Potency > existing.Potency
			s.effects[e.Kind] = &e
			return ApplyUpgraded
		}
	}
	s.effects[e.Kind] = &e
	return ApplyAdded
}

// Has reports whether a given Kind is currently active.
func (s *Stack) Has(k Kind) bool {
	e, ok := s.effects[k]
	return ok && e.Active(time.Now())
}

// Get returns the active effect for a Kind and whether it exists.
func (s *Stack) Get(k Kind) (Effect, bool) {
	e, ok := s.effects[k]
	if !ok {
		return Effect{}, false
	}
	return *e, true
}

// Remove dispels a specific Kind.  Returns true if an effect was present.
func (s *Stack) Remove(k Kind) bool {
	if _, ok := s.effects[k]; ok {
		delete(s.effects, k)
		return true
	}
	return false
}

// DispelOne removes one random debuff (or buff) of the given category.
// Returns the removed effect and true, or zero Effect and false if none present.
func (s *Stack) DispelOne(cat Category) (Effect, bool) {
	for k, e := range s.effects {
		if kindCategory[k] == cat {
			delete(s.effects, k)
			return *e, true
		}
	}
	return Effect{}, false
}

// All returns a snapshot of all currently tracked effects (may include
// effects that have just expired but not yet been swept by Tick).
func (s *Stack) All() []Effect {
	out := make([]Effect, 0, len(s.effects))
	for _, e := range s.effects {
		out = append(out, *e)
	}
	return out
}

// Tick sweeps expired effects and generates DoT/HoT events for the current tick.
// Call once per server tick interval (typically every 3 seconds).
func (s *Stack) Tick(now time.Time) TickResult {
	var res TickResult
	for k, e := range s.effects {
		if !e.ExpiresAt.IsZero() && !now.Before(e.ExpiresAt) {
			res.Expired = append(res.Expired, *e)
			delete(s.effects, k)
			continue
		}
		// Generate per-tick events for active DoT/HoT effects.
		switch k {
		case Poison:
			res.Events = append(res.Events, TickEvent{Kind: Poison, Target: TargetHP, Value: -e.Potency})
		case Regen:
			res.Events = append(res.Events, TickEvent{Kind: Regen, Target: TargetHP, Value: e.Potency})
		case Refresh:
			res.Events = append(res.Events, TickEvent{Kind: Refresh, Target: TargetMP, Value: e.Potency})
		}
	}
	return res
}

// --- Combat integration helpers ---

// NetHastePct returns Haste.Potency − Slow.Potency.
// A positive result means net speed gain; negative means net slowdown.
// Returns 0 if neither Haste nor Slow is active.
func (s *Stack) NetHastePct() int {
	haste := 0
	if e, ok := s.effects[Haste]; ok {
		haste = e.Potency
	}
	slow := 0
	if e, ok := s.effects[Slow]; ok {
		slow = e.Potency
	}
	return haste - slow
}

// PhysDefBonus returns the flat physical defense bonus from Protect (0 if absent).
func (s *Stack) PhysDefBonus() int {
	if e, ok := s.effects[Protect]; ok {
		return e.Potency
	}
	return 0
}

// MagicDefBonus returns the flat magic defense bonus from Shell (0 if absent).
func (s *Stack) MagicDefBonus() int {
	if e, ok := s.effects[Shell]; ok {
		return e.Potency
	}
	return 0
}

// IsParalyzed reports whether the Paralyze effect is active.
// Callers roll a random check against Paralyze.Potency to determine action failure.
func (s *Stack) IsParalyzed() bool { _, ok := s.effects[Paralyze]; return ok }

// IsSilenced reports whether the Silence effect is active.
// Callers should block spellcasting when true.
func (s *Stack) IsSilenced() bool { _, ok := s.effects[Silence]; return ok }

// IsBound reports whether the Bind effect is active.
// Callers should block movement when true.
func (s *Stack) IsBound() bool { _, ok := s.effects[Bind]; return ok }

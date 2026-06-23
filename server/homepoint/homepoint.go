// Package homepoint implements the DragonsNShit Home Point crystal / KO return system.
//
// # Home Point model (FFXI-parity)
//
// Every character has a single Home Point — a crystal location they have previously
// touched. On KO they may Return to it, paying an XP penalty proportional to a
// configurable rate. Setting a new Home Point replaces the previous one.
//
// Home Point logic is pure domain logic with no I/O. The caller (server tick or
// player session handler) owns state persistence.
package homepoint

import (
	"errors"

	"dragonsnshit/server/mob"
)

const (
	DefaultXPPenaltyPct = 8  // percentage of current XP lost on Return
	XPPenaltyMin        = 0  // floor (never go negative)
)

var (
	ErrNoHomePoint = errors.New("no home point set")
	ErrNotKO       = errors.New("character is not KO'd")
)

// Crystal is the Home Point crystal at a specific world location.
type Crystal struct {
	SceneID int
	Pos     mob.Pos
}

// State holds a character's home-point and KO status.
// Not goroutine-safe; callers synchronise via the server tick loop.
type State struct {
	Home      *Crystal
	CurrentXP int
	IsKO      bool
}

// NewState creates a character state with no home point set.
func NewState(xp int) *State { return &State{CurrentXP: xp} }

// SetHome registers a new Home Point crystal, replacing the previous one.
func (s *State) SetHome(sceneID int, pos mob.Pos) {
	s.Home = &Crystal{SceneID: sceneID, Pos: pos}
}

// ReturnHome teleports the character to their Home Point, deducting an XP penalty.
// Returns the crystal location and the actual XP penalty applied.
// Errors: ErrNoHomePoint if no home set, ErrNotKO if character is alive.
func (s *State) ReturnHome() (*Crystal, int, error) {
	if s.Home == nil {
		return nil, 0, ErrNoHomePoint
	}
	if !s.IsKO {
		return nil, 0, ErrNotKO
	}

	penalty := s.CurrentXP * DefaultXPPenaltyPct / 100
	if penalty < XPPenaltyMin {
		penalty = XPPenaltyMin
	}
	s.CurrentXP -= penalty
	if s.CurrentXP < 0 {
		s.CurrentXP = 0
	}
	s.IsKO = false
	return s.Home, penalty, nil
}

// KO marks the character as knocked out.
func (s *State) KO() { s.IsKO = true }

// HasHome reports whether a Home Point crystal has been set.
func (s *State) HasHome() bool { return s.Home != nil }

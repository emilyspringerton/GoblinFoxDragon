// Package combat — death.go: Player HP + KO/Raise system (FFXI-parity).
//
// # Death/Raise model
//
// A player's HP state machine has two states: Alive and KO.
// On taking lethal damage TakeKO() transitions to KO (HP=0). The player
// is not disconnected; they wait for a Raise or choose Home Point return.
// On Raise, HP is restored to 1 and an XP debt (penalty) is applied.
// The XP penalty is a percentage of the XP toward the next level.
//
// XP debt is stored and drained over subsequent kills (50% of each kill's
// XP goes to debt reduction) — this package only computes the initial debt.
package combat

import "errors"

const (
	// DefaultRaisePenaltyPct is the percentage of current-level XP lost on KO.
	// FFXI uses 2500 base XP at certain levels; we use a simpler percentage model.
	DefaultRaisePenaltyPct = 10 // 10% of XP toward next level
	RaiseHPRestore         = 1  // HP after Raise (must earn back via cures/rest)
)

var (
	ErrNotKO     = errors.New("player is not KO'd")
	ErrAlreadyKO = errors.New("player is already KO'd")
)

// HPState represents a player's HP and KO status.
// Not goroutine-safe; callers synchronise via server tick.
type HPState struct {
	Current int
	Max     int
	IsKO    bool
}

// NewHPState creates an alive HPState at full HP.
func NewHPState(maxHP int) *HPState {
	return &HPState{Current: maxHP, Max: maxHP}
}

// TakeDamage applies damage to the player.
// If HP drops to 0 or below, TakeKO() is called automatically.
// Returns true if the damage caused a KO.
func (h *HPState) TakeDamage(dmg int) (causedKO bool, err error) {
	if h.IsKO {
		return false, ErrAlreadyKO
	}
	h.Current -= dmg
	if h.Current <= 0 {
		h.Current = 0
		h.IsKO = true
		return true, nil
	}
	return false, nil
}

// Heal restores HP, clamped to Max. No-op if KO.
func (h *HPState) Heal(amount int) {
	if h.IsKO {
		return
	}
	h.Current += amount
	if h.Current > h.Max {
		h.Current = h.Max
	}
}

// TakeKO forces the player into KO state (HP=0) regardless of current HP.
// Used for instant-death mechanics (falling, doom, etc.).
// Returns ErrAlreadyKO if already down.
func (h *HPState) TakeKO() error {
	if h.IsKO {
		return ErrAlreadyKO
	}
	h.Current = 0
	h.IsKO = true
	return nil
}

// Raise restores the player from KO.
// HP is set to RaiseHPRestore (1).
// xpPenaltyPct is the percentage of currentXP to deduct (e.g. DefaultRaisePenaltyPct).
// Returns the XP penalty applied.
// Returns ErrNotKO if the player is not KO'd.
func (h *HPState) Raise(currentXP int, xpPenaltyPct float64) (int, error) {
	if !h.IsKO {
		return 0, ErrNotKO
	}
	h.Current = RaiseHPRestore
	h.IsKO = false

	penalty := int(float64(currentXP) * xpPenaltyPct / 100.0)
	if penalty < 0 {
		penalty = 0
	}
	return penalty, nil
}

// RaiseDefault calls Raise with DefaultRaisePenaltyPct.
func (h *HPState) RaiseDefault(currentXP int) (int, error) {
	return h.Raise(currentXP, DefaultRaisePenaltyPct)
}

// HPPct returns current HP as a percentage of max (0–100).
func (h *HPState) HPPct() int {
	if h.Max == 0 {
		return 0
	}
	return h.Current * 100 / h.Max
}

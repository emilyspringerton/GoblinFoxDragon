// Package duel implements the consensual PvP duel system for DragonsNShit MUD.
//
// A player challenges another. If the defender accepts within 60s, both enter
// Active duel state. The first player to reach 0 HP (or forfeit) loses.
// Ratings update on completion: winner +25, loser -10.
package duel

import (
	"errors"
	"fmt"
	"time"
)

// Phase is the lifecycle state of a duel.
type Phase string

const (
	PhasePending Phase = "Pending" // challenge issued, waiting for accept
	PhaseActive  Phase = "Active"  // both players are fighting
	PhaseDone    Phase = "Done"    // duel has ended
)

// State represents one active or pending duel.
type State struct {
	Challenger string    // slot key of challenger
	Defender   string    // slot key of defender
	Phase      Phase
	IssuedAt   time.Time // when the challenge was issued
	StartedAt  time.Time // when it became Active
	Winner     string    // slot key of winner; "" while pending/active
}

// Errors returned by duel operations.
var (
	ErrAlreadyDueling  = errors.New("player already in a duel")
	ErrNoPendingDuel   = errors.New("no pending duel challenge")
	ErrNotInDuel       = errors.New("not currently in a duel")
	ErrSelfDuel        = errors.New("cannot duel yourself")
	ErrChallengeExpired = errors.New("challenge expired")
)

// ChallengeTimeout is how long a pending challenge stays open.
const ChallengeTimeout = 60 * time.Second

// RatingChange applied to winner and loser.
const (
	WinRating  = 25
	LossRating = -10
)

// Manager tracks all pending and active duels.
type Manager struct {
	// pending: challengerSlot → *State
	pending map[string]*State
	// active: slot → *State (both challenger and defender map to the same *State)
	active map[string]*State
	// Ratings: slot → duel rating (starts at 1000 implicitly)
	Ratings map[string]int
}

// NewManager creates an empty Manager.
func NewManager() *Manager {
	return &Manager{
		pending: make(map[string]*State),
		active:  make(map[string]*State),
		Ratings: make(map[string]int),
	}
}

// InDuel reports whether slot is in any pending or active duel (as either side).
func (m *Manager) InDuel(slot string) bool {
	if _, ok := m.pending[slot]; ok {
		return true
	}
	if _, ok := m.active[slot]; ok {
		return true
	}
	// Also check as a pending defender.
	for _, st := range m.pending {
		if st.Defender == slot {
			return true
		}
	}
	return false
}

// ActiveDuel returns the active duel for slot, or nil.
func (m *Manager) ActiveDuel(slot string) *State {
	return m.active[slot]
}

// PendingFor returns the pending challenge where slot is the defender, or nil.
func (m *Manager) PendingFor(slot string) *State {
	for _, st := range m.pending {
		if st.Defender == slot {
			return st
		}
	}
	return nil
}

// Challenge issues a challenge from challenger to defender.
// Returns ErrSelfDuel, ErrAlreadyDueling, or nil.
func (m *Manager) Challenge(challenger, defender string, now time.Time) error {
	if challenger == defender {
		return ErrSelfDuel
	}
	if m.InDuel(challenger) {
		return fmt.Errorf("%w: %s", ErrAlreadyDueling, challenger)
	}
	if m.InDuel(defender) {
		return fmt.Errorf("%w: %s", ErrAlreadyDueling, defender)
	}
	st := &State{
		Challenger: challenger,
		Defender:   defender,
		Phase:      PhasePending,
		IssuedAt:   now,
	}
	m.pending[challenger] = st
	return nil
}

// Accept transitions the duel from Pending to Active.
// Returns ErrNoPendingDuel or ErrChallengeExpired.
func (m *Manager) Accept(defender string, now time.Time) (*State, error) {
	st := m.PendingFor(defender)
	if st == nil {
		return nil, ErrNoPendingDuel
	}
	if now.Sub(st.IssuedAt) > ChallengeTimeout {
		delete(m.pending, st.Challenger)
		return nil, ErrChallengeExpired
	}
	delete(m.pending, st.Challenger)
	st.Phase = PhaseActive
	st.StartedAt = now
	m.active[st.Challenger] = st
	m.active[st.Defender] = st
	return st, nil
}

// Forfeit ends the duel with the forfeiting player as loser.
// Returns ErrNotInDuel if slot is not in an active duel.
func (m *Manager) Forfeit(slot string, now time.Time) (*State, error) {
	st := m.active[slot]
	if st == nil {
		// Cancel a pending challenge they issued.
		if pst, ok := m.pending[slot]; ok {
			delete(m.pending, slot)
			pst.Phase = PhaseDone
			return pst, nil
		}
		return nil, ErrNotInDuel
	}
	winner := st.Challenger
	if slot == st.Challenger {
		winner = st.Defender
	}
	return m.finish(st, winner), nil
}

// ReportHP checks whether a duel has ended due to HP loss.
// challHP and defHP are the current HP of challenger and defender.
// Returns (winner slot, true) if the duel ended; ("", false) otherwise.
func (m *Manager) ReportHP(st *State, challHP, defHP int) (winner string, done bool) {
	if st.Phase != PhaseActive {
		return "", false
	}
	if challHP <= 0 {
		finished := m.finish(st, st.Defender)
		return finished.Winner, true
	}
	if defHP <= 0 {
		finished := m.finish(st, st.Challenger)
		return finished.Winner, true
	}
	return "", false
}

// finish records the winner and updates ratings.
func (m *Manager) finish(st *State, winner string) *State {
	loser := st.Challenger
	if winner == st.Challenger {
		loser = st.Defender
	}
	st.Winner = winner
	st.Phase = PhaseDone
	delete(m.active, st.Challenger)
	delete(m.active, st.Defender)

	m.Ratings[winner] += WinRating
	m.Ratings[loser] += LossRating
	return st
}

// Rating returns the duel rating for a slot (default 1000).
func (m *Manager) Rating(slot string) int {
	if r, ok := m.Ratings[slot]; ok {
		return 1000 + r
	}
	return 1000
}

// ExpirePending removes stale pending challenges older than ChallengeTimeout.
// Returns the list of challenger slots whose challenges were removed.
func (m *Manager) ExpirePending(now time.Time) []string {
	var expired []string
	for challenger, st := range m.pending {
		if now.Sub(st.IssuedAt) > ChallengeTimeout {
			delete(m.pending, challenger)
			expired = append(expired, challenger)
		}
	}
	return expired
}

// TopN returns up to n slots sorted by descending rating.
func (m *Manager) TopN(n int) []string {
	type entry struct {
		slot   string
		rating int
	}
	entries := make([]entry, 0, len(m.Ratings))
	for slot, delta := range m.Ratings {
		entries = append(entries, entry{slot: slot, rating: 1000 + delta})
	}
	// Sort descending.
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].rating > entries[i].rating {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	if n > len(entries) {
		n = len(entries)
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = entries[i].slot
	}
	return out
}

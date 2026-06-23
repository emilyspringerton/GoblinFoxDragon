// Package party implements the DragonsNShit 6-player party system.
//
// # Party model (FFXI-parity)
//
// A Party has one Leader and up to 5 additional Members (6 total).
// The leader can invite new members and kick existing ones.
// Any member can leave voluntarily; if the leader leaves, leadership transfers
// to the next member in join order.
//
// XP is split evenly among party members whose position is within range of the
// kill point. Out-of-range members receive nothing (standard FFXI rule).
//
// # Alliance model (S85-02)
//
// Three parties form an Alliance. The leader of party[0] is the Alliance leader.
// Alliance XP sharing is not handled here — each party splits independently.
package party

import (
	"errors"
)

// MaxPartySize is the maximum number of players in a party (leader + 5 members).
const MaxPartySize = 6

// Errors
var (
	ErrPartyFull       = errors.New("party is full")
	ErrAlreadyInParty  = errors.New("player already in party")
	ErrNotInParty      = errors.New("player not in party")
	ErrNotLeader       = errors.New("only the leader can do that")
	ErrCannotKickSelf  = errors.New("leader cannot kick themselves")
	ErrPartyEmpty      = errors.New("party has no members")
)

// Party is a group of up to MaxPartySize players.
// Slot strings are the server's canonical player identifiers (same as zone.Manager slots).
// Not goroutine-safe; callers synchronise via the server tick loop.
type Party struct {
	Leader  string   // slot of the current leader
	members []string // ordered list of non-leader members (join order)
}

// New creates a Party with the given slot as leader.
func New(leaderSlot string) *Party {
	return &Party{Leader: leaderSlot}
}

// Members returns a snapshot of non-leader member slots in join order.
func (p *Party) Members() []string {
	out := make([]string, len(p.members))
	copy(out, p.members)
	return out
}

// All returns all party members including the leader (leader first).
func (p *Party) All() []string {
	out := make([]string, 0, 1+len(p.members))
	out = append(out, p.Leader)
	out = append(out, p.members...)
	return out
}

// Size returns total party size (leader + members).
func (p *Party) Size() int { return 1 + len(p.members) }

// Full reports whether the party is at capacity.
func (p *Party) Full() bool { return p.Size() >= MaxPartySize }

// Has reports whether slot is in the party (leader or member).
func (p *Party) Has(slot string) bool {
	if p.Leader == slot {
		return true
	}
	for _, m := range p.members {
		if m == slot {
			return true
		}
	}
	return false
}

// Invite adds slot to the party as a member.
// Only valid if the caller is leader (callerSlot == p.Leader).
// Returns ErrNotLeader, ErrPartyFull, ErrAlreadyInParty on failure.
func (p *Party) Invite(callerSlot, newSlot string) error {
	if callerSlot != p.Leader {
		return ErrNotLeader
	}
	if p.Full() {
		return ErrPartyFull
	}
	if p.Has(newSlot) {
		return ErrAlreadyInParty
	}
	p.members = append(p.members, newSlot)
	return nil
}

// Kick removes slot from the party.
// Only the leader may kick; leader cannot kick themselves.
// Returns ErrNotLeader, ErrCannotKickSelf, ErrNotInParty on failure.
func (p *Party) Kick(callerSlot, targetSlot string) error {
	if callerSlot != p.Leader {
		return ErrNotLeader
	}
	if targetSlot == p.Leader {
		return ErrCannotKickSelf
	}
	for i, m := range p.members {
		if m == targetSlot {
			p.members = append(p.members[:i], p.members[i+1:]...)
			return nil
		}
	}
	return ErrNotInParty
}

// Leave removes slot from the party voluntarily.
// If the leader leaves, leadership passes to the first remaining member.
// Returns ErrNotInParty if slot is not in the party.
// Returns ErrPartyEmpty if the last member (leader) leaves — caller should disband.
func (p *Party) Leave(slot string) error {
	if slot == p.Leader {
		if len(p.members) == 0 {
			return ErrPartyEmpty // caller should disband the party
		}
		// Transfer leadership to first member.
		p.Leader = p.members[0]
		p.members = p.members[1:]
		return nil
	}
	for i, m := range p.members {
		if m == slot {
			p.members = append(p.members[:i], p.members[i+1:]...)
			return nil
		}
	}
	return ErrNotInParty
}

// XPSplit divides totalXP among party members who are within distFromKill units
// of the kill point. Members beyond rangeLimit receive nothing.
// Returns a slice parallel to All() (leader first, then members in join order).
// If no members are in range, returns all-zero slice.
// XP is divided evenly (integer division); remainder is discarded.
func (p *Party) XPSplit(totalXP int, memberDists []float64, rangeLimit float64) []int {
	all := p.All()
	result := make([]int, len(all))
	if len(memberDists) != len(all) {
		return result
	}
	inRange := 0
	for _, d := range memberDists {
		if d <= rangeLimit {
			inRange++
		}
	}
	if inRange == 0 {
		return result
	}
	each := totalXP / inRange
	for i, d := range memberDists {
		if d <= rangeLimit {
			result[i] = each
		}
	}
	return result
}

// ── Alliance ──────────────────────────────────────────────────────────────────

const MaxAllianceParties = 3

// ErrAllianceFull is returned when a third party attempts to join an alliance
// that already has MaxAllianceParties parties.
var ErrAllianceFull = errors.New("alliance is full")

// Alliance groups up to three parties (18 players max).
// Party at index 0 is the "main party" — its leader is the alliance leader.
type Alliance struct {
	parties [MaxAllianceParties]*Party
	count   int
}

// NewAlliance creates an alliance seeded with the given party as party[0].
func NewAlliance(p *Party) *Alliance {
	a := &Alliance{}
	a.parties[0] = p
	a.count = 1
	return a
}

// AddParty adds a party to the alliance. Returns ErrAllianceFull at capacity.
func (a *Alliance) AddParty(p *Party) error {
	if a.count >= MaxAllianceParties {
		return ErrAllianceFull
	}
	a.parties[a.count] = p
	a.count++
	return nil
}

// Party returns the party at the given index (0-based). Returns nil if out of range.
func (a *Alliance) Party(idx int) *Party {
	if idx < 0 || idx >= a.count {
		return nil
	}
	return a.parties[idx]
}

// AllianceLeader returns the slot of the alliance leader (leader of party[0]).
func (a *Alliance) AllianceLeader() string {
	if a.parties[0] == nil {
		return ""
	}
	return a.parties[0].Leader
}

// PartyCount returns how many parties are in the alliance.
func (a *Alliance) PartyCount() int { return a.count }

// TotalSize returns total player count across all alliance parties.
func (a *Alliance) TotalSize() int {
	n := 0
	for i := 0; i < a.count; i++ {
		n += a.parties[i].Size()
	}
	return n
}

// ── XP Chain (S85-03) ────────────────────────────────────────────────────────

// ChainTimeout is how long without a kill before the chain resets.
const ChainTimeout = 3 * 60 * 1e9 // 3 minutes in nanoseconds (time.Duration)

// ChainBonusPerKill is the XP bonus percentage added per consecutive kill.
const ChainBonusPerKill = 10

// ChainBonusCap is the maximum XP bonus percentage from chain kills.
const ChainBonusCap = 50

// XPChain tracks the kill-chain state for a party.
// Consecutive kills within ChainTimeout each add ChainBonusPerKill % bonus XP,
// capped at ChainBonusCap %.
type XPChain struct {
	Count      int   // current chain length (1 = first kill in chain)
	LastKillAt int64 // UnixNano of last kill; 0 = no chain active
}

// Record registers a kill at nowNano (time.Now().UnixNano()).
// Returns the XP bonus percentage to apply (0–ChainBonusCap).
func (c *XPChain) Record(nowNano int64) int {
	if c.LastKillAt != 0 && nowNano-c.LastKillAt > int64(ChainTimeout) {
		// Chain expired — reset.
		c.Count = 0
	}
	c.Count++
	c.LastKillAt = nowNano

	bonus := (c.Count - 1) * ChainBonusPerKill
	if bonus > ChainBonusCap {
		bonus = ChainBonusCap
	}
	return bonus
}

// Reset clears the chain state.
func (c *XPChain) Reset() {
	c.Count = 0
	c.LastKillAt = 0
}

// Expired reports whether the chain has timed out at nowNano.
func (c *XPChain) Expired(nowNano int64) bool {
	return c.LastKillAt != 0 && nowNano-c.LastKillAt > int64(ChainTimeout)
}

// BonusPct returns the current chain bonus percentage without advancing the chain.
func (c *XPChain) BonusPct() int {
	if c.Count <= 1 {
		return 0
	}
	b := (c.Count - 1) * ChainBonusPerKill
	if b > ChainBonusCap {
		return ChainBonusCap
	}
	return b
}

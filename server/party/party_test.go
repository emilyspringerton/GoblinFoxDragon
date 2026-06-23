package party

import (
	"testing"
	"time"
)

// ── Party basics ──────────────────────────────────────────────────────────────

func TestNew_LeaderSet(t *testing.T) {
	p := New("alice")
	if p.Leader != "alice" {
		t.Errorf("Leader=%q, want alice", p.Leader)
	}
}

func TestNew_SizeOne(t *testing.T) {
	p := New("alice")
	if p.Size() != 1 {
		t.Errorf("Size=%d, want 1", p.Size())
	}
}

func TestAll_LeaderFirst(t *testing.T) {
	p := New("alice")
	_ = p.Invite("alice", "bob")
	all := p.All()
	if all[0] != "alice" {
		t.Errorf("All()[0]=%q, want alice", all[0])
	}
	if all[1] != "bob" {
		t.Errorf("All()[1]=%q, want bob", all[1])
	}
}

func TestHas_Leader(t *testing.T) {
	p := New("alice")
	if !p.Has("alice") {
		t.Error("Has(leader) should be true")
	}
}

func TestHas_Member(t *testing.T) {
	p := New("alice")
	_ = p.Invite("alice", "bob")
	if !p.Has("bob") {
		t.Error("Has(member) should be true")
	}
}

func TestHas_Absent(t *testing.T) {
	p := New("alice")
	if p.Has("stranger") {
		t.Error("Has(absent) should be false")
	}
}

// ── Invite ────────────────────────────────────────────────────────────────────

func TestInvite_AddsToMembers(t *testing.T) {
	p := New("alice")
	if err := p.Invite("alice", "bob"); err != nil {
		t.Fatalf("Invite: %v", err)
	}
	if p.Size() != 2 {
		t.Errorf("Size after invite: %d", p.Size())
	}
}

func TestInvite_NotLeaderReturnsError(t *testing.T) {
	p := New("alice")
	_ = p.Invite("alice", "bob")
	if err := p.Invite("bob", "carol"); err != ErrNotLeader {
		t.Errorf("non-leader invite: got %v, want ErrNotLeader", err)
	}
}

func TestInvite_FullReturnsError(t *testing.T) {
	p := New("alice")
	for _, m := range []string{"b", "c", "d", "e", "f"} {
		_ = p.Invite("alice", m)
	}
	if p.Size() != MaxPartySize {
		t.Fatalf("Size=%d, want %d", p.Size(), MaxPartySize)
	}
	if err := p.Invite("alice", "g"); err != ErrPartyFull {
		t.Errorf("full invite: got %v, want ErrPartyFull", err)
	}
}

func TestInvite_DuplicateReturnsError(t *testing.T) {
	p := New("alice")
	_ = p.Invite("alice", "bob")
	if err := p.Invite("alice", "bob"); err != ErrAlreadyInParty {
		t.Errorf("duplicate invite: got %v, want ErrAlreadyInParty", err)
	}
}

func TestInvite_MaxSizeSix(t *testing.T) {
	p := New("alice")
	for i, m := range []string{"b", "c", "d", "e", "f"} {
		if err := p.Invite("alice", m); err != nil {
			t.Fatalf("invite %d failed: %v", i, err)
		}
	}
	if p.Size() != 6 {
		t.Errorf("max party size: got %d, want 6", p.Size())
	}
}

// ── Kick ─────────────────────────────────────────────────────────────────────

func TestKick_RemovesMember(t *testing.T) {
	p := New("alice")
	_ = p.Invite("alice", "bob")
	if err := p.Kick("alice", "bob"); err != nil {
		t.Fatalf("Kick: %v", err)
	}
	if p.Has("bob") {
		t.Error("bob should be gone after kick")
	}
	if p.Size() != 1 {
		t.Errorf("Size after kick: %d, want 1", p.Size())
	}
}

func TestKick_NotLeaderReturnsError(t *testing.T) {
	p := New("alice")
	_ = p.Invite("alice", "bob")
	_ = p.Invite("alice", "carol")
	if err := p.Kick("bob", "carol"); err != ErrNotLeader {
		t.Errorf("non-leader kick: got %v", err)
	}
}

func TestKick_SelfReturnsError(t *testing.T) {
	p := New("alice")
	if err := p.Kick("alice", "alice"); err != ErrCannotKickSelf {
		t.Errorf("self-kick: got %v, want ErrCannotKickSelf", err)
	}
}

func TestKick_AbsentReturnsError(t *testing.T) {
	p := New("alice")
	if err := p.Kick("alice", "ghost"); err != ErrNotInParty {
		t.Errorf("kick absent: got %v, want ErrNotInParty", err)
	}
}

// ── Leave ─────────────────────────────────────────────────────────────────────

func TestLeave_MemberLeaves(t *testing.T) {
	p := New("alice")
	_ = p.Invite("alice", "bob")
	if err := p.Leave("bob"); err != nil {
		t.Fatalf("Leave: %v", err)
	}
	if p.Has("bob") {
		t.Error("bob should be gone after leave")
	}
}

func TestLeave_LeaderTransfersLeadership(t *testing.T) {
	p := New("alice")
	_ = p.Invite("alice", "bob")
	_ = p.Invite("alice", "carol")
	if err := p.Leave("alice"); err != nil {
		t.Fatalf("leader leave: %v", err)
	}
	if p.Leader != "bob" {
		t.Errorf("new leader should be bob (first member), got %q", p.Leader)
	}
	if p.Has("alice") {
		t.Error("alice should be gone after leaving")
	}
}

func TestLeave_LastMemberReturnsEmpty(t *testing.T) {
	p := New("alice")
	if err := p.Leave("alice"); err != ErrPartyEmpty {
		t.Errorf("last member leave: got %v, want ErrPartyEmpty", err)
	}
}

func TestLeave_AbsentReturnsError(t *testing.T) {
	p := New("alice")
	if err := p.Leave("ghost"); err != ErrNotInParty {
		t.Errorf("absent leave: got %v, want ErrNotInParty", err)
	}
}

// ── XPSplit ───────────────────────────────────────────────────────────────────

func TestXPSplit_EvenSplit(t *testing.T) {
	p := New("alice")
	_ = p.Invite("alice", "bob")
	_ = p.Invite("alice", "carol")
	// All 3 in range.
	dists := []float64{5.0, 5.0, 5.0}
	got := p.XPSplit(300, dists, 20.0)
	for i, v := range got {
		if v != 100 {
			t.Errorf("XPSplit[%d]=%d, want 100", i, v)
		}
	}
}

func TestXPSplit_OutOfRangeExcluded(t *testing.T) {
	p := New("alice")
	_ = p.Invite("alice", "bob")
	_ = p.Invite("alice", "carol")
	// carol is out of range.
	dists := []float64{5.0, 5.0, 25.0}
	got := p.XPSplit(300, dists, 20.0)
	if got[0] != 150 || got[1] != 150 {
		t.Errorf("in-range split: alice=%d bob=%d, want 150 each", got[0], got[1])
	}
	if got[2] != 0 {
		t.Errorf("out-of-range carol: got %d, want 0", got[2])
	}
}

func TestXPSplit_NoneInRange(t *testing.T) {
	p := New("alice")
	_ = p.Invite("alice", "bob")
	dists := []float64{100.0, 100.0}
	got := p.XPSplit(300, dists, 20.0)
	for i, v := range got {
		if v != 0 {
			t.Errorf("all OOR: got[%d]=%d, want 0", i, v)
		}
	}
}

func TestXPSplit_SingleMemberGetsAll(t *testing.T) {
	p := New("alice")
	dists := []float64{0.0}
	got := p.XPSplit(1000, dists, 20.0)
	if got[0] != 1000 {
		t.Errorf("solo XP: got %d, want 1000", got[0])
	}
}

func TestXPSplit_WrongDistLen(t *testing.T) {
	p := New("alice")
	got := p.XPSplit(300, []float64{1.0, 2.0}, 20.0)
	for _, v := range got {
		if v != 0 {
			t.Errorf("wrong dist len: expected zero split, got %d", v)
		}
	}
}

// ── Alliance ─────────────────────────────────────────────────────────────────

func TestAlliance_LeaderIsParty0Leader(t *testing.T) {
	p0 := New("alice")
	a := NewAlliance(p0)
	if a.AllianceLeader() != "alice" {
		t.Errorf("AllianceLeader=%q, want alice", a.AllianceLeader())
	}
}

func TestAlliance_AddParty(t *testing.T) {
	p0 := New("alice")
	p1 := New("bob")
	a := NewAlliance(p0)
	if err := a.AddParty(p1); err != nil {
		t.Fatalf("AddParty: %v", err)
	}
	if a.PartyCount() != 2 {
		t.Errorf("PartyCount=%d, want 2", a.PartyCount())
	}
}

func TestAlliance_CapAtThreeParties(t *testing.T) {
	p0, p1, p2, p3 := New("a"), New("b"), New("c"), New("d")
	a := NewAlliance(p0)
	_ = a.AddParty(p1)
	_ = a.AddParty(p2)
	if err := a.AddParty(p3); err != ErrAllianceFull {
		t.Errorf("4th party: got %v, want ErrAllianceFull", err)
	}
	if a.PartyCount() != 3 {
		t.Errorf("PartyCount=%d, want 3", a.PartyCount())
	}
}

func TestAlliance_TotalSize(t *testing.T) {
	p0 := New("a")
	_ = p0.Invite("a", "b")
	p1 := New("c")
	a := NewAlliance(p0)
	_ = a.AddParty(p1)
	if a.TotalSize() != 3 {
		t.Errorf("TotalSize=%d, want 3", a.TotalSize())
	}
}

func TestAlliance_MaxEighteen(t *testing.T) {
	ps := [3]*Party{New("a"), New("b"), New("c")}
	for _, p := range ps {
		for i := 0; i < MaxPartySize-1; i++ {
			_ = p.Invite(p.Leader, p.Leader+string(rune('0'+i)))
		}
	}
	a := NewAlliance(ps[0])
	_ = a.AddParty(ps[1])
	_ = a.AddParty(ps[2])
	if a.TotalSize() != 18 {
		t.Errorf("max alliance size=%d, want 18", a.TotalSize())
	}
}

func TestAlliance_Party0AllianceLeaderAfterPartyLeaderChange(t *testing.T) {
	p0 := New("alice")
	_ = p0.Invite("alice", "bob")
	a := NewAlliance(p0)
	// alice leaves p0 → bob becomes p0 leader → alliance leader changes
	_ = p0.Leave("alice")
	if a.AllianceLeader() != "bob" {
		t.Errorf("alliance leader after transfer: got %q, want bob", a.AllianceLeader())
	}
}

// ── XPChain ──────────────────────────────────────────────────────────────────

func TestXPChain_FirstKillZeroBonus(t *testing.T) {
	var c XPChain
	bonus := c.Record(time.Now().UnixNano())
	if bonus != 0 {
		t.Errorf("first kill bonus: got %d, want 0", bonus)
	}
}

func TestXPChain_SecondKillTenPercent(t *testing.T) {
	var c XPChain
	now := time.Now().UnixNano()
	c.Record(now)
	bonus := c.Record(now + int64(time.Second))
	if bonus != ChainBonusPerKill {
		t.Errorf("chain 2 bonus: got %d, want %d", bonus, ChainBonusPerKill)
	}
}

func TestXPChain_BonusCappedAt50(t *testing.T) {
	var c XPChain
	now := time.Now().UnixNano()
	var lastBonus int
	for i := 0; i < 10; i++ {
		lastBonus = c.Record(now + int64(i)*int64(time.Second))
	}
	if lastBonus != ChainBonusCap {
		t.Errorf("chain cap: got %d, want %d", lastBonus, ChainBonusCap)
	}
}

func TestXPChain_ResetsAfterTimeout(t *testing.T) {
	var c XPChain
	now := time.Now().UnixNano()
	c.Record(now)
	c.Record(now + int64(time.Second))
	// 4 minutes later — chain should have expired.
	after := now + int64(4*time.Minute)
	bonus := c.Record(after)
	if bonus != 0 {
		t.Errorf("after timeout: bonus=%d, want 0 (chain reset)", bonus)
	}
	if c.Count != 1 {
		t.Errorf("Count after reset: %d, want 1", c.Count)
	}
}

func TestXPChain_Expired(t *testing.T) {
	var c XPChain
	now := time.Now().UnixNano()
	c.Record(now)
	if c.Expired(now + int64(time.Minute)) {
		t.Error("should not be expired after 1 min (timeout=3 min)")
	}
	if !c.Expired(now + int64(4*time.Minute)) {
		t.Error("should be expired after 4 min")
	}
}

func TestXPChain_Reset(t *testing.T) {
	var c XPChain
	c.Record(time.Now().UnixNano())
	c.Reset()
	if c.Count != 0 || c.LastKillAt != 0 {
		t.Error("Reset should zero Count and LastKillAt")
	}
}

func TestXPChain_BonusPct(t *testing.T) {
	var c XPChain
	now := time.Now().UnixNano()
	c.Record(now)
	c.Record(now + int64(time.Second))
	c.Record(now + 2*int64(time.Second))
	// Count=3, so BonusPct = (3-1)*10 = 20%
	if b := c.BonusPct(); b != 20 {
		t.Errorf("BonusPct: got %d, want 20", b)
	}
}

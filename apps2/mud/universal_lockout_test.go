package main

import (
	"testing"
	"time"

	"dragonsnshit/server/job"
)

// JOB_SPELL_SYSTEM_NORTHSTAR.md §1 tests -- founder direction 2026-09-12: "do a 1 s lockout."

func TestCheckUniversalLockout_FirstActionNeverBlocked(t *testing.T) {
	p := newTestPlayer() // p.lastActionAt is the zero value -- "never acted"
	if !p.checkUniversalLockout() {
		t.Fatal("a brand-new player's first-ever action was blocked by the universal lockout")
	}
}

func TestCheckUniversalLockout_BlocksWithinOneSecond(t *testing.T) {
	p := newTestPlayer()
	if !p.checkUniversalLockout() {
		t.Fatal("first action should succeed")
	}
	if p.checkUniversalLockout() {
		t.Fatal("a second action immediately after the first should be blocked by the 1s lockout")
	}
}

func TestCheckUniversalLockout_AllowsAfterOneSecond(t *testing.T) {
	p := newTestPlayer()
	// Backdate lastActionAt instead of a real sleep -- deterministic, no flaky real-time wait.
	p.lastActionAt = time.Now().Add(-universalLockout - time.Millisecond)
	if !p.checkUniversalLockout() {
		t.Fatal("an action a full second (plus slack) after the last one should be allowed")
	}
}

func TestCheckUniversalLockout_BeingBlockedDoesNotExtendTheWindow(t *testing.T) {
	// Real, deliberate property named in the function's own doc comment: spamming during
	// lockout must not perpetually push the window back -- only a check that actually succeeds
	// advances p.lastActionAt.
	p := newTestPlayer()
	start := time.Now().Add(-800 * time.Millisecond) // 800ms into the 1s window already
	p.lastActionAt = start

	for i := 0; i < 5; i++ {
		if p.checkUniversalLockout() {
			t.Fatalf("attempt %d: should still be locked out (800ms < 1s)", i)
		}
	}
	if !p.lastActionAt.Equal(start) {
		t.Errorf("lastActionAt changed from repeated blocked attempts: got %v, want unchanged %v", p.lastActionAt, start)
	}
}

func TestCheckUniversalLockout_SuccessAdvancesTheWindow(t *testing.T) {
	p := newTestPlayer()
	p.lastActionAt = time.Now().Add(-universalLockout - time.Millisecond)
	before := time.Now()
	if !p.checkUniversalLockout() {
		t.Fatal("expected success")
	}
	if p.lastActionAt.Before(before) {
		t.Errorf("a successful check should advance lastActionAt to ~now, got %v (before check started at %v)", p.lastActionAt, before)
	}
	// Immediately after succeeding, the player is locked out again for a fresh window.
	if p.checkUniversalLockout() {
		t.Fatal("expected the fresh window to immediately lock out a follow-up attempt")
	}
}

// TestCmdJA_RespectsUniversalLockout and TestCmdCast_RespectsUniversalLockout confirm both real
// entry points actually call checkUniversalLockout first, not just that the helper itself works
// in isolation -- a real, found-live risk class in this file (a helper existing but not actually
// wired into its real call site) this session has hit before.
func TestCmdJA_RespectsUniversalLockout(t *testing.T) {
	p := newTestPlayer()
	p.jobID = job.MNK
	p.charXP.Level = 1
	p.recastTracker = job.NewRecastTracker(job.MonkAbilities()) // chakra: no target/combat needed
	p.hp = 50
	p.maxHP = 100
	p.lastActionAt = time.Now() // locked out right now

	cmdJA(p, "chakra")
	if p.hp != 50 {
		t.Errorf("chakra while locked out healed anyway (hp went from 50 to %d) -- checkUniversalLockout isn't actually gating cmdJA", p.hp)
	}

	p.lastActionAt = time.Time{} // clear the lockout
	cmdJA(p, "chakra")
	if p.hp == 50 {
		t.Error("chakra with the lockout cleared did not heal at all -- test setup problem, not what this test means to check")
	}
}

func TestCmdCast_RespectsUniversalLockout(t *testing.T) {
	p := newTestPlayer()
	p.mp = 100                  // enough for sneak's 50 MP cost -- isolates the lockout, not a separate MP failure
	p.lastActionAt = time.Now() // locked out right now

	cmdCast(p, "sneak", "")
	if p.isSneaking {
		t.Error("cast sneak while locked out should not have applied Sneak -- checkUniversalLockout isn't actually gating cmdCast")
	}

	p.lastActionAt = time.Time{} // clear the lockout
	cmdCast(p, "sneak", "")
	if !p.isSneaking {
		t.Error("cast sneak with the lockout cleared and enough MP did not apply Sneak -- test setup problem, not what this test means to check")
	}
}

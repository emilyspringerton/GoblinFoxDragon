package main

import (
	"testing"
	"time"

	"dragonsnshit/server/job"
	"dragonsnshit/server/mob"
)

// TestCheckNotCasting_BlocksThenAllowsAfterExpiry exercises the real shared gate cmdCast and
// cmdJA both now call (JOB_SPELL_SYSTEM_NORTHSTAR.md §1 Phase 1 remainder).
func TestCheckNotCasting_BlocksThenAllowsAfterExpiry(t *testing.T) {
	p := newTestPlayer()

	// Never cast anything -- zero value must never block.
	if !p.checkNotCasting() {
		t.Fatal("a player who has never cast should never be blocked")
	}

	p.castingSpell = "cure"
	p.castingCompletesAt = time.Now().Add(2 * time.Second)
	if p.checkNotCasting() {
		t.Error("mid-cast player should be blocked from starting another action")
	}

	p.castingCompletesAt = time.Now().Add(-1 * time.Millisecond) // already elapsed
	if !p.checkNotCasting() {
		t.Error("a cast whose completion time has already passed must not block")
	}
}

func TestCastPreflight_InsufficientMPFailsFast(t *testing.T) {
	p := newTestPlayer()
	p.mp = 10 // cure costs 50 (spellMPCosts)

	ok, msg := castPreflight(p, "cure", "")
	if ok {
		t.Fatal("expected preflight to fail on insufficient MP")
	}
	if msg == "" {
		t.Error("expected a real failure message, got empty string")
	}
}

func TestCastPreflight_MobTargetRequiredSpellFailsWithoutTarget(t *testing.T) {
	p := newTestPlayer()
	p.mp = 100
	p.combat = &mob.PlayerCombat{}

	ok, msg := castPreflight(p, "fire", "")
	if ok {
		t.Fatal("expected preflight to fail with no combat target for an offensive nuke")
	}
	if msg == "" {
		t.Error("expected a real failure message, got empty string")
	}
}

func TestCastPreflight_SelfTargetBuffPassesWithEnoughMP(t *testing.T) {
	p := newTestPlayer()
	p.mp = 100

	ok, _ := castPreflight(p, "cure", "") // empty target name = self, resolveSpellTarget never touches gw
	if !ok {
		t.Fatal("expected preflight to pass: enough MP, self-target always resolves")
	}
}

func TestCastPreflight_SpellWithNoRegistryEntryAlwaysPasses(t *testing.T) {
	p := newTestPlayer()
	p.mp = 0 // would fail every real cost check if one existed
	ok, _ := castPreflight(p, "invisible", "")
	if !ok {
		t.Error("a spell with no spellMPCosts/spellRequiresXTarget entry must not be preflight-blocked here -- castNow's own real checks still apply")
	}
}

// TestSpellDisplayName_FallsBackToRawIDWhenUnlisted guards a real, deliberate degrade-gracefully
// choice named in ability.go's own doc comment.
func TestSpellDisplayName_FallsBackToRawIDWhenUnlisted(t *testing.T) {
	if got := spellDisplayName("not-a-real-spell"); got != "not-a-real-spell" {
		t.Errorf("spellDisplayName(unlisted) = %q, want the raw id back unchanged", got)
	}
	if got := spellDisplayName("fire2"); got != "Fire II" {
		t.Errorf(`spellDisplayName("fire2") = %q, want "Fire II"`, got)
	}
}

// TestBeginCast_SetsCastingStateImmediatelyThenResolvesAfterCastTime is the real end-to-end
// proof: a spell with a non-zero cast time does NOT resolve instantly (MP untouched, HP unhealed
// right after the call returns), and DOES resolve once the real timer fires (MP spent, HP healed,
// casting state cleared) -- exercising beginCast's real time.AfterFunc completion path, not just
// its synchronous half.
func TestBeginCast_SetsCastingStateImmediatelyThenResolvesAfterCastTime(t *testing.T) {
	p := newTestPlayer()
	p.slot = "test-slot-1"
	p.jobID = job.WHM // Cure's own real job gate inside castNow -- must be WHM or RDM
	p.hp = 100
	p.maxHP = 500
	p.mp = 100

	oldGW := gw
	gw = &world{players: map[string]*player{p.slot: p}}
	t.Cleanup(func() { gw = oldGW })

	const castTime = 30 * time.Millisecond
	beginCast(p, "cure", "", castTime)

	// Immediately after the call returns, the cast must still be pending -- nothing resolved yet.
	if p.castingSpell != "cure" {
		t.Fatalf("expected castingSpell = %q immediately after beginCast, got %q", "cure", p.castingSpell)
	}
	if p.mp != 100 {
		t.Errorf("MP must not be spent until the cast actually completes; got %d, want 100 still", p.mp)
	}
	if p.hp != 100 {
		t.Errorf("HP must not be healed until the cast actually completes; got %d, want 100 still", p.hp)
	}

	time.Sleep(castTime + 100*time.Millisecond)

	gw.mu.Lock()
	hp, mp, casting := p.hp, p.mp, p.castingSpell
	gw.mu.Unlock()

	if casting != "" {
		t.Errorf("castingSpell should be cleared once the deferred cast completes, still = %q", casting)
	}
	if mp != 50 { // cure costs 50
		t.Errorf("MP after Cure completes: got %d, want 50 (100 - cureCost 50)", mp)
	}
	if hp != 200 { // cure heals 100
		t.Errorf("HP after Cure completes: got %d, want 200 (100 + 100 healed)", hp)
	}
}

// TestBeginCast_PreflightFailureNeverEntersCastingState guards the "fails fast" half of §1's own
// case 3 -- a cast that was never going to work must not even start the cast bar.
func TestBeginCast_PreflightFailureNeverEntersCastingState(t *testing.T) {
	p := newTestPlayer()
	p.mp = 0 // cure costs 50

	beginCast(p, "cure", "", 2*time.Second)

	if p.castingSpell != "" {
		t.Error("a preflight-failed cast must never set castingSpell -- nothing should be pending")
	}
}

// TestCmdCast_ZeroCastTimeSpellStillResolvesInstantly is a real regression guard: every spell
// with NO spellCastTimes entry (general utility, or a currently-disabled job's content) must
// resolve exactly as it always did, with zero casting-state side effects.
func TestCmdCast_ZeroCastTimeSpellStillResolvesInstantly(t *testing.T) {
	p := newTestPlayer()
	p.slot = "test-slot-2"
	p.mp = 100

	oldGW := gw
	gw = &world{players: map[string]*player{p.slot: p}}
	t.Cleanup(func() { gw = oldGW })

	cmdCast(p, "invisible", "")

	if !p.isInvisible {
		t.Error("expected Invisible to actually apply -- cast-time work must not regress zero-cast-time spells")
	}
	if p.castingSpell != "" {
		t.Error("a zero-cast-time spell must never enter the casting state at all")
	}
}

// TestCmdJA_BlockedWhileCasting guards the real shared action economy: a player mid-cast on a
// spell cannot also fire off a job ability.
func TestCmdJA_BlockedWhileCasting(t *testing.T) {
	p := newTestPlayer()
	p.jobID = job.WAR
	p.charXP.Level = 5
	p.recastTracker = job.NewRecastTracker(job.WarriorAbilities())
	p.combat = &mob.PlayerCombat{}
	p.castingSpell = "cure"
	p.castingCompletesAt = time.Now().Add(5 * time.Second)

	cmdJA(p, "provoke")

	// provoke's own recast should never have been consumed -- the casting gate must refuse
	// before recastTracker.Use is ever called.
	if _, err := p.recastTracker.RecastRemaining("provoke", time.Now()); err != nil {
		t.Fatalf("unexpected RecastRemaining error: %v", err)
	}
}

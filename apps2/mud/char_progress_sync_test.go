package main

import (
	"testing"

	"dragonsnshit/server/xp"
)

// char_progress_sync_test.go tests the pure decision logic (needsProgressBaseline/
// baselineCharProgress/needsLevelSync/flowSyncDelta) directly, without touching gw.iduna --
// gw.iduna is a concrete *idunaclient.Client (not an interface), so no real call site in this
// file has ever been unit-tested against a fake; this matches that same established boundary.
// syncCharLevelAndFlow itself is thin, untested-directly wiring on top of these pure pieces
// (mirroring the already-shipped, already-proven headless sync it generalizes).

func TestNeedsProgressBaseline_TrueForFreshPlayer(t *testing.T) {
	p := &player{charXP: &xp.CharXP{Level: 8}}
	if !needsProgressBaseline(p) {
		t.Error("expected true: headlessSyncedLevel is still its Go zero value")
	}
}

func TestNeedsProgressBaseline_FalseAfterBaselining(t *testing.T) {
	p := &player{charXP: &xp.CharXP{Level: 8, CurrentXP: 2321}, flow: 500}
	baselineCharProgress(p)
	if needsProgressBaseline(p) {
		t.Error("expected false after baselineCharProgress")
	}
}

func TestBaselineCharProgress_CapturesCurrentValuesExactly(t *testing.T) {
	p := &player{charXP: &xp.CharXP{Level: 8, CurrentXP: 2321}, flow: 500}
	baselineCharProgress(p)
	if p.headlessSyncedLevel != 8 || p.headlessSyncedXP != 2321 || p.headlessSyncedFlow != 500 {
		t.Errorf("baseline mismatch: level=%d xp=%d flow=%d", p.headlessSyncedLevel, p.headlessSyncedXP, p.headlessSyncedFlow)
	}
}

// TestNeedsLevelSync_DetectsTheExactFounderReportedScenario guards the real regression: a
// player who leveled from 8 to 11 (WAR, founder's own real report) before an abrupt disconnect
// must be detected as needing a sync -- this is the check that, generalized to every tick for
// every player (not just headless), fixes the "lost 3 levels of real progress" bug.
func TestNeedsLevelSync_DetectsTheExactFounderReportedScenario(t *testing.T) {
	p := &player{charXP: &xp.CharXP{Level: 8, CurrentXP: 2321}}
	baselineCharProgress(p)

	p.charXP.Level = 11
	p.charXP.CurrentXP = 3900
	if !needsLevelSync(p) {
		t.Error("expected needsLevelSync to detect the real level-up from 8 to 11")
	}
}

func TestNeedsLevelSync_FalseWhenUnchanged(t *testing.T) {
	p := &player{charXP: &xp.CharXP{Level: 5, CurrentXP: 100}}
	baselineCharProgress(p)
	if needsLevelSync(p) {
		t.Error("expected false: nothing changed since baseline")
	}
}

func TestNeedsLevelSync_DetectsXPChangeEvenWithoutALevelUp(t *testing.T) {
	p := &player{charXP: &xp.CharXP{Level: 5, CurrentXP: 100}}
	baselineCharProgress(p)
	p.charXP.CurrentXP = 250 // same level, more XP toward the next one
	if !needsLevelSync(p) {
		t.Error("expected true: CurrentXP changed even though Level didn't")
	}
}

func TestFlowSyncDelta_ZeroWhenUnchanged(t *testing.T) {
	p := &player{charXP: &xp.CharXP{}, flow: 100}
	baselineCharProgress(p)
	if got := flowSyncDelta(p); got != 0 {
		t.Errorf("flowSyncDelta = %d, want 0", got)
	}
}

func TestFlowSyncDelta_PositiveWhenFlowIncreased(t *testing.T) {
	p := &player{charXP: &xp.CharXP{}, flow: 100}
	baselineCharProgress(p)
	p.flow = 250
	if got := flowSyncDelta(p); got != 150 {
		t.Errorf("flowSyncDelta = %d, want 150", got)
	}
}

func TestFlowSyncDelta_NegativeWhenFlowDecreased(t *testing.T) {
	p := &player{charXP: &xp.CharXP{}, flow: 100}
	baselineCharProgress(p)
	p.flow = 40
	if got := flowSyncDelta(p); got != -60 {
		t.Errorf("flowSyncDelta = %d, want -60", got)
	}
}

// TestMiningSkillDelta/TestFishingSkillDelta guard the real fix for the founder's own follow-up
// ask: "can we make sure fishing skill persists too?" -- unlike level, 0.0 is a real, legitimate
// "never trained" value here, so there's no baseline sentinel branch to test; every real connect
// path is responsible for explicitly setting syncedMiningSkill/syncedFishingSkill to match
// whatever was loaded from IDUNA (see getOrCreateHeadlessPlayer/handleConn).

func TestMiningSkillDelta_ZeroForFreshUntrainedPlayer(t *testing.T) {
	p := &player{}
	if got := miningSkillDelta(p); got != 0 {
		t.Errorf("miningSkillDelta(untrained) = %v, want 0", got)
	}
}

func TestMiningSkillDelta_DetectsRealGain(t *testing.T) {
	p := &player{miningSkill: 42.5, syncedMiningSkill: 42.0}
	if got := miningSkillDelta(p); got != 0.5 {
		t.Errorf("miningSkillDelta = %v, want 0.5", got)
	}
}

func TestFishingSkillDelta_DetectsRealGain(t *testing.T) {
	p := &player{fishingSkill: 17.5, syncedFishingSkill: 17.0}
	if got := fishingSkillDelta(p); got != 0.5 {
		t.Errorf("fishingSkillDelta = %v, want 0.5", got)
	}
}

func TestFishingSkillDelta_ZeroAfterMatchingBaseline(t *testing.T) {
	// Exactly what a real connect path does: load the persisted value into both the live field
	// and its synced mirror together, so the very first tick sees no false delta.
	p := &player{}
	loaded := 23.5
	p.fishingSkill = loaded
	p.syncedFishingSkill = loaded
	if got := fishingSkillDelta(p); got != 0 {
		t.Errorf("fishingSkillDelta right after loading = %v, want 0", got)
	}
}

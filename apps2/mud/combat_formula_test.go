package main

import (
	"math/rand"
	"testing"

	"dragonsnshit/server/job"
	"dragonsnshit/server/rng"
)

// withSeededCombatRNG swaps combatRNG for a seeded one for the duration of one test, restoring
// the real source after -- deterministic rolls without permanently changing this file's own
// package-level RNG for anything else.
func withSeededCombatRNG(t *testing.T, seed int64) {
	t.Helper()
	old := combatRNG
	combatRNG = rand.New(rand.NewSource(seed))
	t.Cleanup(func() { combatRNG = old })
}

func TestClampPercent(t *testing.T) {
	if got := clampPercent(30, 50, 99); got != 50 {
		t.Errorf("clampPercent(30, 50, 99) = %d, want 50", got)
	}
	if got := clampPercent(150, 50, 99); got != 99 {
		t.Errorf("clampPercent(150, 50, 99) = %d, want 99", got)
	}
	if got := clampPercent(75, 50, 99); got != 75 {
		t.Errorf("clampPercent(75, 50, 99) = %d, want 75 (within range, unchanged)", got)
	}
}

func TestRollDamageVariance_StaysWithinBand(t *testing.T) {
	withSeededCombatRNG(t, 1)
	mean := 100
	lo := int(float64(mean) * combatDamageVarianceLow)
	hi := int(float64(mean) * combatDamageVarianceHigh)
	for i := 0; i < 1000; i++ {
		got := rollDamageVariance(mean)
		if got < lo || got > hi {
			t.Fatalf("rollDamageVariance(%d) = %d, want within [%d, %d]", mean, got, lo, hi)
		}
	}
}

func TestRollDamageVariance_ZeroMeanReturnsZero(t *testing.T) {
	if got := rollDamageVariance(0); got != 0 {
		t.Errorf("rollDamageVariance(0) = %d, want 0", got)
	}
}

func TestReduceIncomingDamage_NeverBelowOne(t *testing.T) {
	// A huge VIT must still leave at least 1 damage on a landed hit -- a hit that connects
	// always does *something*, per §5's own explicit requirement.
	if got := reduceIncomingDamage(10, 1000); got != 1 {
		t.Errorf("reduceIncomingDamage(10, 1000) = %d, want 1 (floor)", got)
	}
}

func TestReduceIncomingDamage_RealReduction(t *testing.T) {
	// VIT 20 -> reduces by 10 (VIT/2).
	if got := reduceIncomingDamage(50, 20); got != 40 {
		t.Errorf("reduceIncomingDamage(50, 20) = %d, want 40", got)
	}
}

func TestPityRoll_DesiredOutcomeMoreLikelyAfterDrought(t *testing.T) {
	withSeededCombatRNG(t, 7)
	// A 1% base chance, with the desired outcome already in a full pity drought, should win far
	// more than 1% of the time -- the real point of the mechanism, same structural check
	// server/rng's own tests already establish, exercised here through this file's own wrapper.
	wins := 0
	const trials = 5000
	for i := 0; i < trials; i++ {
		pity := [2]int{rng.MaxPityTier, 0}
		if pityRoll(1, &pity) {
			wins++
		}
	}
	if wins < trials/10 {
		t.Errorf("pityRoll(1%%, full drought) won %d/%d trials -- expected well above a flat 1%%, pity isn't boosting it", wins, trials)
	}
}

func TestResolvePlayerAutoAttackDamage_MissAlwaysReportsZeroDamage(t *testing.T) {
	withSeededCombatRNG(t, 1)
	p := newTestPlayer()
	p.jobID = job.WAR
	// A real hit-chance base of 90%+ means most of these land -- run enough rolls to see at
	// least one real miss too (repeated hits drive Miss's own pity up over time, per
	// pityRoll's own "winner resets to 0, loser increments" mechanism, so a miss becomes more
	// likely the longer a streak of hits runs), and check every single roll's own shape
	// invariant rather than asserting on one cherry-picked call.
	sawMiss := false
	for i := 0; i < 100; i++ {
		hit, crit, dmg := resolvePlayerAutoAttackDamage(p, 30)
		if !hit {
			sawMiss = true
			if crit || dmg != 0 {
				t.Errorf("a miss must report crit=false, damage=0; got crit=%v damage=%d", crit, dmg)
			}
		}
	}
	if !sawMiss {
		t.Error("expected at least one miss over 100 auto-attack rolls -- suspiciously deterministic")
	}
}

func TestResolvePlayerAutoAttackDamage_STRScalesMeanDamage(t *testing.T) {
	withSeededCombatRNG(t, 3)
	pLowSTR := newTestPlayer()
	pLowSTR.jobID = job.WHM // real lowest-STR original-six job

	pHighSTR := newTestPlayer()
	pHighSTR.jobID = job.WAR // real highest-STR original-six job

	sumLow, sumHigh := 0, 0
	const trials = 500
	for i := 0; i < trials; i++ {
		_, _, dmg := resolvePlayerAutoAttackDamage(pLowSTR, 30)
		sumLow += dmg
		_, _, dmg = resolvePlayerAutoAttackDamage(pHighSTR, 30)
		sumHigh += dmg
	}
	if sumHigh <= sumLow {
		t.Errorf("WAR's real higher STR should deal more average damage than WHM's real lower STR over %d trials: WAR sum=%d, WHM sum=%d", trials, sumHigh, sumLow)
	}
}

func TestResolveMobAutoAttackDamage_HitAppliesVITReduction(t *testing.T) {
	withSeededCombatRNG(t, 9)
	pLowVIT := newTestPlayer()
	pLowVIT.jobID = job.BLM // real lowest-VIT original-six job

	pHighVIT := newTestPlayer()
	pHighVIT.jobID = job.WAR // real highest-VIT original-six job

	sumLow, sumHigh := 0, 0
	const trials = 500
	for i := 0; i < trials; i++ {
		hit, _, dmg := resolveMobAutoAttackDamage(pLowVIT, 40)
		if hit {
			sumLow += dmg
		}
		hit, _, dmg = resolveMobAutoAttackDamage(pHighVIT, 40)
		if hit {
			sumHigh += dmg
		}
	}
	if sumHigh >= sumLow {
		t.Errorf("WAR's real higher VIT should take less total damage than BLM's real lower VIT over %d trials: WAR sum=%d, BLM sum=%d", trials, sumHigh, sumLow)
	}
}

func TestResolveMobAutoAttackDamage_MissReportsNoDamage(t *testing.T) {
	withSeededCombatRNG(t, 1)
	p := newTestPlayer()
	p.jobID = job.WAR
	sawMiss := false
	for i := 0; i < 200; i++ {
		hit, crit, dmg := resolveMobAutoAttackDamage(p, 40)
		if !hit {
			sawMiss = true
			if crit || dmg != 0 {
				t.Errorf("a miss must report crit=false, damage=0; got crit=%v damage=%d", crit, dmg)
			}
		}
	}
	if !sawMiss {
		t.Error("expected at least one miss over 200 mob attack rolls at a ~85%% hit chance -- suspiciously deterministic")
	}
}

// JOB_SPELL_SYSTEM_NORTHSTAR.md §0.5 real-effect tests -- founder direct follow-up: "does boost
// increase your attack? does sneak attack guarantee a critical on the next hit...?"

func TestResolvePlayerAutoAttackDamage_PendingAttackBonusAppliesOnceThenClears(t *testing.T) {
	withSeededCombatRNG(t, 1)
	p := newTestPlayer()
	p.jobID = job.WAR
	// Force a guaranteed hit by giving Hit a full pity drought -- isolates the bonus itself from
	// whether this particular swing happened to land.
	p.pityOwnHit = [2]int{10, 0}
	p.pendingAttackBonus = combatBoostBonus

	hit, _, boosted := resolvePlayerAutoAttackDamage(p, 30)
	if !hit {
		t.Fatal("expected a guaranteed hit (full pity drought on Hit)")
	}
	if p.pendingAttackBonus != 0 {
		t.Error("pendingAttackBonus should be cleared after a landed hit consumes it")
	}

	// A second swing, immediately after, with the SAME seed reset must deal LESS damage than the
	// first (no bonus active anymore) -- a real, structural check that the bonus actually did
	// something, not just that the flag got cleared.
	withSeededCombatRNG(t, 1)
	p2 := newTestPlayer()
	p2.jobID = job.WAR
	p2.pityOwnHit = [2]int{10, 0}
	_, _, unboosted := resolvePlayerAutoAttackDamage(p2, 30)
	if boosted <= unboosted {
		t.Errorf("boosted damage (%d) should exceed the same roll unboosted (%d)", boosted, unboosted)
	}
}

func TestResolvePlayerAutoAttackDamage_PendingAttackBonusNotConsumedByAMiss(t *testing.T) {
	p := newTestPlayer()
	p.jobID = job.WAR
	p.pityOwnHit = [2]int{0, 10} // force a miss: Miss (index 1) fully favored
	p.pendingAttackBonus = combatBoostBonus

	hit, _, _ := resolvePlayerAutoAttackDamage(p, 30)
	if hit {
		t.Fatal("expected a guaranteed miss (full pity drought on Miss)")
	}
	if p.pendingAttackBonus != combatBoostBonus {
		t.Error("a missed swing must not consume pendingAttackBonus -- it should still be armed for the next real attempt")
	}
}

func TestResolvePlayerAutoAttackDamage_PendingGuaranteedCritAppliesOnceThenClears(t *testing.T) {
	withSeededCombatRNG(t, 2)
	p := newTestPlayer()
	p.jobID = job.WAR
	p.pityOwnHit = [2]int{10, 0}  // guaranteed hit
	p.pityOwnCrit = [2]int{0, 10} // crit itself fully NOT favored by the normal roll
	p.pendingGuaranteedCrit = true

	hit, crit, _ := resolvePlayerAutoAttackDamage(p, 30)
	if !hit {
		t.Fatal("expected a guaranteed hit")
	}
	if !crit {
		t.Error("pendingGuaranteedCrit should force a crit even though the normal crit roll fully favors Normal")
	}
	if p.pendingGuaranteedCrit {
		t.Error("pendingGuaranteedCrit should be cleared after a landed hit consumes it")
	}

	// Next swing (crit still disfavored, no more guarantee) should NOT crit.
	_, crit2, _ := resolvePlayerAutoAttackDamage(p, 30)
	if crit2 {
		t.Error("expected no crit on the follow-up swing -- the guarantee should already be consumed")
	}
}

func TestResolvePlayerAutoAttackDamage_PendingGuaranteedCritNotConsumedByAMiss(t *testing.T) {
	p := newTestPlayer()
	p.jobID = job.WAR
	p.pityOwnHit = [2]int{0, 10} // force a miss
	p.pendingGuaranteedCrit = true

	hit, _, _ := resolvePlayerAutoAttackDamage(p, 30)
	if hit {
		t.Fatal("expected a guaranteed miss")
	}
	if !p.pendingGuaranteedCrit {
		t.Error("a missed swing must not consume pendingGuaranteedCrit -- it should still be armed for the next real attempt")
	}
}

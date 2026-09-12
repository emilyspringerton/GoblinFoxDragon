package main

import (
	"math/rand"
	"testing"

	"dragonsnshit/server/gear"
	"dragonsnshit/server/itemdef"
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

// Real, found-live flake fixed (2026-09-12, same root cause as
// TestResolvePlayerAutoAttackDamage_PendingGuaranteedCritNotConsumedByAMiss's own doc comment
// just below in this file): pity[1]=10 heavily biases toward a miss, it does not guarantee one.
func TestResolvePlayerAutoAttackDamage_PendingAttackBonusNotConsumedByAMiss(t *testing.T) {
	withSeededCombatRNG(t, 5)
	p := newTestPlayer()
	p.jobID = job.WAR
	p.pityOwnHit = [2]int{0, 10} // heavily favors a miss, not a hard guarantee -- see above
	p.pendingAttackBonus = combatBoostBonus

	sawMiss := false
	for i := 0; i < 100; i++ {
		hit, _, _ := resolvePlayerAutoAttackDamage(p, 30)
		if !hit {
			sawMiss = true
			if p.pendingAttackBonus != combatBoostBonus {
				t.Error("a missed swing must not consume pendingAttackBonus -- it should still be armed for the next real attempt")
			}
			break
		}
		// A rare "surprise" hit along the way would consume pendingAttackBonus itself (real,
		// expected behavior on a landed hit) -- re-arm it so the loop keeps testing the real
		// invariant this test is about (a MISS never consumes it).
		p.pendingAttackBonus = combatBoostBonus
	}
	if !sawMiss {
		t.Fatal("expected at least one miss over 100 heavily-favored-miss rolls -- suspiciously deterministic")
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

// Real, found-live flake fixed (2026-09-12, hit intermittently 3+ times this same session):
// pity[1]=10 heavily BIASES toward a miss (rng.MarbleBagPick's own Fibonacci-weighted odds), it
// does not GUARANTEE one -- a single-shot "expected a guaranteed miss" assertion was always a
// real, if rare, false failure waiting to happen. Fixed the same way this file's own sibling
// tests already handle a biased-not-certain roll (TestResolvePlayerAutoAttackDamage_
// MissAlwaysReportsZeroDamage, above): loop until a real miss is observed (heavily favored, so
// this resolves in a handful of iterations essentially always) and assert the real invariant on
// that miss, with a real seed so the loop itself is reproducible.
func TestResolvePlayerAutoAttackDamage_PendingGuaranteedCritNotConsumedByAMiss(t *testing.T) {
	withSeededCombatRNG(t, 3)
	p := newTestPlayer()
	p.jobID = job.WAR
	p.pityOwnHit = [2]int{0, 10} // heavily favors a miss, not a hard guarantee -- see above
	p.pendingGuaranteedCrit = true

	sawMiss := false
	for i := 0; i < 100; i++ {
		hit, _, _ := resolvePlayerAutoAttackDamage(p, 30)
		if !hit {
			sawMiss = true
			if !p.pendingGuaranteedCrit {
				t.Error("a missed swing must not consume pendingGuaranteedCrit -- it should still be armed for the next real attempt")
			}
			break
		}
		// A rare "surprise" hit along the way would consume pendingGuaranteedCrit itself (real,
		// expected behavior on a landed hit) -- re-arm it so the loop keeps testing the real
		// invariant this test is actually about (a MISS never consumes it), not accidentally
		// depending on never landing a hit across 100 rolls.
		p.pendingGuaranteedCrit = true
	}
	if !sawMiss {
		t.Fatal("expected at least one miss over 100 heavily-favored-miss rolls -- suspiciously deterministic")
	}
}

// Real, found-live gap (2026-09-12, founder scoping a starting-weapon feature): equipping
// anything, including a weapon's own real "attack"/attribute stats, previously had ZERO effect
// on combat math -- ComputeStats was computed only to print a cosmetic "Stat changes:" line.
// These tests guard the real fix: playerCombatStats/equipAttackBonus actually reading equipped
// gear now.

func testRegistryWithSword(t *testing.T) *itemdef.Registry {
	t.Helper()
	reg := itemdef.NewRegistry()
	if err := reg.LoadJSON([]byte(`[{"id":3,"name":"Sword","category":"weapon",
		"equip_slots":["main","off"],"stack_size":1,"stats":{"attack":10,"str":1}}]`)); err != nil {
		t.Fatalf("load test sword: %v", err)
	}
	return reg
}

func TestPlayerCombatStats_EquippedItemAddsAttributeBonus(t *testing.T) {
	oldReg := itemdefReg
	itemdefReg = testRegistryWithSword(t)
	t.Cleanup(func() { itemdefReg = oldReg })

	unarmed := newTestPlayer()
	unarmed.jobID = job.WAR
	unarmed.equip = gear.NewEquipment()
	unarmedSTR := playerCombatStats(unarmed).STR

	armed := newTestPlayer()
	armed.jobID = job.WAR
	armed.equip = gear.NewEquipment()
	if err := armed.equip.Equip(gear.SlotMainHand, gear.ItemEntry{ItemID: "sword", DefID: 3}); err != nil {
		t.Fatalf("equip: %v", err)
	}
	armedSTR := playerCombatStats(armed).STR

	if armedSTR != unarmedSTR+1 {
		t.Errorf("STR with Sword equipped (+1 str) = %d, want %d (unarmed %d + 1)", armedSTR, unarmedSTR+1, unarmedSTR)
	}
}

func TestPlayerCombatStats_NilEquipDoesNotPanic(t *testing.T) {
	p := newTestPlayer()
	p.jobID = job.WAR
	p.equip = nil // real, defensive case -- a player struct built without a real Equipment
	_ = playerCombatStats(p)
}

func TestEquipAttackBonus_UnarmedIsZero(t *testing.T) {
	p := newTestPlayer()
	p.equip = gear.NewEquipment()
	if got := equipAttackBonus(p); got != 0 {
		t.Errorf("equipAttackBonus(unarmed) = %d, want 0", got)
	}
}

func TestEquipAttackBonus_EquippedSwordAddsRealAttackStat(t *testing.T) {
	oldReg := itemdefReg
	itemdefReg = testRegistryWithSword(t)
	t.Cleanup(func() { itemdefReg = oldReg })

	p := newTestPlayer()
	p.equip = gear.NewEquipment()
	if err := p.equip.Equip(gear.SlotMainHand, gear.ItemEntry{ItemID: "sword", DefID: 3}); err != nil {
		t.Fatalf("equip: %v", err)
	}
	if got := equipAttackBonus(p); got != 10 {
		t.Errorf("equipAttackBonus(Sword equipped) = %d, want 10 (the real Sword's own attack stat)", got)
	}
}

func TestResolvePlayerAutoAttackDamage_EquippedWeaponIncreasesAverageDamage(t *testing.T) {
	oldReg := itemdefReg
	itemdefReg = testRegistryWithSword(t)
	t.Cleanup(func() { itemdefReg = oldReg })

	withSeededCombatRNG(t, 11)
	unarmed := newTestPlayer()
	unarmed.jobID = job.WAR
	unarmed.equip = gear.NewEquipment()

	armed := newTestPlayer()
	armed.jobID = job.WAR
	armed.equip = gear.NewEquipment()
	armed.equip.Equip(gear.SlotMainHand, gear.ItemEntry{ItemID: "sword", DefID: 3})

	sumUnarmed, sumArmed := 0, 0
	const trials = 300
	for i := 0; i < trials; i++ {
		_, _, dmg := resolvePlayerAutoAttackDamage(unarmed, 30)
		sumUnarmed += dmg
		_, _, dmg = resolvePlayerAutoAttackDamage(armed, 30)
		sumArmed += dmg
	}
	if sumArmed <= sumUnarmed {
		t.Errorf("equipping a real weapon should deal more average damage than unarmed over %d trials: armed sum=%d, unarmed sum=%d", trials, sumArmed, sumUnarmed)
	}
}

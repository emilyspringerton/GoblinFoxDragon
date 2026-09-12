package mob

import (
	"testing"
	"time"
)

// ReadyToSwing tests (JOB_SPELL_SYSTEM_NORTHSTAR.md §5) -- split out of TickPlayer so
// apps2/mud can compute a real STR/DEX/VIT/AGI-based damage roll itself instead of always using
// combat's own flat BaseDamage. TickPlayer's own existing tests (worm_test.go) already confirm
// its unchanged external behavior; these are the new function's own direct tests.

func TestReadyToSwing_NoTargetReturnsErrNoTarget(t *testing.T) {
	reg := spawnWormReg("w1", Pos{0, 2, 0})
	combat := &PlayerCombat{}
	mobID, err := reg.ReadyToSwing(combat, Pos{0, 2, 0}, 0, time.Now())
	if err != ErrNoTarget {
		t.Errorf("expected ErrNoTarget, got %v", err)
	}
	if mobID != "" {
		t.Errorf("expected empty mobID, got %q", mobID)
	}
}

func TestReadyToSwing_NotYetTimeReturnsEmptyNilError(t *testing.T) {
	reg := spawnWormReg("w1", Pos{0, 2, 0})
	combat := &PlayerCombat{
		TargetMobID: "w1",
		SwingDelay:  10 * time.Second,
		MeleeRange:  5,
	}
	combat.lastSwing = time.Now() // just swung
	mobID, err := reg.ReadyToSwing(combat, Pos{0, 2, 0}, 0, time.Now())
	if err != nil {
		t.Errorf("expected nil error (not yet time is not a real error), got %v", err)
	}
	if mobID != "" {
		t.Errorf("expected empty mobID when not yet time to swing, got %q", mobID)
	}
}

func TestReadyToSwing_ReadyReturnsMobIDAndAdvancesLastSwing(t *testing.T) {
	reg := spawnWormReg("w1", Pos{0, 2, 0})
	combat := &PlayerCombat{
		TargetMobID: "w1",
		SwingDelay:  100 * time.Millisecond,
		MeleeRange:  5,
	}
	now := time.Now().Add(time.Second)
	mobID, err := reg.ReadyToSwing(combat, Pos{0, 2, 0}, 0, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mobID != "w1" {
		t.Errorf("expected mobID w1, got %q", mobID)
	}
	if !combat.lastSwing.Equal(now) {
		t.Errorf("expected lastSwing advanced to %v, got %v", now, combat.lastSwing)
	}
}

func TestReadyToSwing_OutOfRangeReturnsErrOutOfRange(t *testing.T) {
	reg := spawnWormReg("w1", Pos{100, 2, 100})
	combat := &PlayerCombat{
		TargetMobID: "w1",
		SwingDelay:  100 * time.Millisecond,
		MeleeRange:  5,
	}
	mobID, err := reg.ReadyToSwing(combat, Pos{0, 2, 0}, 0, time.Now().Add(time.Second))
	if err != ErrOutOfRange {
		t.Errorf("expected ErrOutOfRange, got %v", err)
	}
	if mobID != "" {
		t.Errorf("expected empty mobID, got %q", mobID)
	}
}

func TestReadyToSwing_DeadMobReturnsErrMobDeadAndClearsTarget(t *testing.T) {
	reg := spawnWormReg("w1", Pos{0, 2, 0})
	m, _ := reg.Get("w1")
	m.HP = 0
	m.State = StateDead // alive() checks State, not HP directly
	combat := &PlayerCombat{
		TargetMobID: "w1",
		SwingDelay:  100 * time.Millisecond,
		MeleeRange:  5,
	}
	_, err := reg.ReadyToSwing(combat, Pos{0, 2, 0}, 0, time.Now().Add(time.Second))
	if err != ErrMobDead {
		t.Errorf("expected ErrMobDead, got %v", err)
	}
	if combat.TargetMobID != "" {
		t.Errorf("expected TargetMobID cleared on a dead target, got %q", combat.TargetMobID)
	}
}

// TestTickPlayer_StillProducesIdenticalRealHitAfterTheSplit is a real regression guard: the
// split into ReadyToSwing must not change TickPlayer's own observable behavior at all.
func TestTickPlayer_StillProducesIdenticalRealHitAfterTheSplit(t *testing.T) {
	reg := spawnWormReg("w1", Pos{0, 2, 0})
	combat := &PlayerCombat{
		TargetMobID: "w1",
		SwingDelay:  100 * time.Millisecond,
		BaseDamage:  25,
		MeleeRange:  5,
	}
	res, _, err := reg.TickPlayer("p1", combat, Pos{0, 2, 0}, 0, time.Now().Add(time.Second))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Dealt != 25 {
		t.Errorf("expected 25 damage dealt (combat.BaseDamage, unchanged by the split), got %d", res.Dealt)
	}
}

package job

import (
	"testing"
	"time"
)

var baseTime = time.Now()

// ── CharJob ───────────────────────────────────────────────────────────────────

func TestNewCharJob_Valid(t *testing.T) {
	cj, err := NewCharJob(WAR, WHM, 50, 25)
	if err != nil || cj == nil {
		t.Fatalf("NewCharJob: %v", err)
	}
}

func TestNewCharJob_SameJobError(t *testing.T) {
	if _, err := NewCharJob(WAR, WAR, 50, 25); err != ErrSameJob {
		t.Errorf("same job: got %v, want ErrSameJob", err)
	}
}

func TestNewCharJob_NoSub(t *testing.T) {
	cj, err := NewCharJob(WAR, "", 50, 0)
	if err != nil {
		t.Fatalf("no sub: %v", err)
	}
	if cj.EffectiveSubLevel() != 0 {
		t.Error("no sub job should have effective sub level 0")
	}
}

// ── EffectiveSubLevel ─────────────────────────────────────────────────────────

func TestEffectiveSubLevel_Halved(t *testing.T) {
	cj, _ := NewCharJob(WAR, WHM, 50, 30)
	if cj.EffectiveSubLevel() != 15 {
		t.Errorf("EffectiveSubLevel(30)=%d, want 15", cj.EffectiveSubLevel())
	}
}

func TestEffectiveSubLevel_OddLvlFloored(t *testing.T) {
	cj, _ := NewCharJob(WAR, WHM, 50, 25)
	if cj.EffectiveSubLevel() != 12 {
		t.Errorf("EffectiveSubLevel(25)=%d, want 12 (floor)", cj.EffectiveSubLevel())
	}
}

func TestEffectiveSubLevel_L1Sub(t *testing.T) {
	cj, _ := NewCharJob(WAR, WHM, 50, 1)
	if cj.EffectiveSubLevel() != 0 {
		t.Errorf("EffectiveSubLevel(1)=%d, want 0", cj.EffectiveSubLevel())
	}
}

// ── CombinedStats ─────────────────────────────────────────────────────────────

func TestCombinedStats_HigherThanMain(t *testing.T) {
	main, _ := StatsFor(WAR)
	cj, _ := NewCharJob(WAR, WHM, 50, 50)
	combined, err := cj.CombinedStats()
	if err != nil {
		t.Fatalf("CombinedStats: %v", err)
	}
	if combined.BaseHP <= main.BaseHP {
		t.Errorf("combined BaseHP (%d) should exceed main only (%d)", combined.BaseHP, main.BaseHP)
	}
}

func TestCombinedStats_NoSubEqualsMain(t *testing.T) {
	main, _ := StatsFor(WAR)
	cj, _ := NewCharJob(WAR, "", 50, 0)
	combined, _ := cj.CombinedStats()
	if combined.BaseHP != main.BaseHP {
		t.Errorf("no-sub combined BaseHP (%d) != main BaseHP (%d)", combined.BaseHP, main.BaseHP)
	}
}

func TestCombinedStats_UnknownMainJob(t *testing.T) {
	cj := &CharJob{Main: "XXX", Sub: WHM, MainLvl: 50, SubLvl: 25}
	if _, err := cj.CombinedStats(); err != ErrUnknownJob {
		t.Errorf("unknown main: got %v, want ErrUnknownJob", err)
	}
}

func TestCombinedStats_UnknownSubJob(t *testing.T) {
	cj := &CharJob{Main: WAR, Sub: "XXX", MainLvl: 50, SubLvl: 25}
	if _, err := cj.CombinedStats(); err != ErrUnknownJob {
		t.Errorf("unknown sub: got %v, want ErrUnknownJob", err)
	}
}

// ── RecastTracker ─────────────────────────────────────────────────────────────

func tracker() *RecastTracker {
	return NewRecastTracker(WarriorAbilities())
}

func TestRecast_ReadyBeforeUse(t *testing.T) {
	rt := tracker()
	ready, err := rt.Ready("provoke", baseTime)
	if err != nil || !ready {
		t.Errorf("provoke unused: ready=%v err=%v", ready, err)
	}
}

func TestRecast_UnknownAbility(t *testing.T) {
	rt := tracker()
	if _, err := rt.Ready("nonexistent", baseTime); err != ErrAbilityUnknown {
		t.Errorf("unknown ability Ready: got %v, want ErrAbilityUnknown", err)
	}
}

func TestUse_SuccessMarksRecast(t *testing.T) {
	rt := tracker()
	if err := rt.Use("provoke", baseTime, 10); err != nil {
		t.Fatalf("Use provoke: %v", err)
	}
	// immediately after use should be on recast
	ready, _ := rt.Ready("provoke", baseTime)
	if ready {
		t.Error("ability should be on recast immediately after use")
	}
}

func TestUse_RecastExpires(t *testing.T) {
	rt := tracker()
	rt.Use("provoke", baseTime, 10)
	// provoke recast = 30s; advance 31 seconds
	future := baseTime.Add(31 * time.Second)
	ready, _ := rt.Ready("provoke", future)
	if !ready {
		t.Error("provoke should be ready after 31 seconds")
	}
}

func TestUse_OnRecastError(t *testing.T) {
	rt := tracker()
	rt.Use("provoke", baseTime, 10)
	if err := rt.Use("provoke", baseTime.Add(time.Second), 10); err != ErrAbilityOnRecast {
		t.Errorf("recast gate: got %v, want ErrAbilityOnRecast", err)
	}
}

func TestUse_LevelGate(t *testing.T) {
	rt := tracker()
	// provoke MinLevel=5; try at level 4
	if err := rt.Use("provoke", baseTime, 4); err != ErrAbilityLevelGated {
		t.Errorf("level gate: got %v, want ErrAbilityLevelGated", err)
	}
}

func TestRecastRemaining_ZeroWhenReady(t *testing.T) {
	rt := tracker()
	rem, err := rt.RecastRemaining("provoke", baseTime)
	if err != nil || rem != 0 {
		t.Errorf("unused: rem=%v err=%v, want (0, nil)", rem, err)
	}
}

func TestRecastRemaining_PositiveAfterUse(t *testing.T) {
	rt := tracker()
	rt.Use("provoke", baseTime, 10)
	rem, _ := rt.RecastRemaining("provoke", baseTime.Add(5*time.Second))
	// provoke recast=30s, used 5s ago → 25s remaining
	if rem < 24*time.Second || rem > 26*time.Second {
		t.Errorf("recast remaining: got %v, want ~25s", rem)
	}
}

func TestTraitPassive_NoRecast(t *testing.T) {
	// Traits have no recast tracker entry — just check they can be defined
	traits := []Trait{
		{ID: "attack-bonus", Job: WAR, MinLevel: 10},
	}
	if len(traits) == 0 {
		t.Error("trait slice should have entries")
	}
}

// SummonerAbilities (2026-07-31, founder: "zagan beleth vassago as summoner avatars GFD").

func TestSummonerAbilities_HasAllThreeAvatars(t *testing.T) {
	want := map[string]bool{"summon_zagan": false, "summon_beleth": false, "summon_vassago": false}
	for _, a := range SummonerAbilities() {
		if a.Job != SMN {
			t.Errorf("ability %s has Job=%s, want SMN", a.ID, a.Job)
		}
		if _, ok := want[a.ID]; !ok {
			t.Errorf("unexpected ability ID %q in SummonerAbilities", a.ID)
			continue
		}
		want[a.ID] = true
	}
	for id, found := range want {
		if !found {
			t.Errorf("SummonerAbilities is missing %q", id)
		}
	}
}

func TestSummonerAbilities_UsableThroughRecastTracker(t *testing.T) {
	// Real integration check, not just a data shape check -- confirms SummonerAbilities()
	// actually works through the same RecastTracker every other job's abilities do.
	rt := NewRecastTracker(SummonerAbilities())
	if err := rt.Use("summon_zagan", baseTime, 10); err != nil {
		t.Fatalf("Use(summon_zagan): unexpected error %v", err)
	}
	ready, err := rt.Ready("summon_zagan", baseTime)
	if err != nil {
		t.Fatalf("Ready: unexpected error %v", err)
	}
	if ready {
		t.Error("summon_zagan should still be on its own 3-minute recast immediately after use")
	}
}

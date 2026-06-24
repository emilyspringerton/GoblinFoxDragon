package nm_test

import (
	"testing"
	"time"

	"dragonsnshit/server/nm"
)

var (
	testNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	zone1   = 1
)

func makeRegistry(t *testing.T, spawns ...*nm.NMSpawn) *nm.Registry {
	t.Helper()
	r := nm.NewRegistry()
	for _, s := range spawns {
		r.Register(s)
	}
	return r
}

// TestRegistryRegisterAndGet checks basic register/get.
func TestRegistryRegisterAndGet(t *testing.T) {
	r := nm.NewRegistry()
	s := &nm.NMSpawn{ID: "nm-test", RespawnMinutes: 30}
	r.Register(s)
	got := r.Get("nm-test")
	if got == nil || got.ID != "nm-test" {
		t.Errorf("Get: want nm-test, got %v", got)
	}
	if r.Size() != 1 {
		t.Errorf("Size: want 1, got %d", r.Size())
	}
}

// TestRegistryGetUnknown returns nil for unknown NM.
func TestRegistryGetUnknown(t *testing.T) {
	r := nm.NewRegistry()
	if got := r.Get("nobody"); got != nil {
		t.Errorf("want nil, got %v", got)
	}
}

// TestOnKillSchedulesRespawn verifies PendingCount grows after kill.
func TestOnKillSchedulesRespawn(t *testing.T) {
	reg := makeRegistry(t, &nm.NMSpawn{ID: "nm-boss", RespawnMinutes: 60})
	sched := nm.NewNMRespawnScheduler(reg)
	sched.OnKill("nm-boss", zone1, testNow)
	if sched.PendingCount() != 1 {
		t.Errorf("pending: want 1, got %d", sched.PendingCount())
	}
}

// TestOnKillNoRespawnWhenZeroMinutes ignores NMs with RespawnMinutes=0.
func TestOnKillNoRespawnWhenZeroMinutes(t *testing.T) {
	reg := makeRegistry(t, &nm.NMSpawn{ID: "nm-oneshot", RespawnMinutes: 0})
	sched := nm.NewNMRespawnScheduler(reg)
	sched.OnKill("nm-oneshot", zone1, testNow)
	if sched.PendingCount() != 0 {
		t.Errorf("want 0 pending (no respawn), got %d", sched.PendingCount())
	}
}

// TestOnKillUnknownNMIsNoop ensures unknown NM IDs don't panic or schedule.
func TestOnKillUnknownNMIsNoop(t *testing.T) {
	reg := makeRegistry(t)
	sched := nm.NewNMRespawnScheduler(reg)
	sched.OnKill("ghost-nm", zone1, testNow) // should not panic
	if sched.PendingCount() != 0 {
		t.Errorf("want 0 pending, got %d", sched.PendingCount())
	}
}

// TestTickReturnsPopEventAtRespawnTime verifies pop fires at the right time.
func TestTickReturnsPopEventAtRespawnTime(t *testing.T) {
	reg := makeRegistry(t, &nm.NMSpawn{ID: "nm-fast", RespawnMinutes: 5})
	sched := nm.NewNMRespawnScheduler(reg)
	sched.OnKill("nm-fast", zone1, testNow)

	// Tick before respawn time — no pop.
	early := testNow.Add(3 * time.Minute)
	pops := sched.Tick(early)
	if len(pops) != 0 {
		t.Errorf("early tick: want 0 pops, got %d", len(pops))
	}

	// Tick exactly at respawn time — pop fires.
	onTime := testNow.Add(5 * time.Minute)
	pops = sched.Tick(onTime)
	if len(pops) != 1 {
		t.Fatalf("at-time tick: want 1 pop, got %d", len(pops))
	}
	if pops[0].NMID != "nm-fast" {
		t.Errorf("pop NM ID: want nm-fast, got %s", pops[0].NMID)
	}
	if pops[0].ZoneID != zone1 {
		t.Errorf("pop zone: want %d, got %d", zone1, pops[0].ZoneID)
	}
}

// TestTickClearsTimerAfterPop ensures a popped NM doesn't re-pop on the next tick.
func TestTickClearsTimerAfterPop(t *testing.T) {
	reg := makeRegistry(t, &nm.NMSpawn{ID: "nm-clear", RespawnMinutes: 1})
	sched := nm.NewNMRespawnScheduler(reg)
	sched.OnKill("nm-clear", zone1, testNow)

	popTime := testNow.Add(1 * time.Minute)
	if pops := sched.Tick(popTime); len(pops) != 1 {
		t.Fatalf("first tick: want 1 pop, got %d", len(pops))
	}
	if pops := sched.Tick(popTime.Add(time.Minute)); len(pops) != 0 {
		t.Errorf("second tick: want 0 pops (timer cleared), got %d", len(pops))
	}
	if sched.PendingCount() != 0 {
		t.Errorf("pending after pop: want 0, got %d", sched.PendingCount())
	}
}

// TestCancelRemovesTimer verifies Cancel prevents a pop.
func TestCancelRemovesTimer(t *testing.T) {
	reg := makeRegistry(t, &nm.NMSpawn{ID: "nm-cancel", RespawnMinutes: 5})
	sched := nm.NewNMRespawnScheduler(reg)
	sched.OnKill("nm-cancel", zone1, testNow)
	sched.Cancel("nm-cancel")

	if sched.PendingCount() != 0 {
		t.Errorf("pending after cancel: want 0, got %d", sched.PendingCount())
	}
	pops := sched.Tick(testNow.Add(10 * time.Minute))
	if len(pops) != 0 {
		t.Errorf("tick after cancel: want 0 pops, got %d", len(pops))
	}
}

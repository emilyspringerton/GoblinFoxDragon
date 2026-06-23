package nm

import (
	"math/rand"
	"testing"
	"time"
)

var rng = rand.New(rand.NewSource(42))

func baseNM() *NMSpawn {
	return &NMSpawn{
		ID:            "nm-test",
		PlaceholderID: "mob-placeholder",
		SpawnChance:   1.0, // guaranteed for tests
		WindowOpen:    5 * time.Minute,
		WindowClose:   30 * time.Minute,
	}
}

// ── OnPlaceholderKilled ───────────────────────────────────────────────────────

func TestPlaceholderKill_OpensWindow(t *testing.T) {
	s := baseNM()
	now := time.Now()
	s.OnPlaceholderKilled(now)
	if !s.InWindow(now.Add(6 * time.Minute)) {
		t.Error("window should be open after 6 min (after 5min delay)")
	}
}

func TestPlaceholderKill_BeforeWindowOpen(t *testing.T) {
	s := baseNM()
	now := time.Now()
	s.OnPlaceholderKilled(now)
	// 3 minutes after kill — window opens at 5min
	if s.InWindow(now.Add(3 * time.Minute)) {
		t.Error("window should not be open before WindowOpen delay")
	}
}

func TestPlaceholderKill_AfterWindowClose(t *testing.T) {
	s := baseNM()
	now := time.Now()
	s.OnPlaceholderKilled(now)
	if s.InWindow(now.Add(31 * time.Minute)) {
		t.Error("window should be closed after WindowClose")
	}
}

func TestNoPlaceholder_KillNoOp(t *testing.T) {
	s := &NMSpawn{ID: "nm-scheduled", SpawnChance: 1.0}
	s.OnPlaceholderKilled(time.Now())
	if s.InWindow(time.Now()) {
		t.Error("no-placeholder NM should not open window on placeholder kill")
	}
}

// ── TrySpawn ─────────────────────────────────────────────────────────────────

func TestTrySpawn_InWindowGuaranteed(t *testing.T) {
	s := baseNM()
	now := time.Now()
	s.OnPlaceholderKilled(now)
	inWindow := now.Add(10 * time.Minute)
	if !s.TrySpawn(inWindow, rng) {
		t.Error("chance=1.0 should always spawn in window")
	}
}

func TestTrySpawn_OutsideWindow(t *testing.T) {
	s := baseNM()
	if s.TrySpawn(time.Now(), rng) {
		t.Error("TrySpawn before window opened should return false")
	}
}

func TestTrySpawn_MarksSpawned(t *testing.T) {
	s := baseNM()
	now := time.Now()
	s.OnPlaceholderKilled(now)
	inWindow := now.Add(10 * time.Minute)
	s.TrySpawn(inWindow, rng)
	// Second call in same window should always be false (already spawned).
	if s.TrySpawn(inWindow, rng) {
		t.Error("should not spawn twice in same window")
	}
}

func TestTrySpawn_ChanceZeroNeverSpawns(t *testing.T) {
	s := &NMSpawn{
		ID:            "nm-never",
		PlaceholderID: "ph",
		SpawnChance:   0.0,
		WindowOpen:    0,
		WindowClose:   time.Hour,
	}
	now := time.Now()
	s.OnPlaceholderKilled(now)
	for i := 0; i < 100; i++ {
		if s.TrySpawn(now.Add(time.Minute), rng) {
			t.Fatal("chance=0 should never spawn")
		}
		s.spawned = false // reset for next attempt
	}
}

// ── OpenWindow ────────────────────────────────────────────────────────────────

func TestOpenWindow_DirectOpen(t *testing.T) {
	s := &NMSpawn{ID: "nm-sched", SpawnChance: 1.0}
	now := time.Now()
	s.OpenWindow(now, now.Add(time.Hour))
	if !s.InWindow(now.Add(time.Minute)) {
		t.Error("should be in window after OpenWindow")
	}
}

// ── WindowExpired ─────────────────────────────────────────────────────────────

func TestWindowExpired_AfterClose(t *testing.T) {
	s := baseNM()
	now := time.Now()
	s.OnPlaceholderKilled(now)
	if s.WindowExpired(now.Add(29 * time.Minute)) {
		t.Error("window not yet expired at 29min")
	}
	if !s.WindowExpired(now.Add(31 * time.Minute)) {
		t.Error("window should be expired after WindowClose")
	}
}

func TestWindowExpired_FalseIfSpawned(t *testing.T) {
	s := baseNM()
	now := time.Now()
	s.OnPlaceholderKilled(now)
	s.TrySpawn(now.Add(10*time.Minute), rng) // spawns
	// Even past close time — spawned=true means expired=false
	if s.WindowExpired(now.Add(40 * time.Minute)) {
		t.Error("spawned NM should not show as expired")
	}
}

// ── Reset ─────────────────────────────────────────────────────────────────────

func TestReset_ClearsWindow(t *testing.T) {
	s := baseNM()
	now := time.Now()
	s.OnPlaceholderKilled(now)
	s.Reset()
	if s.InWindow(now.Add(10 * time.Minute)) {
		t.Error("after Reset, InWindow should be false")
	}
}

// ── Presets ───────────────────────────────────────────────────────────────────

func TestMeadowNMs_HasKingWorm(t *testing.T) {
	nms := MeadowNMs()
	found := false
	for _, n := range nms {
		if n.ID == "nm-king-worm" {
			found = true
		}
	}
	if !found {
		t.Error("MeadowNMs should include nm-king-worm")
	}
}

func TestSwampNMs_HasMarshLeech(t *testing.T) {
	nms := SwampNMs()
	found := false
	for _, n := range nms {
		if n.ID == "nm-marsh-leech" {
			found = true
		}
	}
	if !found {
		t.Error("SwampNMs should include nm-marsh-leech")
	}
}

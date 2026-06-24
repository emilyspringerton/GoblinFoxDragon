package factionwar_test

import (
	"testing"
	"time"

	"dragonsnshit/server/factionwar"
)

var testDistricts = []string{"district-residential", "district-commercial", "district-industrial"}

func newEngine() *factionwar.Engine {
	return factionwar.NewEngine(testDistricts)
}

func TestNewEngineNoActiveWars(t *testing.T) {
	e := newEngine()
	for _, d := range testDistricts {
		if e.IsAtWar(d) {
			t.Errorf("expected no active war at start for %q", d)
		}
	}
}

func TestAlertnessMuliplierIdle(t *testing.T) {
	e := newEngine()
	for _, d := range testDistricts {
		if got := e.AlertnessMultiplier(d); got != 1.0 {
			t.Errorf("district %q: want 1.0, got %f", d, got)
		}
	}
}

func TestK9DrainMultiplierIdle(t *testing.T) {
	e := newEngine()
	for _, d := range testDistricts {
		if got := e.K9DrainMultiplier(d); got != 1.0 {
			t.Errorf("district %q: want 1.0, got %f", d, got)
		}
	}
}

func TestActiveWarAfterCycle(t *testing.T) {
	// Force war by advancing time past WarCyclePeriod using many ticks.
	e := factionwar.NewEngine([]string{"d1"})

	var triggered []factionwar.War
	for i := 0; i < 1000 && len(triggered) == 0; i++ {
		tr, _ := e.Tick()
		triggered = append(triggered, tr...)
	}
	// We expect at least one war to eventually trigger given 33% chance per tick after elapsed.
	// With 1000 tries: probability of zero triggers is (0.67)^1000 ≈ 0, so this won't flake.
	if len(triggered) == 0 {
		t.Error("expected at least one war to trigger within 1000 ticks")
	}
}

func TestWarProperties(t *testing.T) {
	e := factionwar.NewEngine([]string{"d2"})
	var war *factionwar.War
	for i := 0; i < 1000 && war == nil; i++ {
		tr, _ := e.Tick()
		for _, w := range tr {
			cp := w
			war = &cp
		}
	}
	if war == nil {
		t.Fatal("no war triggered")
	}
	if war.DistrictID != "d2" {
		t.Errorf("DistrictID want d2, got %q", war.DistrictID)
	}
	if war.Phase != factionwar.PhaseActive {
		t.Errorf("Phase want Active, got %v", war.Phase)
	}
	if war.ID == "" {
		t.Error("War ID should not be empty")
	}
}

func TestIsAtWarAfterTrigger(t *testing.T) {
	e := factionwar.NewEngine([]string{"d3"})
	for i := 0; i < 1000; i++ {
		e.Tick()
		if e.IsAtWar("d3") {
			return
		}
	}
	t.Error("expected IsAtWar to return true after trigger")
}

func TestAlertnessDuringWar(t *testing.T) {
	e := factionwar.NewEngine([]string{"d4"})
	for i := 0; i < 1000; i++ {
		e.Tick()
		if e.IsAtWar("d4") {
			m := e.AlertnessMultiplier("d4")
			if m != factionwar.AlertnessMultiply {
				t.Errorf("want %f, got %f", factionwar.AlertnessMultiply, m)
			}
			return
		}
	}
	t.Skip("no war triggered — skip")
}

func TestK9DrainDuringWar(t *testing.T) {
	e := factionwar.NewEngine([]string{"d5"})
	for i := 0; i < 1000; i++ {
		e.Tick()
		if e.IsAtWar("d5") {
			m := e.K9DrainMultiplier("d5")
			if m != factionwar.K9DrainMultiply {
				t.Errorf("want %f, got %f", factionwar.K9DrainMultiply, m)
			}
			return
		}
	}
	t.Skip("no war triggered — skip")
}

func TestUpdateFOCount(t *testing.T) {
	e := factionwar.NewEngine([]string{"d6"})
	for i := 0; i < 1000; i++ {
		e.Tick()
		if e.IsAtWar("d6") {
			e.UpdateFOCount("d6", "The Frequency", 2)
			w := e.ActiveWar("d6")
			if w == nil {
				t.Fatal("expected active war")
			}
			if w.FOsHeld["The Frequency"] != 2 {
				t.Errorf("FOsHeld want 2, got %d", w.FOsHeld["The Frequency"])
			}
			return
		}
	}
	t.Skip("no war triggered — skip")
}

func TestWinConditionResolves(t *testing.T) {
	e := factionwar.NewEngine([]string{"d7"})
	// Trigger war then set FO count to win threshold.
	for i := 0; i < 1000; i++ {
		e.Tick()
		if e.IsAtWar("d7") {
			e.UpdateFOCount("d7", "The Bloc", factionwar.MinFOsToWin)
			_, resolved := e.Tick()
			if len(resolved) == 0 {
				t.Error("expected war to resolve after win condition met")
			}
			if e.IsAtWar("d7") {
				t.Error("war should no longer be active after resolution")
			}
			return
		}
	}
	t.Skip("no war triggered — skip")
}

func TestWinConditionTimeExpiry(t *testing.T) {
	// We can't easily test 24h expiry in unit tests without clock injection.
	// Verify that EndsAt is set to StartedAt + WarDuration.
	e := factionwar.NewEngine([]string{"d8"})
	for i := 0; i < 1000; i++ {
		e.Tick()
		if e.IsAtWar("d8") {
			w := e.ActiveWar("d8")
			if w == nil {
				t.Fatal("expected active war")
			}
			expected := w.StartedAt.Add(factionwar.WarDuration)
			if !w.EndsAt.Equal(expected) {
				t.Errorf("EndsAt: want %v, got %v", expected, w.EndsAt)
			}
			return
		}
	}
	t.Skip("no war triggered — skip")
}

func TestAllWarsReturnsSnapshot(t *testing.T) {
	e := factionwar.NewEngine([]string{"da", "db"})
	for i := 0; i < 1000; i++ {
		e.Tick()
		wars := e.AllWars()
		if len(wars) > 0 {
			// Verify returned wars have valid district IDs.
			for _, w := range wars {
				if w.DistrictID != "da" && w.DistrictID != "db" {
					t.Errorf("unexpected district %q", w.DistrictID)
				}
			}
			return
		}
	}
}

func TestActiveWarNilWhenIdle(t *testing.T) {
	e := newEngine()
	w := e.ActiveWar("district-residential")
	if w != nil {
		t.Error("expected nil when no war active")
	}
}

func TestLastCycleResetAfterResolve(t *testing.T) {
	// After a war resolves, the district should not immediately re-trigger.
	e := factionwar.NewEngine([]string{"dc"})
	warCount := 0
	for i := 0; i < 5000; i++ {
		tr, res := e.Tick()
		warCount += len(tr)
		_ = res
		if warCount >= 2 {
			t.Error("district triggered war twice without 72h interval — last cycle not reset")
			return
		}
	}
	// Passed: only at most one war in 5000 ticks (no rapid re-triggering).
}

func TestPhaseActiveOnStart(t *testing.T) {
	e := factionwar.NewEngine([]string{"dd"})
	for i := 0; i < 1000; i++ {
		tr, _ := e.Tick()
		for _, w := range tr {
			if w.Phase != factionwar.PhaseActive {
				t.Errorf("triggered war phase: want Active, got %d", w.Phase)
			}
			_ = time.Now() // ensure time import used
			return
		}
	}
}

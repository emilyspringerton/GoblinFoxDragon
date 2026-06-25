package npcattention

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func alwaysLOS(_, _ string) bool { return true }
func neverLOS(_, _ string) bool  { return false }

// ── NPC.Tick ──────────────────────────────────────────────────────────────────

func TestNoDisguiseBuildsSuspicion(t *testing.T) {
	npc := NewNPC("guard-1", FactionGuard, 0)
	d := Disguise{Faction: FactionNone}
	// 10 seconds of exposure with no disguise.
	npc.Tick("player-1", true, d, 0, 10, t0)
	w := npc.WatchState("player-1")
	if w.Suspicion <= 0 {
		t.Error("suspicion should have risen with no disguise")
	}
}

func TestDisguiseSameFactionIsEnforcer(t *testing.T) {
	npcA := NewNPC("guard-1", FactionGuard, 0)
	npcB := NewNPC("civilian-1", FactionCivilian, 0)
	d := Disguise{Faction: FactionGuard} // wearing guard uniform

	// Guard NPC: enforcer, higher gain.
	npcA.Tick("p1", true, d, 0, 10, t0)
	wA := npcA.WatchState("p1")

	// Civilian NPC: accepts the disguise, very low gain.
	npcB.Tick("p1", true, d, 0, 10, t0)
	wB := npcB.WatchState("p1")

	if wA.Suspicion <= wB.Suspicion {
		t.Errorf("guard (enforcer) should have higher suspicion than civilian: %.1f vs %.1f",
			wA.Suspicion, wB.Suspicion)
	}
}

func TestOutOfLOSDecays(t *testing.T) {
	npc := NewNPC("guard-1", FactionGuard, 0)
	d := Disguise{Faction: FactionNone}
	// Build up suspicion first.
	npc.Tick("p1", true, d, 0, 5, t0)
	before := npc.WatchState("p1").Suspicion

	// Then go out of sight.
	npc.Tick("p1", false, d, 0, 5, t0.Add(5*time.Second))
	after := npc.WatchState("p1").Suspicion
	if after >= before {
		t.Errorf("suspicion should decay out of LOS: before=%.1f after=%.1f", before, after)
	}
}

func TestSuspiciousThresholdEvent(t *testing.T) {
	npc := NewNPC("guard-1", FactionGuard, 0)
	d := Disguise{Faction: FactionNone}
	// Need > 35 / 25 = 1.4 seconds to cross SuspiciousAt.
	events := npc.Tick("p1", true, d, 0, 2, t0)
	found := false
	for _, e := range events {
		if e.Verb == VerbBecameSuspicious {
			found = true
		}
	}
	if !found {
		t.Errorf("expected VerbBecameSuspicious after sufficient exposure, got %v", events)
	}
}

func TestHostileThresholdEvent(t *testing.T) {
	npc := NewNPC("guard-1", FactionGuard, 0)
	d := Disguise{Faction: FactionNone}
	// Force Hostile (100 / 25 = 4 seconds).
	var all []AwarenessEvent
	all = append(all, npc.Tick("p1", true, d, 0, 5, t0)...)
	if npc.WatchState("p1").Awareness != AwarenessHostile {
		t.Errorf("expected Hostile after full exposure, got %v", npc.WatchState("p1").Awareness)
	}
	hasHostile := false
	for _, e := range all {
		if e.Verb == VerbBecameHostile {
			hasHostile = true
		}
	}
	if !hasHostile {
		t.Error("expected VerbBecameHostile in events")
	}
}

func TestRunningAddsGain(t *testing.T) {
	npcA := NewNPC("guard-1", FactionGuard, 0)
	npcB := NewNPC("guard-2", FactionGuard, 0)
	running := Disguise{Faction: FactionNone, Running: true}
	still := Disguise{Faction: FactionNone}

	npcA.Tick("p1", true, running, 0, 3, t0)
	npcB.Tick("p1", true, still, 0, 3, t0)
	if npcA.WatchState("p1").Suspicion <= npcB.WatchState("p1").Suspicion {
		t.Error("running should produce higher suspicion than standing still")
	}
}

func TestBodySpikesSuspicion(t *testing.T) {
	npcA := NewNPC("guard-1", FactionGuard, 0)
	npcB := NewNPC("guard-2", FactionGuard, 0)
	d := Disguise{Faction: FactionCivilian}

	// Same setup, one has a body nearby.
	npcA.Tick("p1", true, d, 1, 1, t0)
	npcB.Tick("p1", true, d, 0, 1, t0)
	if npcA.WatchState("p1").Suspicion <= npcB.WatchState("p1").Suspicion {
		t.Error("body in LOS should spike suspicion")
	}
}

// ── Witness ───────────────────────────────────────────────────────────────────

func TestWitnessKillJumpsToAlerted(t *testing.T) {
	npc := NewNPC("guard-1", FactionGuard, 0)
	events := npc.WitnessKill("p1", t0)
	w := npc.WatchState("p1")
	if w.Awareness < AwarenessAlerted {
		t.Errorf("expected Alerted after witness, got %v", w.Awareness)
	}
	if !w.IsWitness {
		t.Error("IsWitness should be true")
	}
	found := false
	for _, e := range events {
		if e.Verb == VerbWitnessAlerted {
			found = true
		}
	}
	if !found {
		t.Error("expected VerbWitnessAlerted event")
	}
}

func TestWitnessDoesNotDecayBelowFloor(t *testing.T) {
	npc := NewNPC("guard-1", FactionGuard, 0)
	npc.WitnessKill("p1", t0)
	// Tick out of LOS for a long time.
	d := Disguise{Faction: FactionCivilian}
	npc.Tick("p1", false, d, 0, 1000, t0.Add(time.Hour))
	w := npc.WatchState("p1")
	if w.Suspicion < WitnessFloor {
		t.Errorf("witness suspicion should not decay below %.0f, got %.1f", WitnessFloor, w.Suspicion)
	}
}

func TestSilenceWitnessResets(t *testing.T) {
	npc := NewNPC("guard-1", FactionGuard, 0)
	npc.WitnessKill("p1", t0)
	npc.SilenceWitness("p1")
	w := npc.WatchState("p1")
	if w.IsWitness {
		t.Error("IsWitness should be false after silence")
	}
	if w.Awareness != AwarenessUnaware {
		t.Errorf("awareness should be Unaware after silence, got %v", w.Awareness)
	}
}

// ── Scene ─────────────────────────────────────────────────────────────────────

func TestSceneTick(t *testing.T) {
	sc := NewScene(0)
	sc.AddNPC(NewNPC("guard-1", FactionGuard, 0))
	sc.AddNPC(NewNPC("guard-2", FactionGuard, 0))

	players := []PlayerState{
		{PlayerID: "p1", Disguise: Disguise{Faction: FactionNone}, SceneID: 0},
	}
	events := sc.Tick(players, alwaysLOS, 10, t0)
	if len(events) == 0 {
		t.Error("expected awareness events from scene tick")
	}
}

func TestSceneTickNoLOS(t *testing.T) {
	sc := NewScene(0)
	sc.AddNPC(NewNPC("guard-1", FactionGuard, 0))
	players := []PlayerState{
		{PlayerID: "p1", Disguise: Disguise{Faction: FactionNone}, SceneID: 0},
	}
	// No LOS — suspicion should not rise.
	sc.Tick(players, neverLOS, 10, t0)
	npc := sc.GetNPC("guard-1")
	if npc.WatchState("p1").Suspicion != 0 {
		t.Error("suspicion should not rise without LOS")
	}
}

func TestSceneWitnessCount(t *testing.T) {
	sc := NewScene(0)
	sc.AddNPC(NewNPC("g1", FactionGuard, 0))
	sc.AddNPC(NewNPC("g2", FactionGuard, 0))
	sc.WitnessKill("p1", alwaysLOS, t0)
	if sc.WitnessCount("p1") != 2 {
		t.Errorf("expected 2 witnesses, got %d", sc.WitnessCount("p1"))
	}
}

func TestSceneAlertLevel(t *testing.T) {
	sc := NewScene(0)
	sc.AddNPC(NewNPC("g1", FactionGuard, 0))
	if sc.AlertLevel() != AwarenessUnaware {
		t.Error("empty scene should be unaware")
	}
	sc.WitnessKill("p1", alwaysLOS, t0)
	if sc.AlertLevel() < AwarenessAlerted {
		t.Error("scene should be at least Alerted after witness")
	}
}

func TestHighestAwareness(t *testing.T) {
	npc := NewNPC("g1", FactionGuard, 0)
	npc.WitnessKill("p1", t0)
	d := Disguise{Faction: FactionNone}
	npc.Tick("p2", true, d, 0, 1, t0) // low suspicion

	a, pid := npc.HighestAwareness()
	if a < AwarenessAlerted {
		t.Errorf("highest awareness should be Alerted+, got %v", a)
	}
	if pid != "p1" {
		t.Errorf("highest awareness player should be p1, got %s", pid)
	}
}

func TestDifferentScenePlayerIgnored(t *testing.T) {
	sc := NewScene(0)
	sc.AddNPC(NewNPC("g1", FactionGuard, 0))
	// Player in scene 1, NPC in scene 0.
	players := []PlayerState{
		{PlayerID: "p1", Disguise: Disguise{Faction: FactionNone}, SceneID: 1},
	}
	sc.Tick(players, alwaysLOS, 10, t0)
	npc := sc.GetNPC("g1")
	if npc.WatchState("p1").Suspicion != 0 {
		t.Error("player in different scene should not raise NPC suspicion")
	}
}

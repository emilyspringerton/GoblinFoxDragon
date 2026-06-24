package schedule_test

import (
	"testing"

	"dragonsnshit/server/schedule"
)

func TestRegisterAndCount(t *testing.T) {
	reg := schedule.NewRegistry()
	reg.Register(&schedule.NPCSchedule{NPCID: "npc-a", ZoneAtHour: [24]int{}})
	if reg.Count() != 1 {
		t.Errorf("want 1, got %d", reg.Count())
	}
}

func TestCurrentZoneAfterRegister(t *testing.T) {
	reg := schedule.NewRegistry()
	var arr [24]int
	arr[0] = 42
	reg.Register(&schedule.NPCSchedule{NPCID: "npc-a", ZoneAtHour: arr})
	z, ok := reg.CurrentZone("npc-a")
	if !ok {
		t.Fatal("expected zone to be found")
	}
	if z != 42 {
		t.Errorf("want 42, got %d", z)
	}
}

func TestTickMovesNPC(t *testing.T) {
	reg := schedule.NewRegistry()
	var arr [24]int
	arr[0] = 10
	arr[1] = 20
	reg.Register(&schedule.NPCSchedule{NPCID: "npc-a", ZoneAtHour: arr})
	// Tick at hour 1: NPC should move from zone 10 to zone 20.
	moves := reg.Tick(1)
	if len(moves) != 1 {
		t.Fatalf("want 1 move, got %d", len(moves))
	}
	if moves[0].FromZone != 10 || moves[0].ToZone != 20 {
		t.Errorf("move: want 10→20, got %d→%d", moves[0].FromZone, moves[0].ToZone)
	}
}

func TestTickNoMoveWhenZoneSame(t *testing.T) {
	reg := schedule.NewRegistry()
	var arr [24]int
	for i := range arr {
		arr[i] = 99
	}
	reg.Register(&schedule.NPCSchedule{NPCID: "npc-b", ZoneAtHour: arr})
	moves := reg.Tick(5)
	if len(moves) != 0 {
		t.Errorf("expected no moves when zone unchanged, got %d", len(moves))
	}
}

func TestTickUpdatesCurrentZone(t *testing.T) {
	reg := schedule.NewRegistry()
	var arr [24]int
	arr[0] = 1
	arr[2] = 2
	reg.Register(&schedule.NPCSchedule{NPCID: "npc-c", ZoneAtHour: arr})
	_ = reg.Tick(2)
	z, _ := reg.CurrentZone("npc-c")
	if z != 2 {
		t.Errorf("want zone 2 after tick, got %d", z)
	}
}

func TestTickInvalidHour(t *testing.T) {
	reg := schedule.NewRegistry()
	moves := reg.Tick(24)
	if moves != nil {
		t.Error("expected nil moves for invalid hour 24")
	}
	moves = reg.Tick(-1)
	if moves != nil {
		t.Error("expected nil moves for invalid hour -1")
	}
}

func TestGetSchedule(t *testing.T) {
	reg := schedule.NewRegistry()
	s := &schedule.NPCSchedule{NPCID: "npc-d"}
	reg.Register(s)
	got := reg.Get("npc-d")
	if got == nil {
		t.Fatal("expected non-nil schedule")
	}
	if got.NPCID != "npc-d" {
		t.Errorf("NPCID mismatch")
	}
}

func TestGetMissingReturnsNil(t *testing.T) {
	reg := schedule.NewRegistry()
	if reg.Get("nobody") != nil {
		t.Error("expected nil for unknown NPC")
	}
}

func TestAllReturnsIDs(t *testing.T) {
	reg := schedule.NewRegistry()
	reg.Register(&schedule.NPCSchedule{NPCID: "a"})
	reg.Register(&schedule.NPCSchedule{NPCID: "b"})
	ids := reg.All()
	if len(ids) != 2 {
		t.Errorf("want 2 IDs, got %d", len(ids))
	}
}

func TestExamineFlavor(t *testing.T) {
	s := &schedule.NPCSchedule{
		NPCID:       "test-npc",
		Description: "wanders around",
	}
	flavor := schedule.ExamineFlavor(s, 14)
	if flavor == "" {
		t.Error("expected non-empty flavor")
	}
}

func TestExamineFlavorNil(t *testing.T) {
	if flavor := schedule.ExamineFlavor(nil, 10); flavor != "" {
		t.Errorf("expected empty flavor for nil schedule, got %q", flavor)
	}
}

func TestDefaultSchedulesCount(t *testing.T) {
	if n := len(schedule.DefaultSchedules()); n != 3 {
		t.Errorf("want 3 default schedules, got %d", n)
	}
}

func TestJiangshiWardenNightZone(t *testing.T) {
	s := schedule.JiangshiWardenSchedule
	// Night hours: 20-23 should be zone 50
	for _, h := range []int{20, 21, 22, 23, 0, 1, 2, 3, 4, 5} {
		if s.ZoneAtHour[h] != 50 {
			t.Errorf("jiangshi warden: hour %d want zone 50, got %d", h, s.ZoneAtHour[h])
		}
	}
}

func TestJiangshiWardenDayZone(t *testing.T) {
	s := schedule.JiangshiWardenSchedule
	// Day hours: 6-19 should be zone 51
	for _, h := range []int{6, 7, 8, 9, 12, 18, 19} {
		if s.ZoneAtHour[h] != 51 {
			t.Errorf("jiangshi warden: hour %d want zone 51, got %d", h, s.ZoneAtHour[h])
		}
	}
}

func TestEastwindArchivistWorkHours(t *testing.T) {
	s := schedule.EastwindArchivistSchedule
	for _, h := range []int{8, 10, 12, 17} {
		if s.ZoneAtHour[h] != 60 {
			t.Errorf("archivist: hour %d want zone 60, got %d", h, s.ZoneAtHour[h])
		}
	}
}

func TestHeikeganiDockBossOffHours(t *testing.T) {
	s := schedule.HeikeganiDockBossSchedule
	for _, h := range []int{22, 23, 0, 1, 2, 3, 4, 5} {
		if s.ZoneAtHour[h] != 71 {
			t.Errorf("dock boss: hour %d want zone 71 (home), got %d", h, s.ZoneAtHour[h])
		}
	}
}

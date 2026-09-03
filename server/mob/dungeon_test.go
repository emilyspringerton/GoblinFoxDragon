package mob

import (
	"testing"

	"dragonsnshit/server/worldapi"
)

// TestGenerateDungeonSpawns_BossPresent -- DUNGEON_NORTHSTAR.md Milestone 3's own acceptance:
// "a boss spawn in the final room."
func TestGenerateDungeonSpawns_BossPresent(t *testing.T) {
	layout := worldapi.GenerateDungeonLayout(5)
	spawns := GenerateDungeonSpawns(layout, 0, 42, 5)

	var bossCount int
	wantBoss := DungeonRoster[0].Boss
	for _, m := range spawns {
		if m.Kind == wantBoss {
			bossCount++
			if m.SceneID != 42 {
				t.Errorf("boss spawn has wrong SceneID: got %d, want 42", m.SceneID)
			}
		}
	}
	if bossCount != 1 {
		t.Fatalf("expected exactly 1 boss spawn (%s), got %d", wantBoss, bossCount)
	}
}

// TestGenerateDungeonSpawns_EntranceRoomClear -- the entrance room should stay empty (real,
// deliberate design choice mirroring every other zone's safe-landing convention).
func TestGenerateDungeonSpawns_EntranceRoomClear(t *testing.T) {
	layout := worldapi.GenerateDungeonLayout(5)
	spawns := GenerateDungeonSpawns(layout, 0, 1, 5)
	entrance := layout.Rooms[layout.EntranceIdx]
	for _, m := range spawns {
		// Minions jitter a couple units off center; a mob landing exactly ON the room's own
		// center coordinates would indicate the entrance-skip logic failed.
		if int(m.Pos.X) == entrance.CenterX && int(m.Pos.Z) == entrance.CenterZ {
			t.Errorf("found a spawn at the entrance room's exact center, expected it to stay clear: %+v", m)
		}
	}
}

// TestGenerateDungeonSpawns_Deterministic -- same layout+seed+index must produce the same
// spawn table (needed for the same real client/server-agreement reason Milestone 2's own seed
// determinism matters).
func TestGenerateDungeonSpawns_Deterministic(t *testing.T) {
	layout := worldapi.GenerateDungeonLayout(11)
	a := GenerateDungeonSpawns(layout, 2, 1, 99)
	b := GenerateDungeonSpawns(layout, 2, 1, 99)
	if len(a) != len(b) {
		t.Fatalf("same seed produced different spawn counts: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("same seed produced different spawn %d: %+v vs %+v", i, a[i], b[i])
		}
	}
}

// TestGenerateDungeonSpawns_DungeonIndexWraps -- an out-of-range index (including negative)
// must not panic, matching the doc comment's own promise.
func TestGenerateDungeonSpawns_DungeonIndexWraps(t *testing.T) {
	layout := worldapi.GenerateDungeonLayout(3)
	for _, idx := range []int{-1, -100, 0, 7, 8, 1000} {
		if spawns := GenerateDungeonSpawns(layout, idx, 1, 1); len(spawns) == 0 {
			t.Errorf("dungeonIndex=%d produced no spawns", idx)
		}
	}
}

// TestGenerateDungeonSpawns_EmptyLayout -- defensive: a zero-room layout must not panic.
func TestGenerateDungeonSpawns_EmptyLayout(t *testing.T) {
	spawns := GenerateDungeonSpawns(worldapi.DungeonLayout{}, 0, 1, 1)
	if spawns != nil {
		t.Errorf("expected nil spawns for an empty layout, got %d", len(spawns))
	}
}

// TestGenerateDungeonSpawns_SpawnsRegisterCleanly -- confirms every generated Mob is actually
// acceptable to this package's own real Registry.Spawn (unique IDs, no malformed fields).
func TestGenerateDungeonSpawns_SpawnsRegisterCleanly(t *testing.T) {
	layout := worldapi.GenerateDungeonLayout(21)
	spawns := GenerateDungeonSpawns(layout, 4, 9, 21)
	reg := New()
	for _, m := range spawns {
		if err := reg.Spawn(m); err != nil {
			t.Fatalf("Spawn(%+v) failed: %v", m, err)
		}
	}
}

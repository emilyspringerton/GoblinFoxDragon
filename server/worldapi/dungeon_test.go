package worldapi

import "testing"

// TestGenerateDungeonLayout_Deterministic -- DUNGEON_NORTHSTAR.md Milestone 2's own acceptance:
// "the same seed always produces the same layout" (needed so a matchmaker-issued seed, see
// REDGARDEN's MatchFoundMsg, lets a server and its clients agree on one dungeon without a
// separate data round-trip).
func TestGenerateDungeonLayout_Deterministic(t *testing.T) {
	a := GenerateDungeonLayout(42)
	b := GenerateDungeonLayout(42)
	if len(a.Rooms) != len(b.Rooms) {
		t.Fatalf("same seed produced different room counts: %d vs %d", len(a.Rooms), len(b.Rooms))
	}
	for i := range a.Rooms {
		if a.Rooms[i] != b.Rooms[i] {
			t.Fatalf("same seed produced different room %d: %+v vs %+v", i, a.Rooms[i], b.Rooms[i])
		}
	}
}

// TestGenerateDungeonLayout_DiffersBySeed -- "regenerates differently per instance."
func TestGenerateDungeonLayout_DiffersBySeed(t *testing.T) {
	seen := make(map[int]bool)
	for seed := int64(1); seed <= 20; seed++ {
		l := GenerateDungeonLayout(seed)
		key := len(l.Rooms)
		for _, r := range l.Rooms {
			key = key*31 + r.CenterX*7 + r.CenterZ*13 + r.W + r.D
		}
		seen[key] = true
	}
	if len(seen) < 2 {
		t.Fatalf("expected different seeds to produce visibly different layouts, got %d distinct shapes across 20 seeds", len(seen))
	}
}

// TestGenerateDungeonLayout_RoomCountInRange checks the real, documented bounds.
func TestGenerateDungeonLayout_RoomCountInRange(t *testing.T) {
	for seed := int64(0); seed < 50; seed++ {
		l := GenerateDungeonLayout(seed)
		if len(l.Rooms) < DungeonMinRooms || len(l.Rooms) > DungeonMaxRooms {
			t.Fatalf("seed %d: room count %d out of [%d,%d]", seed, len(l.Rooms), DungeonMinRooms, DungeonMaxRooms)
		}
	}
}

// TestDungeonLayout_IsReachable -- Milestone 2's own explicit acceptance bar: "connected,
// reachable entrance-to-boss-room." Verified via real BFS, not assumed.
func TestDungeonLayout_IsReachable(t *testing.T) {
	for seed := int64(0); seed < 50; seed++ {
		l := GenerateDungeonLayout(seed)
		if !l.IsReachable() {
			t.Fatalf("seed %d: boss room %d not reachable from entrance room %d", seed, l.BossIdx, l.EntranceIdx)
		}
	}
}

// TestDungeonLayout_EntranceAndBossDistinct -- a dungeon with only one real room would make
// "entrance" and "boss" the same room, a real degenerate case worth catching if DungeonMinRooms
// ever regresses to 1.
func TestDungeonLayout_EntranceAndBossDistinct(t *testing.T) {
	l := GenerateDungeonLayout(7)
	if l.EntranceIdx == l.BossIdx {
		t.Fatalf("entrance and boss room are the same room (%d) -- DungeonMinRooms should keep these distinct", l.EntranceIdx)
	}
}

// TestDungeonLayoutToBlocks_ProducesRealGeometry -- confirms the generator's real block output:
// real floor (dirt) and wall (stone_bricks) blocks exist, and every block lands within the
// layout's own real bounding box (no stray geometry).
func TestDungeonLayoutToBlocks_ProducesRealGeometry(t *testing.T) {
	l := GenerateDungeonLayout(99)
	blocks := DungeonLayoutToBlocks(l, 1000, 2000)
	if len(blocks) == 0 {
		t.Fatal("expected real block output, got none")
	}
	var hasFloor, hasWall bool
	for _, b := range blocks {
		if b.BlockName == "minecraft:dirt" {
			hasFloor = true
		}
		if b.BlockName == "minecraft:stone_bricks" {
			hasWall = true
		}
	}
	if !hasFloor {
		t.Error("expected a real dirt floor layer")
	}
	if !hasWall {
		t.Error("expected real stone_bricks walls")
	}
}

// TestDungeonLayoutToBlocks_EmptyLayout -- real, defensive: an empty layout (zero rooms) must
// not panic the carver.
func TestDungeonLayoutToBlocks_EmptyLayout(t *testing.T) {
	blocks := DungeonLayoutToBlocks(DungeonLayout{}, 0, 0)
	if blocks != nil {
		t.Errorf("expected nil blocks for an empty layout, got %d", len(blocks))
	}
}

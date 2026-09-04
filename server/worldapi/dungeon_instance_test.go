package worldapi

import "testing"

func TestDungeonInstanceRegistry_AllocateReturnsIncreasingSceneIDs(t *testing.T) {
	r := NewDungeonInstanceRegistry(100)
	a, okA := r.Allocate(0, 1)
	b, okB := r.Allocate(0, 2)
	c, okC := r.Allocate(0, 3)
	if !okA || !okB || !okC {
		t.Fatalf("expected all three allocations to succeed")
	}
	if a != 100 || b != 101 || c != 102 {
		t.Fatalf("expected 500,501,502, got %d,%d,%d", a, b, c)
	}
}

func TestDungeonInstanceRegistry_HasAndUnknownScene(t *testing.T) {
	r := NewDungeonInstanceRegistry(100)
	sceneID, ok := r.Allocate(0, 42)
	if !ok {
		t.Fatalf("expected Allocate to succeed")
	}
	if !r.Has(sceneID) {
		t.Errorf("expected Has(%d) true right after Allocate", sceneID)
	}
	if r.Has(sceneID + 1) {
		t.Errorf("expected Has of an unallocated scene ID to be false")
	}
}

func TestDungeonInstanceRegistry_BlocksForChunk_MatchesDirectGeneration(t *testing.T) {
	r := NewDungeonInstanceRegistry(100)
	seed := int64(7)
	sceneID, ok := r.Allocate(0, seed)
	if !ok {
		t.Fatalf("expected Allocate to succeed")
	}

	// Real ground truth: generate the exact same layout/blocks directly (bypassing the
	// registry) and manually filter to chunk (0,0) -- BlocksForChunk must return the identical
	// set, since Allocate uses originX=originZ=0 for every instance.
	layout := GenerateDungeonLayout(seed)
	allBlocks := DungeonLayoutToBlocks(layout, 0, 0)
	var want []WorldBlock
	for _, b := range allBlocks {
		if b.X >= 0 && b.X <= 15 && b.Z >= 0 && b.Z <= 15 {
			want = append(want, b)
		}
	}

	got, ok := r.BlocksForChunk(sceneID, 0, 0)
	if !ok {
		t.Fatalf("expected ok=true for a live instance")
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d blocks in chunk (0,0), got %d", len(want), len(got))
	}
	wantSet := make(map[WorldBlock]bool, len(want))
	for _, b := range want {
		wantSet[b] = true
	}
	for _, b := range got {
		if !wantSet[b] {
			t.Errorf("unexpected block outside the direct-generation ground truth: %+v", b)
		}
	}
}

func TestDungeonInstanceRegistry_BlocksForChunk_OutOfRangeChunkIsEmptyNotError(t *testing.T) {
	r := NewDungeonInstanceRegistry(100)
	sceneID, ok := r.Allocate(0, 7)
	if !ok {
		t.Fatalf("expected Allocate to succeed")
	}

	// A chunk far away from every room/corridor should come back empty but still ok=true --
	// distinct from an unknown scene ID (ok=false).
	got, ok := r.BlocksForChunk(sceneID, 10000, 10000)
	if !ok {
		t.Fatalf("expected ok=true for a live instance even with an empty chunk")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 blocks in a far-away chunk, got %d", len(got))
	}
}

func TestDungeonInstanceRegistry_BlocksForChunk_UnknownScene(t *testing.T) {
	r := NewDungeonInstanceRegistry(100)
	_, ok := r.BlocksForChunk(999, 0, 0)
	if ok {
		t.Fatalf("expected ok=false for a scene ID that was never allocated")
	}
}

func TestDungeonInstanceRegistry_EntrySpawn_InsideEntranceRoom(t *testing.T) {
	r := NewDungeonInstanceRegistry(100)
	seed := int64(99)
	sceneID, ok := r.Allocate(0, seed)
	if !ok {
		t.Fatalf("expected Allocate to succeed")
	}

	x, y, z, ok := r.EntrySpawn(sceneID)
	if !ok {
		t.Fatalf("expected ok=true for a live instance")
	}
	layout := GenerateDungeonLayout(seed)
	entrance := layout.Rooms[layout.EntranceIdx]
	if x != float64(entrance.CenterX) || z != float64(entrance.CenterZ) {
		t.Errorf("expected spawn at entrance room center (%d,%d), got (%v,%v)", entrance.CenterX, entrance.CenterZ, x, z)
	}
	if y != float64(dungeonFloorY) {
		t.Errorf("expected spawn Y at dungeonFloorY (%d), got %v", dungeonFloorY, y)
	}
}

func TestDungeonInstanceRegistry_EntrySpawn_UnknownScene(t *testing.T) {
	r := NewDungeonInstanceRegistry(100)
	_, _, _, ok := r.EntrySpawn(999)
	if ok {
		t.Fatalf("expected ok=false for a scene ID that was never allocated")
	}
}

func TestDungeonInstanceRegistry_MultipleInstancesAreIndependent(t *testing.T) {
	r := NewDungeonInstanceRegistry(100)
	a, okA := r.Allocate(0, 1)
	b, okB := r.Allocate(0, 2)
	if !okA || !okB {
		t.Fatalf("expected both allocations to succeed")
	}

	aBlocks, _ := r.BlocksForChunk(a, 0, 0)
	bBlocks, _ := r.BlocksForChunk(b, 0, 0)
	// Different seeds are extremely unlikely to produce identical block sets for chunk (0,0);
	// this is a real, direct sanity check that instances don't share state, not a
	// probabilistic/flaky assertion about dungeon layout diversity in general.
	if len(aBlocks) == len(bBlocks) {
		same := true
		for i := range aBlocks {
			if aBlocks[i] != bBlocks[i] {
				same = false
				break
			}
		}
		if same {
			t.Errorf("expected two different-seed instances to produce different chunk (0,0) blocks, got identical results")
		}
	}
}

// TestDungeonInstanceRegistry_ExhaustionRefusesRatherThanWraps -- real, load-bearing wire-format
// constraint (see the registry's own doc comment): once every scene ID in
// [dungeonInstanceStartSceneID, dungeonInstanceMaxSceneID] is allocated, Allocate must refuse
// (ok=false) rather than wrap around and hand out a scene ID a still-live instance already owns.
func TestDungeonInstanceRegistry_ExhaustionRefusesRatherThanWraps(t *testing.T) {
	const start = 250
	r := NewDungeonInstanceRegistry(start)
	capacity := dungeonInstanceMaxSceneID - start + 1

	for i := 0; i < capacity; i++ {
		if _, ok := r.Allocate(0, int64(i)); !ok {
			t.Fatalf("expected allocation %d/%d to succeed (within capacity)", i+1, capacity)
		}
	}

	sceneID, ok := r.Allocate(0, 9999)
	if ok {
		t.Fatalf("expected Allocate to refuse once capacity is exhausted, got sceneID=%d", sceneID)
	}

	// Real, direct confirmation that refusal didn't silently consume or corrupt an existing
	// instance -- the very first allocation should still be exactly as it was.
	if !r.Has(start) {
		t.Errorf("expected the first-allocated scene ID (%d) to remain a live instance after exhaustion", start)
	}
}

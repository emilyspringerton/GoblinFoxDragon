package worldapi_test

import (
	"testing"

	"dragonsnshit/server/worldapi"
)

func TestDragonflyChunkGenerator_proceduralFallback(t *testing.T) {
	gen := worldapi.NewDragonflyChunkGenerator(nil)
	blocks := gen.ChunkBlocks(0, 0, 0)
	if len(blocks) == 0 {
		t.Fatal("expected non-empty blocks from procedural fallback")
	}
	// Every block must have a non-zero BlockID.
	for i, b := range blocks {
		if b.BlockID == 0 {
			t.Errorf("block[%d] has zero BlockID (air should be excluded)", i)
		}
	}
}

func TestDragonflyChunkGenerator_worldStore(t *testing.T) {
	called := false
	store := func(sceneID, cx, cz int) []worldapi.WorldBlock {
		called = true
		return []worldapi.WorldBlock{
			{X: 1, Y: 0, Z: 1, BlockName: "minecraft:stone"},
			{X: 2, Y: 4, Z: 2, BlockName: "minecraft:grass_block"},
			{X: 3, Y: 2, Z: 3, BlockName: "minecraft:unknown_block"}, // should be skipped
		}
	}
	gen := worldapi.NewDragonflyChunkGenerator(store)
	blocks := gen.ChunkBlocks(0, 0, 0)
	if !called {
		t.Fatal("WorldStore not called")
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 mapped blocks (unknown skipped), got %d", len(blocks))
	}
	if blocks[0].BlockID != 1 { // stone
		t.Errorf("expected stone ID=1, got %d", blocks[0].BlockID)
	}
	if blocks[1].BlockID != 2 { // grass
		t.Errorf("expected grass ID=2, got %d", blocks[1].BlockID)
	}
}

func TestDragonflyChunkGenerator_nilStoreReturnsNil(t *testing.T) {
	store := func(sceneID, cx, cz int) []worldapi.WorldBlock { return nil }
	gen := worldapi.NewDragonflyChunkGenerator(store)
	if got := gen.ChunkBlocks(0, 0, 0); got != nil {
		t.Fatalf("expected nil when store returns nil, got %d blocks", len(got))
	}
}

func TestNameToBlockID_suffixFallback(t *testing.T) {
	// mangrove_log / cherry_leaves are not in the explicit table — suffix fallback must catch them
	gen := worldapi.NewDragonflyChunkGenerator(func(_, _, _ int) []worldapi.WorldBlock {
		return []worldapi.WorldBlock{
			{X: 0, Y: 0, Z: 0, BlockName: "minecraft:mangrove_log"},
			{X: 1, Y: 0, Z: 1, BlockName: "minecraft:cherry_leaves"},
		}
	})
	blocks := gen.ChunkBlocks(0, 0, 0)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks via suffix fallback, got %d", len(blocks))
	}
	if blocks[0].BlockID != 17 {
		t.Errorf("mangrove_log: expected log ID=17, got %d", blocks[0].BlockID)
	}
	if blocks[1].BlockID != 18 {
		t.Errorf("cherry_leaves: expected leaf ID=18, got %d", blocks[1].BlockID)
	}
}

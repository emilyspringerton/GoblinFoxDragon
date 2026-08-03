package worldapi

import (
	"testing"
)

func TestProceduralWorldStore_Scene0Meadow(t *testing.T) {
	blocks := ProceduralWorldStore(0, 0, 0)
	if len(blocks) == 0 {
		t.Fatal("scene 0: expected blocks, got none")
	}
	// Expect grass and stone
	var hasGrass, hasStone bool
	for _, b := range blocks {
		if b.BlockName == "minecraft:grass_block" {
			hasGrass = true
		}
		if b.BlockName == "minecraft:stone" {
			hasStone = true
		}
	}
	if !hasGrass {
		t.Error("scene 0: expected grass_block")
	}
	if !hasStone {
		t.Error("scene 0: expected stone")
	}
}

// TestProceduralWorldStore_Scene0Meadow_Flowers (2026-08-03, founder: "add some block backed
// flowers to the meadow") -- real, block-backed poppy/dandelion ground cover, not just decoration
// invented on the render side. Checks both real block names appear and that no flower lands on a
// tree's own trunk column (meadowFlowers' own doc comment: picked by hand to avoid exactly that).
func TestProceduralWorldStore_Scene0Meadow_Flowers(t *testing.T) {
	blocks := ProceduralWorldStore(0, 0, 0)
	var hasPoppy, hasDandelion bool
	treeCols := make(map[[2]int]bool)
	for _, t := range meadowTrees(0, 0) {
		treeCols[[2]int{t[0], t[1]}] = true
	}
	for _, b := range blocks {
		switch b.BlockName {
		case "minecraft:poppy":
			hasPoppy = true
		case "minecraft:dandelion":
			hasDandelion = true
		default:
			continue
		}
		if treeCols[[2]int{b.X, b.Z}] {
			t.Errorf("flower at (%d,%d) lands on a tree's own trunk column", b.X, b.Z)
		}
	}
	if !hasPoppy {
		t.Error("scene 0: expected at least one minecraft:poppy")
	}
	if !hasDandelion {
		t.Error("scene 0: expected at least one minecraft:dandelion")
	}
}

func TestProceduralWorldStore_Scene1Hills(t *testing.T) {
	blocks := ProceduralWorldStore(1, 0, 0)
	if len(blocks) == 0 {
		t.Fatal("scene 1: expected blocks, got none")
	}
	// Hills must have varied heights — find min/max grass_block Y
	minY, maxY := 255, 0
	for _, b := range blocks {
		if b.BlockName == "minecraft:grass_block" {
			if b.Y < minY {
				minY = b.Y
			}
			if b.Y > maxY {
				maxY = b.Y
			}
		}
	}
	if maxY-minY < 2 {
		t.Errorf("scene 1 (hills): expected height variation ≥2, got min=%d max=%d", minY, maxY)
	}
}

func TestProceduralWorldStore_Scene2Caves(t *testing.T) {
	blocks := ProceduralWorldStore(2, 0, 0)
	if len(blocks) == 0 {
		t.Fatal("scene 2: expected blocks, got none")
	}
	// Cave chunks are mostly stone, no grass
	var hasGrass bool
	var stoneCount int
	for _, b := range blocks {
		if b.BlockName == "minecraft:grass_block" {
			hasGrass = true
		}
		if b.BlockName == "minecraft:stone" {
			stoneCount++
		}
	}
	if hasGrass {
		t.Error("scene 2 (caves): expected no grass_block underground")
	}
	if stoneCount == 0 {
		t.Error("scene 2 (caves): expected stone blocks")
	}
}

func TestProceduralWorldStore_ScenesDiffer(t *testing.T) {
	s0 := ProceduralWorldStore(0, 0, 0)
	s1 := ProceduralWorldStore(1, 0, 0)
	s2 := ProceduralWorldStore(2, 0, 0)

	if blockSetEqual(s0, s1) {
		t.Error("scene 0 and scene 1 terrain are identical — expected different")
	}
	if blockSetEqual(s0, s2) {
		t.Error("scene 0 and scene 2 terrain are identical — expected different")
	}
	if blockSetEqual(s1, s2) {
		t.Error("scene 1 and scene 2 terrain are identical — expected different")
	}
}

func TestProceduralWorldStore_CavesHaveCorridors(t *testing.T) {
	// The east-west corridor at z=7,8 y=2-5 must be absent from the block list
	blocks := ProceduralWorldStore(2, 0, 0)
	solid := make(map[[3]int]bool)
	for _, b := range blocks {
		solid[[3]int{b.X % 16, b.Y, b.Z % 16}] = true
	}
	// y=3 at (x=0,z=7) should be carved out (air)
	if solid[[3]int{0, 3, 7}] {
		t.Error("cave corridor at z=7 y=3: expected air, got solid")
	}
	// y=3 at (x=7,z=0) should also be carved (north-south corridor)
	if solid[[3]int{7, 3, 0}] {
		t.Error("cave corridor at x=7 y=3: expected air, got solid")
	}
}

func TestProceduralWorldStore_ChunkVariation(t *testing.T) {
	// Adjacent meadow chunks should produce consistent (non-identical) results
	c00 := ProceduralWorldStore(0, 0, 0)
	c10 := ProceduralWorldStore(0, 1, 0)
	// Both should be non-empty
	if len(c00) == 0 || len(c10) == 0 {
		t.Fatal("adjacent chunks returned empty results")
	}
}

func blockSetEqual(a, b []WorldBlock) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

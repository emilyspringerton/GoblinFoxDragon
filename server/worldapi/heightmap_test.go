package worldapi

import "testing"

func TestHeightmapChunk_Meadow_Flat(t *testing.T) {
	heights, ok := HeightmapChunk(0, 0, 0)
	if !ok {
		t.Fatal("scene 0 (meadow): expected ok=true")
	}
	for i, h := range heights {
		if h != 4 {
			t.Fatalf("scene 0 (meadow): column %d: expected height 4, got %d", i, h)
		}
	}
}

func TestHeightmapChunk_Hills_MatchesBlockGeneration(t *testing.T) {
	const chunkX, chunkZ = 2, -3
	heights, ok := HeightmapChunk(1, chunkX, chunkZ)
	if !ok {
		t.Fatal("scene 1 (hills): expected ok=true")
	}

	// Cross-check against the real block list hillsChunk emits, so this test breaks if the
	// heightmap extraction (hillsColumnHeight) ever drifts from the block generation it was
	// split out of.
	blocks := ProceduralWorldStore(1, chunkX, chunkZ)
	wantByColumn := make(map[[2]int]int)
	for _, b := range blocks {
		if b.BlockName == "minecraft:grass_block" {
			lx, lz := b.X-chunkX*16, b.Z-chunkZ*16
			wantByColumn[[2]int{lx, lz}] = b.Y
		}
	}
	if len(wantByColumn) != 256 {
		t.Fatalf("expected 256 grass columns, got %d", len(wantByColumn))
	}
	for lx := 0; lx < 16; lx++ {
		for lz := 0; lz < 16; lz++ {
			want := wantByColumn[[2]int{lx, lz}]
			got := int(heights[lx*16+lz])
			if got != want {
				t.Fatalf("column (%d,%d): heightmap=%d, block generation=%d", lx, lz, got, want)
			}
		}
	}
}

func TestHeightmapChunk_Caves_NotSupported(t *testing.T) {
	_, ok := HeightmapChunk(2, 0, 0)
	if ok {
		t.Fatal("scene 2 (caves): expected ok=false, caves are genuinely 3D, no single height per column")
	}
}

func TestHeightmapChunk_Swamp_WaterHigherThanLand(t *testing.T) {
	heights, ok := HeightmapChunk(3, 0, 0)
	if !ok {
		t.Fatal("scene 3 (swamp): expected ok=true")
	}
	waterSet := swampWaterCells(0, 0)
	for i, isWater := range waterSet {
		if isWater && heights[i] != 2 {
			t.Errorf("column %d: water cell should have height 2, got %d", i, heights[i])
		}
		if !isWater && heights[i] != 1 {
			t.Errorf("column %d: land cell should have height 1, got %d", i, heights[i])
		}
	}
}

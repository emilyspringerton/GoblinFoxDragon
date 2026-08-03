package worldapi

import "testing"

// TestHeightmapChunk_Meadow_GentleRoll (2026-08-03, founder: "meadows are not completely flat
// my bro") -- Meadow used to be perfectly flat (height 4 everywhere); now a real, gentle roll,
// range 3-5, much subtler than Hills' own range (2-8). This replaces the old
// TestHeightmapChunk_Meadow_Flat, which asserted the flat behavior this fix deliberately removed.
func TestHeightmapChunk_Meadow_GentleRoll(t *testing.T) {
	heights, ok := HeightmapChunk(0, 0, 0)
	if !ok {
		t.Fatal("scene 0 (meadow): expected ok=true")
	}
	minH, maxH := uint8(255), uint8(0)
	for i, h := range heights {
		if h < 3 || h > 5 {
			t.Fatalf("scene 0 (meadow): column %d: expected height in [3,5] (gentle roll), got %d", i, h)
		}
		if h < minH {
			minH = h
		}
		if h > maxH {
			maxH = h
		}
	}
	if minH == maxH {
		t.Fatalf("scene 0 (meadow): expected real height variation across the chunk, got a uniform %d everywhere", minH)
	}
}

// TestHeightmapChunk_Meadow_MatchesBlockGeneration cross-checks the heightmap against
// meadowChunk's own real block output, same discipline TestHeightmapChunk_Hills_MatchesBlockGeneration
// already established -- catches drift between meadowColumnHeight and the block generation it
// was split out of.
func TestHeightmapChunk_Meadow_MatchesBlockGeneration(t *testing.T) {
	const chunkX, chunkZ = 2, -3
	heights, ok := HeightmapChunk(0, chunkX, chunkZ)
	if !ok {
		t.Fatal("scene 0 (meadow): expected ok=true")
	}
	blocks := ProceduralWorldStore(0, chunkX, chunkZ)
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

// TestColumnHeight_MatchesHeightmapChunk cross-checks ColumnHeight (the new single-column
// lookup, added for apps2/server-go's own real ground collision) against HeightmapChunk (the
// already-tested whole-chunk version) across several chunks including negative coordinates --
// the two must never disagree, since they're describing the same real terrain to two different
// callers (one HTTP client, one same-process Go caller).
func TestColumnHeight_MatchesHeightmapChunk(t *testing.T) {
	for _, scene := range []int{0, 1, 3} {
		for _, chunk := range [][2]int{{0, 0}, {2, -3}, {-1, -1}, {5, 4}} {
			chunkX, chunkZ := chunk[0], chunk[1]
			want, wantOK := HeightmapChunk(scene, chunkX, chunkZ)
			if !wantOK {
				t.Fatalf("scene %d chunk (%d,%d): HeightmapChunk unexpectedly not ok", scene, chunkX, chunkZ)
			}
			for lx := 0; lx < 16; lx++ {
				for lz := 0; lz < 16; lz++ {
					wx, wz := chunkX*16+lx, chunkZ*16+lz
					got, gotOK := ColumnHeight(scene, wx, wz)
					if !gotOK {
						t.Fatalf("scene %d (%d,%d): ColumnHeight unexpectedly not ok", scene, wx, wz)
					}
					if got != want[lx*16+lz] {
						t.Fatalf("scene %d world(%d,%d) [chunk (%d,%d) local (%d,%d)]: "+
							"ColumnHeight=%d, HeightmapChunk=%d", scene, wx, wz, chunkX, chunkZ,
							lx, lz, got, want[lx*16+lz])
					}
				}
			}
		}
	}
}

func TestColumnHeight_Caves_NotSupported(t *testing.T) {
	_, ok := ColumnHeight(2, 0, 0)
	if ok {
		t.Fatal("scene 2 (caves): expected ok=false")
	}
}

func TestFloorDiv16_NegativeCoordinates(t *testing.T) {
	cases := []struct{ v, want int }{
		{0, 0}, {15, 0}, {16, 1}, {31, 1}, {32, 2},
		{-1, -1}, {-16, -1}, {-17, -2}, {-32, -2}, {-33, -3},
	}
	for _, c := range cases {
		if got := floorDiv16(c.v); got != c.want {
			t.Errorf("floorDiv16(%d) = %d, want %d", c.v, got, c.want)
		}
	}
}

package worldapi

// Heightmap exposure (SMOOTH_TERRAIN_NORTHSTAR.md §3.1/Milestone 1, founder: "render the
// dragonfly biomes smooth with trees"). ProceduralWorldStore's column-derived scenes already
// compute a height per (x,z) before emitting blocks -- this exposes that value directly instead
// of making a client reconstruct it by fetching and scanning the full block list from /chunks.
// Only scenes whose generation genuinely reduces to one height per column are supported; Caves
// (scene 2) is a real 3D solid grid and has no single height per column (see
// SMOOTH_TERRAIN_NORTHSTAR.md §4) -- HeightmapChunk returns ok=false for it, not a guess.

// HeightmapChunk fills a 16x16 grid (row-major, index = local x*16+local z, matching
// swampWaterCells' own layout convention) with the generated surface height for every column in
// chunk (chunkX, chunkZ). ok is false when sceneID has no height-per-column view at all.
func HeightmapChunk(sceneID, chunkX, chunkZ int) (heights [256]uint8, ok bool) {
	const chunkSize = 16
	switch sceneID {
	case 0: // Meadow -- real gentle rolling terrain, shares meadowChunk's own formula (2026-08-03,
		// founder: "meadows are not completely flat my bro" -- no longer a hardcoded flat 4)
		for lz := 0; lz < chunkSize; lz++ {
			for lx := 0; lx < chunkSize; lx++ {
				wx := chunkX*chunkSize + lx
				wz := chunkZ*chunkSize + lz
				heights[lx*chunkSize+lz] = uint8(meadowColumnHeight(wx, wz))
			}
		}
		return heights, true
	case 1: // Hills -- real per-column variation, shares hillsChunk's own formula
		for lz := 0; lz < chunkSize; lz++ {
			for lx := 0; lx < chunkSize; lx++ {
				wx := chunkX*chunkSize + lx
				wz := chunkZ*chunkSize + lz
				heights[lx*chunkSize+lz] = uint8(hillsColumnHeight(wx, wz))
			}
		}
		return heights, true
	case 3: // Swampville -- flat, water cells sit one block higher (the water surface itself)
		waterSet := swampWaterCells(chunkX, chunkZ)
		for i := range heights {
			if waterSet[i] {
				heights[i] = 2
			} else {
				heights[i] = 1
			}
		}
		return heights, true
	default:
		var zero [256]uint8
		return zero, false
	}
}

// ColumnHeight returns the real generated surface height for a single world column (wx, wz) --
// the same per-column math HeightmapChunk uses, without generating the other 255 columns of the
// chunk it belongs to (backend-unification, 2026-08-03, real Y-axis ground collision in
// apps2/server-go -- a per-tick, per-player lookup, wasteful to route through the whole-chunk
// HeightmapChunk for). Same scene support as HeightmapChunk (Meadow/Hills/Swampville; Caves
// returns ok=false, no single height per column).
func ColumnHeight(sceneID, wx, wz int) (height uint8, ok bool) {
	switch sceneID {
	case 0: // Meadow -- real gentle rolling terrain, shares meadowChunk's own formula
		return uint8(meadowColumnHeight(wx, wz)), true
	case 1: // Hills -- real per-column variation
		return uint8(hillsColumnHeight(wx, wz)), true
	case 3: // Swampville -- flat, water cells one block higher
		chunkX, chunkZ := floorDiv16(wx), floorDiv16(wz)
		lx, lz := wx-chunkX*16, wz-chunkZ*16
		waterSet := swampWaterCells(chunkX, chunkZ)
		if waterSet[lx*16+lz] {
			return 2, true
		}
		return 1, true
	default:
		return 0, false
	}
}

// floorDiv16 is real floor division by 16 -- Go's own `/` truncates toward zero for negative
// operands (same as C), which gives the wrong chunk for negative world coordinates (wx=-1 should
// be chunk -1, truncating division gives chunk 0). World columns go negative routinely (this is
// an origin-centered coordinate system, confirmed by every scenes_test.go/heightmap_test.go case
// already using negative chunk coords), so this isn't an edge case to skip.
func floorDiv16(v int) int {
	if v >= 0 {
		return v / 16
	}
	return -((-v + 15) / 16)
}

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
	case 0: // Meadow -- flat, see meadowChunk's own grassY
		for i := range heights {
			heights[i] = 4
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

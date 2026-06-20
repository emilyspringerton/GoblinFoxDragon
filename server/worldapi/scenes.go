package worldapi

import "math"

// ProceduralWorldStore is a scene-differentiated WorldStore function for
// SHANKPIT scenes. Pass it to NewDragonflyChunkGenerator to give each SHANKPIT
// scene visually distinct Dragonfly-sourced terrain.
//
// sceneID=0 → flat meadow (grass surface + oak trees, classic FPS arena feel)
// sceneID=1 → rolling hills (sinusoidal height variation, open sightlines)
// sceneID=2 → stone caves (underground corridors carved into solid stone)
// all other → flat meadow fallback
func ProceduralWorldStore(sceneID, chunkX, chunkZ int) []WorldBlock {
	switch sceneID {
	case 1:
		return hillsChunk(chunkX, chunkZ)
	case 2:
		return cavesChunk(chunkX, chunkZ)
	default:
		return meadowChunk(chunkX, chunkZ)
	}
}

// ── scene 0: flat meadow ──────────────────────────────────────────────────────

// meadowChunk generates a flat grassy meadow: stone base, dirt layer, grass surface, sparse trees.
func meadowChunk(chunkX, chunkZ int) []WorldBlock {
	const (
		chunkSize = 16
		stoneTop  = 2 // stone from y=0..stoneTop
		dirtY     = 3
		grassY    = 4
	)
	out := make([]WorldBlock, 0, 512)
	for z := 0; z < chunkSize; z++ {
		for x := 0; x < chunkSize; x++ {
			wx := chunkX*chunkSize + x
			wz := chunkZ*chunkSize + z
			for y := 0; y <= stoneTop; y++ {
				out = append(out, WorldBlock{X: wx, Y: y, Z: wz, BlockName: "minecraft:stone"})
			}
			out = append(out, WorldBlock{X: wx, Y: dirtY, Z: wz, BlockName: "minecraft:dirt"})
			out = append(out, WorldBlock{X: wx, Y: grassY, Z: wz, BlockName: "minecraft:grass_block"})
		}
	}
	// Trees: slightly more than procedural
	trees := meadowTrees(chunkX, chunkZ)
	for _, t := range trees {
		out = append(out, worldTree(chunkX, chunkZ, t[0], t[1], grassY+1)...)
	}
	return out
}

func meadowTrees(chunkX, chunkZ int) [][2]int {
	// Use a deterministic per-chunk hash to scatter trees
	h := chunkX*31 + chunkZ*17
	switch h % 5 {
	case 0:
		return [][2]int{{4, 4}, {12, 11}}
	case 1:
		return [][2]int{{7, 3}}
	case 2:
		return [][2]int{{2, 9}, {13, 5}, {8, 13}}
	case 3:
		return [][2]int{{5, 7}}
	default:
		return nil
	}
}

// ── scene 1: rolling hills ────────────────────────────────────────────────────

// hillsChunk generates rolling terrain with height variation from 3–7 blocks.
// No trees — open sightlines for long-range combat.
func hillsChunk(chunkX, chunkZ int) []WorldBlock {
	const chunkSize = 16
	out := make([]WorldBlock, 0, 1024)
	for z := 0; z < chunkSize; z++ {
		for x := 0; x < chunkSize; x++ {
			wx := chunkX*chunkSize + x
			wz := chunkZ*chunkSize + z
			// Sinusoidal height: 4 ± 2 blocks variation
			height := 4 + int(math.Round(
				2.0*math.Sin(float64(wx)*0.35) +
					1.5*math.Cos(float64(wz)*0.28) +
					0.8*math.Sin(float64(wx+wz)*0.17),
			))
			if height < 2 {
				height = 2
			}
			if height > 8 {
				height = 8
			}
			// Stone base
			for y := 0; y <= height-2; y++ {
				out = append(out, WorldBlock{X: wx, Y: y, Z: wz, BlockName: "minecraft:stone"})
			}
			// Dirt sub-surface
			if height >= 1 {
				out = append(out, WorldBlock{X: wx, Y: height - 1, Z: wz, BlockName: "minecraft:dirt"})
			}
			// Grass cap
			out = append(out, WorldBlock{X: wx, Y: height, Z: wz, BlockName: "minecraft:grass_block"})
		}
	}
	return out
}

// ── scene 2: stone caves ──────────────────────────────────────────────────────

// cavesChunk generates a solid stone mass with two carved corridors:
//   - East-west corridor along z=7,8 at y=2-4
//   - North-south corridor along x=7,8 at y=2-4
//
// The dirt floor in corridors gives the bot traction.
func cavesChunk(chunkX, chunkZ int) []WorldBlock {
	const (
		chunkSize  = 16
		caveFloor  = 2
		caveCeil   = 5
		solidTop   = 8
		corridorA1 = 7
		corridorA2 = 8
	)

	type solid [3]int
	solid3D := make([][]bool, solidTop+1)
	for y := range solid3D {
		solid3D[y] = make([]bool, chunkSize*chunkSize)
	}
	idx := func(x, z int) int { return z*chunkSize + x }

	// Fill everything solid
	for y := 0; y <= solidTop; y++ {
		for i := range solid3D[y] {
			solid3D[y][i] = true
		}
	}

	// Carve east-west corridor (all x, z=7..8, y=caveFloor..caveCeil)
	for z := corridorA1; z <= corridorA2; z++ {
		for x := 0; x < chunkSize; x++ {
			for y := caveFloor; y <= caveCeil; y++ {
				solid3D[y][idx(x, z)] = false
			}
		}
	}

	// Carve north-south corridor (x=7..8, all z, y=caveFloor..caveCeil)
	for x := corridorA1; x <= corridorA2; x++ {
		for z := 0; z < chunkSize; z++ {
			for y := caveFloor; y <= caveCeil; y++ {
				solid3D[y][idx(x, z)] = false
			}
		}
	}

	out := make([]WorldBlock, 0, 1024)
	for z := 0; z < chunkSize; z++ {
		for x := 0; x < chunkSize; x++ {
			wx := chunkX*chunkSize + x
			wz := chunkZ*chunkSize + z
			for y := 0; y <= solidTop; y++ {
				if !solid3D[y][idx(x, z)] {
					continue
				}
				// Dirt floor in corridors (y == caveFloor-1 adjacent to air)
				name := "minecraft:stone"
				if y == caveFloor-1 && !solid3D[caveFloor][idx(x, z)] {
					name = "minecraft:dirt"
				}
				out = append(out, WorldBlock{X: wx, Y: y, Z: wz, BlockName: name})
			}
		}
	}
	return out
}

// ── shared helpers ────────────────────────────────────────────────────────────

// worldTree places an oak tree with trunk base at world coords (wx,wz), bottom of trunk at y=baseY.
func worldTree(chunkX, chunkZ, lx, lz, baseY int) []WorldBlock {
	wx := chunkX*16 + lx
	wz := chunkZ*16 + lz
	out := make([]WorldBlock, 0, 12)
	for dy := 0; dy < 4; dy++ {
		out = append(out, WorldBlock{X: wx, Y: baseY + dy, Z: wz, BlockName: "minecraft:oak_log"})
	}
	leafY := baseY + 4
	for dz := -1; dz <= 1; dz++ {
		for dx := -1; dx <= 1; dx++ {
			out = append(out, WorldBlock{X: wx + dx, Y: leafY, Z: wz + dz, BlockName: "minecraft:oak_leaves"})
		}
	}
	out = append(out, WorldBlock{X: wx, Y: leafY + 1, Z: wz, BlockName: "minecraft:oak_leaves"})
	return out
}

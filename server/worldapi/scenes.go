package worldapi

import "math"

// ProceduralWorldStore is a scene-differentiated WorldStore function for
// SHANKPIT scenes. Pass it to NewDragonflyChunkGenerator to give each SHANKPIT
// scene visually distinct Dragonfly-sourced terrain.
//
// sceneID=0 → flat meadow (grass surface + oak trees, classic FPS arena feel)
// sceneID=1 → rolling hills (sinusoidal height variation, open sightlines)
// sceneID=2 → stone caves (underground corridors carved into solid stone)
// sceneID=3 → Swampville (clay/mud floor, shallow water pools, dead mangrove trees)
// all other → flat meadow fallback
func ProceduralWorldStore(sceneID, chunkX, chunkZ int) []WorldBlock {
	switch sceneID {
	case 1:
		return hillsChunk(chunkX, chunkZ)
	case 2:
		return cavesChunk(chunkX, chunkZ)
	case 3:
		return swampChunk(chunkX, chunkZ)
	// TRAPX city districts / TYLER scene cluster (S122-01 + S123-01)
	case 200, 201, 202, 203, 204, 205, 206, 207:
		return urbanChunk(sceneID, chunkX, chunkZ)
	default:
		return meadowChunk(chunkX, chunkZ)
	}
}

// urbanChunk generates a flat concrete city block with apartment walls (scenes 200–204).
// Block palette: stone_bricks (concrete), chiseled_stone (road), oak_planks (interior floors).
// District 200 (Residential): apartment stacks. 201 (Commercial): strip-mall facades.
// 202 (Industrial): warehouse walls. 203 (Underground): tunnel ceiling + floor.
// 204 (Abandoned): crumbled stone_bricks variants.
func urbanChunk(sceneID, chunkX, chunkZ int) []WorldBlock {
	out := make([]WorldBlock, 0, 256)
	// Ground plane: concrete road surface
	roadBlock := "minecraft:stone_bricks"
	if sceneID == 203 { // underground: coarse stone floor
		roadBlock = "minecraft:cobblestone"
	}
	for lx := 0; lx < 16; lx++ {
		for lz := 0; lz < 16; lz++ {
			wx := chunkX*16 + lx
			wz := chunkZ*16 + lz
			out = append(out, WorldBlock{X: wx, Y: 64, Z: wz, BlockName: roadBlock})
			// Sub-surface: solid stone
			for dy := 1; dy <= 3; dy++ {
				out = append(out, WorldBlock{X: wx, Y: 64 - dy, Z: wz, BlockName: "minecraft:stone"})
			}
		}
	}

	// District-specific features.
	switch sceneID {
	case 200: // Residential: apartment block on north and east edges
		out = append(out, urbanApartmentBlock(chunkX, chunkZ, 0, 65, 0, 5)...)
		out = append(out, urbanApartmentBlock(chunkX, chunkZ, 12, 65, 0, 5)...)
	case 201: // Commercial: strip-mall facade on north edge
		out = append(out, urbanFacadeRow(chunkX, chunkZ, 0, 65, 4)...)
	case 202: // Industrial: warehouse walls on perimeter
		out = append(out, urbanWarehouseWalls(chunkX, chunkZ)...)
	case 203: // Underground: tunnel ceiling at Y=72
		for lx := 0; lx < 16; lx++ {
			for lz := 0; lz < 16; lz++ {
				wx := chunkX*16 + lx
				wz := chunkZ*16 + lz
				out = append(out, WorldBlock{X: wx, Y: 72, Z: wz, BlockName: "minecraft:stone"})
			}
		}
	case 204: // Vatican Corridors: crumbled stone with salt-water channels
		for _, pos := range [][3]int{{3, 65, 3}, {7, 65, 9}, {11, 65, 4}, {1, 65, 13}} {
			wx := chunkX*16 + pos[0]
			wz := chunkZ*16 + pos[2]
			for dy := 0; dy < pos[1]-64; dy++ {
				out = append(out, WorldBlock{X: wx, Y: 64 + dy, Z: wz, BlockName: "minecraft:cracked_stone_bricks"})
			}
		}
		// Narrow water channel along center (z=7)
		for lx := 0; lx < 16; lx++ {
			wx := chunkX*16 + lx
			wz := chunkZ*16 + 7
			out = append(out, WorldBlock{X: wx, Y: 64, Z: wz, BlockName: "minecraft:water"})
		}
	case 205: // Osaka Underport: flooded docks with iron bulkhead doors
		for lx := 0; lx < 16; lx++ {
			for lz := 0; lz < 16; lz++ {
				wx := chunkX*16 + lx
				wz := chunkZ*16 + lz
				if lz < 3 || lz > 12 { // flooded edges
					out = append(out, WorldBlock{X: wx, Y: 64, Z: wz, BlockName: "minecraft:water"})
				}
			}
		}
		// Iron bulkhead doors on east and west walls
		for lz := 3; lz <= 12; lz++ {
			wx0 := chunkX * 16
			wx1 := chunkX*16 + 15
			wz := chunkZ*16 + lz
			for dy := 0; dy < 3; dy++ {
				out = append(out, WorldBlock{X: wx0, Y: 65 + dy, Z: wz, BlockName: "minecraft:iron_bars"})
				out = append(out, WorldBlock{X: wx1, Y: 65 + dy, Z: wz, BlockName: "minecraft:iron_bars"})
			}
		}
	case 206: // Kuroshio Coast: rocky shoreline with kelp and cliff face
		for lx := 0; lx < 16; lx++ {
			for lz := 0; lz < 16; lz++ {
				wx := chunkX*16 + lx
				wz := chunkZ*16 + lz
				if lz > 10 { // ocean side
					out = append(out, WorldBlock{X: wx, Y: 64, Z: wz, BlockName: "minecraft:water"})
					// Kelp scattered
					if (lx+lz+chunkX+chunkZ)%4 == 0 {
						out = append(out, WorldBlock{X: wx, Y: 65, Z: wz, BlockName: "minecraft:kelp"})
					}
				} else {
					out = append(out, WorldBlock{X: wx, Y: 64, Z: wz, BlockName: "minecraft:gravel"})
				}
			}
		}
		// Cliff face on north edge (z=0..2): packed stone columns
		for lx := 0; lx < 16; lx++ {
			wx := chunkX*16 + lx
			for lz := 0; lz <= 2; lz++ {
				wz := chunkZ*16 + lz
				for dy := 0; dy < 4; dy++ {
					out = append(out, WorldBlock{X: wx, Y: 65 + dy, Z: wz, BlockName: "minecraft:stone"})
				}
			}
		}
	case 207: // Bacon's Table: collapsed warehouse, neon shrine-gate arch
		// Scattered debris floor
		for _, pos := range [][2]int{{2, 3}, {5, 8}, {9, 4}, {13, 11}, {7, 7}, {3, 13}} {
			wx := chunkX*16 + pos[0]
			wz := chunkZ*16 + pos[1]
			out = append(out, WorldBlock{X: wx, Y: 65, Z: wz, BlockName: "minecraft:soul_sand"})
		}
		// Shrine-gate arch: two pillars + crossbar at entrance (z=0)
		for _, lx := range []int{5, 10} {
			wx := chunkX*16 + lx
			wz := chunkZ * 16
			for dy := 0; dy < 4; dy++ {
				out = append(out, WorldBlock{X: wx, Y: 65 + dy, Z: wz, BlockName: "minecraft:crimson_planks"})
			}
		}
		// Crossbar
		for lx := 5; lx <= 10; lx++ {
			wx := chunkX*16 + lx
			wz := chunkZ * 16
			out = append(out, WorldBlock{X: wx, Y: 68, Z: wz, BlockName: "minecraft:crimson_planks"})
		}
	}
	return out
}

func urbanApartmentBlock(chunkX, chunkZ, lx, baseY, lz, height int) []WorldBlock {
	out := make([]WorldBlock, 0, height*4)
	wx := chunkX*16 + lx
	wz := chunkZ*16 + lz
	for dy := 0; dy < height; dy++ {
		out = append(out, WorldBlock{X: wx, Y: baseY + dy, Z: wz, BlockName: "minecraft:stone_bricks"})
		out = append(out, WorldBlock{X: wx + 1, Y: baseY + dy, Z: wz, BlockName: "minecraft:stone_bricks"})
		out = append(out, WorldBlock{X: wx, Y: baseY + dy, Z: wz + 1, BlockName: "minecraft:stone_bricks"})
	}
	return out
}

func urbanFacadeRow(chunkX, chunkZ, lx, baseY, height int) []WorldBlock {
	out := make([]WorldBlock, 0, 16*height)
	for x := 0; x < 16; x++ {
		wx := chunkX*16 + lx + x
		wz := chunkZ * 16
		for dy := 0; dy < height; dy++ {
			out = append(out, WorldBlock{X: wx, Y: baseY + dy, Z: wz, BlockName: "minecraft:smooth_stone"})
		}
	}
	return out
}

func urbanWarehouseWalls(chunkX, chunkZ int) []WorldBlock {
	out := make([]WorldBlock, 0, 64)
	baseY := 65
	height := 6
	for lx := 0; lx < 16; lx++ {
		for _, lz := range []int{0, 15} {
			wx := chunkX*16 + lx
			wz := chunkZ*16 + lz
			for dy := 0; dy < height; dy++ {
				out = append(out, WorldBlock{X: wx, Y: baseY + dy, Z: wz, BlockName: "minecraft:iron_bars"})
			}
		}
	}
	return out
}

// ── scene 0: flat meadow ──────────────────────────────────────────────────────

// meadowChunk generates a flat grassy meadow: stone base, dirt layer, grass surface, sparse trees.
// meadowColumnHeight (2026-08-03, founder: "meadows are not completely flat my bro" -- real,
// grounded feedback, correcting a real design choice, not a rendering bug: meadowChunk had
// always used a hardcoded flat grassY=4 for every column, both here and in the heightmap
// exposure that mirrors it). A gentle rolling terrain, much subtler than hillsColumnHeight's own
// dramatic variation (amplitude ~3, range 2-8) -- meadows should read as a soft, walkable roll,
// not hills. Range 3-5, centered on the original flat height so existing content built around
// "meadow height 4" (worldTree's own base, apps2/battlegrounds_gui's own client fallback
// groundY=4) stays reasonable, not shifted wholesale.
func meadowColumnHeight(wx, wz int) int {
	height := 4 + int(math.Round(
		0.8*math.Sin(float64(wx)*0.15)+
			0.6*math.Cos(float64(wz)*0.13),
	))
	if height < 3 {
		height = 3
	}
	if height > 5 {
		height = 5
	}
	return height
}

func meadowChunk(chunkX, chunkZ int) []WorldBlock {
	const (
		chunkSize = 16
		stoneTop  = 2 // stone from y=0..stoneTop
		dirtY     = 3
	)
	out := make([]WorldBlock, 0, 512)
	for z := 0; z < chunkSize; z++ {
		for x := 0; x < chunkSize; x++ {
			wx := chunkX*chunkSize + x
			wz := chunkZ*chunkSize + z
			grassY := meadowColumnHeight(wx, wz)
			for y := 0; y <= stoneTop; y++ {
				out = append(out, WorldBlock{X: wx, Y: y, Z: wz, BlockName: "minecraft:stone"})
			}
			for y := stoneTop + 1; y < grassY; y++ {
				out = append(out, WorldBlock{X: wx, Y: y, Z: wz, BlockName: "minecraft:dirt"})
			}
			out = append(out, WorldBlock{X: wx, Y: grassY, Z: wz, BlockName: "minecraft:grass_block"})
		}
	}
	// Trees: slightly more than procedural. Rooted at the real height under each tree's own
	// (x,z), not a fixed grassY -- otherwise trees would float above or sink into the now-real
	// rolling ground.
	trees := meadowTrees(chunkX, chunkZ)
	for _, t := range trees {
		wx, wz := chunkX*chunkSize+t[0], chunkZ*chunkSize+t[1]
		out = append(out, worldTree(chunkX, chunkZ, t[0], t[1], meadowColumnHeight(wx, wz)+1)...)
	}
	return out
}

// meadowTrees returns this chunk's real tree spots, deterministic per (chunkX, chunkZ) so a
// client asking for the same chunk twice always gets the same forest (apps2/battlegrounds_gui's
// own town_meadow_tree_positions mirrors this exactly, hash and all -- "kept in sync by hand" per
// its own doc comment). Density bumped 2026-08-03, founder: "i dont see any updates yet expanding
// our meadow scene adding more trees" -- the original 5-bucket table topped out at 3 trees per
// chunk (and one bucket, h%5==4, had none at all), which read as "empty field," not "meadow." Every
// bucket now returns 5-6 trees with no bare bucket, still the same real, reproducible hash.
func meadowTrees(chunkX, chunkZ int) [][2]int {
	// Use a deterministic per-chunk hash to scatter trees
	h := chunkX*31 + chunkZ*17
	switch h % 5 {
	case 0:
		return [][2]int{{2, 2}, {4, 11}, {9, 3}, {12, 9}, {6, 14}, {14, 5}}
	case 1:
		return [][2]int{{3, 6}, {8, 2}, {11, 12}, {5, 9}, {14, 14}, {2, 13}}
	case 2:
		return [][2]int{{2, 9}, {13, 5}, {8, 13}, {5, 2}, {11, 8}, {3, 14}}
	case 3:
		return [][2]int{{5, 7}, {9, 12}, {2, 4}, {13, 10}, {7, 2}, {12, 14}}
	default:
		return [][2]int{{6, 6}, {10, 3}, {3, 11}, {13, 13}, {8, 8}}
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
			height := hillsColumnHeight(wx, wz)
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

// hillsColumnHeight returns the same sinusoidal height hillsChunk builds blocks up
// to for world column (wx, wz), clamped to 2-8. Split out (SMOOTH_TERRAIN_NORTHSTAR.md
// Milestone 1, "backend heightmap exposure") so the /heightmap endpoint and hillsChunk's
// own block generation can never drift apart — one formula, two callers, not two copies
// of the same math to keep in sync by hand.
func hillsColumnHeight(wx, wz int) int {
	height := 4 + int(math.Round(
		2.0*math.Sin(float64(wx)*0.35)+
			1.5*math.Cos(float64(wz)*0.28)+
			0.8*math.Sin(float64(wx+wz)*0.17),
	))
	if height < 2 {
		height = 2
	}
	if height > 8 {
		height = 8
	}
	return height
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

// ── scene 3: Swampville ───────────────────────────────────────────────────────

// swampChunk generates low-lying swamp terrain:
//   - y=0 stone base, y=1 clay floor, y=1 water pools (deterministic per-chunk)
//   - sparse mud patches at surface
//   - dead mangrove trees (jungle log + sparse leaves, no canopy density)
//
// The flat y=1 surface matches the Swampville zone SpawnY=1.
func swampChunk(chunkX, chunkZ int) []WorldBlock {
	const chunkSize = 16
	out := make([]WorldBlock, 0, 512)

	// Per-chunk water-pool mask: some local coords are water at y=1.
	waterSet := swampWaterCells(chunkX, chunkZ)
	isWater := func(lx, lz int) bool {
		return waterSet[lx*chunkSize+lz]
	}

	for z := 0; z < chunkSize; z++ {
		for x := 0; x < chunkSize; x++ {
			wx := chunkX*chunkSize + x
			wz := chunkZ*chunkSize + z

			// Stone base
			out = append(out, WorldBlock{X: wx, Y: 0, Z: wz, BlockName: "minecraft:stone"})

			if isWater(x, z) {
				// Shallow water cell: clay bottom + water surface
				out = append(out, WorldBlock{X: wx, Y: 1, Z: wz, BlockName: "minecraft:clay"})
				out = append(out, WorldBlock{X: wx, Y: 2, Z: wz, BlockName: "minecraft:water"})
			} else {
				// Dry land: clay with occasional mud patch
				floor := "minecraft:clay"
				h := (x*7 + z*13 + chunkX*3 + chunkZ*5) & 0xF
				if h < 3 {
					floor = "minecraft:mud"
				}
				out = append(out, WorldBlock{X: wx, Y: 1, Z: wz, BlockName: floor})
			}
		}
	}

	// Dead mangrove trees on dry land only.
	for _, t := range swampTrees(chunkX, chunkZ) {
		lx, lz := t[0], t[1]
		if !isWater(lx, lz) {
			out = append(out, swampTree(chunkX, chunkZ, lx, lz, 2)...)
		}
	}

	return out
}

// swampWaterCells returns a flat [16×16]bool marking water cells for a chunk.
// Uses a cheap deterministic hash — about 25% of cells are water, forming
// irregular scattered pools.
func swampWaterCells(chunkX, chunkZ int) [256]bool {
	var mask [256]bool
	seed := chunkX*1031 + chunkZ*7919
	for z := 0; z < 16; z++ {
		for x := 0; x < 16; x++ {
			h := seed ^ (x*431) ^ (z*521)
			h = (h ^ (h >> 4)) * 0x45d9f3b
			h = (h ^ (h >> 16))
			if h&0x3 == 0 { // ~25% water
				mask[x*16+z] = true
			}
		}
	}
	return mask
}

func swampTrees(chunkX, chunkZ int) [][2]int {
	h := chunkX*53 + chunkZ*37
	switch h % 6 {
	case 0:
		return [][2]int{{3, 3}, {11, 10}}
	case 1:
		return [][2]int{{6, 2}, {13, 13}}
	case 2:
		return [][2]int{{2, 8}}
	case 3:
		return [][2]int{{9, 5}, {4, 12}, {13, 2}}
	case 4:
		return [][2]int{{1, 1}, {14, 14}}
	default:
		return [][2]int{{7, 9}}
	}
}

// swampTree places a dead mangrove tree: bare trunk (3 tall) + sparse leaf crown.
func swampTree(chunkX, chunkZ, lx, lz, baseY int) []WorldBlock {
	wx := chunkX*16 + lx
	wz := chunkZ*16 + lz
	out := make([]WorldBlock, 0, 8)
	for dy := 0; dy < 3; dy++ {
		out = append(out, WorldBlock{X: wx, Y: baseY + dy, Z: wz, BlockName: "minecraft:jungle_log"})
	}
	// Sparse leaf crown — four cardinal leaves only (dead tree look)
	leafY := baseY + 3
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		out = append(out, WorldBlock{X: wx + d[0], Y: leafY, Z: wz + d[1], BlockName: "minecraft:jungle_leaves"})
	}
	out = append(out, WorldBlock{X: wx, Y: leafY, Z: wz, BlockName: "minecraft:jungle_leaves"})
	return out
}

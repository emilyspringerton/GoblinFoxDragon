package worldapi

import "math/rand"

// dungeon.go — DUNGEON_NORTHSTAR.md Milestone 2 ("Seeded room/corridor generator... Multi-room
// layout, connected, reachable entrance-to-boss-room, regenerates differently per instance").
// Real, deliberately minimal first generator, per the northstar's own §3.2 scope note: "Diablo
// 2's own approach (preset room 'tiles' from a pool, stitched together with connectivity rules)
// is the closest real precedent to build toward -- not attempted here, this section only
// establishes where the generator lives and what it emits." This is that first, honest slice:
// a seeded sequence of rectangular rooms connected by straight corridors in a linear chain
// (room i -> room i+1), which guarantees reachability entrance-to-boss-room BY CONSTRUCTION
// (every room has exactly one path in from the previous room and one path out to the next) --
// real Diablo-style preset-tile stitching with branching/loops is a genuinely separate, later
// improvement, not built here.
//
// Emits the same WorldBlock-shaped output worldapi's existing scene generators already produce
// (per the northstar's own §3.2), using the same solid-grid-then-carve representation
// cavesChunk already demonstrates (see scenes.go) -- generalized from a fixed 16x16 chunk to an
// arbitrary-size grid sized to the whole dungeon layout, since a dungeon instance is generated
// once per instance/seed, not queried per streamed chunk the way the persistent-zone scenes are.
// Wiring this into the real chunk-streaming HTTP path (so a live dungeon server could actually
// serve it) is real, separate, later work (Milestone 4/1), not attempted here.

const (
	// DungeonMinRooms/DungeonMaxRooms bound how many rooms a seeded layout gets -- a real,
	// deliberately small range for this first generator (Diablo 2's own real dungeons run much
	// larger; this is the minimal slice the northstar's own §3.2 scopes for now).
	DungeonMinRooms = 5
	DungeonMaxRooms = 8

	dungeonRoomMinSize = 5
	dungeonRoomMaxSize = 9
	dungeonRoomGapMin  = 6  // minimum center-to-center gap along the placement axis
	dungeonRoomGapMax  = 12
	dungeonFloorY      = 64
	dungeonCeilY       = 68
	dungeonCorridorW   = 3 // corridor width, matching a room-sized doorway rather than a 1-wide crawl
)

// DungeonRoom is one rectangular room in dungeon-local space (not world chunk space -- the
// caller positions the whole layout wherever it needs to, same as any other WorldBlock producer).
type DungeonRoom struct {
	CenterX, CenterZ int
	W, D             int // width (X extent) and depth (Z extent)
}

// DungeonCorridor connects two rooms by index (into DungeonLayout.Rooms) with a straight,
// L-shaped path (X-then-Z, matching cavesChunk's own axis-aligned corridor carving).
type DungeonCorridor struct {
	FromRoom, ToRoom int
}

// DungeonLayout is one fully-generated, seeded dungeon instance -- real topology (which rooms
// exist, how they connect), not yet block-carved.
type DungeonLayout struct {
	Seed        int64
	Rooms       []DungeonRoom
	Corridors   []DungeonCorridor
	EntranceIdx int // always 0 -- named explicitly so callers don't hardcode the assumption
	BossIdx     int // always len(Rooms)-1
}

// GenerateDungeonLayout builds a real, seeded multi-room layout: the same seed always produces
// the same layout (real determinism, needed so the matchmaker's own per-match seed -- see
// REDGARDEN's MatchFoundMsg, Milestone 1 -- can let a server and its clients agree on one
// dungeon without a separate data round-trip), and different seeds produce different room
// counts/sizes/placements.
func GenerateDungeonLayout(seed int64) DungeonLayout {
	rng := rand.New(rand.NewSource(seed))
	n := DungeonMinRooms + rng.Intn(DungeonMaxRooms-DungeonMinRooms+1)

	rooms := make([]DungeonRoom, 0, n)
	x, z := 0, 0
	for i := 0; i < n; i++ {
		w := dungeonRoomMinSize + rng.Intn(dungeonRoomMaxSize-dungeonRoomMinSize+1)
		d := dungeonRoomMinSize + rng.Intn(dungeonRoomMaxSize-dungeonRoomMinSize+1)
		rooms = append(rooms, DungeonRoom{CenterX: x, CenterZ: z, W: w, D: d})

		// Alternate placement axis (X then Z then X...) so the dungeon snakes rather than
		// running in one dead-straight line -- real, minimal variety without a full graph.
		gap := dungeonRoomGapMin + rng.Intn(dungeonRoomGapMax-dungeonRoomGapMin+1)
		if i%2 == 0 {
			x += gap
		} else {
			z += gap
		}
	}

	corridors := make([]DungeonCorridor, 0, n-1)
	for i := 0; i < n-1; i++ {
		corridors = append(corridors, DungeonCorridor{FromRoom: i, ToRoom: i + 1})
	}

	return DungeonLayout{
		Seed:        seed,
		Rooms:       rooms,
		Corridors:   corridors,
		EntranceIdx: 0,
		BossIdx:     n - 1,
	}
}

// IsReachable does a real BFS over the corridor graph confirming the entrance room can actually
// reach the boss room -- Milestone 2's own explicit acceptance bar ("reachable entrance-to-
// boss-room"), verified rather than assumed true "by construction" without checking.
func (l DungeonLayout) IsReachable() bool {
	adj := make(map[int][]int, len(l.Rooms))
	for _, c := range l.Corridors {
		adj[c.FromRoom] = append(adj[c.FromRoom], c.ToRoom)
		adj[c.ToRoom] = append(adj[c.ToRoom], c.FromRoom)
	}
	visited := make(map[int]bool, len(l.Rooms))
	queue := []int{l.EntranceIdx}
	visited[l.EntranceIdx] = true
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == l.BossIdx {
			return true
		}
		for _, next := range adj[cur] {
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}
	return len(l.Rooms) == 1 && l.EntranceIdx == l.BossIdx
}

// DungeonLayoutToBlocks carves the layout into a real WorldBlock list using the same
// solid-grid-then-carve representation cavesChunk already demonstrates (scenes.go): fill a
// bounding solid grid with stone, carve room interiors and corridor paths to air, and lay a
// dirt floor at the carved boundary. originX/originZ let the caller place the whole dungeon
// anywhere in world space (e.g. offset per-instance so multiple concurrent dungeon instances
// don't collide in the same coordinate space -- a real, separate placement concern the caller
// owns, not decided here).
func DungeonLayoutToBlocks(l DungeonLayout, originX, originZ int) []WorldBlock {
	if len(l.Rooms) == 0 {
		return nil
	}
	first := l.Rooms[0]
	minX, maxX := first.CenterX-first.W/2-1, first.CenterX+first.W/2+1
	minZ, maxZ := first.CenterZ-first.D/2-1, first.CenterZ+first.D/2+1
	for _, r := range l.Rooms[1:] {
		if x0, x1 := r.CenterX-r.W/2-1, r.CenterX+r.W/2+1; x0 < minX || x1 > maxX {
			if x0 < minX {
				minX = x0
			}
			if x1 > maxX {
				maxX = x1
			}
		}
		if z0, z1 := r.CenterZ-r.D/2-1, r.CenterZ+r.D/2+1; z0 < minZ || z1 > maxZ {
			if z0 < minZ {
				minZ = z0
			}
			if z1 > maxZ {
				maxZ = z1
			}
		}
	}
	width := maxX - minX + 1
	depth := maxZ - minZ + 1
	if width <= 0 || depth <= 0 {
		return nil
	}

	solidTop := dungeonCeilY
	solid := make([][]bool, solidTop+1)
	for y := range solid {
		solid[y] = make([]bool, width*depth)
		for i := range solid[y] {
			solid[y][i] = true
		}
	}
	idx := func(lx, lz int) int { return lz*width + lx }
	carve := func(lx, lz int) {
		if lx < 0 || lx >= width || lz < 0 || lz >= depth {
			return
		}
		for y := dungeonFloorY; y <= solidTop-1; y++ {
			solid[y][idx(lx, lz)] = false
		}
	}

	for _, r := range l.Rooms {
		x0, x1 := r.CenterX-r.W/2, r.CenterX+r.W/2
		z0, z1 := r.CenterZ-r.D/2, r.CenterZ+r.D/2
		for wx := x0; wx <= x1; wx++ {
			for wz := z0; wz <= z1; wz++ {
				carve(wx-minX, wz-minZ)
			}
		}
	}
	for _, c := range l.Corridors {
		a, b := l.Rooms[c.FromRoom], l.Rooms[c.ToRoom]
		half := dungeonCorridorW / 2
		// X-leg at a.CenterZ, then Z-leg at b.CenterX -- axis-aligned, matching cavesChunk's
		// own corridor-carving style.
		xLo, xHi := a.CenterX, b.CenterX
		if xLo > xHi {
			xLo, xHi = xHi, xLo
		}
		for wx := xLo; wx <= xHi; wx++ {
			for dz := -half; dz <= half; dz++ {
				carve(wx-minX, a.CenterZ+dz-minZ)
			}
		}
		zLo, zHi := a.CenterZ, b.CenterZ
		if zLo > zHi {
			zLo, zHi = zHi, zLo
		}
		for wz := zLo; wz <= zHi; wz++ {
			for dx := -half; dx <= half; dx++ {
				carve(b.CenterX+dx-minX, wz-minZ)
			}
		}
	}

	out := make([]WorldBlock, 0, width*depth*4)
	for lz := 0; lz < depth; lz++ {
		for lx := 0; lx < width; lx++ {
			wx := originX + minX + lx
			wz := originZ + minZ + lz
			for y := 0; y <= solidTop; y++ {
				if !solid[y][idx(lx, lz)] {
					continue
				}
				name := "minecraft:stone_bricks"
				if y == dungeonFloorY-1 && !solid[dungeonFloorY][idx(lx, lz)] {
					name = "minecraft:dirt" // real floor, same convention cavesChunk uses
				}
				out = append(out, WorldBlock{X: wx, Y: y, Z: wz, BlockName: name})
			}
		}
	}
	return out
}

package worldapi

import "sync"

// dungeon_instance.go -- DUNGEON_NORTHSTAR.md Milestone 1 ("instancing"), real correction and
// real, minimal v0 slice (2026-09-04).
//
// Real, decisive correction to this northstar's own Milestone 1 acceptance text: it was written
// assuming REDGARDEN's per-match architecture ("New `--server-bin` spawns via the matchmaker's
// existing fork/exec path... here's the port") -- but Milestone 4's own acceptance text names
// "Battlegrounds' existing combat HUD" as the real dungeon-render client, and Battlegrounds
// (apps2/battlegrounds_gui) talks to THIS repo's own apps2/server-go, a single, long-running,
// persistent-world UDP process (checked directly: apps2/server-go/main.go opens exactly one
// net.ListenUDP for the whole game, not a fresh process per match) -- not a REDGARDEN-style
// per-match spawned server. Wiring dungeon entry through REDGARDEN's matchmaker would mean the
// dungeon never actually reaches the client that's supposed to render it. The real, load-bearing
// mechanism this persistent-world server already has for "send a player somewhere new" is
// telecrystal travel (server/telecrystal + apps2/server-go's own PacketTelecrystalUse handler:
// validate -> IDUNA scene/pos update -> PacketSceneChange-shaped ack) -- real "instancing" here
// means allocating a fresh scene ID + seed and reusing that exact same travel mechanism, not
// forking a new OS process.
//
// DungeonInstanceRegistry is the real, minimal v0 that makes that possible: each Allocate call
// reserves the next scene ID in a dedicated dungeon range and eagerly generates that instance's
// full layout/blocks/spawns from Milestone 2/3's own already-real, already-tested generators
// (GenerateDungeonLayout, DungeonLayoutToBlocks), cached in memory so every subsequent
// GET /chunks?scene=<id> for that instance serves consistent geometry. Real, honest limitation,
// named not hidden: this is in-process memory only -- an apps2/server-go restart loses every
// live dungeon instance (a player mid-dungeon would need to re-enter, generating a fresh one).
// Real, honest, deliberately NOT done here: party-roster passthrough (Milestone 1's own other
// named gap) -- Allocate always creates a brand-new solo instance per call, there is no concept
// yet of multiple players sharing the same instance by request.
//
// Real, load-bearing wire-format constraint, checked directly before picking a scene ID range:
// PacketTelecrystalAck/PacketSceneChange (apps2/server-go/main.go) encode the scene ID as a
// single `uint8` byte (`ack[1] = uint8(crystal.TargetScene)`) -- any scene ID above 255 would
// silently truncate on the wire, corrupting the very travel ack this registry exists to drive.
// Real, checked-not-assumed existing scene usage stays entirely below 208 (meadow/hills/caves/
// swamp at 0-3, the 8 TRAPX city districts at 200-207), so dungeonInstanceStartSceneID..255
// (48 possible concurrent instances) is real, currently-free range. Once exhausted, Allocate
// returns ok=false rather than silently wrapping around and colliding with a still-live
// instance -- this registry has no instance-release/expiry mechanism yet (a player leaving a
// dungeon doesn't free its scene ID), so wrapping would eventually hand two different players
// the same scene ID for two different dungeons. A real, named v0 boundary: 48 concurrent
// dungeon instances server-wide, until release/expiry is built as real, separate follow-up.
const (
	dungeonInstanceStartSceneID = 208
	dungeonInstanceMaxSceneID   = 255
)

// dungeonInstance is one live, already-generated dungeon (real, computed once at Allocate time,
// not lazily -- a fresh instance's layout is small enough, per DungeonMinRooms/DungeonMaxRooms's
// own real bounds, 5-8 rooms, that eager generation is real, cheap, and avoids a second caching
// layer around lazy-on-first-chunk-request generation).
type dungeonInstance struct {
	layout DungeonLayout
	blocks []WorldBlock
}

// DungeonInstanceRegistry holds every live dungeon instance this apps2/server-go process has
// allocated. Safe for concurrent use -- apps2/server-go's own UDP read loop is single-threaded
// today (see gameWorld's own doc comment in apps2/server-go/main.go for the same real
// "single-threaded main loop, no mutex needed" reasoning), but worldapi's own HTTP /chunks
// endpoint (server/worldapi/worldapi.go) runs on Go's own net/http default concurrent handler
// goroutines -- a real, live concurrent-access path this registry needs to be correct under.
type DungeonInstanceRegistry struct {
	mu          sync.RWMutex
	instances   map[int]*dungeonInstance
	nextSceneID int
}

// NewDungeonInstanceRegistry creates a registry whose first Allocate call returns startSceneID.
// Real production callers should pass dungeonInstanceStartSceneID; a distinct start is exposed
// for tests so they don't collide with each other or with real scene IDs.
func NewDungeonInstanceRegistry(startSceneID int) *DungeonInstanceRegistry {
	return &DungeonInstanceRegistry{
		instances:   make(map[int]*dungeonInstance),
		nextSceneID: startSceneID,
	}
}

// Allocate generates a brand-new dungeon instance from seed and reserves the next scene ID for
// it, returning that scene ID. dungeonIndex is accepted (not yet used by this v0) so callers
// already have the right call shape for Milestone 3's own GenerateDungeonSpawns, which takes
// the same seed+sceneID+dungeonIndex triple -- wiring real mob spawns into a live instance is
// real, separate follow-up (this registry only carries block geometry today).
//
// ok is false once every scene ID in this registry's real, wire-format-bounded range is already
// in use (see the registry's own doc comment above for why this refuses rather than wraps
// around) -- callers must handle this as a real "no dungeon slots free right now" outcome, not
// assume Allocate always succeeds.
func (r *DungeonInstanceRegistry) Allocate(dungeonIndex int, seed int64) (sceneID int, ok bool) {
	r.mu.Lock()
	if r.nextSceneID > dungeonInstanceMaxSceneID {
		r.mu.Unlock()
		return 0, false
	}
	sceneID = r.nextSceneID
	r.nextSceneID++
	r.mu.Unlock()

	layout := GenerateDungeonLayout(seed)
	blocks := DungeonLayoutToBlocks(layout, 0, 0) // each instance owns its own scene ID, so
	// local layout space IS that scene's world space -- no offset needed, unlike the shared
	// TRAPX city districts' own chunk-grid origin math.

	r.mu.Lock()
	r.instances[sceneID] = &dungeonInstance{layout: layout, blocks: blocks}
	r.mu.Unlock()
	return sceneID, true
}

// Has reports whether sceneID is a live, allocated dungeon instance.
func (r *DungeonInstanceRegistry) Has(sceneID int) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.instances[sceneID]
	return ok
}

// BlocksForChunk returns just the blocks of a live instance falling inside the requested 16x16
// chunk (chunkX*16..chunkX*16+15, chunkZ*16..chunkZ*16+15, any Y) -- the real per-chunk slice
// GET /chunks?scene=N&cx=X&cz=Z needs, filtered from the instance's own full, already-generated
// block list. ok is false when sceneID names no live instance (the caller should fall back to
// ProceduralWorldStore's own default in that case, same as any other unrecognized scene ID).
func (r *DungeonInstanceRegistry) BlocksForChunk(sceneID, chunkX, chunkZ int) (blocks []WorldBlock, ok bool) {
	r.mu.RLock()
	inst, found := r.instances[sceneID]
	r.mu.RUnlock()
	if !found {
		return nil, false
	}
	x0, x1 := chunkX*16, chunkX*16+15
	z0, z1 := chunkZ*16, chunkZ*16+15
	out := make([]WorldBlock, 0)
	for _, b := range inst.blocks {
		if b.X >= x0 && b.X <= x1 && b.Z >= z0 && b.Z <= z1 {
			out = append(out, b)
		}
	}
	return out, true
}

// EntrySpawn returns the real spawn position for a live instance -- its entrance room's own
// center, at the real dungeon floor height (dungeonFloorY, the same constant
// DungeonLayoutToBlocks itself carves the walkable space starting from). ok is false when
// sceneID names no live instance.
func (r *DungeonInstanceRegistry) EntrySpawn(sceneID int) (x, y, z float64, ok bool) {
	r.mu.RLock()
	inst, found := r.instances[sceneID]
	r.mu.RUnlock()
	if !found {
		return 0, 0, 0, false
	}
	entrance := inst.layout.Rooms[inst.layout.EntranceIdx]
	return float64(entrance.CenterX), float64(dungeonFloorY), float64(entrance.CenterZ), true
}

# Smooth Voxel Terrain — Northstar

*Written 2026-08-02. Founder, after seeing Town's flat checkerboard ground and thinking ahead to
future zones: "ok the town we have is flat and thats cool but when we have other zones we are
going to want biome like stuff including different elevations is it possible to use the voxel
stuff in dragonfly but we show it as a smooth terrain?" → "yea use dragonfly but make it look
natural not like blocks interpolate or whatever" -- confirmed: keep Dragonfly's voxel chunks as
the real backend data, but render them as a smooth, interpolated heightfield in the client, not
Minecraft-style blocky cubes.*

*Extended 2026-08-03, prompted by the real telecrystal work (Dragon Gate -> Meadow): "ok so can
we warp from town to the zone backed by dragonfly?" -> "the northstar of that is we want it to
render the dragonfly biomes smooth with trees" -> "like a nice minecraft meadow biome but we
render it with our frontend." Two real additions to the original scope: §3.5 (trees -- found
while re-checking the backend that Meadow's own chunk generation already scatters real oak trees,
a reusable asset, not new backend work) and §3.6 (the Town <-> Dragonfly bridge question this
session's own telecrystal work surfaced: Town's Dragon Gate currently sends a character to
`apps2/mud`'s own "Meadow," which is real backend state but has NO relationship to
`server/worldapi`'s Dragonfly-voxel Meadow at all -- they're the same NAME on two backends that,
per `DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`, don't share state).*

---

## 1. What exists today (found while scoping this)

**Backend** (`server/worldapi`): a single endpoint, `GET /chunks?scene=N&cx=X&cz=Z` on port 7070
(`worldapi.New(worldapi.NewDragonflyChunkGenerator(worldapi.ProceduralWorldStore))`, wired in
`apps2/server-go/main.go`), returns a flat JSON array of `VoxelBlock{X,Y,Z uint8; BlockID uint16}`
for one 16x16 chunk. The actual block content comes from `scenes.go`'s `ProceduralWorldStore` --
a hardcoded per-`sceneID` switch, not real noise-based terrain and not persisted/player-editable:
every call regenerates blocks from scratch. Scene 1 ("Hills") already computes a height per (x,z)
column via a sine function before emitting solid blocks up to that height -- so a heightmap is
already conceptually present in the generation logic, it's just never exposed as one; the API
only ever hands back the fully-expanded block list. Scene 2 ("Caves") is genuinely 3D (a
`[y][x*16+z]bool` solid grid, not column-derived), so not every scene reduces to a heightmap --
see §4. There is no biome type/enum/struct anywhere in the Go backend; `sceneID` is informally
the closest thing to a biome selector today.

**Client** (`apps2/battlegrounds_gui/src/main.c`): every `Mesh` is position+normal vertices
(6 floats/vertex, `upload_mesh`/`draw_mesh`) through one shared shader that takes a single flat
`uColor` uniform per draw call -- there is no per-vertex color/UV attribute in the pipeline at
all today. `town_draw_ground` tiles a flat unit quad N times with `mat4_translate`/`mat4_scale`,
alternating a hardcoded grey/brown `uColor` per tile -- pure flat geometry, zero elevation
anywhere in the client (movement, camera focus, and click-to-move all assume y=0 unconditionally).

**SHANKPIT's own client** (`apps2/lobby/src/render_voxel.c`) already renders real `VoxelBlock`
data from this same `/chunks` endpoint, but as literal per-block textured cubes via legacy
`glBegin(GL_QUADS)` -- exactly the blocky look this doc is moving away from, not reusable as-is.

## 2. The mechanism

A heightfield mesh reuses the client's *existing* pos+normal vertex format unchanged -- elevation
is just a per-vertex Y instead of a constant, and normals come from the height gradient (finite
differences between neighboring samples) instead of a hardcoded +Y. No shader rewrite needed for
elevation itself. What's actually new:

1. **Backend**: a heightmap-shaped view of column-reducible scenes (expose the height-per-column
   `ProceduralWorldStore` scenes already compute internally, rather than making the client
   reconstruct height by scanning a raw block array it has to fetch anyway).
2. **Client**: a new mesh-generation function that samples that heightmap on a grid, applies
   smoothing between neighboring samples (not stair-stepped per-block steps), and builds
   triangles + gradient-derived normals -- structurally the same shape as `town_draw_ground`'s
   own tile loop, just per-vertex-height instead of flat.
3. **Movement/camera/click-to-move** need to start reading terrain height instead of assuming 0 --
   real, necessary follow-on work, not attempted in the terrain renderer itself.

```
Today:   VoxelBlock[] (full 3D block list) -> SHANKPIT's blocky per-cube renderer
New:     height-per-column (+ biome id)     -> smooth heightfield mesh, pos+normal, existing shader
```

## 3. Architecture

### 3.1 Backend: expose height, not just blocks

Add a heightmap-shaped response for scenes whose generation is already column-derived (Hills,
Meadow, Swampville, the flat 200-series set-pieces) -- either a new endpoint
(`GET /heightmap?scene=N&cx=X&cz=Z`, returning `{height []uint8, biome []uint8}` per column) or an
additive field on the existing `/chunks` response. A genuinely 3D scene (Caves) has no single
height-per-column and is explicitly out of scope for this doc -- see §4.

### 3.2 Client: heightfield mesh generator

New function, same shape as `town_draw_ground`: for a chunk-sized grid, sample the heightmap at
each (x,z), interpolate between samples (bilinear or cosine easing, not nearest-neighbor) so
slopes read as continuous rather than stepped, compute each vertex's normal from the local height
gradient, and emit triangles through the existing `upload_mesh`/`draw_mesh` path -- no new GL
state, no shader changes, same VAO/VBO layout every other mesh in this client already uses.

### 3.3 Biome coloring (separable from elevation)

Cheapest real option without touching the shared shader: flat color per mesh-chunk keyed off the
dominant biome in that chunk, same "one `uColor` per draw call" convention Town's ground already
uses -- visible banding at biome boundaries, but zero pipeline risk. True smooth color blending
across biome edges needs a new per-vertex-color attribute and a shader variant that interpolates
it -- a real, separable second piece of work, not required for "smooth terrain" itself (elevation
is the part that reads as "blocky" today; flat-shaded color regions do not).

### 3.4 Movement, camera, click-to-move

Every place `main.c` currently assumes ground is y=0 (avatar rendering, `screen_to_ground`'s ray
cast, camera focus height, WASD movement) needs to instead sample the local heightfield. Real,
necessary work once a zone actually has elevation -- not attempted for Town (which stays
intentionally flat) and not attempted in the terrain-rendering milestones below.

### 3.5 Trees (2026-08-03, founder: "we want it to render the dragonfly biomes smooth with trees"
-- "like a nice minecraft meadow biome but we render it with our frontend")

Real, grounded finding: trees are NOT new backend work. `server/worldapi/scenes.go`'s own
`meadowChunk` already scatters real oak trees (`meadowTrees`'s deterministic per-chunk hash +
`worldTree`'s trunk/canopy block placement, `minecraft:oak_log`/`minecraft:oak_leaves`) as part of
Meadow's normal chunk generation -- confirmed live, 2026-08-03: a real `/chunks?scene=0` call
returned 1308 blocks including 8 real tree-log blocks. Swampville has its own dead-mangrove
variant (`swampTree`, `jungle_log`/`jungle_leaves`). This is real, existing, already-seeded
content -- the gap is purely on the render side.

Trees are NOT part of the heightfield mesh (§3.2) -- they're discrete props, identified in the
`/chunks` block list by `BlockName` (any `*_log`/`*_leaves` match), positioned at their own real
world (x,y,z). For visual consistency with the rest of this client's art style (every existing
model -- heroes, worm mobs, Town's buildings -- is simple stacked-primitive geometry through the
same `upload_mesh`/`draw_mesh` pipeline, not sprites), a tree should be a small procedural mesh
(cylinder-ish trunk + a faceted/rounded canopy blob) built the same way, not a billboard --
billboards were the right call for DUNGEON_NORTHSTAR's mobs specifically because those are
*moving creatures* needing cheap animation frames; a tree is static world dressing, closer in kind
to a building than a monster.

### 3.6 The Town <-> Dragonfly bridge (open question, not resolved here)

Founder: "ok so can we warp from town to the zone backed by dragonfly?" Real gap found while
scoping this: Town's own Dragon Gate (this session's telecrystal work) currently sends a character
to `apps2/mud`'s own "Meadow" -- real backend state, but a TEXT-ONLY MUD zone with zero
relationship to `server/worldapi`'s Dragonfly-voxel "Meadow." They share a name and a scene-ID
convention (0) but are served by two completely separate backends
(`docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`'s own finding: apps2/mud and apps2/server-go "don't
share state at all"). A real bridge needs, at minimum: (a) this terrain-rendering work landing in
`apps2/battlegrounds_gui` at all (§3.1-3.4, all still NOT STARTED), and (b) a real answer to
whether a Town character's own session can move into `apps2/server-go`'s own UDP protocol
world -- a genuine protocol switch, not just a position update -- or whether the two backends need
real unification first (the two-backends audit's own closing recommendation: "port apps2/mud's
RPG logic to run inside apps2/server-go's loop"). Not designed here; flagged honestly as the real
open question this doc's own title question raises.

### 3.7 Explicitly out of scope, named rather than silently ignored

**World sculpting** (founder: "on some level we have the ability to sculpt the world not clear
how that will work") -- real aspiration, zero design here. `ProceduralWorldStore` is a fully
deterministic, regenerate-every-call stub (§1) -- no persistence layer exists for it to write
player edits into. A real sculpting feature needs real block storage first (a genuine, separate,
large piece of infrastructure), not attempted or estimated in this doc.

**Real Minecraft Bedrock client connectivity** (founder: "can we... connect from my phones
minecraft, to debug") -- checked directly, 2026-08-03: this repo does not vendor the real
`df-mc/dragonfly` Bedrock server library (`go.mod` has no such dependency, confirmed via search).
"Dragonfly" here is this project's own codename; `apps2/server-go`'s real UDP game protocol (now
running on :6970, see Milestone 0.5) is a from-scratch custom protocol (its own packet format,
IDUNA-JWT auth) -- NOT real Bedrock/RakNet networking. A real phone Minecraft client cannot talk
to it regardless of port. Achieving that would mean integrating real Bedrock-protocol
compatibility (what libraries like the actual df-mc/dragonfly exist to solve) -- a separate, much
larger project, not a config change, and not attempted here.

**Update, same day:** founder forked the real `github.com/df-mc/dragonfly` to
`emilyspringerton/dragonfly`. Confirmed genuine -- `go.mod` module path is
`github.com/df-mc/dragonfly`, zero commits ahead of `df-mc/dragonfly`'s own `master` (a clean
fork, no local changes yet). Built and ran it vanilla (`go run .` equivalent, Go 1.26 toolchain,
real `sandertv/gophertunnel` + `sandertv/go-raknet` deps): it stands up a real RakNet/Bedrock
listener on UDP `:19132`, logs `mc-version=1.26.30`, and would accept a real phone Minecraft
client today, unmodified, on the same LAN or with the port forwarded -- this closes the "connect
from my phone's Minecraft, to debug" ask, using vanilla upstream content (its own default
superflat/whatever world), not GoblinFoxDragon's own Meadow biome.

Getting the *real* Meadow content (the trees, the biomes this doc is actually about) reachable
from a real Bedrock client is a separate, much larger integration than running the vanilla
binary: it means writing a `df-mc/dragonfly` world/chunk provider that sources blocks from
`server/worldapi`'s `ProceduralWorldStore` instead of dragonfly's own leveldb-backed world, plus
wiring player movement/interaction through dragonfly's own entity/handler model. Not designed or
estimated here -- flagged as the real next open question if Bedrock-client debug access to actual
GoblinFoxDragon content (not vanilla dragonfly content) becomes a priority.

## 4. What this is not

Not a general voxel/marching-cubes renderer -- true overhangs, caves, and arbitrary 3D voxel
shapes (Scene 2's own genuinely-3D caves) don't reduce to a single height-per-column and are
explicitly out of scope; caves keep whatever rendering approach they already have (or get one
separately, later). Not a replacement for `/chunks` or SHANKPIT's own blocky renderer -- SHANKPIT
is a different client with its own art direction; this doc only changes how
`apps2/battlegrounds_gui` renders terrain. Not real persisted/player-editable terrain --
`ProceduralWorldStore` stays exactly the deterministic-per-call stub it is today; smoothing how
it's *rendered* doesn't change how it's *generated*. Not attempted for Town, which stays flat by
design (§3.4).

## 5. Milestones

| # | Milestone | Acceptance | Status |
|---|---|---|---|
| 0 | This northstar | Written, registered in golden-docs-index | DONE |
| 0.5 | apps2/server-go running + seeded | Supervised (systemd), non-conflicting port, `/chunks` serves real block+tree data | DONE (2026-08-03) |
| 1 | Backend heightmap exposure | New endpoint or field returns per-column height (+ biome id) for column-derived scenes (Hills first) | NOT STARTED |
| 2 | Client heightfield mesh | New mesh-gen function renders a smooth, interpolated (not stair-stepped) terrain surface for one test scene, reusing the existing pos+normal pipeline unchanged | NOT STARTED |
| 3 | Biome flat-coloring | Terrain chunks colored by dominant biome, same flat-per-draw-call convention as Town's ground | NOT STARTED |
| 4 | Movement/camera elevation awareness | WASD movement, click-to-move, and camera focus read real terrain height instead of assuming 0, for the same test scene | NOT STARTED |
| 5 | Smooth per-vertex biome blending (stretch) | New vertex-color attribute + shader variant blends biome color continuously across chunk/biome boundaries | NOT STARTED |

## 6. Related docs

- `docs2/MMO_NORTHSTAR.md` -- zone/scene model this design layers terrain onto
- `docs2/HEADLESS_SESSION_NORTHSTAR.md` §3.4 -- the "second scene" (non-Town, walkable world)
  concept this terrain work is meant to eventually serve
- `docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md` -- apps2/mud vs apps2/server-go split;
  `server/worldapi` lives on the apps2/server-go side

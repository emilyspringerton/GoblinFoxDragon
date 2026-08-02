# Smooth Voxel Terrain — Northstar

*Written 2026-08-02. Founder, after seeing Town's flat checkerboard ground and thinking ahead to
future zones: "ok the town we have is flat and thats cool but when we have other zones we are
going to want biome like stuff including different elevations is it possible to use the voxel
stuff in dragonfly but we show it as a smooth terrain?" → "yea use dragonfly but make it look
natural not like blocks interpolate or whatever" -- confirmed: keep Dragonfly's voxel chunks as
the real backend data, but render them as a smooth, interpolated heightfield in the client, not
Minecraft-style blocky cubes.*

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

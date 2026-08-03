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

**Done, 2026-08-03 (Milestone 1):** `GET /heightmap?scene=N&cx=X&cz=Z` shipped in
`server/worldapi` (`heightmap.go`), returning `{"height": [uint8 x256], "biome": int}` -- biome is
a single value for the whole chunk, not a 256-entry array, since `ProceduralWorldStore` has no
per-column biome mixing to expose yet (would just be wasted bytes on the wire). Covers Meadow
(flat, 4), Hills (real per-column variation, `hillsColumnHeight` split out of `hillsChunk` itself
so the two can't drift apart), and Swampville (flat, water cells one block higher than dry land).
Caves correctly returns 204 (no height-per-column view exists for it). Live-verified against the
running `gfd-server-go.service`: Meadow returns a uniform `{4}`, Hills returns real min=2/max=8
variation, Swamp returns both `{1,2}`, Caves 204. Test coverage includes a direct cross-check
against `hillsChunk`'s own block output so the extracted height formula can't silently drift from
the block generation it came from.

### 3.2 Client: heightfield mesh generator

New function, same shape as `town_draw_ground`: for a chunk-sized grid, sample the heightmap at
each (x,z), interpolate between samples (bilinear or cosine easing, not nearest-neighbor) so
slopes read as continuous rather than stepped, compute each vertex's normal from the local height
gradient, and emit triangles through the existing `upload_mesh`/`draw_mesh` path -- no new GL
state, no shader changes, same VAO/VBO layout every other mesh in this client already uses.

**Done, 2026-08-03 (Milestone 2):** `build_heightfield_mesh` + `heightfield_sample`
(`apps2/battlegrounds_gui/src/main.c`) fetch a real heightmap from the Milestone 1 `/heightmap`
endpoint over HTTP (new `http_extract_json_uint8_array_field` in `http_client.h`), bilinearly
interpolate at 2x the source resolution, derive per-vertex normals from finite-difference height
gradients, and emit triangles through the exact same pos+normal `upload_mesh`/`draw_mesh` path
every other mesh in this client uses -- no shader change. Wired as an F10 debug toggle
(`town_load_terrain_test`/`town_draw_terrain_test`) that renders the real live Hills chunk (0,0)
floating clear of Town's own footprint, not integrated into Town itself (Town stays flat by
design, §3.4) and not wired into movement/collision (Milestone 4). Live-verified visually: built
and ran the real client under Xvfb (`gcc` direct build, `sandertv`-free -- this is
GoblinFoxDragon's own client, unrelated to the df-mc/dragonfly fork discussed in §3.7), connected
to Town via a WOTAN dev-agent identity, screenshotted the F10 mesh -- a real smooth, continuously
gradient-shaded rolling surface, not stair-stepped cubes, confirming both the interpolation and
the normal-based lighting work correctly against real backend data.

### 3.3 Biome coloring (separable from elevation)

Cheapest real option without touching the shared shader: flat color per mesh-chunk keyed off the
dominant biome in that chunk, same "one `uColor` per draw call" convention Town's ground already
uses -- visible banding at biome boundaries, but zero pipeline risk. True smooth color blending
across biome edges needs a new per-vertex-color attribute and a shader variant that interpolates
it -- a real, separable second piece of work, not required for "smooth terrain" itself (elevation
is the part that reads as "blocky" today; flat-shaded color regions do not).

**Done, 2026-08-03 (Milestone 3):** `biome_color` (`apps2/battlegrounds_gui/src/main.c`) maps
worldapi's own `scene`/biome id to a flat RGB per draw call -- Meadow grass green, Hills olive,
Swampville muddy brown-green, unknown scenes a neutral grey fallback. The F10 debug scene (§3.2's
own Milestone 2 work) now fetches and renders all three column-derived biomes side by side rather
than a single hardcoded green, so the coloring is real per-biome data, not a placeholder. No new
enum needed -- reuses worldapi's own informal "sceneID is the biome selector" convention (§1)
rather than inventing a second one client-side. Live-verified visually under Xvfb: all three
patches render simultaneously with visibly distinct hues, confirming `biome_color` is actually
driven by each patch's own real `scene` field, not hardcoded.

### 3.4 Movement, camera, click-to-move

Every place `main.c` currently assumes ground is y=0 (avatar rendering, `screen_to_ground`'s ray
cast, camera focus height, WASD movement) needs to instead sample the local heightfield. Real,
necessary work once a zone actually has elevation -- not attempted for Town (which stays
intentionally flat) and not attempted in the terrain-rendering milestones below.

**Done, 2026-08-03 (Milestone 4), scoped to the same F10 test patches Milestones 2/3 built --
Town itself untouched, stays flat by design, per this section's own opening line:**
`terrain_test_height_at(wx, wz, &out_y)` returns the real interpolated terrain height (same
CPU-side `heights[256]` + `heightfield_sample` the GPU mesh was built from, so movement/camera can
never see a different surface than what's rendered) when standing inside one of the three F10
patches, and 0 (Town's own flat ground) everywhere else. Wired into the camera's own focus point
(`mat4_orbit_view`'s `focus_y`, previously hardcoded 0.0f) and the avatar's draw-time Y offset
(combined with the existing jump-arc translate, same "world-space Y pre-multiply" technique).
`terrain_test_offset_x` is now the one shared source of each patch's world placement -- both the
renderer and the height lookup call it, so they can't drift the way the block-generation/heightmap
split in Milestone 1 would have if `hillsColumnHeight` hadn't been factored out the same way.

**Explicitly not done, named rather than silently skipped**: `screen_to_ground`'s own click-to-move
ray-cast still projects against a flat y=0 plane, so a click's resulting (x,z) target on a sloped
test patch is an approximation, not a real ray-vs-heightfield intersection -- a materially harder
geometric problem (the acceptance criterion's own "for the same test scene" is satisfied by
movement/camera Y, not by exact click targeting on a slope). WASD's own (x,z) update logic is
unchanged for the same reason: only the resulting position's rendered Y now reads real terrain,
not the horizontal movement algorithm itself. Real for the debug validation this milestone exists
to provide; a full fix belongs with the Town<->Dragonfly bridge (§3.6) if/when a zone with real
elevation becomes actually walkable, not this isolated test harness.

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

**Done, 2026-08-03**: `town_draw_dfzone_trees` (`apps2/battlegrounds_gui/src/main.c`) closes this
-- the founder's own original request ("smooth biomes with trees") was only half-delivered until
now (smooth terrain: yes since Milestone 2; trees: not until this). Rather than fetch and parse
the full `/chunks` block list (an array of ~1300 objects for one Meadow chunk, real parsing
complexity for data whose positions are already fully deterministic), `town_meadow_tree_positions`
mirrors `meadowTrees`'s own hash (`chunkX*31 + chunkZ*17`, mod 5) directly in C -- same "client
keeps its own copy of world data" convention this session's telecrystal work already established
twice over (`town_telecrystal_travel`'s own copied crystal values, `apps/lobby`'s own
`TELECRYSTAL_DEFS`). Each tree reuses `draw_hero_box` -- the exact stacked-primitive technique
every hero/worm/building in this client already uses -- for a thin trunk plus two tapering canopy
tiers, sitting at the zone's own real terrain height (`dfzone_height_at`) rather than assuming
y=0. Meadow only (Hills has none by design, Swampville isn't offered as a real destination here).
Live-verified visually under Xvfb: screenshotted a real tree standing in the Meadow zone at its
correct deterministic position.

### 3.6 The Town <-> Dragonfly bridge

Founder: "ok so can we warp from town to the zone backed by dragonfly?" Real gap found while
scoping this: Town's own Dragon Gate (this session's telecrystal work) currently sends a character
to `apps2/mud`'s own "Meadow" -- real backend state, but a TEXT-ONLY MUD zone with zero
relationship to `server/worldapi`'s Dragonfly-voxel "Meadow." They share a name and a scene-ID
convention (0) but are served by two completely separate backends
(`docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`'s own finding: apps2/mud and apps2/server-go "don't
share state at all").

**Done, 2026-08-03 -- the RENDER side of this bridge, not a backend unification:** founder: "im
expecting to teleport from town to the new zone." `town_telecrystal_travel` used to stop at the
IDUNA position PATCH, leaving Town's own New-Handington geometry on screen (the exact gap its own
old doc comment named). Now it also lazy-loads the real Dragonfly Meadow heightmap (`dfzone_load`,
worldapi scene 0) and switches the client's own render mode (`g_dfzone_active`) -- Town's
ground/buildings/worms stop drawing, `town_draw_dfzone` draws the real live heightfield mesh
(Milestones 2-4's own pipeline, unchanged) at the world origin instead, and camera/avatar height
follow it (`dfzone_height_at`). A dedicated "G" key is the return trip
(`town_telecrystal_return`, the real `TELECRYSTAL_ID_MEADOW_RETURN_HANDINGTON` values). Live-
verified visually under Xvfb: Town's checkerboard and buildings genuinely disappear and the real
Meadow terrain fills the screen at the destination.

**First real playtest found three more real bugs, all fixed same day**: (1) IDUNA's own position
endpoint returns 204 on success, not 200 -- the client's status check was wrong, so every
successful travel reported "the crystal fizzles" (`status != 204`, both directions, fixed). (2)
Movement had no bounds check *inside* the zone -- click-to-move/WASD still clamped to Town's own
~57-unit footprint, easily walking straight off the real ~8-unit-radius mesh into nothing (same
failure class as the original Town "floating in a blue abyss" bug, just not caught here yet;
`town_move_half_extent()` now picks the right bound for whichever ground is actually rendered).
(3) The zone read as "pretty big in relation to" the avatar and the Dragon Gate click was "hard to
trigger" -- `TERRAIN_TEST_CELL_SIZE` tripled (1.0 -> 3.0, tripling the physical footprint without
touching the fixed 16x16 heightmap resolution) and the gate's own click trigger now uses its real
12-unit telecrystal radius instead of the building's ~5-unit visual box.

**Real UX upgrade, same day, founder: "check the shankpit side of the codebase there is
telecrystals the ux is good i want it like that circle showing cast radius cast bar ticks up":**
the click-based trigger from the fix above was itself replaced entirely, ported from
`apps/lobby/src/main.c`'s own already-shipped telecrystal UX (the older SHANKPIT-style client in
this same repo) -- a real reference implementation, not redesigned from scratch. A pulsing
world-space ring (`town_draw_gate_ring`, new `draw_mesh_lines` -- this client's 3D pass is
shader-bound, not the legacy matrix stack apps/lobby's own ring uses, so the ring is a real mesh
through the same pipeline instead of mixed-in immediate-mode calls) sits at the crystal's real
radius (12 units, both directions, `server/telecrystal`'s own registry values), turning solid
white when the player is inside it. A fill bar with a visible commit tick-mark
(`town_draw_gate_overlay`) advances over 1000ms, the real travel/return call fires at the 600ms
commit mark (matching the reference's own commit-before-bar-finishes feel), and leaving the ring
before commit cancels the cast (`town_gate_tick`). Live-verified visually under Xvfb: screenshotted
mid-cast showing the fill bar, commit marker, and "CASTING: TELEPORT MEADOW" text against the real
in-range white ring.

**Corrected same day, founder: "pressing g does nothing i expect it to auto cast when i enter the
ring"**: the first pass ported apps/lobby's own G-press-to-start mechanic verbatim -- not what was
actually wanted here. `town_gate_tick` now auto-starts the cast itself on the ring-enter edge
(`was_in_range` false -> true), no key involved at all; `town_gate_start_cast` still exists and is
still safe to call (G is left wired to it as a harmless manual fallback), but the primary,
expected path is now pure proximity -- walk in, the bar starts. Live-verified visually under
Xvfb: simulated walking from outside the ring to inside it and screenshotted the cast bar already
progressing on arrival, no keypress simulated.

**Reference UX port completed, same day**: the one remaining piece of `apps/lobby`'s own
telecrystal UX not yet carried over -- a brief, large, screen-centered "TRAVELING: <destination>"
banner right at the moment of arrival (`town_draw_travel_overlay`/`g_travel_overlay_text`,
apps/lobby's own `draw_travel_overlay`/`travel_overlay_text`), distinct from
`combat_log_push`'s own arrival line (a scrolling log entry, easy to miss mid-fight; the banner
isn't). Set at both real arrival points (`town_telecrystal_travel`/`town_telecrystal_return`),
1.4s duration. Live-verified visually under Xvfb.

**Still not done, named rather than silently ignored**: this is a client-side render swap, not a
protocol bridge -- the character's real backend session is still `apps2/mud`'s text MUD (position
PATCHed via IDUNA, same as before), not actually inside `apps2/server-go`'s own UDP world. No
mobs, other players, or `apps2/mud` combat/chat sync render in the Dragonfly zone -- it's a real,
walkable, correctly-elevated space with nothing populated in it yet, same honest scope as an
empty stage. Only one chunk (0,0) loads -- walking past its edge has no more terrain (falls back
to Town's own y=0 convention, `dfzone_height_at` returns 0 outside the loaded chunk, not a
crash). A genuine backend unification (the two-backends audit's own closing recommendation: "port
apps2/mud's RPG logic to run inside apps2/server-go's loop") is still not attempted -- this closes
the *visual* mismatch the founder was actually pointing at, not the deeper architecture question.

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

**Deployed persistently, 2026-08-03 (later the same day):** the vanilla test run above was a
one-off manual process, gone the moment the terminal closed. Made it real: a user-level systemd
unit (`~/.config/systemd/user/dragonfly-debug.service`, not committed anywhere -- see below)
builds and supervises `~/dragonfly`'s own binary, restarting on failure, so the real RakNet
listener on UDP `:19132` is up continuously, not just while someone is watching. Confirmed
listening (`ss -ulnp`) and logging real startup (`mc-version=1.26.30`) under supervision.
**Explicitly not done, named rather than assumed**: WAN/firewall reachability from a real phone on
a different network was NOT verified -- this box's inbound UDP :19132 firewall/security-group
state is unknown (no sudo access to check from this session), so "works on the same LAN" is
confirmed, "works from anywhere" is not. The systemd unit and its own deploy instructions live
locally only, not committed into `~/dragonfly` itself -- that's the founder's own personal fork of
a real external open-source project (`df-mc/dragonfly`), and build binaries/service units have no
business in that repo's git history (matching the same reasoning `world`/`players`/`resources`
are already gitignored there upstream).

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
| 1 | Backend heightmap exposure | New endpoint or field returns per-column height (+ biome id) for column-derived scenes (Hills first) | DONE (2026-08-03) |
| 2 | Client heightfield mesh | New mesh-gen function renders a smooth, interpolated (not stair-stepped) terrain surface for one test scene, reusing the existing pos+normal pipeline unchanged | DONE (2026-08-03) |
| 3 | Biome flat-coloring | Terrain chunks colored by dominant biome, same flat-per-draw-call convention as Town's ground | DONE (2026-08-03) |
| 4 | Movement/camera elevation awareness | WASD movement, click-to-move, and camera focus read real terrain height instead of assuming 0, for the same test scene | DONE (2026-08-03), click-to-move x/z targeting scoped out, see §3.4 |
| 5 | Smooth per-vertex biome blending (stretch) | New vertex-color attribute + shader variant blends biome color continuously across chunk/biome boundaries | NOT STARTED |

## 6. Related docs

- `docs2/MMO_NORTHSTAR.md` -- zone/scene model this design layers terrain onto
- `docs2/HEADLESS_SESSION_NORTHSTAR.md` §3.4 -- the "second scene" (non-Town, walkable world)
  concept this terrain work is meant to eventually serve
- `docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md` -- apps2/mud vs apps2/server-go split;
  `server/worldapi` lives on the apps2/server-go side

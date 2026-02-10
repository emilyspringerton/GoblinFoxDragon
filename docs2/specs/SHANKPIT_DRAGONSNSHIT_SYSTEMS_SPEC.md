# SHANKPIT × DRAGONSNSHIT — Systems Spec (v0 → Persistent World Bridge)

## 0) North Star

We’re building one stack that can ship fast arcade multiplayer (Shankpit) while steadily upgrading into a persistent world backend (DragonsNShit). The key trick is: don’t rewrite the client—swap the authority + persistence behind it.

## 1) System Map

### 1.1 Client (Shankpit runtime)

SDL/OpenGL immediate-mode client that:

- sends UDP user commands (UserCmds) and receives snapshots
- already has an explicit “use” intent via `BTN_USE` in UserCmd creation
- `SHANKPIT_CONSTRUCT`

Needs to become scene-aware (portals / streaming / persistence) without changing feel.

### 1.2 Server (Shankpit UDP authority)

UDP server accepts `PACKET_CONNECT`, assigns a slot, respawns a player, then returns `PACKET_WELCOME` and includes `scene_id` in the header.

`SHANKPIT_CONSTRUCT`

Broadcasts snapshots containing `NetPlayer.scene_id` per entity.

`SHANKPIT_CONSTRUCT`

Already has a `ServerState` concept of scene transitions (`scene_id`, `pending_scene`, `transition_timer`) but it’s not yet turning into real player-level world travel.

`SHANKPIT_CONSTRUCT`

### 1.3 Simulation + Physics

There is an explicit “scene” concept and portal geometry:

- Portal location/radius constants exist.
- `scene_portal_triggered()` exists and checks radius.
- `scene_spawn_point()` / `scene_force_spawn()` exist for respawn per scene.

`SHANKPIT_CONSTRUCT`

Critical current constraint: physics scene selection is implemented as a global `phys_set_scene(scene_id)` that swaps active map pointers.

`SHANKPIT_CONSTRUCT`

That’s fine for “one scene at a time,” but it blocks real multi-scene coexistence.

### 1.4 DragonsNShit backend (persistent world target)

The construct already declares the plan: run DragonsNShit as a fork of Dragonfly and wire via Go modules.

`SHANKPIT_CONSTRUCT`

This backend should eventually become:

- authoritative persistence (inventory/skills/cities/economy)
- world/chunk streaming and entity authority
- “bridge protocol” from Shankpit client into persistent simulation (incremental adoption)

### 1.5 Build / CI / “Construct”

Current Construct generation is hardcoded to a tiny curated list (only a few shankpit files).

`SHANKPIT_CONSTRUCT`

This is a major reason DragonsNShit feels “second class”: the repo may contain it, but the primary artifact doesn’t.

## 2) Guiding Principles

- Authority first: server decides reality; client is “a good liar” for smoothness.
- Add persistence behind a stable wire contract: avoid forcing client rewrites.
- Keep UDP until value demands TCP: only switch when persistence consistency actually needs it.
- Visibility = velocity: if it’s not in the Construct + CI artifacts, it “doesn’t exist.”

## 3) The Big Blind Spots (things that will bite us)

### 3.1 Scene correctness is not real yet

We have scene ids on the wire and on players, but physics uses a global active scene (`phys_set_scene`).

`SHANKPIT_CONSTRUCT`

This creates “it mostly works in single-player / single-scene” illusions and breaks the moment you want:

- two players in different scenes simultaneously
- per-scene collision, weapons, projectiles, vehicles, AI

### 3.2 “Portal system exists” but isn’t capturing value

Portal data + trigger function exist, and the client has a “use” button pathway (`BTN_USE`), but the end-to-end loop is missing:

- server-authoritative travel action
- per-player scene isolation (physics + interactions)
- client transition rules + filtering by scene

This is classic 80% done, 0% value.

### 3.3 Construct/CI is hiding the merged reality

Construct is currently the canonical “what is the system?” artifact, but it only cats a small list of files.

`SHANKPIT_CONSTRUCT`

So any DragonsNShit bridge work can be real and still feel invisible because:

- not included in Construct
- not promoted in build artifacts
- not reviewable by agents/automation that rely on the Construct

### 3.4 Persistence boundary is undefined

We have a plan to integrate a Dragonfly-based backend, but the contract boundary isn’t specified yet:

- What does Shankpit own vs Dragon server own?
- What can remain UDP “eventually consistent” vs what needs strict ordering?
- How does identity persist (account/character) vs ephemeral client slots?

### 3.5 Security & spoofing risks (will matter sooner than you think)

Even pre-persistence, portals + scene travel become a cheat surface unless the server validates:

- travel only when truly inside portal radius (server-side)
- cooldown/debounce to stop oscillation abuse
- per-scene isolation so cross-scene attacks don’t exist

## 4) “80% Done but Value Not Captured Yet” List (mostly)

### A) Scenes exist… but as metadata

`NetPlayer.scene_id` is transmitted in snapshots.

`SHANKPIT_CONSTRUCT`

Welcome includes `scene_id`.

`SHANKPIT_CONSTRUCT`

But the runtime behavior hasn’t cashed that check yet (render/filter/interact per scene).

### B) Portal geometry + trigger exist… but travel doesn’t

Constants + trigger logic are already there.

`SHANKPIT_CONSTRUCT`

But no authoritative travel pipeline.

### C) Player input has “USE”… but it’s not a first-class gameplay verb

`BTN_USE` exists and is wired into user command creation.

`SHANKPIT_CONSTRUCT`

Value arrives when `USE` controls portals/vehicles/world interactions consistently.

### D) Transition fields exist in server state… but no state machine is harvesting them

`scene_id`, `pending_scene`, `transition_timer` are sitting there.

`SHANKPIT_CONSTRUCT`

Value arrives when they become a deliberate travel/transition state machine (even if still UDP).

### E) DragonsNShit plan exists… but no “bridge seam” is enforced

Integration plan is stated.

`SHANKPIT_CONSTRUCT`

Value arrives when there’s a minimal interface that Shankpit calls (even if it’s a stub), so the direction is inevitable.

### F) The merged world exists… but the Construct denies it

Curated file list is the bottleneck.

`SHANKPIT_CONSTRUCT`

Value arrives when Construct includes the dragon bridge surfaces + any world/persistence code touched.

## 5) v0→v1 Milestones (what “done” looks like)

### Milestone 1: Portal travel works under UDP (authoritative)

- Server validates portal radius (`scene_portal_triggered`) and a `USE` action.
- Server flips player scene + spawns safely (using scene spawn helpers).
- Client switches scene based on server data and filters other players by scene.

### Milestone 2: Per-player scene isolation (physics + interactions)

- Replace “global active map” behavior with “scene-scoped collision queries” (or a minimal per-entity set/reset wrapper).
- Ensure no cross-scene damage/vehicle/projectile interaction.

### Milestone 3: Dragon backend seam

- Add a resolver like `portal_resolve_destination()` / `world_backend_*()` that the Shankpit server calls.
- Initially stubbed; later routes to Dragonfly fork.

### Milestone 4: Construct becomes the truth

Construct generation expanded from the hardcoded list to include:

- all scene/portal code
- all bridge/backend seam code
- dragonsnshit server integration surfaces

(so “second class” disappears structurally)

## 6) Operational Notes

- Stay UDP for now. Use server authority + snapshots as the stabilizer.
- The moment persistence starts to matter (inventory economy, durable actions), we’ll decide:
  - keep UDP + reliability layer for specific messages, or
  - migrate “durable actions” to TCP while leaving movement on UDP.

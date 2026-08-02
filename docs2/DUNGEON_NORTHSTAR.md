# Instanced Dungeons — Northstar

*Written 2026-08-02. Founder: "ok can you make a dungeon like diablo 2 dungeons?" -> scoped:
instanced (spawned per-group like a Battlegrounds match, ends when cleared/left, not a persistent
walkable zone) and procedural (rooms/corridors reassembled from a tile set, different layout each
visit, not a fixed hand-built map like Town) -> "use some of the arena heroes as dungeon mobs and
bosses" -> "also the different minions whatever im uploading some art in a second." Art direction
pending -- this doc covers the architecture; art/enemy-roster specifics land as a follow-up once
shared, named as an open gap below rather than guessed at.*

---

## 1. What exists today (found while scoping this)

**Instancing**: REDGARDEN's `apps/matchmaker/src/main.c` already spawns a fresh server process
per match via real `fork()` + `execl(server_bin, ..., "--port", port_str[, "--lobby-size", n])` --
and critically, `server_bin` is already a runtime flag, not hardcoded to arena specifically: the
same matchmaker binary already spawns two different server binaries today (`red_garden_server`
for the card-RTS, `red_garden_arena_server` for team arena), selected purely by which `--server-bin`
+ `--lobby-size` the matchmaker itself was launched with. The spawn mechanism (fork/exec, port
assignment, `PACKET_MATCH_FOUND` telling the client which port) is generic and reusable as-is.
What's NOT generic yet: argument passing is hardcoded to exactly `--port`/`--lobby-size` -- no
seed, party roster, or instance-config passthrough exists, and `PACKET_MATCH_FOUND` carries only a
port number, nothing telling the client which *mode* it's joining.

**Procedural generation**: `server/worldapi/scenes.go`'s "Caves" scene (sceneID 2) carves voxel
corridors into a solid grid, but it is fully deterministic (no `rand`/seed anywhere) and always
emits the exact same fixed cross-shaped two-corridor intersection -- not a room/corridor topology,
no randomization, no door/connectivity logic. Useful as a reference for *how to represent carved
voxel geometry as block lists* (solid-grid-then-carve, dirt-floor-on-boundary), but too simple in
kind to extend into multi-room dungeon layout -- a real generator gets built new on top of that
representation pattern, not grown out of Caves itself.

**Mobs and zones**: `server/mob`'s `Registry.Spawn(m Mob)` is already generic -- takes any
`Mob{Kind, SceneID, pos, ...}`, not restricted to worms; `caves.go` already spawns multiple kinds
(`cave-bat`, `cave-spider`, `skeleton`) into zone 2. Every existing spawn table
(`MeadowWormSpawns`, `TownSquareWormSpawns`, `CavesSpawns`) is a hand-written fixed list called
once at startup, though -- there's no "spawn kind X at position Y in scene Z" runtime API driven
by external input yet, just the primitive it'd be built on. `zone/zone.go`'s `Manager.AddZone`
registers new zone IDs at runtime, not just the five defaults -- nothing blocks creating/tearing
down ephemeral per-instance zone IDs, the Manager just has no explicit "destroy zone" (an unused
zone with zero players is harmless to leave registered).

**Client rendering**: none of `apps2/battlegrounds_gui`'s three existing modes (local arena demo,
networked instanced PvP, Town Square) fit "walk a generated 3D space and fight mobs." Town's own
worms are hand-synced decorative constants with no live position/HP feed at all -- real Town
combat is entirely textual, through `/api/town/command` to the MUD, not positional. A dungeon
explore-and-fight render mode is new work, though it can start from Town's box-art style and the
arena's own movement/camera code rather than from nothing.

**Arena heroes as mobs/bosses** (founder direction): REDGARDEN's 28 hero kits already have real
ability/AI logic (`apps/arena_bot`, `apps/client/bot_main.c`) built for PvP. Using them as
dungeon enemies means a dungeon encounter is "fight an AI-controlled hero kit," not a new enemy
AI system built from scratch -- the real work is *driving* that existing AI as a hostile NPC
inside a dungeon instance rather than as a bot-pool PvP participant, plus whatever minion/boss
distinction the founder's pending art direction implies (not yet specified).

## 2. The mechanism

```
Matchmaker (new dungeon queue, or party-invite flow -- not yet decided, see open questions)
    -> fork/exec a new "dungeon server" binary (new, not arena_server) on its own port
    -> dungeon server generates a fresh room/corridor layout (new generator, seeded per instance)
    -> populates it with mob/boss spawns drawn from the arena hero roster (reusing existing
       hero AI, repurposed as hostile NPCs) plus whatever minion types the pending art defines
    -> client (new render mode) connects, walks the generated space, fights
    -> instance ends on clear/wipe/leave, process exits, port freed -- same lifecycle arena
       matches already have
```

## 3. Architecture

### 3.1 Instancing

New `--server-bin` (a dungeon server, not `red_garden_arena_server`) reusing the matchmaker's
existing generic fork/exec spawn path. Needs: (a) a way to pass a seed and party roster into the
spawned process (today's matchmaker only passes `--port`/`--lobby-size` -- extend the spawn args,
or have the dungeon server pull its own config from IDUNA by an instance ID passed at spawn), (b)
`PACKET_MATCH_FOUND` (or an equivalent) telling the client it's joining a dungeon, not an arena
match, since the client currently assumes arena protocol implicitly from which port it queued on.

### 3.2 Procedural room/corridor generation

New generator, built on the solid-grid-then-carve representation Caves already demonstrates but
with real topology: place a seeded sequence of rooms, connect them with corridors, ensure
reachability from entrance to exit/boss room. Diablo 2's own approach (preset room "tiles" from a
pool, stitched together with connectivity rules) is the closest real precedent to build toward --
not attempted here, this section only establishes where the generator lives and what it emits
(the same `WorldBlock`-shaped output `worldapi` already knows how to serve).

### 3.3 Mob and boss population

Uses `server/mob`'s existing `Registry.Spawn` primitive, but needs a new runtime spawn-table
generator (seeded alongside the room layout, not a hand-written fixed list) that places
mob/minion spawns per room and a boss spawn in the final room. The mobs/bosses themselves are
REDGARDEN's existing arena hero kits, driven by their existing AI (`apps/arena_bot`) but as
hostile NPCs rather than PvP participants -- the real new work is that repurposing, not new AI.
Minion variety/roster is explicitly the founder's pending art-direction drop (§1), not designed
here.

### 3.4 Client render mode

New mode in `apps2/battlegrounds_gui`: walk a generated 3D space (starting from the arena's own
movement/camera/click-to-move code, Town's box-art style for geometry), fight hero-kit-driven
mobs using Battlegrounds' existing combat rendering (health bars, ability slots, hit feedback --
all already built, just pointed at PvE targets instead of PvP).

## 4. What this is not

Not a persistent walkable zone -- ends and tears down like a Battlegrounds match, per founder
direction. Not built on Caves' own generator -- that scene stays exactly as it is; a dungeon gets
its own new, seeded, multi-room generator. Not a new enemy AI system -- mobs/bosses are REDGARDEN's
existing hero kits repurposed as hostile NPCs, not new content built from scratch. Not
art-directed yet -- minion/boss specifics wait on the founder's pending upload, named as an open
gap rather than guessed at.

## 5. Open questions (not resolved here)

- How does a party actually queue into a dungeon -- a new matchmaker pool (like the existing
  arena/card-RTS ones), a direct party-invite flow, or something else?
- Does clearing a dungeon grant real rewards (loot, XP) through IDUNA's existing character/gold
  columns, same as `MMO_NORTHSTAR.md`'s economy systems, or is a first pass combat-only?
- Difficulty/scaling: fixed per dungeon, or scaled to party size/level like real Diablo 2?

## 6. Milestones

| # | Milestone | Acceptance | Status |
|---|---|---|---|
| 0 | This northstar | Written, registered in golden-docs-index | DONE |
| 1 | Dungeon server binary + instancing | New `--server-bin` spawns via the matchmaker's existing fork/exec path, client receives a real "you're in a dungeon, here's the port" message | NOT STARTED |
| 2 | Seeded room/corridor generator | Multi-room layout, connected, reachable entrance-to-boss-room, regenerates differently per instance | NOT STARTED |
| 3 | Mob/boss population from arena heroes | Seeded spawn table places hero-kit-driven hostile NPCs per room + a boss in the final room, using existing `apps/arena_bot` AI | NOT STARTED |
| 4 | Client dungeon render mode | New mode walks the generated space, fights populated mobs, using Battlegrounds' existing combat HUD | NOT STARTED |
| 5 | Art direction incorporated | Minion/boss roster and visual style from the founder's art drop, once shared | NOT STARTED |
| 6 | Rewards + party queue flow | Resolves the open questions in §5 | NOT STARTED |

## 7. Related docs

- `docs2/REDGARDEN_GUI_NORTHSTAR.md` -- Battlegrounds' own instancing model, matchmaker/arena_server this design's spawn path reuses
- `docs2/HEADLESS_SESSION_NORTHSTAR.md` -- the other real-space-to-explore design (Town/worm zone), a persistent-zone counterpart to this instanced one
- `docs2/SMOOTH_TERRAIN_NORTHSTAR.md` -- if dungeon geometry ever wants natural (not blocky) rendering rather than Town's box-art style, that plan applies here too
- `docs2/MMO_NORTHSTAR.md` -- IDUNA character/gold schema any real reward system (§5) would use

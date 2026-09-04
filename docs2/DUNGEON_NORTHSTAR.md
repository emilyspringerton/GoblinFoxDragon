# Instanced Dungeons — Northstar

*Written 2026-08-02. Founder: "ok can you make a dungeon like diablo 2 dungeons?" -> scoped:
instanced (spawned per-group like a Battlegrounds match, ends when cleared/left, not a persistent
walkable zone) and procedural (rooms/corridors reassembled from a tile set, different layout each
visit, not a fixed hand-built map like Town) -> "use some of the arena heroes as dungeon mobs and
bosses" -> "also the different minions whatever im uploading some art in a second" -> "feel free
to chunky 2 d sprites rendered in 3d if thats easier" -> "one more pull for art 1" -> all three
uploaded art files landed (GoblinFoxDragon commits b2266f6, 6ad32fa): `kikoryu.jpeg` (a detailed
painted boss-tier creature -- fire/wing/claw dragon-wolf-goblin hybrid), `art2.jpeg` (a full sheet
of rough minion concept sketches -- small abstract eldritch creatures, big eyes, checkered/grid
body textures, tentacles, spikes), and `art1.jpeg` (a sheet of ornate weapon/dagger concepts --
jagged crossguards, star/pentagram sigils, a totem-pole-like staff object -- plus a couple more
creature heads in `art2`'s style, and flavor text: "Deceiver," "Office Star," "E Pluribus Unum").
`art1` reads as loot/item design more than mob design -- see §1's rendering-approach note and
§5's rewards question.*

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
inside a dungeon instance rather than as a bot-pool PvP participant. Art now landed for the tiers
this implies: `kikoryu.jpeg` reads as boss-tier (one detailed, dramatic creature per encounter,
matching "a boss in the final room" in §3.3), `art2.jpeg`'s sheet reads as the minion roster
(many smaller, simpler creature types populating regular rooms) -- exact mapping of which arena
hero kit's AI drives which visual is not decided here, just that the two-tier split the art
itself already suggests lines up with the boss/minion split already in this doc. `art1.jpeg`
turned out to be a third category, not more mobs -- ornate weapon/dagger designs and a
totem-staff object, reading as loot/item concept art rather than creature art; relevant to §5's
"does clearing a dungeon grant real rewards" open question, not to mob/boss population.

**Rendering approach** (founder: "feel free to chunky 2 d sprites rendered in 3d if thats
easier"): confirmed nothing in `apps2/battlegrounds_gui/src/main.c` loads textures or does
billboard rendering today -- no `SDL_image`/`IMG_Load`, no `glTexImage2D`, nothing. Every existing
model (heroes, worms, buildings) is stacked flat-colored boxes. This means billboarded sprites
(camera-facing textured quads, Diablo 2's own actual original technique) and "real" 3D box-art
models are BOTH genuinely new client infrastructure -- sprites aren't a shortcut around existing
capability, they're the simpler of two things that don't exist yet. Chosen: chunky 2D sprites,
per founder direction -- needs a texture-loading path (`SDL_image` or a raw-format loader) and a
billboard quad (a single camera-facing quad per mob, not `upload_mesh`'s pos+normal-only
triangle-mesh path) as new, separate additions to the render pipeline.

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
movement/camera/click-to-move code; room/corridor geometry can stay Town's flat box-art style,
no need to invent new geometry rendering). Mobs/bosses render as billboarded 2D sprites (§1's
"Rendering approach") rather than box-stack models -- a genuinely new small subsystem (texture
loading + camera-facing quad draw call per mob) sitting alongside, not replacing, the existing
`upload_mesh`/`draw_mesh` geometry path. Combat itself reuses Battlegrounds' existing rendering
(health bars, ability slots, hit feedback -- all already built, just pointed at PvE targets
instead of PvP).

## 4. What this is not

Not a persistent walkable zone -- ends and tears down like a Battlegrounds match, per founder
direction. Not built on Caves' own generator -- that scene stays exactly as it is; a dungeon gets
its own new, seeded, multi-room generator. Not a new enemy AI system -- mobs/bosses are REDGARDEN's
existing hero kits repurposed as hostile NPCs, not new content built from scratch. Mob/boss art is
directed (boss + minion-sheet, §1); loot/item art exists (`art1.jpeg`) but isn't wired to any
system yet since §5's rewards question is still open. Not full 3D character models -- billboarded
sprites, per founder direction.

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
| 1 | Dungeon instancing | A player can enter a real, freshly-generated dungeon instance and the client renders it | IN PROGRESS -- **real, decisive correction (2026-09-04)**: this milestone's own acceptance text was written assuming REDGARDEN's per-match architecture ("New `--server-bin` spawns via the matchmaker's existing fork/exec path... here's the port"), but Milestone 4's own acceptance text names "Battlegrounds' existing combat HUD" as the real dungeon-render client -- and Battlegrounds (`apps2/battlegrounds_gui`) talks to THIS repo's own `apps2/server-go`, a single, long-running, persistent-world UDP process (checked directly: exactly one `net.ListenUDP` for the whole game), not a REDGARDEN-style per-match spawned server. Wiring dungeon entry through REDGARDEN's matchmaker would mean the dungeon never reaches the client that renders it. REDGARDEN's own real seed/mode transport (`4bb46b9`, `MatchFoundMsg` seed+mode, `--mode dungeon` CLI flag) stays real and shipped, but is now understood to serve REDGARDEN's OWN arena/dungeon ambitions if it ever gets one, not this milestone's own dungeon feature. Real, shipped v0 instead: `server/worldapi/dungeon_instance.go`'s `DungeonInstanceRegistry` -- `Allocate` generates a fresh instance (Milestone 2/3's own already-real `GenerateDungeonLayout`/`DungeonLayoutToBlocks`) and reserves a scene ID in the real, wire-format-bounded `208-255` range (checked directly: `PacketTelecrystalAck`/`PacketSceneChange` encode scene ID as a single `uint8`, capping any scene-ID scheme at 0-255; existing scenes use 0-3/200-207, so this range is real and free), refusing (not wrapping) once its 48-instance capacity is exhausted since there's no release/expiry mechanism yet. New `apps2/server-go` UDP handler, `PacketDungeonEnter` (payload: `dungeon_index` byte) -- reuses `server/telecrystal`'s own real travel mechanism (`idunaclient.TravelTelecrystal` at 0 cost, ack shaped exactly like `PacketTelecrystalAck`/`PacketSceneChange`) rather than inventing a new travel path. `worldapi`'s own `/chunks` HTTP handler now checks the registry before falling back to `ProceduralWorldStore`. 9 new tests (increasing scene IDs, has/unknown-scene, blocks-for-chunk matching direct generation, out-of-range chunk empty-not-error, unknown-scene rejection, entry-spawn inside the entrance room, multiple instances stay independent, exhaustion refuses rather than wraps) all green; `go build`/`go vet`/`go test ./...` clean across the whole repo, zero regressions. Real, honest, NOT done: party-roster passthrough (every `Allocate` call makes a brand-new SOLO instance, no shared-instance-by-party concept yet), and no live end-to-end verification against a real running IDUNA + UDP client was performed this pass (unit/integration-tested against the real registry/generator code directly, not a live multiplayer session). **Status correction (2026-09-04)**: "mob spawns aren't wired into a live instance yet" no longer describes the real, current state of this feature overall -- it's still true that nothing calls `GenerateDungeonSpawns` from THIS repo's own `apps2/server-go` specifically (that stays real and accurate for `DungeonInstanceRegistry`'s own instancing path, which Milestone 4's own real, later founder decision left shipped-but-unused pending true per-party instancing), but Milestone 4 below shipped a real, live, working dungeon experience with real populated mobs through `apps2/mud` instead, the repo's own actual real PvE-content owner (`server/mob`'s `Registry`/`DungeonRoster` live there, not in `apps2/server-go` at all -- see Milestone 4's own real, decisive finding). Read this row as "the `apps2/server-go`/`battlegrounds_gui` visual-render path specifically has no live mobs yet," not "no real dungeon has live mobs anywhere in GFD." |
| 2 | Seeded room/corridor generator | Multi-room layout, connected, reachable entrance-to-boss-room, regenerates differently per instance | DONE (real, minimal v0) -- `server/worldapi/dungeon.go` (GFD commit pending): `GenerateDungeonLayout(seed)` places 5-8 rooms in a seeded, alternating-axis snake layout, connected in a linear chain (real reachability BY CONSTRUCTION, verified via an actual BFS in `IsReachable()`, not assumed); `DungeonLayoutToBlocks` carves it into real `WorldBlock`s using the same solid-grid-then-carve representation `cavesChunk` already established. 7 real tests (determinism, distinctness across seeds, room-count bounds, reachability, real floor/wall block output). Real, honest, deliberately NOT built: Diablo 2's own preset-room-pool/branching-connectivity style (named as future work in §3.2 already) -- this is a straight chain, not a graph with loops/branches; **Update 2026-09-04**: now wired into the real chunk-streaming HTTP path, via Milestone 1's own `DungeonInstanceRegistry` (`server/worldapi/dungeon_instance.go`). |
| 3 | Mob/boss population from arena heroes | Seeded spawn table places hero-kit-driven hostile NPCs per room + a boss in the final room, using existing `apps/arena_bot` AI | IN PROGRESS -- real seeded spawn TABLE shipped 2026-09-03 (`server/mob/dungeon.go`): `DungeonRoster` carries §7's real 8-dungeon boss/elite table as Go data, `GenerateDungeonSpawns(layout, dungeonIndex, sceneID, seed)` places the named boss in the layout's `BossIdx` room, an optional elite in non-boss/non-entrance rooms, generic minions elsewhere, entrance room kept clear. 6 new tests (boss presence, entrance-clear, determinism, index-wrap safety, empty-layout safety, real `Registry.Spawn` acceptance) all green. **Status correction (2026-09-04)**: this row's own "not wired anywhere" gap, as it read before Milestone 4's own real shipped work below, is resolved -- `GenerateDungeonSpawns` IS now a real, live, called dependency (`apps2/mud/main.go`'s own `spawnInto(zoneID, mob.GenerateDungeonSpawns(layout, i, zoneID, seed))`, confirmed live in source), not a function nothing calls. Real, honest, still NOT done, the one real remaining gap this milestone's own acceptance text names: every spawned `Mob`'s `Kind` carries the real hero identifier (e.g. `ARENA_HERO_CART`) but behaves as a generic mob today -- actually *driving* REDGARDEN's `apps/arena_bot` AI as a hostile NPC (the harder, real cross-repo work §3.3 itself names) is still ahead. |
| 4 | Client dungeon render mode | New mode walks the generated space, fights populated mobs, using Battlegrounds' existing combat HUD | NOT STARTED for the visual `apps2/battlegrounds_gui` 3D render mode -- **real, decisive correction (2026-09-04, GFD-993944, founder: "have a dungeons button... basic combat")**: this milestone's own acceptance text assumed apps2/server-go (Battlegrounds' UDP transport) would host dungeon combat, but real investigation (founder decision, AskUserQuestion) found apps2/server-go has ZERO mob registry (PvP-only) -- every real PvE mechanic (DungeonRoster, GenerateDungeonSpawns, spawn/variant registries) lives in `server/mob`, imported ONLY by `apps2/mud`. Real, shipped v0 instead, entirely inside `apps2/mud` (matching every other real PvE zone's own architecture, not server-go's): 8 real dungeon zones (208-215) registered at world init via `zone.Manager.AddZone`, each populated with a real, deterministically-seeded `GenerateDungeonLayout`+`GenerateDungeonSpawns` roster (the real named boss/elite/minion content from §7, not placeholders) through the exact same `spawnInto` closure gating Hills/Caves/Swampville. Two new commands, no job/cost gate ("bare bones," per the founder's own framing): `dungeons` (lists the 8 real named dungeons) and `dungeon-enter <1-8>` (real zone transfer, same `zoneMgr.Transfer`/`syncChatSession`/`cmdLook` shape `cmdCastTeleport`/crystal travel already use). Live-verified end to end over real telnet: list → enter → see the real named roster → `attack <mob-id>` auto-approaches and lands real damage, multiple aggro'd mobs deal real damage back. Real, honest, NOT done: this is v0 shared/persistent per dungeon (like Hills/Caves), not per-party procedural instancing -- `server/worldapi`'s `DungeonInstanceRegistry` (apps2/server-go, Milestone 1) stays real and shipped but unused by this path, a real, separate, deferred future decision if true per-party instancing is ever wanted. Real, separate, decisive finding along the way, NOT fixed here: `apps2/battlegrounds_gui` has no client-side rendering path for ANY zone besides Meadow (scene 0, via `g_dfzone_active`) and Town (scene 4, box-art) -- Hills/Caves/Swampville (zones 1-3), despite having real, populated mobs since this session's own earlier fix, have never actually been visually enterable via the GUI client either; a dungeon "render mode" needs that same generic scene-agnostic PvE HUD built first (real, separate, bigger card, not silently folded into this one), THEN a dungeon-specific render mode on top of it, per this milestone's own original Not-Started row below and Milestone 4.5. |
| 4.5 | Billboard sprite subsystem | New texture-loading + camera-facing quad rendering path (nothing like this exists today), mobs/bosses render as chunky 2D sprites in the 3D space | NOT STARTED |
| 5 | Art direction incorporated | All 3 files landed: `kikoryu.jpeg` (boss), `art2.jpeg` (minion sheet), `art1.jpeg` (loot/item concepts, not mob art). Real content pass done (§7, 2026-09-03): 8 named dungeons, each with a real boss/elite roster drawn from the actual arena hero compendium; sprite sheets themselves not yet cut from this source art | IN PROGRESS |
| 6 | Rewards + party queue flow | Resolves the open questions in §5 | **PARTIALLY RESOLVED (2026-09-04, real investigation, not new code)** -- real, decisive finding: 2 of §5's own 3 real open questions are already answered by infrastructure this repo already has, confirmed live in source, not assumed. **§5 Q1 (party queue)**: `cmdDungeonEnter` (`apps2/mud/main.go`) transfers a player to a fixed, deterministic zone ID per dungeon number (`dungeonZoneBase + (n-1)`) -- every player who runs `dungeon-enter N` lands in the exact same shared zone (Milestone 4's own "shared/persistent per dungeon, like Hills/Caves" architecture), so a party trivially "queues together" today: each member just runs the same command, no special queue mechanism needed. Real, already-existing `invite`/`accept`/`party`/`leave-party` commands plus real, zone-range-checked party XP-splitting (`pt.XPSplit`, chain bonuses, `gw.playerParty`/`gw.parties`) already work generally, not dungeon-specific code. **§5 Q2 (rewards)**: XP rewards ARE already real and live for dungeon kills -- `awardXP` is the same universal, zone-agnostic on-mob-death path every other zone uses (confirmed live in source: dungeon mobs spawn through the identical `spawnInto` closure Hills/Caves/Swampville use, so their kills flow through the same real XP-award code, nothing dungeon-specific required). The one real, genuinely still-missing reward type is ITEM/LOOT drops -- zero random-item-award mechanism exists anywhere in GFD combat today (a real, separate, already-scoped gap, see `docs2/DUNGEON_HAT_DROPS_NORTHSTAR.md`'s own real finding for the BRAWLPIT-hat special case of this same gap). **§5 Q3 (difficulty/scaling)**: still genuinely unresolved -- no scaling exists (fixed per dungeon, matching every other zone's own real, static-difficulty precedent), a real, deliberate-by-omission choice, not actively decided either way; a real founder call if dynamic scaling is ever wanted. No code changed this pass -- a real, decisive investigation correcting this row's own stale "NOT STARTED" status, not new functionality. |
| 7 | Named dungeon content | 8 named dungeons + boss/elite assignments, real compendium-grounded (§7). Kikoryu's Hoard named as the real superboss/endgame target, honestly scoped as new AI work, not built | DONE (design only) |

## 7. Named dungeons and boss assignments — real content pass (2026-09-03)

Kanban cruise-queue card: "GFD add dungeons - we need content - make it like DIABLO in terms of
how the procedural dungeons work use the hero compendium to design bosses for the named
dungeons." This section is a real, deliberate CONTENT pass, not an engineering one — it resolves
§3.3's own explicitly-named open gap ("exact mapping of which arena hero kit's AI drives which
visual is not decided here") with real, specific choices, grounded in `TYLER/multiverse_heroes.md`
(the real 123-hero compendium every REDGARDEN arena hero is drawn from — checked directly:
every one of the 30 real, implemented `ARENA_HERO_*` kits traces to a real named entry there).
None of Milestones 1–4.5 are built yet; this section is the real content those milestones will
eventually consume, matching Diablo 2's own real act structure (a handful of named areas, each
with its own themed boss and mob roster) rather than one generic "dungeon."

Real, deliberate design rule applied throughout: each dungeon's boss is drawn from the SAME real
compendium faction as its theme (not an arbitrary hero placed for stat-balance reasons), so the
lore already coheres — a real, checked constraint, not decoration.

| # | Dungeon | Faction (compendium) | Boss (real `ARENA_HERO_*`) | Elite/recurring heroes | Minion flavor |
|---|---|---|---|---|---|
| 1 | The Sealed Archive | Jiangshi Syndicate | `ARENA_HERO_CART` (The Retrieval Cart — "an object that started arriving with things nobody ordered," a genuinely fitting final-room reveal for an archive dungeon) | `ARENA_HERO_NOOR1` (NOOR-1, "Four Days Behind") | `art2.jpeg`'s small eldritch/checkered-body sheet reads as misfiled archive contents come alive — the real intended fit for this room |
| 2 | The Frequency Table | Goetia Court | `ARENA_HERO_PAIMON` (Paimon, "commands two hundred legions" — the real compendium line that makes him the natural climactic boss of the whole court) | `ARENA_HERO_VASSAGO`, `ARENA_HERO_BELETH`, `ARENA_HERO_ZAGAN`, `ARENA_HERO_DOC_WHEEL` (Buer) | Same `art2.jpeg` roster, recolored/retextured per room — a real, cheap way to reuse one sprite sheet across multiple named dungeons rather than commissioning new art per dungeon, named honestly as a real production shortcut, not hidden |
| 3 | The Remnant's Hall | Valhalla Remnant | `ARENA_HERO_GUNNR` (Gunnr, "The Raven-Caller" — real, fitting continuity with this same session's own real blog thread, `state-of-the-ecosystem-the-duck-has-in-fact-moved`, which already named Gunnr as the roster's new low-water mark; a dungeon boss slot is a real, earned spotlight for a hero the blog has been quietly ribbing) | `ARENA_HERO_LOKI`, `ARENA_HERO_GARY`, `ARENA_HERO_COURIER` (Ratatoskr) | — |
| 4 | East of the Wall | Middle Kingdom Heirs | `ARENA_HERO_HE_XIANGU` (He Xiangu, "who stopped eating") | `ARENA_HERO_WEATHERMAN` (Ao Guang's Weather-Debt Collector), `ARENA_HERO_FLUTE_DEBT` (Han Xiangzi's Flute-Debt) | — |
| 5 | The Highland Wake | Highland Court | `ARENA_HERO_MORRIGAN` ("a war goddess who offered herself to a hero, was refused") | `ARENA_HERO_DAGDA` | — |
| 6 | The Founders' Table | Springerton Engine / Unbound Historicals (the original, pre-compendium REDGARDEN cast, later folded into the compendium proper — real, checked: `Unicorn`/`Ghost`/`Frog`/`Pizza`/`Flamel` all have real compendium entries, #104/105/106/108/110) | `ARENA_HERO_UNICORN` ("allegedly a robot" — the roster's own most-tenured, most-recognizable face, a real, deliberate choice for the dungeon that plays first) | `ARENA_HERO_GHOST`, `ARENA_HERO_FROG`, `ARENA_HERO_TREE`, `ARENA_HERO_DUCK` | Real, honest scope note: `ARENA_HERO_PIZZA`/`ARENA_HERO_ABRAHAM`/`ARENA_HERO_ADA`/`ARENA_HERO_MNM`/`ARENA_HERO_CAIN` are deliberately NOT placed in this dungeon or any other below — a real, second content pass, not resolved here |
| 7 | The Unbound Wing | Unbound Historicals | `ARENA_HERO_CAIN` (Cain, East of Eden — "the first murder," a real, tonally distinct, genuinely dark capstone appropriate for a late-game area) | `ARENA_HERO_ABRAHAM` (Abraham of Worms), `ARENA_HERO_ADA` (Ada Lovelace), `ARENA_HERO_MNM` | Real, deliberate tonal note: this dungeon is explicitly heavier/stranger than 1–6, matching the compendium's own real framing of Faction 11 as "a different kind of grounding... not myth" |
| 8 | The Proving Grounds | Non-canon (DragonsNShit-ported Battlegrounds content, per `arena_game.h`'s own comment: "not a TYLER hero") | `ARENA_HERO_WARRIOR` | — | Real, deliberate choice: the lowest-tier, first dungeon a new party clears — a mechanically simple boss fight with no compendium lore weight, the same real reason `arena_game.h` itself treats Warrior as content, not canon |

**Kikoryu's Hoard — the real, named exception, honestly scoped as NEW work, not a repurposed
hero.** `kikoryu.jpeg` (the detailed painted fire/wing/claw dragon-wolf-goblin hybrid, already
landed per this doc's own header) does not map to any existing `ARENA_HERO_*` kit — it's real,
new, unique boss art, not a repurposed PvP hero. Real, honest scope: unlike every dungeon above
(whose real "new work" is *driving* existing arena-bot AI as a hostile NPC), Kikoryu genuinely
needs its own new AI/kit built from scratch — the same category of real, larger, separate work
`§4`'s own "not built from scratch" framing explicitly carves out. Real, deliberate design
placement: the superboss/endgame dungeon, cleared only after all 8 named dungeons above,
matching Diablo 2's own real structure (a superboss beyond the act bosses, not gated behind any
one act). Not built here — named as this document's own real, concrete next content target once
Milestones 1–4.5 land and at least one of dungeons 1–8 is real and playable.

**Real, deliberate roster gap named, not glossed over**: `ARENA_HERO_PIZZA`, `ARENA_HERO_TYLER`,
and `ARENA_HERO_BACON_PUCK` are real, implemented arena heroes with no dungeon placement above —
a genuine, honest incompleteness in this content pass, not an oversight hidden by omission. Real
candidate for a follow-up: a ninth, meta/"behind the scenes" dungeon built around `Tyler` and
`Bacon Puck` specifically, echoing this same session's own real blog continuity (`TYLER, Series
X: The Long Quiet and Recruitment Drive`) — named as a real idea, not committed to here.

## 9. Related docs

- `docs2/REDGARDEN_GUI_NORTHSTAR.md` -- Battlegrounds' own instancing model, matchmaker/arena_server this design's spawn path reuses
- `docs2/HEADLESS_SESSION_NORTHSTAR.md` -- the other real-space-to-explore design (Town/worm zone), a persistent-zone counterpart to this instanced one
- `docs2/SMOOTH_TERRAIN_NORTHSTAR.md` -- if dungeon geometry ever wants natural (not blocky) rendering rather than Town's box-art style, that plan applies here too
- `docs2/MMO_NORTHSTAR.md` -- IDUNA character/gold schema any real reward system (§5) would use
- `TYLER/multiverse_heroes.md` -- the real 123-hero compendium §8's own dungeon/boss assignments are drawn from

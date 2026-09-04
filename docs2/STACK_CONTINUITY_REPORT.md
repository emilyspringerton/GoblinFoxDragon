# GoblinFoxDragon — Stack Continuity Report

**GFD-DOX-124** ("do a deep dive stack continuity report linking off the readme fully update the
readme with the current state of the world and current direction of the project"). Real, checked
directly against the actual source tree, `git log`, and this repo's own `docs2/*_NORTHSTAR.md`
files as of **2026-09-04** (950 real commits, most recent the same day) — not a restatement of
old docs, a fresh read of what's actually here right now. Linked from `README.md`.

## 1. What GoblinFoxDragon actually is today

A real, playable, server-authoritative FFXI-style MMO stack ("DragonsNShit") plus a real,
separate MOBA/arena PvP mode ("Battlegrounds"), sharing one C client
(`apps2/battlegrounds_gui`) and split across two real backends with genuinely different
architectures and, as of this session, a genuinely different real content boundary between them
(§3). Not a finished game — a working, rough, actively-developed vertical slice with real combat,
real jobs, a real economy, and a growing PvE content pipeline (dungeons, Notorious Monsters), sitting
alongside an entirely separate real-time PvP arena mode.

## 2. The apps (real binaries, what each one is)

| App | Language | Role |
|---|---|---|
| `apps2/mud` | Go | **The real PvE/MMO server.** Telnet + HTTP text-command MUD backend — jobs, spells, mob AI/combat, loot, quests, auction house, guilds, dungeons, Notorious Monsters, party/XP, conquest, world crisis, TRAPX city sim. The single biggest, most actively developed binary in this repo. Runs under `gfd-mud.service` (systemd), ports `:2323` (telnet) / `:7171` (WorldEvent HTTP API). |
| `apps2/server-go` | Go | **The real PvP/arena server.** Server-authoritative UDP backend for Battlegrounds matches — real-time snapshot/attack protocol, matchmaking, HPState/TPState combat (shared `server/combat` code with `apps2/mud`). **Has zero mob registry** — confirmed directly this session (GFD-x-123/124's own investigation) — PvP-only, no PvE content lives here. Also serves the real Bedrock-protocol voxel backend + `/heightmap`/`/chunks` HTTP API (`worldapi`, `:7070`), consumed by both `battlegrounds_gui`'s own Town/Meadow terrain and `WEAKNIGHT_BEDROCK_RACERS`. |
| `apps2/battlegrounds_gui` | C99/SDL2/GL | **The real client.** One binary, two real modes: Town/Meadow (talks to `apps2/mud` over HTTP text commands + polls `worldapi` for terrain) and Battlegrounds (talks to `apps2/server-go` over UDP). Real IDUNA email+password login (with real LOG IN/SIGN UP buttons as of this session), a real 6-slot ability bar now backed by a real PARENA mod (§4), a real Inventory screen, real combat log, real King/buff HUD, WASM build deployed live at `okemily.com/battlegrounds/`. Forked from REDGARDEN's `apps/arena` at commit `61baafb` — diverges forward, not a live REDGARDEN checkout. |
| `apps/lobby` | C99/SDL2 | SHANKPIT-lineage lobby/menu client; hosts the EduScript "Architect's Orb" terminal (a real, separate, small puzzle-scripting VM — not PARENA, see `docs2/MOD_SURFACE_NORTHSTAR.md` §3 for why the two coexist). |
| `apps/server` | C | Legacy/optional plain UDP server, kept as a fallback build target in CI; not the real, current backend for anything above. |
| `apps2/crystal` | Go | Standalone boids/ecosystem simulation that produces the real Meadow goblin/fox NPC seed file (`data/crystal_seed_meadow.json`) `apps2/mud` optionally loads at startup and `battlegrounds_gui` renders client-side. |
| `apps2/client-go`, `apps2/wsudprelay` | Go | Smaller support/bridge utilities (a Go reference client, a WebSocket↔UDP relay for the WASM build's own networking needs). |

## 3. The real, decisive architectural boundary found this session

`apps2/mud` (Go, PvE) and `apps2/server-go` (Go, PvP/UDP) are genuinely separate worlds with no
shared mob/combat-target model — confirmed directly, not assumed, while scoping GFD-x-123/124's
mod event broker and GFD-993944's dungeon work. Every real PvE mechanic (`server/mob`,
`server/nm`, `server/spawn`, `server/mobdrop`, `server/mobvariant`, dungeon generation) is
imported **only** by `apps2/mud`. `apps2/server-go` shares `server/combat`'s HP/TP state
machinery with `apps2/mud` (so player-vs-player damage math is consistent) but has never had a
mob/NPC concept at all. Two real, concrete consequences already acted on this session:
- **Dungeons route through `apps2/mud`, not `apps2/server-go`** (GFD-993944) — even though the
  DUNGEON_NORTHSTAR's own original Milestone 1 wired instance-entry through `apps2/server-go`'s
  UDP transport, that work is now understood to be on the wrong side of this boundary and stays
  unused for this feature (not deleted — a real, future per-party-instancing path could still use it).
- **The new mod event broker (`server/modevent`) lives in `apps2/mud`**, not `apps2/server-go`,
  for the same reason.

## 4. Mods-first: the real, current PARENA/BURROW integration state

GFD had **zero** PARENA mod-surface presence until this session (checked directly against
`docs2/MOD_SURFACE_NORTHSTAR.md` §2's own finding). Two real, separate, now-live integration
paths exist, matching the two real host languages in this repo:

- **C client target** (`apps2/battlegrounds_gui`, via the real `parena build` CLI): first real
  mod, `PARENA/stdlib/gfd/action_bar_mod.prn` → `on-gfd-ability-for-slot`, generated C committed
  at `packages/simulation/action_bar_mod.c`, decides which ability lives in which action-bar
  slot per job. Static-link pattern (compile once, commit the `.c`, call by name) — the same
  proven-in-production shape `ECOWAR/docs/ARENA_API.md` documents for 8 real mods there,
  deliberately chosen over the still-unscoped federated EduScript↔PARENA process model in
  `MOD_SURFACE_NORTHSTAR.md` §3a and over PAPERCRAFT's heavier `dlopen`-based `--mods-manifest`
  runtime-loading pattern (neither needed, since this client's release model is static-linked
  native + WASM builds, not live-reload).
- **Go server target** (`apps2/mud`, via BURROW's own Go emission target): first real mod,
  `PARENA/stdlib/gfd/nm_bonus_mod.prn` → `OnGfdMobDeathXpBonusPercent`, compiled via
  `burrow build -o x.go` into `apps2/mud/internal/burrowgen/nm_bonus_mod_gen.go` (committed),
  wired through the new `server/modevent.Broker` (real, generic, string-named,
  I32-typed pub/sub — any core code or mod can define a new event without touching the broker).
  **This is the first real host anywhere in the whole monorepo to actually consume BURROW's Go
  emission target** — shipped since 2026-08-30, never used by a real host until this.

Real, honest, current scope limit on both: this buys `.prn`-edit-and-recompile extensibility
(reassign a slot, add a new decision branch), not hot-reload or third-party/runtime modding — no
`dlopen` surface exists in either host. A real, separate, bigger ask along these same lines
(`GFD-NM-124`'s own explicitly-deferred "plugins call special abilities via special named
functions") is the natural next consumer of `server/modevent` once there's a real event worth
publishing for it.

## 5. Major initiatives — real, current status

| Initiative | Doc | Real status |
|---|---|---|
| Mob spawn/variant/NM data model | `MOB_SPAWN_NORTHSTAR.md` | **Shipped, all 5 phases.** Data-driven spawn toggles (`data/mob_spawns.json`), FFXI-style zone grid coords, drop tables (`data/mob_drops.json`), difficulty-tier variants (`data/mob_variants.json`), dungeon roster override (`data/dungeon_roster.json`) — all with a real IDUNA admin GUI page each. Found and fixed a real, long-standing bug along the way: Hills/Caves had never actually spawned mobs since S189, despite the code existing. |
| Notorious Monsters | (folded into `MOB_SPAWN_NORTHSTAR.md`'s own spawn interface, `server/nm`) | **Live and data-driven.** `server/nm`'s real placeholder-kill/window/respawn engine now reachable from `data/mob_spawns.json` itself (GFD-NM-123), with a real IDUNA admin editor (GFD-NM-124). Found and fixed a real bug in the same pass: Hills/Caves' own 4 pre-written NMs had never actually been wired into the live trigger loop. |
| Dungeons | `DUNGEON_NORTHSTAR.md` | **v0 real and playable, text/telnet-only.** 8 real named dungeons (real boss/elite content drawn from the actual 30-hero arena compendium), seeded room-corridor generation, real combat — live-verified end to end (enter, fight, take and deal real damage) via `apps2/mud`. **Not** yet visually enterable via `battlegrounds_gui` (§6) or truly per-party-instanced (v0 is one shared, persistent zone per dungeon, like Hills/Caves). |
| Mod surface / event broker | `MOD_SURFACE_NORTHSTAR.md`, §4 above | **First real slices shipped on both hosts**, generalizable pattern proven, most of the doc's own bigger asks (destructible environments, faction hooks, METALVERSE terminal mode, the federated EduScript↔PARENA process model) still real, scoped, not started. |
| Inventory / equipment | `INVENTORY_EQUIPMENT_NORTHSTAR.md`, `INVENTORY_PERSISTENCE_NORTHSTAR.md` | Real in-memory inventory + a real FFXI-style Inventory screen in `battlegrounds_gui` (GFD-INV-93911). **Real, known, unfixed gap, deliberately scoped not built**: `apps2/mud`'s own `p.inventory` is never persisted to IDUNA on either the telnet or headless path — a headless session (the GUI's only path) idle-evicts after 10 minutes with zero durability. Root cause of the live "Auction House shows no inventory" report. Real 3-phase fix scoped (`INVENTORY_PERSISTENCE_NORTHSTAR.md`), founder decision: scope it, don't build yet. |
| Login/onboarding | (no dedicated NORTHSTAR yet) | Real IDUNA email+password login, real LOG IN/SIGN UP buttons (root-caused a real SDL2/AltGr hotkey conflict on Windows along the way). `GFD-ONBOARD-123` (a full multi-screen sign-up flow: email/password/confirm/honor-code/character-creation) is real, queued, not yet started. |
| Admin/back-office tooling | (no dedicated NORTHSTAR; lives in IDUNA) | Real, growing set of IDUNA admin pages for GFD content: Items, Mob Drops, Mob Spawns (+NM), Dungeon Roster — all real CRUD APIs + cream/gold UI pages, all editing the real JSON files `apps2/mud` loads at startup. Machine-name fields (mob kind, dungeon boss) now have real autocomplete (GFD-XX-123), matching `PRRJECT_FATBABY`'s own ticker-search UX. |
| Performance | (no dedicated NORTHSTAR) | One real, measured fix shipped outside this repo but directly serving it: IDUNA's `backlog.ParseFile` (the kanban↔`EMILY/BACKLOG.md` bridge every GFD card in this whole report went through) was re-parsing a 29,808-line file on every call (~250ms); now cached (~29µs), an ~8,600x real speedup (GFD-OPTIM-1244). `apps2/mud`'s own core game-loop performance (a separate, real, founder-flagged concern, `GFD-994001`) has not been profiled or touched yet. |

## 6. Real, current, honestly-named gaps (not hidden anywhere)

- **Client rendering only covers Meadow (scene 0) and Town (scene 4).** Checked directly this
  session: `battlegrounds_gui`'s own scene-switch logic (`g_dfzone_active`) is hardcoded to those
  two scene IDs. Hills, Caves, Swampville — and now all 8 new dungeon zones — have real, live,
  populated mobs and real combat, but are reachable only via telnet/headless text commands, never
  visually, via the GUI client today. This is the real, single biggest gap between "the server
  has real PvE content" and "a player can actually see and click on it," and is a real
  prerequisite for `DUNGEON_NORTHSTAR.md`'s own Milestone 4 (client dungeon render mode).
- **Inventory has no durability** (§5) — a real, live, reported bug, root-caused and scoped, not fixed.
- **Battlegrounds (the PvP arena mode) is currently reported broken, and now root-caused**
  (`GFD-BG-12444`, `docs2/BATTLEGROUNDS_MIGRATION_NORTHSTAR.md`): GFD has no native
  matchmaker/arena-server at all — every real match is served by REDGARDEN's own live
  processes, and GFD's own forked client simulation code (`protocol.h`, `arena_game.c`) has
  drifted 78/1161 real lines behind REDGARDEN's current, still-evolving copy. A real,
  founder-level decision among 3 named options (full sync, pin to the fork commit's binaries, or
  give GFD its own native server) is needed before a fix — not resolved here.
- **Every job shares one level, and now root-caused** (`GFD-XX-1249`,
  `docs2/PER_JOB_LEVELS_NORTHSTAR.md`) — real correction to this report's own earlier framing:
  the single shared level is actually real, deliberate FFXI-parity design (FFXI itself shares one
  level across jobs), not a bug. The founder's ask is a genuine, decisive deviation toward
  FFXIV/WoW-style independent per-job leveling — real in-memory refactor scoped (buildable), real
  durability blocked on a new IDUNA schema (same class of gap as inventory persistence, §5).
- **`docs/NORTHSTAR.md`** (note: `docs/`, not `docs2/`) is the repo's oldest architecture doc,
  dated 2026-06-14, and was already flagged as stale by the README itself before this report —
  still true, not refreshed here; `docs2/*_NORTHSTAR.md` are the real, current per-system specs.

## 7. Real, current direction

Read off the actual pattern of what's been asked for and shipped, not speculated: **PvE content
depth first, PvP maintenance mode.** Dungeons, Notorious Monsters, mob spawn/variant tooling,
inventory, and onboarding have all seen real, recent, concrete work; Battlegrounds (PvP) has seen
a report that it's broken and no recent feature work. **Mods-first is now a real, working
discipline on both hosts** (C client, Go server), not just a stated intention — the near-term
trajectory is more core mechanics (job-abilities, NM special abilities, future spawn/loot
decisions) moving onto `action_bar_mod`'s and `nm_bonus_mod`'s own pattern rather than staying as
hardcoded Go/C. **Admin tooling is catching up to content systems** — every new JSON-driven
mechanic this session shipped its own IDUNA page in the same pass, not as a later followup. The
real, current, unresolved question the whole session's own trail keeps bumping into: whether
GFD's PvE content (Hills/Caves/Dungeons) ever gets a real client render path, or whether
`battlegrounds_gui` stays a Meadow/Town/Battlegrounds-only client with everything else played
text-first — not decided, the single most consequential open architecture question for this
repo's own near-term roadmap.

## Related

- `README.md` — this report is linked from there; keep both in sync going forward.
- `docs2/MOB_SPAWN_NORTHSTAR.md`, `docs2/DUNGEON_NORTHSTAR.md`, `docs2/MOD_SURFACE_NORTHSTAR.md`,
  `docs2/INVENTORY_PERSISTENCE_NORTHSTAR.md` — the real, per-system specs this report synthesizes.
- `CHANGELOG.md` — the real, dated, commit-linked trail every claim above is drawn from.

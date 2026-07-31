# REDGARDEN-as-GUI — Northstar

*Written 2026-07-31. Founder, real-time, verbatim: "can we graft redgarden frontend onto GFD mud
as a gui to make our mmorpg?" → "i dont care how you do it fork redgarden into GFD write the
northstar this is the mmo. this is dragonsnshit" → "cli will continue to work" → "redgarden as a
gui" → "like old school runescape."*

**Status:** Spec only, no code yet — milestone 0 of this doc's own table.

**Relationship to `docs2/MMO_NORTHSTAR.md`:** amends it, doesn't replace it. Every system that
doc already specs — IDUNA-backed character/item/guild schema, item provenance chains, the
economy, World Crisis phases, Telecrystal scene travel, EduScript VM scripting — stays exactly
as designed. This doc changes one thing: **who renders it.** MMO_NORTHSTAR's own "Integration
Architecture" diagram named "C/SDL2 Client (SHANKPIT runtime, extended)" as the frontend; that
line is superseded by this doc. See §7 for the exact edit made to that file.

---

## 1. Three-sentence version

DragonsNShit already has a real, deep MMORPG backend — 22 FFXI-parity jobs, skillchains and
magic bursts, enmity, conquest, NM spawns and treasure pools, crafting guilds, parties and
linkshells, all live in `apps2/mud`'s 7,300-line Go server — but the only way to play it today
is a text telnet session on `:2323`. REDGARDEN is a separate, working, real-time 3D SDL2/OpenGL
MOBA client with exactly the rendering and input machinery a visual MMO client needs (click-to-
move, hero silhouettes, cast rings, HP bars, item-shop UI, a minimap) and none of a from-scratch
build's cost. Fork REDGARDEN's *client* — not its combat sim — into GFD as a second, parallel
frontend to the same MUD server: the telnet interface keeps working unchanged, and a REDGARDEN-
shaped GUI client renders the exact same characters, zones, and combat, OSRS-style (click-to-move
third-person, chunky legible geometry, skill-training loop) instead of FPS/instanced-MOBA-style.

---

## 2. What actually exists today, checked directly (not assumed)

**`apps2/mud` (DragonsNShit's real server, `GoblinFoxDragon/apps2/mud/main.go`, 7,310 lines):**
telnet/TCP on `:2323` (`nc localhost 2323` or `telnet localhost 2323`), one Go process, all
server packages wired into a single 1Hz game loop. Real, shipped systems (per this repo's own
CHANGELOG, not the MMO_NORTHSTAR milestone table below — see §2.1 on why those disagree): 22 FFXI
jobs + sub-jobs with combined stats, level/XP to L99, enmity (hate table, AoE cure, overaggro),
death/raise with XP penalty, status effects (Poison/Paralyze/Slow/Silence/Bind/Haste/Regen/
Refresh/Protect/Shell), skillchains + magic bursts (14 resonances, 3 tiers), TP weapon skill
points, auto-attack + mob AI state machine, NM spawn conditions/aggro types/treasure pool
(lot/pass/resolve), crafting guilds (8 types) + HQ synthesis, conquest (3 nations, weekly tick),
parties/alliances/XP chains, linkshell guilds, chat (say/tell/yell/guild), mining skill, home
point, field manuals, IDUNA JWT auth already gating `PacketConnect` (S76-01). Four zones exist
today: Meadow, Hills, Caves, Swampville — each hand-authored as a block of Go inside
`initWorld()` (`docs2/HERO_BRIDGE_PREREQUISITES.md` already named externalizing this into a data
format as the real prerequisite for adding more zones/lore content; unrelated to this doc's own
scope but worth knowing before assuming zones are cheap to add).

**REDGARDEN (`/home/fatbaby/REDGARDEN`, sibling repo, this session's own primary focus):** a
working C99/SDL2 real-time client-server MOBA. `apps/arena` is the relevant piece — a
**shader-based modern-GL client** (`GL/gl.h` + `SDL_GL_GetProcAddress`, deliberately no GLU
dependency, confirmed via `ldd`: no `libGLU` linked at all — this matters because it's the one
REDGARDEN build target that's actually portable without the GLU packaging problems the rest of
this monorepo's SDL2/OpenGL apps have hit repeatedly). It already has, real and working: click-
to-move with an animated target-ring marker; a full hero-silhouette-from-boxes rendering system
(28 heroes, each visually distinct, `draw_hero_model`/`draw_hero_box_facing`); HP bars, HUD,
minimap; a Q/W/R ability-slot system with cast rings, dodgeable skill-shot projectiles, and
ground-AoE zone circles; a 27-item shop panel with pagination and an active-item hotkey; a real
click-to-pick draft/character-select screen; connect-ticket HMAC-SHA256 auth (`packages/common/
hmac_sha256.h`, same scheme as shankpit-460) over a custom UDP wire protocol
(`packages/common/protocol.h`). None of this is placeholder or spec-only — it's the actual client
built and iterated on this whole session.

**The fit:** DragonsNShit has the RPG simulation depth and no real-time visual client; REDGARDEN
has a proven real-time visual client and no RPG simulation depth (its own combat sim,
`arena_game.c`, is a MOBA hero-kit balance engine, not an MMO job/skill system, and isn't what
this doc proposes reusing — see §4). Grafting one onto the other is cheaper than building either
half from scratch a second time.

### 2.1 A stale-doc note, found while writing this

`docs2/MMO_NORTHSTAR.md`'s own milestone table (last updated 2026-06-21) lists milestones 2-8
("IDUNA player identity" through "Season system") as NOT STARTED. That's stale: `apps2/mud` has
shipped a large, real, adjacent-but-different body of MMO systems work since (see the CHANGELOG
list above) under a different milestone numbering (`S76`-`S87`) that MMO_NORTHSTAR's own table
never got updated to reflect. This doc doesn't attempt to reconcile the two milestone tables —
that's its own follow-up, named honestly rather than silently built on top of.

---

## 3. What this is not

Not a live network bridge between two already-running, separately-maintained servers. Not
Minecraft/Bedrock protocol support in REDGARDEN's client, and not a requirement that players run
Minecraft — `apps2/mud` is a telnet server, not a Bedrock server, and Dragonfly's own Bedrock
protocol machinery (the base this repo forks) is not the transport this doc proposes using at
all. Not a replacement for the telnet interface — per founder direction ("cli will continue to
work"), telnet stays a fully-supported first-class client for this server, forever, alongside the
new GUI, the same way a MUD and a rendered client can both be real interfaces to one authoritative
game.

---

## 4. Architecture

```
                    ┌─────────────────────────────────────────────┐
                    │           apps2/mud (Go, :2323 + new)        │
                    │  single authoritative game loop (1Hz tick)   │
                    │                                               │
   telnet/nc  ──────┼──▶ text listener (:2323, UNCHANGED)          │
   (CLI players)     │       │                                      │
                    │       ▼                                      │
                    │  ┌─────────────────────────────────────┐    │
                    │  │   shared internal action dispatch    │    │
                    │  │   (move, attack, cast, craft, chat,  │    │
                    │  │    party, trade, pool/lot/pass...)   │    │
                    │  └─────────────────────────────────────┘    │
                    │       ▲                                      │
   REDGARDEN  ───────┼──▶ new binary listener (:PORT, new)         │
   GUI client         │   forked from packages/common/protocol.h    │
   (this doc)         │   (connect-ticket HMAC auth, UDP snapshot   │
                    │    cadence) — decodes UserCmd-style input    │
                    │    into the SAME dispatch calls telnet uses  │
                    └─────────────────────────────────────────────┘
```

One authoritative Go process, one game-state, two client protocols in front of it. A telnet
player and a REDGARDEN-GUI player can occupy the same zone, fight the same NM, and see each
other's chat — because they're both just clients of the same dispatch layer, not two different
games that happen to share a name.

### 4.1 What forks over from REDGARDEN largely as-is

The **rendering and input machinery**, not the combat sim: `apps/arena/src/main.c`'s SDL2/OpenGL
pipeline (camera orbit/zoom, hero-silhouette-from-boxes system, HUD/HP-bar/minimap text drawing,
the Q/W/R ability-slot UI with cast rings/projectiles/zone circles, the item-shop panel
chrome, the click-to-pick screen shell), `packages/common/protocol.h`'s packet shape and
connect-ticket handshake, `packages/common/hmac_sha256.h` verbatim.

### 4.2 What does not fork over, and gets rewritten against apps2/mud's own systems instead

`arena_game.c` itself — REDGARDEN's hero-kit balance sim (28 heroes' Q/W/R numbers, MOBA-specific
concepts like a fixed 20-slot lobby, a match/draft/winner lifecycle) is not what an MMO character
needs; a MUD character doesn't have a "match end." What replaces it: `apps2/mud`'s own real,
shipped combat/job/skillchain/enmity system stays fully authoritative. REDGARDEN's Q/W/R
slot-and-cast-ring vocabulary becomes the **visual grammar** a job's real abilities render through
— e.g. a Warrior's weapon skill fires the same "cast ring + cooldown sweep" UI REDGARDEN already
draws for Ghost's Q, but the number and effect come from `apps2/mud`'s own weapon-skill system,
not from any REDGARDEN hero's stat block. This is the single most important design call this doc
makes: **REDGARDEN contributes the rendering grammar; DragonsNShit contributes the RPG mechanics
underneath it.** No REDGARDEN hero identity (Ghost, Tyler, Gunnr, etc.) is intended to appear in
DragonsNShit at all — the MUD has its own multiverse-lore hero content pipeline already
(`docs2/HERO_CONTENT_FRAMEWORK.md`, `HERO_BRIDGE_PREREQUISITES.md`), separate from REDGARDEN's own
roster.

### 4.3 The world-rendering gap (the honest biggest unknown)

REDGARDEN's client draws a small, flat, bounded arena (`ARENA_HALF_EXTENT`) with a handful of box
obstacles — it has never rendered real terrain, a multi-zone world, or anything resembling
Meadow/Hills/Caves/Swampville's actual layout. Making REDGARDEN's renderer show a real MMO zone is
new engine work, not a port. The recommended starting scope (see Milestone 4) is deliberately
small: render one zone (Meadow) as a flat plane with box-obstacle placeholders for its real
terrain features, matching REDGARDEN's own established "boxes for now" convention (the same one
every hero silhouette and every piece of jungle terrain in REDGARDEN already uses) rather than
attempting real heightfield/voxel terrain on day one.

---

## 5. What it should feel like to play — "like old school runescape" (founder's own reference)

Third-person, click-to-move (REDGARDEN already has this exactly), chunky/legible low-poly-ish
geometry (REDGARDEN's box-silhouette hero system is stylistically already close to this, not a
mismatch to bridge), a skill-training progression loop (matches `apps2/mud`'s own real 22-job,
L99, mining/crafting/skillchain systems far better than it would ever match a MOBA's match-based
kit-power loop), inventory/equipment panels and a persistent chat log rather than a lobby/draft
flow. Concretely: keep REDGARDEN's camera/click-to-move/HUD-chrome/item-panel conventions:
replace REDGARDEN's draft-pick screen and match/lobby concepts entirely (an MMO character doesn't
draft a hero once per match — it exists persistently, per §4.2).

---

## 6. Milestones

| # | Milestone | Acceptance | Status |
|---|---|---|---|
| 0 | This NORTHSTAR | Written, registered in golden-docs-index, MMO_NORTHSTAR's frontend line updated to point here | DONE |
| 1 | Protocol bridge spike | `apps2/mud` gains a second listener speaking REDGARDEN's connect-ticket handshake + a minimal read-only snapshot packet (own character position/HP/zone + nearby mobs) | NOT STARTED |
| 2 | Fork the client shell | REDGARDEN's `apps/arena` client forked into `GoblinFoxDragon/apps2/` (new app dir); MOBA-specific match/lobby/draft-pick concepts stripped; camera/hero-rendering/HUD/ability-slot-UI machinery kept | NOT STARTED |
| 3 | Input bridge | Click-to-move and Q/W/R slot presses from the forked client decode into `apps2/mud`'s existing action dispatch (the same calls telnet's `move`/`attack`/`cast` commands already make) | NOT STARTED |
| 4 | First real zone rendered | Meadow rendered as a flat plane + box-obstacle placeholders (§4.3's scoped-down terrain approach), not REDGARDEN's fixed arena | NOT STARTED |
| 5 | One job's abilities wired to the slot UI | A single real FFXI job's abilities (proposal: Warrior — the simplest kit) drive REDGARDEN's cast-ring/cooldown/projectile rendering end-to-end, proving the "REDGARDEN renders, MUD simulates" seam on one real, non-placeholder case | NOT STARTED |
| 6 | CLI/GUI coexistence validated | One player on telnet, one on the GUI client, same zone: each sees the other's chat, movement, and combat in real time | NOT STARTED |
| 7 | IDUNA-backed persistent character shared correctly | MMO_NORTHSTAR's own already-speced `characters`/`items`/`guilds` IDUNA schema (§2 of that doc) is the source of truth for both client surfaces — no state exists only on one side | NOT STARTED |

---

## 7. Edit made to `docs2/MMO_NORTHSTAR.md`

That doc's "Integration Architecture" section named the frontend as "C/SDL2 Client (SHANKPIT
runtime, extended)." Updated to point at this doc instead — see that file's own diff for the
exact wording. Every other section of MMO_NORTHSTAR (IDUNA schema, item provenance, guild system,
economy, World Crisis, Telecrystal travel) is unchanged and still the systems-design source of
truth this doc builds on top of.

---

## 8. Open questions, not resolved here

- Which transport for the new listener — UDP (matching REDGARDEN's own current choice, better
  fit for real-time position snapshots) or TCP (matching `apps2/mud`'s existing telnet listener,
  simpler to reuse connection-handling code already in `main.go`)? Leaning UDP for parity with
  REDGARDEN's own proven snapshot cadence, not decided here.
- How does REDGARDEN's real-time HP-delta-driven `attack_flash`/`heal_flash` visual-effect idiom
  (this whole session's own established pattern for "reconstruct a combat event from a state
  delta, no explicit event packet needed") map onto `apps2/mud`'s skillchain/magic-burst system,
  which has much richer real combat-event semantics than a flat HP delta can carry? Probably
  needs the snapshot format to carry a genuine event list, not just position/HP, unlike
  REDGARDEN's own deliberately-minimal wire protocol — a real protocol-design decision, not
  resolved here.
- Zone-authoring format (`HERO_BRIDGE_PREREQUISITES.md`'s own named gap) is a prerequisite for
  rendering *more than* Meadow, but not for Milestone 4's scoped-down single-zone proof.
- Which REDGARDEN systems besides ability-slot UI are worth reusing — the shop-panel chrome maps
  naturally onto DragonsNShit's own crafting/AH systems, but that mapping isn't designed here.

---

## 9. Related docs

| Doc | Location |
|---|---|
| DragonsNShit product systems design (source of truth this doc builds on) | `GoblinFoxDragon/docs2/MMO_NORTHSTAR.md` |
| GFD engine/studio northstar | `GoblinFoxDragon/docs/NORTHSTAR.md` |
| Zone-authoring gap (unrelated prerequisite, worth knowing) | `GoblinFoxDragon/docs2/HERO_BRIDGE_PREREQUISITES.md` |
| Hero/lore content process | `GoblinFoxDragon/docs2/HERO_CONTENT_FRAMEWORK.md` |
| REDGARDEN's own current architecture and client history | `REDGARDEN/NORTHSTAR.md` §3.5 |
| REDGARDEN wire protocol (fork source) | `REDGARDEN/packages/common/protocol.h` |
| REDGARDEN connect-ticket auth (fork source) | `REDGARDEN/packages/common/hmac_sha256.h` |

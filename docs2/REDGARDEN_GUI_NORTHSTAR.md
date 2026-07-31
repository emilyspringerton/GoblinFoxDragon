# REDGARDEN = DragonsNShit's Battlegrounds — Northstar

*Written 2026-07-31. Founder, real-time, verbatim: "can we graft redgarden frontend onto GFD mud
as a gui to make our mmorpg?" → "i dont care how you do it fork redgarden into GFD write the
northstar this is the mmo. this is dragonsnshit" → "cli will continue to work" → "redgarden as a
gui" → "like old school runescape."*

**Status:** Spec only, no code yet — milestone 0 of this doc's own table.

**CORRECTION 2026-07-31 (major, supersedes §§1/4/5/6 below as they originally read):**
founder: *"some of the docs say we arent bringing redgardens gameplay just the ui thats not right
i want dragonsnshit mmo to feel like redgarden like battlegrounds for dragonsnshit is redgarden."*
The original version of this doc got the core call backwards — it said REDGARDEN contributes only
"rendering grammar" and DragonsNShit's own systems replace REDGARDEN's actual gameplay
underneath. Wrong. **REDGARDEN's full gameplay — heroes, abilities, items, node-capture map, the
whole `arena_server` simulation, not just its renderer — ships essentially as-is, as DragonsNShit's
Battlegrounds: an instanced PvP mode reachable from the persistent world**, the same relationship
WoW's Battlegrounds/Arena have to WoW's open world, or FFXI's own self-contained minigames
(Chocobo Racing, Triple Triad) have to FFXI's main game. This is a **simpler** architecture than
the original version of this doc, not a more complex one — REDGARDEN needs no gameplay changes at
all, `arena_server`/`apps/matchmaker`/`apps/arena` work as-is; the only new work is a portal/queue
entry point in the persistent world and reward-crediting back to the player's persistent
character via IDUNA. §§1, 4, 5, and 6 below are rewritten to reflect this. The two-backends
finding (`docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`) is still real and still worth fixing for the
persistent-world layer itself, but REDGARDEN's own bridge no longer waits on it — see §4.

**CORRECTION 2, 2026-07-31 (refines the correction above — read this one last):** founder:
*"like not the same literal game loop maybe but we want to amend our ould systems like
skillchains etc work with redgarden affordances."* The correction above over-corrected into full
decoupling — two rosters, two ability systems, connected only by identity+rewards. Refined: **the
process/loop separation stays** (Battlegrounds is still its own spawned-per-match `arena_server`
process, not merged into `apps2/mud`'s or `apps2/server-go`'s own tick loop) — but the **ability
content** cast through REDGARDEN's own Q/W/R slots is `apps2/mud`'s real job/weapon-skill/
skillchain system, ported into `arena_game.c`'s own slot machinery, not REDGARDEN's fixed
28-hero kit roster left untouched. A Battleground combatant picks a **job** (Warrior, Black Mage,
...), that job's real weapon skills/spells are what's bound to Q/W/R, rendered through
REDGARDEN's own real cast-ring/projectile/zone-circle vocabulary, and skillchain resonance
genuinely triggers between two players' casts — shown with REDGARDEN's own visual language (a
real skillchain flash, not a generic hit). REDGARDEN's real-time tick loop, node-capture map,
item shop, and match structure all stay exactly as built. §§4.1/4.2 and §5 and the milestone
table below are rewritten again to reflect this — this is now the load-bearing version, not the
correction above it.

**Update 2026-07-31 (earlier, now superseded by both corrections above):** wrote a packet-level
bridge spec, `docs2/specs/REDGARDEN_MUD_BRIDGE_SPEC.md`, designing a new listener bolted onto
`apps2/mud` that would translate REDGARDEN input into `apps2/mud`'s own RPG action calls. No
longer the plan — REDGARDEN doesn't need translating into anything; it runs its own combat
directly. Kept for its still-real gap-finding (`apps2/mud` has no continuous movement server-side,
`cmdAttack`/`cmdGo` details), not deleted.

**Relationship to `docs2/MMO_NORTHSTAR.md`:** amends it, doesn't replace it. Every system that
doc already specs — IDUNA-backed character/item/guild schema, item provenance chains, the
economy, World Crisis phases, Telecrystal scene travel, EduScript VM scripting — stays exactly
as designed and governs the *persistent world* half of the product. This doc adds the other half:
**REDGARDEN, wholesale, is the Battlegrounds half.** MMO_NORTHSTAR's own "Integration
Architecture" diagram named "C/SDL2 Client (SHANKPIT runtime, extended)" as the frontend; that
line is superseded by this doc. See §7 for the exact edit made to that file.

---

## 1. Three-sentence version

DragonsNShit's persistent world (`apps2/mud`'s real, deep FFXI-parity RPG systems — 22 jobs,
skillchains, enmity, conquest, crafting guilds, all live today, telnet-only, `:2323`) is one half
of the product; REDGARDEN's real-time combat framework — its own `arena_server`/`apps/matchmaker`
process, click-to-move, Q/W/R ability-slot UI with cast rings, dodgeable projectiles, ground-AoE
zone circles, item shop, and node-capture map — becomes **DragonsNShit's Battlegrounds**: an
instanced PvP mode a persistent-world character queues into, the same relationship WoW
Battlegrounds or FFXI's own self-contained minigames have to their respective main games, spawned
as its own separate process per match, same as REDGARDEN already does today — not merged into the
persistent world's own game loop. What's different from REDGARDEN's own current build: the
abilities cast through those Q/W/R slots are `apps2/mud`'s real job weapon-skills and spells
(Warrior, Black Mage, ...) ported into REDGARDEN's slot machinery, with real skillchain resonance
between players' casts, rendered through REDGARDEN's own visual language — not REDGARDEN's
existing 28 fixed hero kits left untouched, and not a from-scratch reinvention of `apps2/mud`'s
own real skillchain math either. The persistent world stays OSRS-flavored (click-to-move, chunky
legible geometry, skill-training loop, per the founder's own reference); Battlegrounds stays
REDGARDEN's own real-time feel, carrying real DragonsNShit mechanics through it — two distinct
feels on purpose, connected by shared systems where it counts, the same way WoW's own overworld
and Battlegrounds don't pretend to be the same game mode but still share the same class kit.

---

## 2. What actually exists today, checked directly (not assumed)

**`apps2/mud` (DragonsNShit's persistent-world server, `GoblinFoxDragon/apps2/mud/main.go`,
7,310 lines):** telnet/TCP on `:2323` (`nc localhost 2323` or `telnet localhost 2323`), one Go
process, all server packages wired into a single 1Hz game loop. Real, shipped systems (per this
repo's own CHANGELOG, not the MMO_NORTHSTAR milestone table below — see §2.1 on why those
disagree): 22 FFXI jobs + sub-jobs with combined stats, level/XP to L99, enmity (hate table, AoE
cure, overaggro), death/raise with XP penalty, status effects (Poison/Paralyze/Slow/Silence/Bind/
Haste/Regen/Refresh/Protect/Shell), skillchains + magic bursts (14 resonances, 3 tiers), TP
weapon skill points, auto-attack + mob AI state machine, NM spawn conditions/aggro types/treasure
pool (lot/pass/resolve), crafting guilds (8 types) + HQ synthesis, conquest (3 nations, weekly
tick), parties/alliances/XP chains, linkshell guilds, chat (say/tell/yell/guild), mining skill,
home point, field manuals, IDUNA JWT auth already gating `PacketConnect` (S76-01, though its own
`idunaclient` field is otherwise dead — see `DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`). Four zones
exist today: Meadow, Hills, Caves, Swampville — each hand-authored as a block of Go inside
`initWorld()` (`docs2/HERO_BRIDGE_PREREQUISITES.md` already named externalizing this into a data
format as the real prerequisite for adding more zones/lore content; unrelated to this doc's own
scope but worth knowing before assuming zones are cheap to add).

**`apps2/server-go` (found 2026-07-31, `DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`):** a second,
separate real backend — UDP on `:6969`, real IDUNA-JWT-authenticated `PacketConnect`, real
IDUNA-backed Telecrystal travel/crafting/skill-XP, FPS-shaped (SHANKPIT-derived) hitscan combat.
Not this doc's concern directly — it's part of the persistent-world side's own internal split,
not something REDGARDEN's Battlegrounds bridge needs to touch. See that doc for the full finding.

**REDGARDEN (`/home/fatbaby/REDGARDEN`, sibling repo, this session's own primary focus):** a
complete, working C99/SDL2 real-time client-server MOBA — not just a client, the whole stack.
`apps/arena_server` is the authoritative combat simulation (28 heroes' real kits — Q/W/R
abilities, dodgeable skill-shot projectiles, ground-AoE zone abilities, towers gating node
capture, a 27-item shop economy, real draft/pick phase); `apps/matchmaker` pairs queued clients
and spawns a dedicated `arena_server` per match (`fork`+`exec`, one match per process); `apps/
arena` is the **shader-based modern-GL client** (`GL/gl.h` + `SDL_GL_GetProcAddress`, no GLU
dependency — confirmed via `ldd`, the one build target in this monorepo that's actually portable
without the GLU packaging problems every other SDL2/OpenGL app here has hit) with real click-to-
move, hero-silhouette rendering (`draw_hero_model`/`draw_hero_box_facing`), HP bars/HUD/minimap,
the Q/W/R ability-slot UI with cast rings, a shop panel with pagination, and the draft/pick
screen; connect-ticket HMAC-SHA256 auth (`packages/common/hmac_sha256.h`, same scheme as
shankpit-460) over `packages/common/protocol.h`'s UDP wire protocol. None of this is placeholder
— it's the actual, complete system built and iterated on this whole session, and **all of it
ships to Battlegrounds unchanged.**

**The fit:** two complete, working systems that don't need to be merged into one — they need a
door between them. REDGARDEN doesn't need DragonsNShit's RPG depth (it has its own real
combat/economy/progression, scoped to a match); DragonsNShit's persistent world doesn't need
REDGARDEN's combat model grafted into its own job/skillchain system (that would flatten
`apps2/mud`'s real depth into something worse than either system alone). The door is IDUNA:
shared player identity, and a reward-crediting call after a Battleground match ends, same shape
`arena_server`'s own `report_match_result`/WOTAN reporting already is.

### 2.1 A stale-doc note, found while writing this

`docs2/MMO_NORTHSTAR.md`'s own milestone table (last updated 2026-06-21) lists milestones 2-8
("IDUNA player identity" through "Season system") as NOT STARTED. That's stale: `apps2/mud` has
shipped a large, real, adjacent-but-different body of MMO systems work since (see the CHANGELOG
list above) under a different milestone numbering (`S76`-`S87`) that MMO_NORTHSTAR's own table
never got updated to reflect. This doc doesn't attempt to reconcile the two milestone tables —
that's its own follow-up, named honestly rather than silently built on top of.

---

## 3. What this is not

Not a live network bridge translating REDGARDEN's input into DragonsNShit's own RPG action calls
(the original version of this doc's plan) — REDGARDEN's own simulation stays authoritative for
Battlegrounds matches, full stop. Not Minecraft/Bedrock protocol support in REDGARDEN's client,
and not a requirement that players run Minecraft. Not a replacement for the telnet interface —
per founder direction ("cli will continue to work"), telnet stays a fully-supported first-class
way to play the persistent world, forever; Battlegrounds is an additional mode reachable from it,
not a replacement for it. Not a merge of REDGARDEN's hero roster into `apps2/mud`'s own
multiverse-lore hero content pipeline (`HERO_CONTENT_FRAMEWORK.md`) — those stay two separate
rosters serving two separate modes, same as a persistent-world MMO character and their
Battleground loadout are conventionally decoupled identities in the genre this doc is modeling.

---

## 4. Architecture

```
DragonsNShit persistent world (apps2/mud, telnet :2323 — unchanged)
  │
  │  player reaches a Battlegrounds entry point (portal / queue NPC / command —
  │  exact UX not designed here, matches SHANKPIT's own portal_resolve_destination
  │  precedent, THE_BRIDGE_SPEC.md)
  ▼
IDUNA (:8080) — shared identity layer
  │  mints a REDGARDEN connect-ticket for this player's own IDUNA identity,
  │  same HMAC scheme REDGARDEN/shankpit-460 already use
  ▼
REDGARDEN's own real, unchanged stack
  apps/matchmaker  →  spawns a dedicated apps/arena_server per match (existing behavior)
  apps/arena client →  connects with the minted ticket, plays a real, unchanged REDGARDEN match
  │
  │  match ends — arena_server's own report_match_result / WOTAN reporting,
  │  extended to also credit the player's persistent DragonsNShit character
  ▼
IDUNA — credits rewards (gil, faction/conquest points, cosmetics — not designed here) to the
        persistent character row
```

Two complete, separately-authoritative systems, connected only at the identity/reward seam. No
shared game loop, no packet-translation layer, no combat-system unification — the thing the
original version of this doc spent most of its length designing turns out not to be needed at
all once Battlegrounds is the right frame.

### 4.1 What forks over from REDGARDEN unchanged

The **process/loop, the real-time framework, and the affordances**: `apps/arena_server`'s tick
loop and match structure, `apps/matchmaker`'s spawn-per-match pattern, `apps/arena`'s rendering
(camera, hero-silhouette system, HUD/HP bars/minimap), the Q/W/R ability-slot UI with its cast
rings, the dodgeable-projectile and ground-AoE-zone-circle rendering, the item-shop panel, the
node-capture map/objective structure, `packages/common/protocol.h`'s packet shape, `packages/
common/hmac_sha256.h`'s connect-ticket auth. Battlegrounds is still its own separately-spawned
process per match — **not** merged into `apps2/mud`'s or `apps2/server-go`'s own 1Hz/UDP loop
(founder: "not the same literal game loop maybe").

### 4.2 What gets amended: the ability content itself

**Correction 2's actual content, not just a process boundary**: the specific Q/W/R abilities a
Battleground combatant casts are not REDGARDEN's own 28 fixed hero kits left untouched —
`apps2/mud`'s real job/weapon-skill/skillchain system (§2's own inventory: 22 jobs, TP weapon
skills, 14 skillchain resonances across 3 tiers, magic bursts) is ported into `arena_game.c`'s
own ability-slot machinery as new content, replacing what a "hero" means in this mode: you pick a
**job**, not a REDGARDEN hero, and that job's real abilities render through REDGARDEN's existing
cast-ring/projectile/zone-circle vocabulary — a Warrior's real weapon skill fires the same visual
language Ghost's Q already uses, but the number, cooldown, and TP cost come from `apps2/mud`'s
own real weapon-skill system, not a REDGARDEN hero stat block. Skillchains are the one genuinely
new mechanic this requires in `arena_game.c`: two players' ability casts on the same target within
a resonance window need to detect and score a chain, same math `apps2/mud`'s own
`skillchain.Chain` already implements — ported, not reinvented — and rendered as a real, distinct
visual event (not folded into the generic `attack_flash` every other REDGARDEN hit already uses).
REDGARDEN's own 28-hero roster and `apps2/mud`'s own multiverse-lore hero content pipeline
(`HERO_CONTENT_FRAMEWORK.md`) stay separate from this — this doc isn't claiming Ghost or Tyler
become jobs, or that a job becomes a REDGARDEN hero; it's the *ability system* that ports, not
either roster.

### 4.3 The Battlegrounds-entry UX (not designed here)

Portal, NPC, queue command, or something else — `apps2/mud`'s own zone-transfer mechanism
(`cmdGo`) is the closest existing precedent (a discrete transition point, not continuous
movement), but the exact player-facing flow for "leave the persistent world, enter a
Battleground" isn't designed in this pass.

---

## 5. What each mode should feel like to play

**Persistent world** — "like old school runescape" (founder's own reference): third-person,
click-to-move, chunky/legible low-poly-ish geometry, a skill-training progression loop
(`apps2/mud`'s own real 22-job/L99/mining/crafting/skillchain depth already matches this far
better than a match-based kit-power loop would), inventory/equipment panels, a persistent chat
log. This is `apps2/mud`'s (eventually `apps2/server-go`-unified, per the two-backends audit)
own domain, not REDGARDEN's.

**Battlegrounds** — REDGARDEN's exact real-time feel: click-to-move, buy items, cast Q/W/R, dodge
skill-shots, fight over nodes, win or lose a real match, chain skillchains for bonus damage — but
what you pick going in is a **job** (Warrior, Black Mage, ...), not one of REDGARDEN's 28 named
heroes, and what fires through those Q/W/R slots is that job's real weapon skills and spells
(§4.2). No OSRS-ification of this half — the founder's own words are the spec here: *"i want
dragonsnshit mmo to feel like redgarden... battlegrounds for dragonsnshit is redgarden"* — and
*"we want our old systems like skillchains etc [to] work with redgarden affordances."*

---

## 6. Milestones

| # | Milestone | Acceptance | Status |
|---|---|---|---|
| 0 | This NORTHSTAR | Written, registered in golden-docs-index, MMO_NORTHSTAR's frontend line updated to point here | DONE |
| 0.5 | Two-backends audit | Found `apps2/server-go` alongside `apps2/mud` — real for the persistent-world layer, decoupled from REDGARDEN's own bridge (see §4) — `docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md` | DONE |
| 0.75 | Battlegrounds correction | Found REDGARDEN's own full gameplay, not just its renderer, is the right thing to ship — this doc rewritten §§1/4/5/6 | DONE |
| 0.8 | Job/skillchain-affordance correction | Refined 0.75: process stays separate, but ability *content* is `apps2/mud`'s real job/skillchain system ported into `arena_game.c`'s slot machinery, not REDGARDEN's fixed hero kits untouched — §§4.1/4.2/5 rewritten again | DONE |
| 1 | One job ported as real REDGARDEN ability content | Warrior (proposal: simplest real kit in `apps2/mud`'s job system) — its real weapon skills wired into `arena_game.c`'s Q/W/R slots, rendered through REDGARDEN's existing cast-ring/projectile UI, numbers/cooldowns/TP cost sourced from `apps2/mud`'s own `skillchain`/job packages, not invented | NOT STARTED |
| 2 | Skillchain resonance in `arena_game.c` | Port `apps2/mud`'s own `skillchain.Chain` detection/scoring into REDGARDEN's tick loop; new, distinct visual event (not folded into the generic `attack_flash`) when two players' casts chain | NOT STARTED |
| 3 | Entry-point hook | Persistent world gains a Battlegrounds entry point (portal/NPC/command, §4.3) that mints a real REDGARDEN connect-ticket via IDUNA for the player's own identity, lets them pick a job (not a REDGARDEN hero), and hands off to `apps/matchmaker` | NOT STARTED |
| 4 | Reward-credit hook | `arena_server`'s match-end reporting extended to also credit the player's persistent DragonsNShit character via IDUNA (gil/faction points/cosmetics — exact reward shape not designed here) | NOT STARTED |
| 5 | End-to-end validated | A real persistent-world character queues, picks Warrior, plays a real match casting real weapon skills through REDGARDEN's UI, chains a real skillchain, and returns to the persistent world with a real credited reward | NOT STARTED |
| 6 | (Optional, separate track) Persistent-world backend unification | `docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`'s own recommendation — unify `apps2/mud`'s RPG logic into `apps2/server-go`'s loop. Valuable for the persistent-world half on its own merits; not a blocker for Milestones 1-5 above | NOT STARTED |

---

## 7. Edit made to `docs2/MMO_NORTHSTAR.md`

That doc's "Integration Architecture" section named the frontend as "C/SDL2 Client (SHANKPIT
runtime, extended)." Updated to point at this doc instead — see that file's own diff for the
exact wording. Every other section of MMO_NORTHSTAR (IDUNA schema, item provenance, guild system,
economy, World Crisis, Telecrystal travel) is unchanged and still the systems-design source of
truth for the persistent-world half of the product.

---

## 8. Open questions, not resolved here

- Exact Battlegrounds-entry UX (§4.3) — portal, NPC, or command.
- Exact reward shape credited back to the persistent character (gil? faction/conquest points?
  cosmetic unlocks tied to Battleground performance?) — a real design pass, not named here.
- Does Battleground participation ever need to be gated by persistent-world state (level
  minimums, faction standing, an unlock quest), or is it open to any character from the start?
  Founder call, not a technical question.
- Whether `apps2/mud`'s telnet players and REDGARDEN's Battleground players are ever meant to see
  each other's *presence* (e.g. a persistent-world "so-and-so just won a Battleground" broadcast)
  — a nice-to-have social hook, not designed here.
- `docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`'s own unification recommendation (Milestone 4 above)
  — real, valuable, and entirely independent of this doc's own critical path now.

---

## 9. Related docs

| Doc | Location |
|---|---|
| DragonsNShit product systems design (persistent-world source of truth) | `GoblinFoxDragon/docs2/MMO_NORTHSTAR.md` |
| GFD engine/studio northstar | `GoblinFoxDragon/docs/NORTHSTAR.md` |
| Two-backends audit (persistent-world internal split, decoupled from this doc's critical path) | `GoblinFoxDragon/docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md` |
| Original packet-bridge design (superseded by the Battlegrounds correction above, kept for its gap-finding) | `GoblinFoxDragon/docs2/specs/REDGARDEN_MUD_BRIDGE_SPEC.md` |
| Zone-authoring gap (unrelated prerequisite, worth knowing) | `GoblinFoxDragon/docs2/HERO_BRIDGE_PREREQUISITES.md` |
| Hero/lore content process (persistent-world's own separate roster) | `GoblinFoxDragon/docs2/HERO_CONTENT_FRAMEWORK.md` |
| REDGARDEN's own current architecture and match/matchmaker history | `REDGARDEN/NORTHSTAR.md` §3.5, §13 |
| REDGARDEN wire protocol (used as-is) | `REDGARDEN/packages/common/protocol.h` |
| REDGARDEN connect-ticket auth (used as-is) | `REDGARDEN/packages/common/hmac_sha256.h` |

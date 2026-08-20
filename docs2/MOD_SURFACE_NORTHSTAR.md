# NORTHSTAR — GFD Mod Surface (Vintage Story / WC3 World Editor reference)

**Status:** Draft v0.1 — scoping only, no implementation.
**Date:** 2026-08-20.

**Founder framing, verbatim, across several fragments this session:** "develop a mod surface
first" → "expand it naturally for the domains we are using like gta7 and the GFD core rpg which
is actually human verified" → "when designing the mod surface for gfd graphics engine edu edition
with virtual machine make sure to reference vintage story in terms of their mod surface and the og
frozen throne whatever the wc3 map editor let you do — think of redgarden almost as a map editor
that is a game that doesn't have the map editing built into it" → "all of the features should use
our new mod surface" → "lua or whatever the heck" → "i invented a language let me see if i can
find it" → "especially the fps edu edition." Followed by an explicit standing rule for this repo:
**"game expansions need to be mod api first then add the feature — that is the new MO for GFD."**
Every item below (solid buildings, destructible environments, skate-culture tech, faction hooks,
shader accessibility toggle) is scoped as a mod-surface consumer, not a bolt-on, per that rule.

---

## 1. What already exists — checked directly, not assumed

The founder's instinct that "the virtual machine" already exists is correct. It's the **EduScript
VM** (`packages/education/`: `edu_lexer.c/h` → `edu_parser.c/h` → `edu_bytecode.c/h` →
`edu_vm.c/h`), fully scoped in `docs2/EDUCATION_CURRICULUM_NORTHSTAR.md` (2026-07-23) and already
live — not a proposal. A real stack machine (`int stack[256]`, `int vars[64]`, 17 opcodes covering
`let`/`if`/`while`), bound to real physical world objects via `edu_bindings.h`'s `EduWorldState`
(crates, gates, bridges, portals, switches, up to 4 marked enemies) through 17 callable builtins
(`set_switch`, `move_crate`, `open_gate`, `raise_bridge`, `stabilize_portal`, `mark_enemy`,
`spawn_prop`, `set_entity_pos`, etc.). It runs today in `apps/lobby/src/main.c` as the "Architect's
Orb" terminal (F6 toggle / F7 compile / F8 run / F9 reset, 8 script slots) — a real,
playable, compile-and-run puzzle loop, not a mockup.

**The real gap this doc has to close is not "build a VM" — it's "the VM only reaches one of
GFD's two clients."** `apps2/battlegrounds_gui` (the FPS client — REDGARDEN's `apps/arena` forked
at commit `61baafb`, C99/SDL2, modern-GL) has **zero plugin/mod/script-loading mechanism** —
confirmed directly, no hits beyond unrelated bot-AI inference files. That's the client the
founder specifically flagged as the priority ("especially the fps edu edition"), and it's the one
every new feature below (solid buildings, destructibility, skate tech) targets first.

**EduScript's own real limit, already flagged in its own northstar (§3):** no arrays, no
user-defined functions, no recursion. That's a real prerequisite to fix before EduScript can carry
anything richer than the current single Architect Trial puzzle — flagged there, still true here,
not re-solved in this doc.

## 2. Reference points, applied honestly rather than name-dropped

**Warcraft III's World Editor / Frozen Throne trigger system** is a GUI trigger-action editor
(events → conditions → actions) that compiles down to JASS, a real scripting language, under the
hood. EduScript's existing binding pattern — world objects (gates, bridges, switches) exposed as
callable builtins from a small script language — is structurally the same idea already, just
without JASS's GUI-trigger front end and without WC3's breadth (WC3 triggers can touch nearly
anything in the map: unit creation, AI, dialogue, camera, game state). **EduScript today is a
narrow slice of what WC3 triggers do** — puzzle-object manipulation only, nothing about spawning
units, changing terrain, or driving NPC behavior. Closing that gap (more binding categories, not
a different paradigm) is the real work, not switching engines.

**Vintage Story's mod API** is a much deeper reference point than WC3's: mods are full C#
assemblies (or simpler JSON-patch mods for content-only changes) that can define new block types
and their behaviors, entity behaviors, world-generation rules, and crafting recipes — systemic
hooks into the simulation itself, not just scripted one-off puzzle logic. This is the real bar
"mod API first" is aiming at: **building solidity, destructibility, and skate-surface behavior as
data-driven properties on block/prop types that a script or config can define, not as hardcoded
if-branches in engine C code.** That's the concrete, checkable difference between "mod-API-first"
and "feature-first-with-a-mod-surface-bolted-on."

**The founder's own framing — REDGARDEN as "almost a map editor that is a game that doesn't have
the map editing built into it"** — reads as: the simulation/rendering engine underneath
(`apps2/battlegrounds_gui`, forked from `apps/arena`) is already real and functional; what it's
missing is the authoring layer WC3 shipped alongside its actual game (the World Editor itself),
not a different game engine. EduScript, ported and deepened, is that authoring layer's real
starting point — it's a scripting console without a world-editing UI around it yet.

## 3. Scripting language — genuinely open, not decided here

The founder floated "lua or whatever the heck" and separately mentioned having invented a custom
language and needing to locate it. **Not resolved in this doc.** EduScript itself is a real
incumbent worth weighing against both: it's already integrated, already bound to world objects,
and already running in production in `apps/lobby` — the cost of extending it (arrays, functions)
is likely much lower than integrating Lua fresh or standing up a from-scratch custom language,
but "lower cost" isn't the same as "founder's actual intent," especially if the invented language
is meant to be a real product feature (a differentiator), not just an implementation detail. Real
next step: the founder locates the invented-language reference, then a real three-way comparison
(EduScript-extended vs. Lua vs. the invented language) happens before committing — not guessed at
here.

## 4. Concrete next features, all mod-API-first per the new GFD MO

- **Solid buildings (~80% collidable, GTA3-style):** should be a per-prop-type flag
  (`solid: true/false`) read by the mod surface's binding layer, not a hardcoded collision
  special-case — the ~20% non-solid carve-out (doors, breakable/thin structures, decorative
  clutter) is itself mod-surface-expressible data, not an engine exception list.
- **Destructible-environments engine:** real, existing prior art for the *design intent* — not
  the code — lives in a separate repo, `/home/fatbaby/skateboard/NORTHSTAR.md` (lowercase
  `skateboard`, not the empty `SKATEBOARD` — confirmed via direct read), which independently
  specs "R6 Siege-style non-voxel destructible geometry": authored mesh damage/reveal states, not
  a voxel system, where breaching a wall opens new lines of sight and movement at once. **Given
  GFD's destructible-environments ask and the skate repo's own destructibility spec describe the
  same underlying mechanic for two different games, they should share one real system, not become
  two independent implementations** — flagged here as a real recommendation, not yet built or
  confirmed with the founder as final.
- **Skate-culture tech, pulled from `/home/fatbaby/skateboard/NORTHSTAR.md`:** that doc's real
  core concept — "the city itself is the skatepark," curbs/rails/gaps/rooftops doing double duty
  as both terrain and movement lines, set in the same TrapX-universe tone GTA7 already uses on
  Minecraft — is itself a mod-surface concern: which surfaces are skate-grindable/ollie-able
  should be a per-surface-type property, same pattern as the solidity flag above, not a
  hardcoded list of specific meshes.
- **Faction hooks baked into the APIs:** TRAPX's real Faction Reputation system
  (`SHANKPIT/docs2/TRAPX_NORTHSTAR.md`, and GTA7's own live `FactionManager`) is the closest
  existing mechanical precedent (see `REDGARDEN/NORTHSTAR.md` §29's addendum, 2026-08-20, for the
  same cross-reference made for ECOWAR) — GFD's own faction hooks should expose the same shape of
  binding (join/reputation/rank-gated effects) through the mod surface rather than a bespoke
  GFD-only faction system.
- **Shader accessibility toggle:** an on/off switch for any shader-driven visual effect
  (matching S181-01's cel-shading pass in REDGARDEN's own `apps/arena`, GFD's direct upstream)
  should be a real client setting, not tied to the mod surface itself — flagged here for
  completeness since the founder raised it in the same breath as the graphics work, but it's a
  client-settings concern, not a modding concern.

## 4a. METALVERSE terminal mode — ticker charts + news feeds (2026-08-20 addendum)

Founder, real-time, after settling on **METALVERSE** as the name for `apps2/battlegrounds_gui`
specifically (not an abstract repo concept — see §5's own note on that naming thread): "can we
build a mode into metalverse with heavy terminal integration that lets you spawn ticker stock
charts and news feeds in a similar way to /gta7tv spawns screens in gta7."

**Real, already-built data source to back this — checked directly, not assumed:** FatBaby's
`signalapi` (`PRRJECT_FATBABY/internal/apiserver/server.go`, real HTTP endpoints, already live on
`:9091`) already exposes exactly the data this feature needs: `/v1/movers-history/{ticker}` (real
historical price data — the actual backing for a "ticker stock chart"), `/v1/entities/{ticker}`
and `/v1/press-releases/` (real company/news data — the backing for a "news feed"),
`/v1/signals` and `/v1/governance-signals`. Nothing needs inventing on the data side; this is a
real client integration against a real existing API, not new data infrastructure.

**Real, already-built rendering precedent to build from:** GTA7's `/gta7tv` (a real Minecraft
`TextDisplay` entity, Paper-plugin-specific) doesn't port directly — `apps2/battlegrounds_gui` is
a custom SDL2/OpenGL engine, not Minecraft. The two closer precedents actually inside this engine
family: (1) the EduScript VM's "Architect's Orb" terminal in `apps/lobby` (F6 toggle / F7 compile
/ F8 run / F9 reset — real, working, in-world terminal UI, the closest existing analog to "heavy
terminal integration"), and (2) `apps2/battlegrounds_gui`'s own existing 2D HUD pass (the same
immediate-mode GL layer this doc's WASM work already characterized in
`apps2/battlegrounds_gui/wasm/README.md`). A "spawn a screen" mode most naturally means a
world-anchored renderable panel (a textured quad in 3D space, not just a 2D HUD overlay) that can
be populated with either live chart data (rendered client-side from `movers-history` JSON) or news
text (from `press-releases`) — a real, new rendering primitive this engine doesn't have yet, not
a reskin of something that exists.

**Scope call, consistent with the rest of this doc:** per the founder's own new standing MO ("game
expansions need to be mod api first then add the feature"), this should be a mod-surface-defined
panel type (spawn-a-typed-panel, panel type + data source as script/config-driven properties) once
the mod surface itself exists — not a hardcoded feature bolted directly onto `main.c`. Blocked on
the same open question as everything else in §4: the scripting-language decision (§3). Not started
here — data source and rendering precedent identified and grounded, implementation not begun.

## 5. A separate, bigger open question — flagged, not decided here

The founder also floated, in the same real-time stretch, unifying both GFD clients (`apps/lobby`
and `apps2/battlegrounds_gui`) under **PITVIPER** (the SDL2 terminal emulator repo with Emily
Prime integration hooks) — and beyond that, PITVIPER potentially becoming a genuine **"multiverse
portal"**: a single launcher/hub across SHANKPIT, REDGARDEN, GFD, BRAWLPIT, and whatever else, not
just a home for GFD's own two clients. This is a real, much bigger cross-repo architecture
decision than the mod-surface scope in this doc — logged via `emily observe` per Principle 18,
not resolved or scoped here. Worth its own northstar pass once the founder has more to say about
it (the doc also referenced "the osaka garage" without enough context to interpret confidently —
logged verbatim rather than guessed at).

## 6. Status

Scoping only. No milestone plan, no code. Real next steps, in order: (1) founder resolves the
scripting-language question (§3); (2) EduScript's arrays/functions gap gets closed if EduScript is
the chosen path; (3) EduScript's binding layer gets ported to `apps2/battlegrounds_gui` (currently
zero mod surface there); (4) the shared destructible-geometry system between GFD and
`/home/fatbaby/skateboard` gets confirmed with the founder before either side starts building its
own.

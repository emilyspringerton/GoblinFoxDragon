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

## 3. Scripting language — RESOLVED 2026-08-24, two-VM model

Originally left open (see history below, kept for the real reasoning trail): the founder floated
"lua or whatever the heck" and separately mentioned having invented a custom language (PARENA,
since located — see `PARENA/NORTHSTAR.md`) and needing to locate it. **Resolved, founder
real-time, 2026-08-24**, after a deferred discussion earlier the same session on how PARENA could
"turbocharge" EduScript: **don't extend EduScript's own opcode table.** EduScript stays exactly
the small, easy-to-learn, no-arrays/no-functions/no-recursion 17-op VM it already is — that
minimalism is the actual feature, not a gap to close. A **second VM** — PARENA, which compiles
ahead-of-time to real C rather than interpreting toy bytecode — handles anything needing real
data structures, recursion, or genuine instruction-level hacking (the founder's own stated
end-goal: mod-surface "instruction hacks" that transfer to real ARM/x86 architectures, which a
17-op puzzle VM can never resemble but PARENA's real C/assembly output can). The two VMs connect
via **federated process operation**, not in-process embedding/linking — see §3a below for the
concrete shape.

**Real, checkable consequence of this decision**: the "extend EduScript's arrays/functions gap"
next-step named in the original version of this section (and echoed in
`EDUCATION_CURRICULUM_NORTHSTAR.md`) is **superseded** — EduScript's own opcode set is not the
thing that grows. Advanced mod-surface capability grows on the PARENA side, reached from EduScript
scripts via the federated-process interop once that layer exists. `EDUCATION_CURRICULUM_NORTHSTAR
.md`'s own "Phase 0: array opcode + user-defined functions" plan should be re-read against this
resolution before anyone picks it up — it may no longer be the right next increment for EduScript
itself; flagged here, not edited there, since that doc's own phase plan wasn't re-scoped in this
same pass.

**Old, superseded framing kept for history**: "EduScript itself is a real incumbent worth weighing
against both [Lua and PARENA]: the cost of extending it (arrays, functions) is likely much lower
than integrating Lua fresh or standing up a from-scratch custom language... a real three-way
comparison (EduScript-extended vs. Lua vs. the invented language) happens before committing." That
three-way comparison didn't end up being how the decision actually got made — the founder decided
directly, real-time, once PARENA's own real existence and status were established this session,
rather than running the formal comparison this section originally proposed. Recorded so a future
reader doesn't wonder why the planned comparison never appears elsewhere.

## 3a. Federated process operation — the EduScript↔PARENA interop shape (2026-08-24)

**North star, founder real-time, verbatim: "north star is full erlang scheduler system."** The
concrete model for how EduScript (small, sandboxed, in-process puzzle VM) and PARENA (native-
compiled, real-instruction-hacking VM) talk to each other is **not** a function-call-style
embedding (PARENA linked into the game process, called like a library) — it's **federated
processes**, styled on Erlang/BEAM: independent, isolated processes/schedulable-units that
communicate by message-passing, with the supervision/fault-isolation properties BEAM is actually
known for (a crashing PARENA-side process doesn't take the game process down with it — "let it
crash," contained, restarted, not a shared-fault-domain embed).

**Why this shape, not embedding, reasoned from the founder's own stated goals:**
- EduScript's whole value is being small, safe, and puzzle-scoped — sandboxing that by running it
  in-process next to arbitrary PARENA-compiled native code (which is explicitly NOT sandboxed —
  the entire point is instruction-level ARM/x86-transferable hacking) would undermine exactly the
  property EduScript is being kept around for.
- "Federated" + Erlang-scheduler framing implies each VM instance is its own schedulable unit,
  not a shared address space — matches BEAM's actual process model (green threads, no shared
  memory, mailbox message-passing) more than a traditional OS-process IPC pipe, though the two
  aren't mutually exclusive (BEAM's own distribution protocol runs process-to-process messaging
  over real OS sockets between real OS processes/nodes too).

**Real, honest status: north star only, nothing scoped or built.** Real open questions a future
scoping pass needs to answer, not guessed at here:
1. Does "federated process operation" mean literally reimplementing a BEAM-style preemptive
   green-thread scheduler (a substantial VM-runtime project on its own, independent of both
   EduScript and PARENA), or does it mean real OS processes + a message-passing IPC protocol
   between them (lighter-weight, faster to reach, less ambitious than "full Erlang scheduler")?
   The founder's own "north star is full erlang" phrasing suggests the former is the eventual
   target even if the latter is the real near-term stepping stone — not resolved here which one
   is the actual next build increment.
2. What's the message/mailbox contract between an EduScript script and a federated PARENA
   process — what triggers a PARENA process to spawn, what data crosses the boundary, what a
   crash/timeout/restart looks like from the EduScript script's own point of view?
3. Where does this scheduler layer actually live — a new package in this repo, in PARENA itself
   (a real runtime component, not just a compiler), or a separate new repo? Not decided.
4. Relationship to PARENA's own already-stated multi-target ambition (C/JVM/TS/Wasm, per
   `PARENA/NORTHSTAR.md`) — a federated PARENA process is presumably always the C target
   (native, ARM/x86-adjacent, matching the instruction-hacking goal), not the others; not
   confirmed with the founder.

**Connected idea, same real-time stretch, not yet scoped to a specific game/repo:** founder —
"our game will have a phone system built on the same resiliency patterns as the real world." This
lines up directly with Erlang/OTP's own real origin (Ericsson telecom switching — supervision
trees, let-it-crash, hot code reload were built FOR phone-system reliability specifically, not a
metaphor borrowed after the fact), so an in-game phone/comms mechanic riding on this same
federated-process scheduler would be using the pattern for exactly what it was invented for, not
just borrowing its name. Which game/repo this belongs to (GFD? SHANKPIT? REDGARDEN? a new one?)
wasn't specified — logged here as a connected open question, not scoped further.

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

## 7. GFD-MACRO-0012 — real, first PARENA mod slice shipped (2026-09-04)

**This section supersedes §6's "currently zero mod surface" claim for one narrow, real slice.**
Founder, kanban: "GFD macro system make sure we tie the action frame stuff into a parena mod
based shape so we can easily allow for extension at the action bar affordance" — a direct
continuation of GFD-AF-01939's just-shipped 6-slot ability bar.

**Real, deliberate choice: NOT §3a's federated EduScript↔PARENA process model.** That model is
still "north star only, nothing scoped or built" (4 real open questions, no message contract, no
host location decided) — standing it up just to give the action bar a mod hook would have been a
substantial, unscoped detour from a much smaller real ask. Instead, this reuses the ALREADY LIVE,
proven-in-production pattern `ECOWAR/docs/ARENA_API.md` documents for 8 real mods: a PARENA module
compiled once via the real `parena build` CLI, its generated `.c` committed, called by its emitted
C name directly from host code — no dlopen/dlsym runtime loading (PAPERCRAFT's heavier
`--mods-manifest` pattern), since GFD's release model (native + WASM, both statically linked at
build time) doesn't need live-reload.

**Shipped**: `PARENA/stdlib/gfd/action_bar_mod.prn` → `on-gfd-ability-for-slot`, real decision
logic (a job×slot branch, not a bare trigger — matching `card_effect_mod.prn`'s own "real decision
logic" tier, not the earlier "mod is the trigger" tier) — generated C committed at
`apps2/battlegrounds_gui/packages/simulation/action_bar_mod.c`, wired into `town_ability_for_slot`
(`src/main.c`) in place of its old hardcoded job/slot `if`-chain. I32-only ABI (job id, slot id in;
ability id out) — same VS0 ceiling every ECOWAR/PAPERCRAFT mod hits; host C still owns the actual
per-ability-id command string/label/cast-timing table (mod decides WHICH ability, host owns what
it DOES, same split every real mod in this monorepo uses). Verified: a standalone C probe against
the generated function (20 real assertions, all 7 jobs × every real slot) plus full real builds —
native (`gcc`, links clean) and WASM (`build_wasm.sh`, redeployed to
`https://okemily.com/battlegrounds/`, all 4 artifacts curl-200-verified).

**Real, found-live build gap, fixed**: PARENA's own current `runtime/parena_runtime.h` pulls in
SDL2/SDL_ttf and POSIX pty/socket/process helpers (added since this repo's own copy would have
been pinned) that this I32-only mod never calls but Emscripten's libc can't even compile
(`forkpty` has no WASM equivalent). Fixed by pinning GFD's own `packages/simulation/
parena_runtime.h` to PARENA's original, minimal 41-line version (`git rev 9bdf91e`) — same
"repo pins its own compatible copy" precedent `ECOWAR/packages/simulation/parena_runtime.h`
already set (confirmed by diff, not assumed). Documented on `action_bar_mod_host.h`'s own header.

**Real, honest scope of "extension" this buys, stated plainly (not oversold)**: reassigning which
ability lives in which slot, or extending an EXISTING job into a still-unassigned slot, is now a
`.prn` edit + `parena build` + a normal recompile — no more hunting through `town_ability_for_slot`
's C body for the job/slot DECISION. This is NOT hot-reload and NOT third-party/runtime modding —
no dlopen surface exists here, and adding a wholly NEW ability (not just moving an existing one)
still needs one new row in host C's own ability-id table. Real, unstarted next increment if this
direction continues: the same pattern applied to other GFD-side per-job/per-slot decisions (e.g.
`apps2/mud`'s own `blmSpells` map), and, if the founder wants closer-to-runtime extensibility
later, revisiting PAPERCRAFT's dlopen `--mods-manifest` pattern for this client specifically —
neither started, both real, separate future decisions, not resolved here.

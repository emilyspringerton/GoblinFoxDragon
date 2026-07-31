# DragonsNShit Has Two Non-Unified Backends — Audit + Bridge-Target Correction

*Written 2026-07-31, same day as `REDGARDEN_GUI_NORTHSTAR.md` and
`docs2/specs/REDGARDEN_MUD_BRIDGE_SPEC.md`. Founder: "continue dragons n shit" (continuing the
"do the docs first" direction).*

**This corrects a real, load-bearing wrong assumption in both of today's earlier docs.** Both
were written against `apps2/mud` as if it were *the* DragonsNShit backend. It's real and it's
deep, but it's only half the picture — checked directly while starting Milestone 1's actual
implementation, not assumed. There's a second, separately-built backend, `apps2/server-go`, and
the two don't talk to each other.

---

## 1. The two backends, checked directly

### `apps2/mud` (what both earlier docs were written against)
Telnet on `:2323`, 7,310-line single file. Real, deep RPG simulation: 22 FFXI jobs, skillchains,
enmity, conquest, NM spawns, crafting guilds, parties/linkshells (full list in
`REDGARDEN_GUI_NORTHSTAR.md` §2). **No real IDUNA persistence** — an `idunaclient.Client` is
imported and instantiated (`main.go:603,808`) but never actually called anywhere in the file
(grep confirms zero method calls beyond construction). `player.hp/mp/tp/inventory/gil/jobID` are
plain in-memory Go struct fields, gone on restart. **No binary/UDP protocol at all** — text lines
only.

### `apps2/server-go` (found today, not in either earlier doc)
Real UDP server on `:6969` (`packages2/common`'s own wire protocol — `PacketConnect`,
`PacketUserCmd`, `PacketChat`, etc.). **Real IDUNA persistence, actually called**: `PacketConnect`
verifies a real IDUNA JWT (`idunaauth.NewVerifier`) before accepting a client; `PacketTelecrystalUse`
fetches the character from IDUNA, validates the crystal/scene/gold via `telecrystal.Validate`,
deducts gold and updates scene/position via `idunaClient.TravelTelecrystal` — a complete, working,
IDUNA-backed round trip; `PacketCraftRequest` validates reagent ownership via IDUNA, runs
`craft.Attempt`, destroys reagents and creates the new item via IDUNA; `PacketSkillXP` grants
capped skill XP via `idunaClient.IncrementSkill`; a World Crisis phase-machine ticks and
broadcasts every 250ms, PATCHing IDUNA on phase change. **Combat is SHANKPIT-shaped, not
RPG-shaped**: `PacketUserCmd`'s `Buttons&BtnAttack` triggers `player.HandleShankFire` — a hitscan
raycast weapon fire, the same FPS combat model SHANKPIT itself uses. No job system, no
skillchains, no weapon skills, no enmity — none of `apps2/mud`'s RPG depth exists here.
**`PacketSnapshot` (packet type 2) is defined in the shared protocol but never sent anywhere** —
confirmed via repo-wide grep, only the definition file references it. So there's no broadcast of
other players' positions to a connected client today; each client only sees its own voxel-chunk
stream, hit impacts, and chat.

### A third piece, also found today: `apps2/lobby`
An 884-line C/SDL2 client already being built against `apps2/server-go`'s exact protocol
(`packages2/common`, `PacketConnect`, etc.) — a real, if much smaller and less complete, precedent
for exactly what `REDGARDEN_GUI_NORTHSTAR.md` proposes REDGARDEN become. It has the same `GL/
glu.h` dependency that has blocked builds repeatedly across this monorepo (REDGARDEN's own
`apps/lobby`, `shankpit-460` — same root cause each time, `libglu1-mesa-dev` missing). REDGARDEN's
`apps/arena` client is shader-based modern GL with no GLU dependency at all (confirmed via `ldd`,
see `REDGARDEN_GUI_NORTHSTAR.md` §2) — strictly more portable and, per that doc's own inventory,
far more feature-complete (hero rendering, ability-slot UI, shop panel, minimap, draft screen vs.
`apps2/lobby`'s much smaller surface).

---

## 2. What this means for the bridge target

Today's `REDGARDEN_MUD_BRIDGE_SPEC.md` designed a *new* UDP listener bolted onto `apps2/mud`,
because at the time that looked like the only real backend worth bridging to. That's now the
wrong call, for a concrete reason, not a stylistic one: **`apps2/server-go` already IS the real-
time UDP/JWT-authenticated protocol server REDGARDEN's own client architecture expects** —
`PacketConnect` + JWT auth, `PacketUserCmd` + button/movement fields, snapshot-shaped state,
chat — the same *shape* of system REDGARDEN's `arena_server` already is, just Bedrock/FPS-flavored
today instead of MOBA/RPG-flavored. Bolting a second, parallel listener onto the text MUD would
mean building REDGARDEN's own connect-ticket/UDP/snapshot machinery a second time from scratch,
duplicating work `apps2/server-go` already did.

**The real, harder problem this surfaces**: `apps2/server-go` has the right protocol shape but
none of `apps2/mud`'s RPG depth; `apps2/mud` has the RPG depth but the wrong protocol (or none).
Grafting REDGARDEN onto either one alone gets you a GUI with half the game missing. The
IDUNA `characters`/`character_skills`/`character_equipment`/`character_inventory` schema
(`IDUNA/migrations/truestore/202606230001_mmo_schema.sql`, `...0002_mmo_inventory.sql`) already
exists and is already what `apps2/server-go` writes to — checked directly, this schema has real
room for `apps2/mud`'s own job/skill/gil/equipment concepts, which is why the recommendation
below is "port the RPG logic," not "invent a new schema."

**Revised recommendation**: the actual prerequisite ahead of REDGARDEN's own bridge work is a
DragonsNShit-internal unification — port `apps2/mud`'s combat/job/skillchain/craft logic to run
*inside* `apps2/server-go`'s authoritative loop, backed by IDUNA's already-existing schema,
replacing (or running alongside, TBD) `HandleShankFire`'s hitscan model with a real weapon-skill/
job-ability resolution path. Once that exists, REDGARDEN's bridge target is `apps2/server-go`
directly — no new listener needed, REDGARDEN becomes a peer of `apps2/lobby` against the same
already-real protocol, and `PacketSnapshot` (defined, unused) is the natural extension point for
broadcasting other players' state, which REDGARDEN's own client already knows how to render.

This is a bigger, more honest scope than either earlier doc assumed. Not resolved here — this
doc's job is naming the real shape of the problem, not solving it.

---

## 3. Status of today's earlier docs

- `REDGARDEN_GUI_NORTHSTAR.md`: core thesis (fork REDGARDEN's rendering/input machinery, not its
  combat sim, onto DragonsNShit; telnet/CLI keeps working) still holds. Its Milestone 1
  description ("apps2/mud gains a second listener") is superseded by this doc's §2 — updated in
  place with a pointer here.
- `docs2/specs/REDGARDEN_MUD_BRIDGE_SPEC.md`: the packet-mapping table (`ArenaAttackCmd`→
  `cmdAttack` etc.) assumed `apps2/mud` as the bridge target and is now superseded, not deleted —
  its own gap-finding (`apps2/mud` has no continuous movement, hero-slot vs. string-ID targeting
  mismatch) is still real and still relevant once the unification in §2 lands, just not as "build
  a second listener on the text server" framing. Marked superseded in place, pointing here.

---

## 4. Open questions, not resolved here

- Does the RPG-vs-FPS combat unification replace `HandleShankFire` outright, or does
  `apps2/server-go` need to support both combat models (SHANKPIT's own FPS client still needs
  hitscan)? Real product-scope question, founder call.
- Is `apps2/mud`'s telnet interface unified onto the same authoritative loop as `apps2/server-go`
  too (one server process, two listeners — telnet AND UDP, both driving the same IDUNA-backed
  state), or do they stay genuinely separate servers reading/writing the same IDUNA rows
  independently? The former is architecturally cleaner (one game loop, matching
  `REDGARDEN_GUI_NORTHSTAR.md` §4's own "one authoritative process" diagram) but is real
  server-merge work, not named further here.
- `apps2/lobby`'s own fate — retired in favor of REDGARDEN, kept as a second GUI option, or
  something else? Not a technical question, a founder call.

---

## 5. Related docs

| Doc | Location |
|---|---|
| REDGARDEN-as-GUI northstar (amended by this doc) | `GoblinFoxDragon/docs2/REDGARDEN_GUI_NORTHSTAR.md` |
| apps2/mud bridge spec (superseded by this doc) | `GoblinFoxDragon/docs2/specs/REDGARDEN_MUD_BRIDGE_SPEC.md` |
| DragonsNShit product systems design | `GoblinFoxDragon/docs2/MMO_NORTHSTAR.md` |
| IDUNA MMO schema (the shared persistence layer both backends should use) | `IDUNA/migrations/truestore/202606230001_mmo_schema.sql`, `...0002_mmo_inventory.sql` |

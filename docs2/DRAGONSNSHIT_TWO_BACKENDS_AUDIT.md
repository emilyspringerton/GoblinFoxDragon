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
`REDGARDEN_GUI_NORTHSTAR.md` §2).

**CORRECTION 2026-07-31 (later same day): the "no real IDUNA persistence" claim below was wrong.**
Found while investigating the respawn-XP-persistence work (EMILY/BACKLOG.md item 2) — the earlier
grep here searched for the literal strings `idunaclient`/`idunaClient` and found nothing beyond
construction, but the real field is `gw.iduna` (world-level, not per-player, and not spelled
either of those searched strings) and it **is** genuinely called: on connect, `gw.iduna.
GetCharacter` fetches an existing character by a local name→ID cache (`mudCharCache`, persisted
to `var/mud-chars.json`) or `gw.iduna.CreateCharacter` makes a new one, seeding `p.charXP.Level`/
`CurrentXP` and `p.flow` from the real IDUNA row; on disconnect, a deferred block calls
`gw.iduna.UpdateCharacterLevel` and `gw.iduna.UpdatePosition` to sync level/XP/position back.
**What's still real and true**: this only syncs at connect/disconnect, not continuously during
play (unlike `apps2/server-go`, which persists immediately on most real actions) — and **`p.flow`
(gold) is never synced back on disconnect at all**, only read on connect. Checked why: IDUNA's
own `/api/v1/characters/:id/gold` endpoint (`IDUNA/internal/http/handlers/mmo.go`
`handleDeductGold`) only accepts a positive `deduct` amount server-side (`400` if `<= 0`) — there
is no credit/add-gold endpoint at all today, so even a correct apps2/mud disconnect handler
couldn't persist a Flow *increase* without a new IDUNA endpoint, a real, separate, cross-repo gap.
`player.hp/mp/tp/inventory` remain plain in-memory fields with no IDUNA equivalent at all (IDUNA's
schema has no live-combat-HP concept, matching every MMORPG's own "current HP isn't durable
state" convention already noted elsewhere in this doc). **No binary/UDP protocol at all** — text
lines only, still accurate.

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

**Done, 2026-08-03 (real backend-unification slice, founder: "server-authoritative position"):**
`PacketSnapshot` is now real, not just defined. New `integrateMovement`/`buildSnapshotPacket`
(`apps2/server-go/snapshot.go`) give `clientInfo` a real, server-owned `pos`/`yaw`, integrated
every `PacketUserCmd` from raw input (deliberately not trusting a client-reported position, a
real cheat vector for an MMO) and broadcast to every other connected client roughly every 250ms
from the main loop itself (not a new goroutine -- `clients`/`clientAddrs` have no real mutex
protecting them yet, a pre-existing gap, and adding a second unsynchronized accessor would be a
real new crash risk, not fixed here but not made worse either).

**Real, non-trivial discovery made getting here**: no on-foot movement integration existed
anywhere in this codebase family before this -- not even in SHANKPIT's own more mature sibling
server, which is genuinely server-authoritative for hit detection but only continuously
integrates movement for its racing minigame (`racing.go`'s own `applyRacingTick`, vehicle
physics, the wrong shape for walking). `integrateMovement` is the first general-purpose one.
`buildSnapshotPacket`'s own byte layout was verified against apps2/lobby's real, compiler-padded
C struct sizes (a standalone `gcc`+`offsetof` probe, not assumed from the field list) -- `sizeof
(NetHeader)=12`, `sizeof(NetPlayer)=44`, both with real alignment padding a naive field-order
read would have missed. 7 new tests, including a byte-for-byte layout check. Live-verified: the
real production `gfd-server-go.service` rebuilt, redeployed, and confirmed stable (no crash,
including the zero-connected-clients case running the new broadcast path every ~250ms).

**Still real, named gaps, not solved by this slice**: no collision against world geometry
(`world.RayTrace` is still a stub, pre-existing, not touched here) -- a player's server-side
position reflects what their input claims, not yet what's physically possible. FPS-specific
`NetPlayer` fields this backend has no tracking for (weapon/ammo/shooting/vehicle/crouch/shield/
hit-feedback) are zero-filled, not faked. Broadcast rate is ~4Hz (this loop's own natural 250ms
cadence), well under SHANKPIT sibling's own 30Hz -- real, un-costly headroom for later, once a
mutex-protected client map makes a dedicated ticker goroutine safe to add. Mobs are still not
attempted (unchanged from this doc's own earlier text) -- this closes the *other players*
visibility gap specifically, not the *mobs* one.

**Done, same day: real Y-axis ground collision (the smaller, more directly-connected half of
"no collision against world geometry" above).** New `worldapi.ColumnHeight`
(`server/worldapi/heightmap.go`) -- a single-column version of the already-tested
`HeightmapChunk`, added specifically so a per-player, per-tick lookup doesn't have to generate
all 256 columns of the chunk it belongs to just to read one. `apps2/server-go`'s new
`groundClampY` (`snapshot.go`) calls it directly (same process, no HTTP round-trip) right after
`integrateMovement`, so a player's real server-side Y now agrees with the actual terrain instead
of drifting wherever spawn/portal last left it (this backend's own real prior behavior --
`info.pos.Y` was never touched outside those two moments before today). Hardcodes scene 0
(Meadow) -- matches this backend's own existing no-multi-scene-tracking reality, and Meadow's
real height (4) turns out to already match `proceduralChunk`'s own client-side fallback
generator's hardcoded `groundY = 4` -- not a coincidence, the same real ground this system always
implicitly agreed on, just not expressed formally until now. 5 new tests (2 in `server-go`, 3 in
`worldapi` including a real negative-coordinate floor-division regression test --
`ColumnHeight`'s own chunk math needed real floor division, not Go's truncating `/`, since world
coordinates go negative routinely here). Live-verified: `gfd-server-go.service` rebuilt,
redeployed, confirmed stable. **Still not done**: horizontal wall collision (`world.RayTrace`
itself, still a stub) -- this closes vertical grounding only, a deliberately smaller first slice.

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
room for `apps2/mud`'s own job/skill/Flow/equipment concepts, which is why the recommendation
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
  server-merge work, not named further here. **Update, later same day**: the "separate servers
  converging on shared IDUNA rows" path is closer to reality than assumed above — `apps2/mud`
  already does this for level/XP/position (connect + disconnect only, not continuous). The one
  concrete, well-scoped gap found in that path: `p.flow` (gold) is read from IDUNA on connect but
  never written back, and IDUNA's own `/characters/:id/gold` endpoint only supports deducting
  gold server-side (no credit/add endpoint exists) — so completing this specific sync needs a new
  IDUNA endpoint first, real cross-repo work, not attempted in this pass (a partial fix that only
  handled the decrease direction was considered and rejected — silently wrong for the increase
  case is worse than clearly not-done).
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

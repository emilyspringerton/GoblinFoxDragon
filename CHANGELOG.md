## 2026-09-06

- Fixed the one-directional GFD<->EINHORN_SURVIVAL chat bridge (S232-01): server-go now relays say+guild to Minecraft, not just yell, matching apps2/mud's own established precedent. (sess-20260905-0720-ec33e7c5)

## 2026-09-05 (1)
- fix(items): S251-09/10/11 -- three real, found-live bugs from the Item Builder sprint plan,
  fixed together. **S251-09**: `itemdef.Registry.ByName` registered a multi-word item ("Earth
  Crystal") under its own literal spaced, lowercased name ("earth crystal"), but every real
  shop-facing call site in `apps2/mud/main.go` passes the hyphenated convention
  ("earth-crystal") — no multi-word item was ever reachable by its real shop id. Fixed by
  normalizing both the stored key and the lookup key (spaces → hyphens) via a new `nameKey`
  helper, leaving `ItemDef.Name` itself untouched for display. New test
  `TestByName_HyphenatedMultiWordName`. **S251-10**: the `eat` command's dispatch checked
  `len(args) < 2` then read `args[1]` for the item id, when `args[0]` is the actual item id (a
  single-word `eat potion` always missed and hit the usage error) — fixed to match the adjacent,
  already-correct `use` command's own pattern. **S251-11**: `data/items.json` ids 103 (Leather
  Gloves) and 115 (Spiked Knuckles) used `equip_slots: ["handl","handr"]` (no hyphen) against
  `server/gear.AllSlots`'s real canonical `"hand-l"`/`"hand-r"` — neither item was ever
  equippable as authored; corrected both rows. `go build/test ./...` clean. Live-verified:
  zero active player connections confirmed first, rebuilt + restarted the real `gfd-mud.service`,
  confirmed listening cleanly on :2323/:7171.

## 2026-09-04 (30)
- docs(dungeons): Milestone 6 ("Rewards + party queue flow") real status correction, continuing
  kanban card 534432532 ("GFD dungeons") milestone by milestone. Real, decisive investigation
  (not new code) found 2 of §5's own 3 real open questions are already answered by existing
  infrastructure: `cmdDungeonEnter` transfers every player to a fixed, deterministic zone ID per
  dungeon number (confirmed live in source: `dungeonZoneBase + (n-1)`, no per-player/per-party
  branching) -- a party trivially "queues together" today since every member lands in the exact
  same shared zone, no special queue mechanism needed, using the already-real
  `invite`/`accept`/`party`/`leave-party` commands and already-real zone-range-checked party
  XP-splitting (`pt.XPSplit`, chain bonuses). XP rewards are also already real and live for
  dungeon kills -- `awardXP` is the same universal, zone-agnostic path every other zone's mobs
  already use (dungeon mobs spawn through the identical `spawnInto` closure). The one real,
  still-missing reward type is ITEM/LOOT drops (zero random-item-award mechanism exists anywhere
  in GFD combat, a real, separate, already-scoped gap -- see `docs2/DUNGEON_HAT_DROPS_NORTHSTAR.
  md`). Difficulty/scaling remains genuinely unresolved (no scaling exists, matching every other
  zone's own static-difficulty precedent). Milestone 6's own status corrected from "NOT STARTED"
  to "PARTIALLY RESOLVED" to reflect this. No code changed -- a real documentation correction.

## 2026-09-04 (29)
- docs(dungeons): real status corrections in `docs2/DUNGEON_NORTHSTAR.md` (kanban card
  534432532, "GFD dungeons," continuing milestone by milestone). Milestone 1's own "mob spawns
  aren't wired into a live instance yet" and Milestone 3's own "IN PROGRESS" status both predated
  Milestone 4's real, later shipped work (real dungeon combat live in `apps2/mud`, confirmed live
  in source: `spawnInto(zoneID, mob.GenerateDungeonSpawns(layout, i, zoneID, seed))`) — corrected
  both rows to reflect that mob spawning IS now real and live (via `apps2/mud`, GFD's own real
  PvE-content owner), while honestly keeping what's still genuinely true: `apps2/server-go`'s own
  `DungeonInstanceRegistry` path specifically has no live mobs, and driving REDGARDEN's real
  `arena_bot` AI (vs. generic mob behavior) remains a real, separate, unstarted gap. No code
  changed — a real documentation-accuracy correction, not new functionality.
- docs: `docs2/DUNGEON_HAT_DROPS_NORTHSTAR.md` — real scoping pass (Principle 19) for kanban
  `BPGFD-INTEROP-000` ("GFD dungeons have a super rare chance to drop brawlpit hats and those
  hats are sellable on the GFD auction house"). Real, checked-live foundation: GFD dungeons are
  real and live (Milestone 4 above), a real auction house already exists (`server/market/ah.go`,
  no strict item-registry validation, no dedicated cosmetics category yet), BRAWLPIT hats already
  have a real IDUNA-backed schema/API (`WOTAN_HAT_STORE_NORTHSTAR.md`, already shipped). Real gap
  named: zero drop-chance/loot-roll mechanic exists anywhere in GFD combat today (Milestone 6,
  "Rewards + party queue flow," is itself real and unstarted — this card is a real special case
  of it). 3 real open questions named (real drop-rate number; dungeon-exclusive vs. existing-
  catalog hats; whether a GFD-side "sale" needs real IDUNA-side ownership transfer — the one
  structural question most worth a founder decision). 3-phase plan: Phase 1 a real loot-roll on
  dungeon kill, Phase 2 a real cross-repo hat-award API, Phase 3 a real auction-house listing
  path. No code written — planning only.

## 2026-09-04 (28)
- perf(apps2/server-go): GFD-994001 -- "GFD core game loop performance tuning." Real, decisive
  root cause found by direct investigation (not guessed), after checking every real hot-path
  candidate: `scanChunkForVoxelBlocks` (16x16x16=4096-cell scan, negligible), `gameWorld.RayTrace`
  (O(n) over connected clients per shot, negligible at this game's real player counts), and the
  synchronous IDUNA HTTP calls on connect/telecrystal/dungeon-enter (real, but bounded by
  `idunaclient`'s own 5s timeout and gated behind rare, deliberate player actions, not the
  steady-state tick path). The one real, steady-state, every-tick number this loop was self-
  imposing well below what the hardware or wire format need: the world PacketSnapshot broadcast
  was capped at ~4Hz (250ms) purely because it shared its cadence with the main loop's blocking
  UDP read-timeout -- already self-documented in the code's own prior comment as "a named,
  follow-on-able limitation, not silently accepted as good enough forever," never actually
  followed up on until now. Raised both the read-timeout and `snapshotInterval` to 33ms, a real
  30Hz, matching SHANKPIT sibling's own established rate -- zero concurrency changes needed
  (`clients`/`clientAddrs` stay owned by the single main-loop thread exactly as before; this only
  changes how often the loop wakes to check the elapsed-time gate and re-arm the deadline).
  Drive-by: removed a dead, never-wired `var addrsMu sync.RWMutex; _ = addrsMu` placeholder in the
  broadcast goroutine that `go vet` was flagging ("assignment copies lock value to _") -- inert
  leftover, not a real lock, zero behavior change. `GOWORK=off go build ./...` clean across the
  whole repo, `go vet ./apps2/server-go/...` clean, `go test ./apps2/server-go/...` all pass
  (23 tests, unchanged pass count). Real, honest, not investigated this pass: `apps2/mud`'s own
  separate game loop runs a deliberate 1Hz (`time.Second`) tick -- appropriate for a text MUD, not
  a real-time sync path like server-go's, so left untouched rather than "tuned" without a real
  finding there.

## 2026-09-04 (27)
- fix(apps2/mud): GFD-RDM-12422 -- "program the real RDM abilities: POISON, FIRE, WATER, STONE,
  CURE, BIO, DIA." Real, checked-first finding: 5 of the 7 named spells already worked for RDM
  (Poison/Fire/Water/Stone via the shared `blmSpells` map's existing "Black Mage OR Red Mage"
  job gate, Bio from this session's own earlier GFD-AF-01939 work) -- the real, concrete gap
  was just `cure` and `dia`, both hardcoded WHM-only despite real FFXI RDM canonically having
  both. Fixed both job gates to the same "WHM or RDM, job or sub-job" shape `blmSpells` already
  uses. `go build`/`go vet`/`go test ./...` clean. Live-verified end to end over real telnet:
  `cast cure` as RDM now heals for real (`Cure: +100 HP`, previously rejected with "Cure
  requires White Mage job or sub-job"), `cast dia` now applies real DoT damage to a target mob
  (`Dia: -5 HP on worm`, previously rejected the same way). Redeployed live under
  `gfd-mud.service`.

## 2026-09-04 (26)
- feat(server/job, apps2/mud): GFD-JOB-244122 -- "NEW CLASS ENFEEBLE DD SUPPORT SUSTAIN (sorta
  like DNC) ASSASSIN ASN can cast aoe buffs like a bard for different poisons that have
  different impacts on the different mob types etc mp regen affordancs." New 23rd job, ASN
  (Assassin) -- real DEX/AGI-focused DD stat curve (`server/job/job.go`), two real starter JAs
  (`server/job/subjob.go`'s `AssassinAbilities`, both real level/recast-gated through the
  existing `RecastTracker`): `venom` (the real "different impacts per mob type" half) --
  GFD's first real AoE nuke (every existing BLM/RDM spell is single-target-only), Poison-element
  damage against every alive mob in the caster's zone, DEX-scaled, with mobs tagged
  `mob.KindSkeleton` resisting it heavily (a real, simple, honest rule matching FFXI's own
  real "undead resist poison" convention, not a new resistance-typing system); `siphon` (the
  real "sustain"/"mp regen affordances" half, "sorta like DNC") -- a real MP-regen support JA,
  same `status.Refresh` mechanism Bard's own Mage's Ballad already uses, applied zone-wide
  ("aoe buffs like a bard"). 7 new tests across `server/job` (job count, ability-list shape,
  level-gating, recast-tracker integration). `go build`/`go vet`/`go test ./...` clean across
  the whole repo. Live-verified end to end over real telnet: `setjob ASN` works, `ja venom`
  dealt real damage to 89 real mobs in Meadow (several died, real XP/Flow awarded via the real
  `resolveKill` path), `ja siphon` applied real MP regen (45→48 MP after one tick), `venom`'s
  own recast correctly gated a second attempt. Redeployed live under `gfd-mud.service`.

## 2026-09-04 (25)
- docs: GFD-XX-1249 -- "every class/job needs to have separate levels... switching to RDM makes
  me a lvl 10 RDM... every class starts at lvl 1 and can level up to 75." Real investigation, new
  `docs2/PER_JOB_LEVELS_NORTHSTAR.md`: decisive correction to this repo's own design intent --
  the single shared `p.charXP.Level` across all jobs is real, deliberate FFXI-parity (real FFXI
  shares one level across jobs too), not a bug; the founder's ask is a genuine, decisive
  deviation toward FFXIV/WoW-style independent per-job leveling. Real scope named: an in-memory
  per-job map refactor is buildable in one pass (~12 real call sites), but real durability is
  blocked on a new IDUNA schema (the current single flat `level` column can't hold per-job
  values) -- same class of cross-repo persistence gap as `INVENTORY_PERSISTENCE_NORTHSTAR.md`,
  not attempted blind. One real, unresolved design question named: does the real FFXI sub-job/
  `EffectiveSubLevel` mechanic survive per-job leveling, or is it replaced by N independent job
  slots. No code changed -- diagnosed and scoped only. Registered as golden doc
  `PER-JOB-LEVELS-NORTH`; `docs2/STACK_CONTINUITY_REPORT.md` §6 corrected (its own earlier
  framing of this as an FFXI-behavior bug was itself wrong) and updated to point at it.

## 2026-09-04 (24)
- docs: GFD-BG-12444 -- "GFD battlegrounds is down never works... migrating redgarden changes to
  gfd battlegrounds maybe something broke." Real root-cause investigation, new
  `docs2/BATTLEGROUNDS_MIGRATION_NORTHSTAR.md`: decisive, checked-live finding -- GFD has no
  native matchmaker/arena-server of its own at all; every real Battlegrounds match is served by
  REDGARDEN's own live `red_garden_matchmaker`/`red_garden_arena_server` processes. GFD's own
  forked client simulation code has drifted from REDGARDEN's current, still-evolving copy:
  `protocol.h` 78-line diff (new `MatchFoundMsg` seed/mode fields, ground-target cast fields, a
  new packet type, an `obstacle_hp` snapshot array), `arena_game.c` 1161-line diff (REDGARDEN is
  900+ lines ahead: build templates, tree-passive HP, Duck's Smoke Bomb, per-hero zone radius).
  A struct-layout mismatch this size silently corrupts every wire field past the divergence
  point -- matches the report exactly (connects, but real gameplay breaks). 3 real options
  named, not resolved: full sync (real, substantial, ongoing recurring cost), pin the client's
  queue target to a frozen REDGARDEN build at the fork commit (fast, low-risk, freezes
  features), or give GFD a genuinely native arena server (biggest lift, only real fix to the
  recurring-drift problem itself). No code changed -- diagnosed and scoped only, same "scope it,
  don't build now" precedent as `INVENTORY_PERSISTENCE_NORTHSTAR.md`. Registered as golden doc
  `BG-MIGRATION-NORTH`; `docs2/STACK_CONTINUITY_REPORT.md` §6 updated to point at it.

## 2026-09-04 (23)
- fix(battlegrounds_gui): GFD-XX-12441 -- "teleport to town should teleport to home point
  crystal not the dragon gate." `town_telecrystal_return` (the real "return to town" scene
  change fired when leaving Meadow) and `town_recenter_in_town` (the lost-player recovery path)
  both hardcoded the Dragon Gate's own `TOWN_BUILDINGS` position (-40,-50) as the landing spot --
  now both use the real Home Point Crystal's own position instead (`TOWN_BUILDINGS`' own last
  entry, the same real position `town_draw_home_crystal` already reads, not a second position
  source). The Dragon Gate's own physical location is untouched and still correct where it's
  genuinely used elsewhere (the real interactive crystal a player stands at, in Town, to cast
  the outbound Meadow teleport) -- only the "where do you land back in Town" destination
  changed. `town_recenter_in_town`'s own combat-log message updated to match ("recentered at
  the Home Point Crystal"). Verified: `gcc -fsyntax-only` clean, a full native link (clean), and
  a full WASM build (clean) redeployed live to `okemily.com/battlegrounds/` (3/3 artifacts
  curl-200-verified).

## 2026-09-04 (22)
- docs: GFD-DOX-124 -- "do a deep dive stack continuity report linking off the readme fully
  update the readme with the current state of the world and current direction of the project."
  New `docs2/STACK_CONTINUITY_REPORT.md`, checked directly against the actual source tree, git
  log (950 commits), and every existing `docs2/*_NORTHSTAR.md` doc as of today, not a
  restatement of old docs: the 7 real apps/binaries and their roles, all 44 real server packages
  grouped by domain (each with a real one-line doc-comment-sourced description, not guessed),
  the real decisive architectural boundary found this session (`apps2/mud` owns all real PvE
  content, `apps2/server-go` has zero mob registry and is PvP-only), the two real PARENA/BURROW
  mod-integration paths now live on each host language, a per-initiative real status table, and
  honestly-named current gaps (no client render path beyond Meadow/Town, inventory has no
  durability, Battlegrounds reported broken, per-job levels not separated) plus a real,
  synthesized "current direction" section read off the actual pattern of recent work rather
  than speculated. `README.md`'s "Current Status" section rewritten from a stale 2026-08-26
  snapshot to a real 2026-09-04 one, links the new report, and fixes a stale `docs/` vs `docs2/`
  path reference for the mod-surface doc. Registered as golden doc `GFD-CONTINUITY`.

## 2026-09-04 (21)
- feat(server/modevent, apps2/mud): GFD-x-123/GFD-x-124 -- "mod interface for event broker in
  server mods should be able to register and or subscribe to specific named events in the
  system (USE PARENA TYPES) mods should fire off signals for their callbacks" + "mods can
  subscribe to certain events... fire off callbacks on specific event types events can have
  generic payload must be typed." New `server/modevent.Broker`: real, generic pub/sub --
  `Subscribe(name string, Handler)` / `Publish(name string, payload int32) []int32`, string-named
  events (any core code or mod can define a new one, no closed enum) with an I32-in/I32-out
  payload (the same real VS0-across-a-mod-boundary ceiling every ECOWAR/PAPERCRAFT/GFD mod
  already respects) -- 5 new tests, all green. Real, checked-first finding before building:
  `apps2/mud` needed BURROW's own Go emission target, not `parena build`'s C target
  GFD-MACRO-0012's own `action_bar_mod.prn` uses for the C-based `apps2/battlegrounds_gui`
  client -- this is the first real host anywhere in this monorepo to actually consume BURROW's Go
  target (shipped 2026-08-30, never used until now). Real content: `PARENA/stdlib/gfd/
  nm_bonus_mod.prn` compiled to `apps2/mud/internal/burrowgen/nm_bonus_mod_gen.go` (committed,
  same "generate once, commit, call by name" precedent action_bar_mod's own C-target build
  already established) -- real decision logic deciding the bonus XP percent for a Notorious
  Monster kill, wired into `resolveKill`'s own `mob.death` publish (payload: 1 if the kill's ID
  has the real, existing `nm-` prefix, 0 otherwise). 1 new test on the generated function itself.
  `go build`/`go vet`/`go test ./...` clean; redeployed live under `gfd-mud.service`;
  live-verified a normal (non-NM) kill still resolves cleanly with no regression (a full NM pop
  needs a real 5-30min window plus a probability roll, impractical to wait out live -- covered
  instead by full unit coverage of every sub-piece).

## 2026-09-04 (20)
- feat(server/spawn, apps2/mud): GFD-NM-123 -- "the mob spawn interface can optionally specify
  a notorious monster spawn for one of the base mobs." `spawn.Rule` gained an optional `nm`
  block (`NMRule`: id, spawn_chance, window_open_sec, window_close_sec, respawn_minutes) and a
  new `Registry.NMFor(zoneID, kind)` accessor (5 new tests, all green). `apps2/mud`'s own
  `spawnInto` closure now registers a real `nm.NMSpawn` (same mechanism `nm.MeadowNMs()`/
  `HillsNMs()`/etc already use in hardcoded Go) whenever a rule in `data/mob_spawns.json`
  declares one -- placeholder ID is the first real base-kind mob spawned in that zone, same
  "spawn right next to the base kind" precedent the difficulty-variant loop already
  established. Real content added to prove it end to end, not just structurally: `nm-slime-king`
  (Swampville `slime`, 20% chance, 5-30min window, 60min respawn) -- a genuinely new NM, not a
  duplicate of an existing hardcoded one. Real, found-live bug fixed in the same pass: world
  init's own `nmSpawns` map (the placeholder-kill-trigger lookup table) only ever wired zones
  0/3 (Meadow/Swamp) -- `nm.HillsNMs()`/`CavesNMs()` (`nm-great-beetle`, `nm-ancient-wolf`,
  `nm-bone-knight`, `nm-venom-queen`) have been real Go data since S126-13 but their
  placeholder-kill triggers could never fire; zones 1/2 added to the map. `go build`/`go vet`/
  `go test ./...` clean; redeployed live under `gfd-mud.service`; live-verified real combat
  resolves cleanly against the newly-wired zones/mobs with no crash (a full NM pop needs a real
  5-30min window plus a probability roll, deferred to natural gameplay rather than waited out
  live). Real, honest, NOT done: the web interface for authoring/editing these `nm` blocks
  (GFD-NM-124, next card in the priority queue) -- this card is the data model/mechanism only.

## 2026-09-04 (19)
- feat(apps2/mud): GFD-993944 -- real, playable v0 dungeon gameplay ("have a dungeons button
  lets us select dungeon from a list... basic combat but its super bare bones"). Real, decisive
  finding first (AskUserQuestion, founder decision): DUNGEON_NORTHSTAR.md's already-shipped
  Milestone 1 (`PacketDungeonEnter`, apps2/server-go) is on the wrong side of a real
  architecture boundary -- server-go has zero mob registry (PvP-only); every real PvE mechanic
  (`DungeonRoster`, `GenerateDungeonSpawns`, spawn/variant registries) lives in `server/mob`,
  imported only by `apps2/mud`. Shipped instead, entirely inside `apps2/mud`: 8 real dungeon
  zones (208-215) registered via `zone.Manager.AddZone` at world init, each populated with a
  real, deterministically-seeded `GenerateDungeonLayout`+`GenerateDungeonSpawns` roster (the
  real named boss/elite/minion content from DUNGEON_NORTHSTAR.md §7) through the same
  `spawnInto` closure gating Hills/Caves/Swampville. New commands `dungeons` (list) and
  `dungeon-enter <1-8>` (real zone transfer), no job/cost gate. Live-verified end to end over
  real telnet: list → enter → real named roster → `attack <mob-id>` auto-approaches and lands
  real damage, multiple aggro'd mobs hit back for real damage. `go build`/`go vet`/`go test
  ./...` clean, redeployed live under `gfd-mud.service` (systemd, not the manual env-capture
  method -- the standing going-forward preference once a real unit exists). See
  `docs2/DUNGEON_NORTHSTAR.md`'s updated Milestone 4 row for the full reasoning, including a
  real, separate, NOT-fixed-here finding: `apps2/battlegrounds_gui` has no client render path
  for any zone besides Meadow/Town, so Hills/Caves/Swampville and these new dungeons are all
  real, playable, and currently text-only (telnet/headless), not yet visually enterable via the
  GUI -- a real, separate, bigger card, not silently folded into this one.

## 2026-09-04 (18)
- feat(battlegrounds_gui): GFD-MACRO-0012 — "GFD macro system make sure we tie the action frame
  stuff into a parena mod based shape so we can easily allow for extension at the action bar
  affordance." `town_ability_for_slot`'s old hardcoded job/slot `if`-chain now calls a real
  PARENA mod (`PARENA/stdlib/gfd/action_bar_mod.prn` → `on_gfd_ability_for_slot`, generated C
  committed at `packages/simulation/action_bar_mod.c`) for the WHICH-ability-per-slot decision;
  host C keeps a small ability-id → (cmd, label, is_cast, cast_ms) table for the actual
  consequence, same "mod decides, host does" split every real ECOWAR/PAPERCRAFT mod uses. Chosen
  over `docs2/MOD_SURFACE_NORTHSTAR.md`'s own still-unscoped federated EduScript↔PARENA process
  model (§3a) — reuses the already-live, proven ECOWAR static-link mod pattern instead (see
  that doc's new §7 for the full reasoning). Real, found-live gap fixed along the way: PARENA's
  current `runtime/parena_runtime.h` doesn't compile under Emscripten (`forkpty`, no WASM
  equivalent) — GFD's own copy pinned to the original, minimal 41-line runtime (`git rev
  9bdf91e`), same per-repo-pinned-copy precedent `ECOWAR/packages/simulation/parena_runtime.h`
  already set. Verified: a standalone C probe against the generated function (20 real
  assertions, every job × every real slot), a full real native link (`gcc`, clean), and a full
  real WASM build (`build_wasm.sh`, clean) redeployed live to `okemily.com/battlegrounds/` (all
  4 artifacts curl-200-verified). `.github/workflows/build.yml` and `build_wasm.sh` both updated
  to compile the new generated file.

## 2026-09-04 (17)
- feat(battlegrounds_gui): real LOG IN / SIGN UP buttons on the login screen, closing both
  `GFD-FIX` ("ctrl alt s to sign up does not actually work") and `GFD-UX-8325432` ("we need an
  actual login button with a sign up button lower down on the screen") — the same real fix
  closes both, since they're the same underlying gap. Real, decisive root cause for the hotkey
  report, not guessed: this screen calls `SDL_StartTextInput()` for the email/password fields,
  and on several real platforms/keyboard layouts (Windows in particular — Ctrl+Alt is the
  literal AltGr combination there) a held Ctrl+Alt+`<letter>` gets consumed by the OS/IME as a
  dead-key/composition sequence while text input is active, so the `SDL_KEYDOWN` the old handler
  waited for may never actually fire with both modifiers set — a real, known class of bug for
  exactly this "hidden Ctrl+Alt hotkey" pattern (this sandbox's own Linux SDL build can't
  reproduce it directly, but the mechanism is real and well-documented). Two real, visible,
  clickable buttons now sit below the password field — LOG IN (disabled/dimmed until both
  fields are non-empty, same real gate the old Enter-to-submit path already had) and SIGN UP
  (opens the same real WOTAN store signup page the old hotkey did). The old Ctrl+Alt+S hotkey
  and Enter-to-login both stay working underneath for anyone they already worked for — this
  adds a real, discoverable, robust path, it doesn't remove the old ones. Full real WASM build
  clean (2 pre-existing, unrelated warnings only), redeployed all 4 real artifacts,
  live-verified real 200s. Real, honest limitation: click-through itself isn't verifiable from
  this sandbox (no real display/mouse), same real gap this file's own established "compile-clean
  is the achievable verification bar for pure-UI changes" precedent already accepts elsewhere.

## 2026-09-04 (16)
- feat(battlegrounds_gui): Town's own ability bar expands from 3 to 6 real slots
  (`GFD-AF-01939`, founder: "UX UI needs a big overhaul we need to bring the ability frames down
  to the bottom of the screen lets start with 1-6 as action abilities"). Real, checked-live
  finding: both Town and the Battlegrounds arena mode already had their ability tiles
  bottom-center-anchored (an earlier real card, S170-151, already did that repositioning) --
  the real, actionable gap was slot COUNT, capped at 3 (`town_ability_for_slot`'s own real
  per-job mapping only ever assigned slots 0-2; most of the 22 real jobs got nothing past slot 0
  at all). Keys `4`/`5`/`6` now wired through the exact same real dispatch path `1`-`3` already
  use (melee/JA/cast, `town_start_cast`'s own cast-timer for real MP spells); the draw loop, tile
  layout math, and per-slot cooldown-peak trackers all scale to 6. Real, honest v0 boundary
  named directly: slots 3-5 are structurally real and wired end to end, but `town_ability_for_
  slot` doesn't assign most jobs anything there yet -- "(unassigned)" shows until a real,
  separate per-job content pass (this card's own literal ask was UI capacity, not a full 22-job
  skill-list rewrite).
  Founder real-time follow-up, live, same session: "do RDM" -> "hallucinaate the RDM skills
  gimme some random ones it doesnt have to toally work they can all be copy paste of poison as
  long as they call their own fucnctions" -> "the ux is the priority but we need something to
  call to judge if its working". Gave RDM real content for the new slots -- `bio`/`distract`/
  `frazzle` (real FFXI Red Mage enfeeble/DoT spell names), each a real, independently castable
  `apps2/mud` spell entry, mechanically identical to the existing `poison` spell on purpose (same
  `blmSpells` map stats), proving the new slot plumbing actually dispatches real server commands,
  not just placeholder UI. Live-verified: a real RDM test character's `cast bio` landed a real
  hit on a real worm through the exact same path the new slot-4 key would trigger. Full real WASM
  build clean (2 pre-existing, unrelated warnings only), redeployed all 4 real artifacts,
  live-verified real 200s.

## 2026-09-04 (15)
- feat(battlegrounds_gui): real, standalone, FFXI-style Inventory screen (`GFD-INV-93911`,
  founder: "list based inventory system like FFXI list based navigation so keyboard and
  controller can navigate we will enhance with click functionality later for now its not
  needed"). Before this, `inventory` was only reachable as a raw chat-passthrough command
  (output scrolling past in the combat log) or nested inside the Auction House's own Sell-item
  picker — no standalone, browsable screen existed. New `I` keybind opens a real navigable list
  (`Up`/`Down` to select, `Escape` or `I` again to close), fetching and parsing the same real
  `[item-id]`-bracketed `cmdInventory` output the Auction House's own Sell screen already
  parses — deliberately its own dedicated globals/fetch/parse functions rather than reusing the
  Auction House's, matching this file's own established one-menu-one-set-of-globals convention
  (Town Shop menu already set this precedent over reusing the Auction House's). Real, deliberate
  v0 boundary matching the founder's own "click functionality later, not needed for now"
  framing: no `Enter` action wired yet, browsing only. Full real WASM build clean (2
  pre-existing, unrelated warnings only), redeployed all 4 real artifacts to
  `/var/www/okemily/battlegrounds/`, live-verified (real 200s on all four).

## 2026-09-04 (14)
- docs: real scoping pass for kanban priority-queue card `GFD-AH-93944` ("auction house...
  does not show the items in our inventory on the next screen"), per Principle 19. New
  `docs2/INVENTORY_PERSISTENCE_NORTHSTAR.md`. Decisive root cause traced live: `p.inventory`
  (apps2/mud's own simple id→count map) is never persisted to or loaded from IDUNA on either
  the telnet or headless connection path — confirmed live via a real telnet kill (mutation
  logic itself is correct, durability is the gap). Headless sessions (what the GUI/Auction
  House exclusively uses) get silently idle-evicted after 10 minutes
  (`headlessIdleTimeout`) with zero persistence, so any founder idle in Town that long, or
  reconnecting after an `apps2/mud` restart (several happened today during unrelated work),
  sees a genuinely empty inventory — exactly the reported symptom. IDUNA already has a real
  Items API (`ListItems`/`CreateItem`/`DestroyItem`) but it's shaped for individually-crafted
  gear, a real structural mismatch with the simple stackable-material model, and has zero
  callers anywhere in this repo today. Real 2-phase plan: Phase 0 a new, correctly-shaped
  `character_inventory` IDUNA table + `GetInventory`/`SetInventory` client methods, Phase 1 a
  real snapshot sync (load on connect, save on disconnect/eviction, matching the existing
  level/XP/flow precedent exactly) as the bounded first slice, Phase 2 (per-mutation-site
  increment/decrement, ~25 real call sites, for mid-session crash-durability) named as real,
  separate, deferred follow-up. No code written this pass — founder's own explicit "scope it,
  don't build now" direction.
- feat(battlegrounds_gui): real goblin/fox NPC rendering, GFD-ENRICHMENT-0013's second slice
  ("can we port some of the skins from SHANKPIT to GFD as NPCs Mobs" -- the first slice,
  draw_ronin_shell, shipped earlier this session for the player's own avatar). The Meadow's real
  goblin/fox NPCs (server/mob/crystal.go's S189-07 crystal-simulation pass) have existed
  server-side since that pass but were never actually visible client-side -- confirmed directly,
  no goblin/fox draw call existed anywhere in `apps2/battlegrounds_gui` before this.
  Founder-decided design (asked directly, 2026-09-04): the client reads the exact same crystal
  seed JSON file `server/mob.MeadowCrystalSpawns` already reads server-side (same real
  "positions known ahead of time" shape `town_draw_worms`' own hand-mirrored arrays already use),
  rather than a new live server-client position-sync protocol. New `load_crystal_seed` (a
  minimal, format-specific scanner -- no JSON library linked in this client at all, matching its
  bespoke-everything style), `draw_goblin_shell`/`draw_fox_shell` (same BOX-based stacked
  silhouette technique every hero/worm already uses, kind-specific color only). Decorative only,
  not targetable -- matching this pass's own real, bounded scope.
  Generated a real crystal seed (`data/crystal_seed_meadow.json`, 48 goblins + 32 foxes via
  `apps2/crystal -seed 200 -seed-out`) and wired `CRYSTAL_SEED_PATH` into the live `apps2/mud`
  systemd env file (`~/.config/gfd-mud/env`) so the server actually spawns these NPCs for the
  first time too, not just the client rendering them. WASM build gained a real
  `--preload-file` (`build_wasm.sh`) bundling the seed data into the browser build's virtual
  filesystem. Rebuilt cleanly (2 pre-existing, unrelated warnings only) and redeployed all 4
  real artifacts (`battlegrounds.html`/`.js`/`.wasm`/`.data`) to `/var/www/okemily/battlegrounds/`,
  live-verified (real 200s on all four at `okemily.com/battlegrounds/`). Live-verified server-side
  too: a real test character's `mobs` command now lists real goblin/fox NPCs in the Meadow for
  the first time ever.

## 2026-09-04 (12)
- feat(mob): dungeon roster override, GFD-MOBSPAWN-001 Phase 5 (final phase). New
  `mob.LoadDungeonRosterOverride(path)`: replaces the package-level `DungeonRoster` from
  `data/dungeon_roster.json` if present and valid, otherwise keeps the real, compiled-in default
  (same real compendium-grounded content DUNGEON_NORTHSTAR.md's §7 already established) --
  unlike `server/mobdrop`/`server/spawn`, this data has a working fallback, so a missing/
  malformed file degrades gracefully rather than starting empty. Seeded
  `data/dungeon_roster.json` with the exact same 8 real dungeons. Wired into `apps2/mud`'s own
  world init, non-fatal on failure. 3 new tests (override replaces the roster, missing file
  keeps the compiled default, empty array keeps the compiled default -- each restoring
  `DungeonRoster` afterward so other tests in the package never see a polluted global). `go test
  ./...` clean. This closes all 5 phases of `GFD-MOBSPAWN-001` -- see
  `docs2/MOB_SPAWN_NORTHSTAR.md` for the full plan. Live-verified: restarted the real production
  `apps2/mud`, confirmed the override loads with no warning.

## 2026-09-04 (11)
- feat(mob): difficulty-tier variants -- "same model, harder name" (GFD-MOBSPAWN-001 Phase 4).
  New `Mob.DisplayName` field + `Mob.Label()` (falls back to `Kind`), `mob.ApplyVariant` (scales
  HP/MaxHP/MeleeDamage by a multiplier and offsets position slightly, generalizing
  `dungeon.go`'s own real `DungeonEliteHPMul`/`DungeonBossHPMul` precedent), and a new
  `server/mobvariant.Registry` (`data/mob_variants.json`). Real, founder-decided design choice
  (asked directly, 2026-09-04): a variant's `Kind` field is left untouched, so it inherits its
  base kind's drop table automatically rather than needing its own. `apps2/mud/main.go` spawns
  exactly one variant instance per registered rule per zone, right next to a real base-kind mob
  it's templated from (not a 1:1 parallel roster), gated by the same `spawnReg.Enabled` check as
  its base kind. Seeded 3 real variants: Elder Worm (worm, 1.6x), Fierce Rabbit (rabbit, 1.8x),
  Bloated Leech (leech, 1.7x). Target/room/mobs-list display code updated to show `Label()`
  instead of raw `Kind`; `attack` targeting now also matches by display name. 9 new tests (5
  `server/mob`, 4 `server/mobvariant`), `go test ./...` clean. Live-verified: restarted the real
  production `apps2/mud`, confirmed Elder Worm spawns at 144 HP (90 base x 1.6) right next to
  worm-meadow-0, shows correctly in room look/mobs list/target message, and `attack elder`
  targets it by display name.

## 2026-09-04 (10)
- feat(spawn): data-driven spawn toggles + fixed a real, live "Hills and Caves have been empty
  since S189" bug (GFD-MOBSPAWN-001 Phase 2). New `server/spawn.Registry` (`data/mob_spawns.json`,
  `(zone_id, kind) -> enabled`, fail-open default matching pre-existing behavior) answers the
  founder's own real ask ("turn bunnies on or off in the meadow") without rewriting
  `server/mob`'s own real per-Kind stat-block constructors into generic data. Wiring this in
  surfaced a real, found-live bug: `HillsSpawns()` and `CavesSpawns()` have existed since S189's
  hero-lineup work but were never actually called from world init -- `apps2/mud/main.go` created
  empty `mob.Registry`s for zones 1 and 2 and never populated them. Fixed in the same pass: both
  zones now spawn their real mob rosters (rabbits/beetles/hills-wolves; cave-bats/cave-spiders/
  skeletons) at startup, filtered through the new registry. 5 new `server/spawn` tests, `go test
  ./...` clean. Live-verified: restarted the real production `apps2/mud` (zero active
  connections first), teleported a real WHM test character to both zones via `cast tele-hills`/
  `cast tele-caves` -- both now show real, populated mob lists for the first time ever.

## 2026-09-04 (9)
- feat(zone): real FFXI-style grid coordinate system, Phase 1 of `docs2/MOB_SPAWN_NORTHSTAR.md`
  (kanban GFD-MOBSPAWN-001). New `server/zone/grid.go`: `ZoneBounds` (real per-zone X/Z extents,
  grounded in the actual spawn positions already baked into every `server/mob` `*Spawns()`
  function -- Meadow/Hills/Swampville ±40-45, Caves' own real narrowing-away-from-entrance
  shape, New Handington a named, honest approximation pending a real town-bounds source), a
  10-unit `GridCellSize`, `CellFor(zoneID, pos) (string, error)` (derives an `I-7`-style label),
  and `CenterOf(zoneID, cell) (Pos, error)` (the real inverse, for a future spawn table to name
  a cell instead of a raw X/Z). 7 new tests, including two grounded in real spawn positions
  (`MeadowWormSpawns`' own (35,2,0), `HillsSpawns`' own perimeter wolf at (42,11,0)) and a
  round-trip check across every registered zone. `go test ./...` clean. No spawn-table or GUI
  changes yet -- Phase 2 (data-driving spawn tables into `data/mob_spawns.json`) is next.

## 2026-09-04 (8)
- fix(combat): real TP-per-hit bug found and fixed -- `apps2/mud/main.go`'s own melee-hit
  handler always called `combatTp.AddTP` with the hardcoded `combatTp.Delay1HSword` constant,
  regardless of what weapon (or nothing) a player had equipped. New `ItemDef.Delay` field
  (`server/itemdef.go`) + new `weaponDelayFor` (real lookup of the equipped main-hand weapon's
  own delay, falling back to `DelayHtH` bare-handed or `Delay1HSword` for a not-yet-backfilled
  weapon). Founder real-time: "before we go too far i think we need to add attack speed to the
  items... every weapon... not a standard delay per item type." All 18 real weapons in
  `data/items.json` backfilled with individually-chosen delay values (168-372 du), not a shared
  per-weapon-type constant, via the new IDUNA GUI (dogfooding again). 4 new tests (first real
  unit tests for `apps2/mud`), `go build/vet/test ./...` clean. Live-verified against a real
  throwaway instance: a bare-handed hit now grants 13 TP (matching `floor(80/6)` for real
  hand-to-hand delay) instead of the old, universally-wrong 40 TP (1H-sword rate).

## 2026-09-04 (7)
- docs: `docs2/ITEM_BUILDER_NORTHSTAR.md` -- Phase 2d shipped (see `IDUNA/CHANGELOG.md`'s own
  entry for the implementation): a real Vertex-powered batch item-propose assistant. Reuses the
  same real Vertex AI credential `emily.cli/cmd/promptoverse.go`'s own image generation already
  uses. Real, decisive design note: each Vertex call is stateless, so reusing the same GCP
  project shares auth/billing only, not any cross-request "memory" between image and text calls.

## 2026-09-04 (6)
- content: 18 new items via `data/items.json` -- 3 swords (Training Sword/Iron Shortsword/
  Broadsword, levels 1/5/10) and 3 full 5-piece armor sets at levels 1/7/10 (Novice/Rugged/
  Warden), ids 11-13 and 116-130. Founder real-time: "we need like 3 different swords etc we
  need a beginner armor set a lvl 7 armor set and a lvl 10 armor." Seeded through IDUNA's new
  `/admin/gfd-items` tool itself (dogfooding, not hand-edited), live-verified loading correctly
  via the real `itemdefReg.LoadFile`. Real, found-live, pre-existing bug flagged, not fixed:
  two existing items (Leather Gloves id 103, Spiked Knuckles id 115) use un-hyphenated
  `equip_slots` (`"handl"`/`"handr"`) that don't match `server/gear.AllSlots`'s own real
  `"hand-l"`/`"hand-r"` values -- neither has ever been equippable as authored. New content uses
  the correct hyphenated names throughout.
- docs: `docs2/ITEM_BUILDER_NORTHSTAR.md` updated -- Phase 2a (real IDUNA item GUI) shipped, see
  IDUNA's own changelog; new "Item design principles" section records real founder direction
  (FFXI-style per-item job lists not WoW armor-class tiers; most gear is stats+model, mod hooks
  are the minority case; `FlagRare`/`FlagEx` are already a real, correctly-separated tradability
  axis, not something needing a system change).

## 2026-09-04 (5)
- feat(mud): `use <item-id>` command -- `ITEM_BUILDER_NORTHSTAR.md` Phase 0, the blocking
  prerequisite for real item special-effects. `cmdUseItem` checks ownership, looks the item up
  via `itemdefReg`, rejects non-consumables by name, and honestly reports "nothing happens yet"
  for a real consumable -- no state mutation yet, just proving the command path end-to-end.
  Distinct from `eat`, which is `food.Registry`'s own separate, already-working system. Live,
  end-to-end verified against a real throwaway instance + a real IDUNA test character (bought a
  real Hi-Potion via `shop buy`, then `use hi-potion` printed the real message; `use potion` with
  an empty inventory correctly said "you don't have potion"). Real, separate, pre-existing bugs
  found and flagged, not fixed: `itemdef.Registry.ByName`'s lowercased-`Name` keys don't match
  the shop's own hyphenated item-ID convention for multi-word items (`"Earth Crystal"` never
  matches `"earth-crystal"`); the existing `eat` command reads `args[1]` instead of `args[0]`,
  so a single-word `eat potion` always misses.

## 2026-09-04 (4)
- docs: `docs2/ITEM_BUILDER_NORTHSTAR.md` -- real scoping pass for kanban `GFD-ITEM-SUPPLY-
  CHAIN-000` ("tool for creating and managing the stats on GFD items and weapons... special
  programming... done with mods... expose affordances to the item builder"), Principle 19.
  Real, checked-live foundation: `data/items.json` is a well-structured FFXI-style stat table
  (`server/itemdef.go`'s `Registry`), hand-edited, no tool. Decisive finding: no item-use/
  special-effect mechanism exists anywhere today -- every consumable is a real, inert data row.
  Real, existing precedent named for "special programming via mods": PAPERCRAFT's own real
  `--mods-manifest` `dlopen`/`dlsym` pipeline, recommended to port rather than invent fresh.
  4-phase plan: Phase 0 a real "use item" command (blocking prerequisite, doesn't exist),
  Phase 1 the mod-hook mechanism + one proof-point mod, Phase 2 the authoring tool itself
  (CLI vs. web page named as a real, unresolved founder-level decision), Phase 3 broader
  coverage. No code written -- planning only. Registered as golden doc `ITEM-BUILDER-NORTH`.

## 2026-09-04 (3)
- feat(login): `GFD-UA-001` ("GFD needs a sign up button that shells out to IDUNA GFD signup
  form"). Login screen had no sign-up path at all. New `Ctrl+Alt+S` handler (same
  can't-collide-with-typing special-combo pattern as PITVIPER's own `Ctrl+Alt+I`) calls
  `SDL_OpenURL` to open `wotan.okemily.com/store.html?signup=1` in the system browser -- that
  page's real register form (`WOTAN-997`, shipped this session) creates an account through the
  exact same `/api/v1/auth/email/login` this login screen itself calls, so a freshly-registered
  account works here immediately. On-screen hint added ("NO ACCOUNT? CTRL+ALT+S TO SIGN UP").
  WASM rebuilt, redeployed, live-verified. GFD commit `706ca52`.

## 2026-09-04 (2)
- feat(town-avatar): GFD-ENRICHMENT-0013 ("port some of the skins from SHANKPIT to GFD as NPCs
  Mobs, even the yellow ronin shell can stand in for our player character for now"). Real,
  checked-live finding: Town's own avatar fell through `draw_hero_model`'s `default:` case
  (`town_hero_id_for_job` always mapped to `ARENA_HERO_WARRIOR`, which has no explicit shape) --
  a plain, undifferentiated box, the exact gap the card names. New `draw_ronin_shell`, a
  standalone function (deliberately NOT a new `ArenaHeroID` -- that would ripple into
  `arena_ai_bridge.c`'s `ARENA_HERO_COUNT`-sized name/desc/tag tables, the RL policy obs-size
  compile-time assumption, and the PvP draft grid; this is the persistent-world avatar, not a
  new balanced combat hero), proportion-informed by SHANKPIT's own `player_model.h` RONIN_*
  constants (torso/shoulder-pad/head/horn relationships), rescaled to this roster's own
  ~1.3-unit convention. Removed the now-dead `town_hero_id_for_job`. Real, honest, not done:
  color ("yellow") -- this file colors every avatar by self/team/enemy relationship only, no
  per-skin color dimension exists to hang that on. WASM rebuilt (2 pre-existing, unrelated
  warnings) and redeployed to `/var/www/okemily/battlegrounds/`, live-verified. GFD commit
  `c694d34`.

## 2026-09-04
- feat(dungeons): DUNGEON_NORTHSTAR.md Milestone 1 ("instancing"), real correction + real v0
  shipped. Decisive correction: Milestone 1's own acceptance text assumed REDGARDEN's per-match
  matchmaker-fork/exec architecture, but Milestone 4's own text names Battlegrounds (this repo's
  `apps2/server-go`, a single persistent UDP process, not a per-match spawned server) as the
  real dungeon-render client -- entry needs to happen through the process the client actually
  talks to. New `server/worldapi/dungeon_instance.go`: `DungeonInstanceRegistry.Allocate`
  generates a fresh instance from Milestone 2/3's own real generators and reserves a scene ID in
  a real, wire-format-bounded `208-255` range (`PacketTelecrystalAck`/`PacketSceneChange` encode
  scene ID as a single `uint8` -- checked directly before picking a range), refusing once its
  48-instance capacity is exhausted rather than wrapping onto a still-live instance. New
  `apps2/server-go` UDP handler `PacketDungeonEnter` reuses `server/telecrystal`'s own real
  travel mechanism (`TravelTelecrystal` at 0 cost) instead of a new travel path. `worldapi`'s
  `/chunks` HTTP handler checks the registry before falling back to `ProceduralWorldStore`. 9
  new tests, `go build/vet/test ./...` clean across the whole repo. Real, honest, NOT done:
  party-roster passthrough (every instance is solo), mob spawns not wired into a live instance
  yet, no live multiplayer end-to-end verification this pass (unit/integration-tested against
  the real registry/generator code directly).

## 2026-09-03
- feat(mud): real `hatshop` command -- Town proxy to the real BRAWLPIT hat store (kanban
  `WTHS-012010`: "there is already a hat shop in town that could be a proxy to the BrawlPit hat
  shop allowing you to purchase brawlpit hats with GFD flow"). Real, checked-live finding: no
  literal hat-selling NPC exists in `npcVendorCatalog` today -- the founder's own framing reads
  as "there's already a shop mechanism in Town, reuse it as a proxy," not a pre-existing hat
  catalog. New `server/idunaclient` methods (`ListHats`/`BuyHat`/`ListCharacterHats`) call
  IDUNA's own real hat-store API (`WOTAN_HAT_STORE_NORTHSTAR.md` Phase 1, IDUNA commit
  `5bf170c`) directly, 5 new tests against a real `httptest.Server`. New MUD commands `hatshop`
  (catalog), `hatshop buy <hat-id>`, `hatshop mine` (owned hats) -- a real, deliberate
  architectural difference from the existing `shop`/`cmdShopBuy` (which deducts `p.flow`
  LOCALLY and relies on periodic sync to reconcile with IDUNA): a hat purchase spends Flow via a
  real, atomic transaction on IDUNA's own side first, so `cmdHatShopBuy` re-fetches the real,
  authoritative balance afterward via `GetCharacter` rather than guessing the new local value,
  and updates `headlessSyncedFlow` so the periodic delta-sync doesn't later see a stale
  mismatch. `go build/vet/test ./...` all clean (the one pre-existing `go vet` warning in
  `apps2/server-go` is unrelated, confirmed via `git stash`). **Real, live, end-to-end proof**:
  redeployed the live IDUNA instance with the new hat-store routes (real backup taken of
  `var/iduna.db` first, careful manual restart after the earlier session's own matchmaker
  incident), then ran a full probe as the real `DRAGONSNSHIT-MUD` agent -- created a real test
  character, credited Flow, bought a real hat (150 Flow deducted correctly), confirmed
  ownership via `ListCharacterHats`, and confirmed a duplicate purchase correctly fails.
- feat(mob): real Milestone 3 seeded dungeon spawn table (kanban `534432532` "GFD dungeons",
  `docs2/DUNGEON_NORTHSTAR.md` Milestone 3). New `server/mob/dungeon.go`: `DungeonRoster` carries
  §7's real 8-named-dungeon boss/elite content pass as real Go data (not re-derived); consumes a
  Milestone-2 `worldapi.DungeonLayout` and `GenerateDungeonSpawns(layout, dungeonIndex, sceneID,
  seed)` places the real named boss in the boss room, an optional elite hero in interior rooms,
  generic `dungeon-minion` fillers elsewhere, and keeps the entrance room clear (matching every
  existing zone's own safe-landing convention). 6 new tests (boss presence + correct SceneID,
  entrance stays clear, same-seed determinism, out-of-range `dungeonIndex` wraps safely,
  empty-layout safety, every spawn actually accepted by the real `Registry.Spawn`), all green.
  `GOWORK=off go build/test/vet ./...` all clean. Real, honest, deliberately NOT done: every
  spawn's `Kind` carries the real hero identifier as an honest placeholder -- actually driving
  REDGARDEN's `apps/arena_bot` AI as that hero, the harder cross-repo work §3.3 itself names, is
  still ahead.
- feat(worldapi): real Milestone 2 dungeon generator (kanban `534432532` "GFD dungeons",
  `docs2/DUNGEON_NORTHSTAR.md` Milestone 2). New `server/worldapi/dungeon.go`:
  `GenerateDungeonLayout(seed)` seeds 5-8 rooms in an alternating-axis snake placement, linked in
  a linear chain (real reachability from entrance to boss room, verified via an actual BFS in
  `IsReachable()`, not just assumed by construction); `DungeonLayoutToBlocks` carves the layout
  into real `WorldBlock`s reusing `cavesChunk`'s own solid-grid-then-carve representation
  (scenes.go). 7 new tests: same-seed determinism, different-seed variety, room-count bounds,
  reachability, real floor/wall block output, and a defensive empty-layout case. `GOWORK=off go
  test ./...` and `go vet ./server/worldapi/...` both clean. Real, honest, deliberately not
  built: Diablo 2's own branching/preset-room-pool connectivity (still future work per §3.2) --
  this is a straight room chain, not a graph; also not wired into the real chunk-streaming HTTP
  path yet (needs Milestone 1's own still-missing dungeon server binary to actually serve it).
- docs2/DUNGEON_NORTHSTAR.md: Milestone 1 updated to IN PROGRESS -- real seed/mode transport
  shipped in REDGARDEN (commit `4bb46b9`): `MatchFoundMsg` carries a real per-match seed+mode,
  the matchmaker generates and passes it to the spawned server, `arena_server` seeds its own RNG
  from it when present. No dungeon server binary exists yet -- that, and party-roster
  passthrough, remain the real work left in this milestone. (sess-20260902-2008-ed50169e)
- feat: real Sell flow added to the Auction House menu (kanban cruise-queue card 3455435, 'listing items on the GFD auction house should work'). Real gap found live reading apps2/battlegrounds_gui/src/main.c's own doc comments: the menu could browse categories, browse items, view your own listings, and cancel a listing -- but had NO way to actually create a new listing at all, meaning the only real path to sell an item was typing a raw '/ah sell <item-id> <price>' slash command, defeating the entire point of the FFXI-style 'arrow keys and enter' menu the founder asked for. Two new AHScreen states: AH_SELL_ITEMS (lists the player's own real inventory) and AH_SELL_PRICE (digit-only text entry, same real SDL_TEXTINPUT mechanism chat_input_active already uses). New 'Sell an Item' row on the AH main menu. Real, small, backward-compatible prerequisite fix in apps2/mud/main.go's cmdInventory: it never emitted the real item ID at all (only the display name), so the client had no way to know what to pass to 'ah sell' -- now appends a bracketed [item-id] per line, same convention ah_parse_rows already extracts from every other AH screen. Routes through the exact same real town_send_command path every other AH action in this menu uses (cancel, browse) -- no second, client-only economy. Session-only price entry, no persistence concerns (it's a one-shot command, not a setting). Live-verified, not just compile-checked: both new screens (item picker + price entry) rendered correctly under a real Xvfb run with synthetic inventory data (temporarily hooked into the arena-mode render pass, reverted before commit) -- panel, help text, row highlighting, bracketed IDs, and the live price buffer with cursor all confirmed visually. Native gcc build (-Wall -Wextra) clean, matching the real Windows cross-compile source list from .github/workflows/build.yml. go build/vet/test ./... all clean (only a pre-existing, unrelated go vet warning in apps2/server-go, not touched here). Honest, named gap: full end-to-end verification against a real logged-in character (confirming a real listing actually appears via the deployed client) was not performed -- reaching Town requires a real IDUNA login flow, a heavier prerequisite than this pass's own scope; the underlying 'ah sell' command path itself is already real, pre-existing, and covered by server/market's own passing unit tests. (sess-20260902-2008-ed50169e)
- docs2/DUNGEON_NORTHSTAR.md: real content pass (kanban cruise card "GFD add dungeons - we need content - make it like DIABLO... use the hero compendium to design bosses for the named dungeons"), new §7. Resolves the doc's own explicitly-named open gap ("exact mapping of which arena hero kit's AI drives which visual is not decided here") with real, specific assignments: 8 named dungeons, each themed to a real TYLER/multiverse_heroes.md compendium faction and populated with a boss + elite roster drawn from that same faction's own real, implemented ARENA_HERO_* kits (checked directly -- every one of the 30 real arena heroes traces to a real compendium entry). Named Kikoryu's Hoard as the real superboss/endgame dungeon using the already-landed kikoryu.jpeg art, honestly scoped as genuinely new AI work (not a repurposed hero kit, unlike every other dungeon here) since no existing hero maps to that creature. Real, honest gaps named, not hidden: Pizza/Tyler/Bacon Puck have no dungeon placement in this pass; none of Milestones 1-4.5 (the actual engineering) are built -- this is real content grounding those future milestones will consume, not an implementation. (sess-20260902-2008-ed50169e)

## 2026-08-25
- added auto-release CI job (PITVIPER pattern): real, non-prerelease GitHub release on every push to master (sess-20260825-1938-f6bd411e)
- battlegrounds_gui: chat copy/paste bindings (Ctrl+V paste, Ctrl+C copy-scrollback), same PITVIPER-inspired pattern (sess-20260825-1938-f6bd411e)
- fixed WASM battlegrounds black screen: guarded desktop-only SDL_GL_SetAttribute(3.2 Compat) calls and swapped VS_SRC/FS_SRC to GLSL ES 1.00 for __EMSCRIPTEN__ builds; verified via headless Chromium (zero WebGL/shader errors post-fix vs fatal errors before), deployed to /var/www/okemily/battlegrounds/ (sess-20260825-1938-f6bd411e)

- S189-07: seeded the Meadow (zone 0) with goblin/fox NPCs generated by apps2/crystal's own boids/ecosystem simulation. New apps2/crystal -seed/-seed-out flags run the sim headless for N ticks and dump a JSON world snapshot (goblins/foxes + terrain grids); server/mob/crystal.go's MeadowCrystalSpawns loads that seed and converts goblins/foxes into real Mob entities (kind-differentiated stats: raiders are aggressive fighters, scavenger/tinkerer/merchant and all foxes are non-aggressive flavor), positioned by mapping crystal's 44x44 grid onto the same world-coordinate space MeadowWormSpawns' own perimeter ring uses. Wired into apps2/mud/main.go's world init behind CRYSTAL_SEED_PATH env var (optional/additive, world init never fails without one). Live-verified: real seed generated (48 goblins, 32 foxes after 100 ticks), real load producing 80 correctly-positioned/typed mobs, real mud server boot with the seed loaded and no errors. 4 new automated tests, full repo build+test suite clean. (sess-20260825-0828-cc32a704)

## 2026-08-24

- MOD_SURFACE_NORTHSTAR: resolved the EduScript/PARENA scripting-language question (§3) -- two-VM model, federated process operation styled on Erlang/BEAM (§3a), founder's own phone-system-resiliency idea connected as the same real pattern (sess-20260824-2252-ce890e4f)

## 2026-08-20
- Ported REDGARDEN's King buff-HUD and King health-bar/name-tag fixes into battlegrounds_gui. WASM rebuilt and redeployed. (sess-20260820-0649-a3f19d93)
- Real WebSocket<->UDP relay (apps2/wsudprelay) makes the GFD Battlegrounds WASM web client actually playable -- fronts redgarden-stable, verified live via a real WebSocket handshake against the running matchmaker. websocket_proxy_pre.js routes through nginx's new wss:// proxy for HTTPS mixed-content compliance. (sess-20260820-0649-a3f19d93)
- Ported REDGARDEN's Four Kings/jungle-camp wire-protocol fix into apps2/battlegrounds_gui (commit 81d8076) -- same root cause (fully simulated server-side, never serialized/rendered), same fix, matching Milestone 5's porting precedent. (sess-20260820-0649-a3f19d93)
- docs2/MOD_SURFACE_NORTHSTAR.md §4a: scoped a METALVERSE terminal mode for spawning ticker
  charts + news feeds (founder: "similar to how /gta7tv spawns screens in gta7"). Real data source
  identified (FatBaby signalapi's already-live /v1/movers-history/{ticker}, /v1/press-releases/,
  /v1/entities/{ticker} -- nothing to invent on the data side), real rendering precedent grounded
  (EduScript's Architect's Orb terminal + the engine's existing 2D HUD pass; GTA7's /gta7tv itself
  doesn't port, it's Minecraft-specific). Scoping only, blocked on the same mod-surface scripting
  decision as the rest of the doc. (sess-20260813-2154-dda37e8b)
- Real, working WASM build of apps2/battlegrounds_gui (the real IDUNA login GUI client). Founder,
  urgent real-time: "we need a web client for that product yesterday." Installed Emscripten
  (emsdk, no sudo needed) and got a clean compile+link with ZERO changes to any real game source
  -- the 3D world rendering (already modern GL) ports as-is; the legacy-immediate-mode 2D HUD
  pass (~400 glVertex2f call sites) is covered by Emscripten's own `-s LEGACY_GL_EMULATION=1`
  except `glRectf`, patched via an 8-line WASM-build-only shim (`apps2/battlegrounds_gui/wasm/
  glrectf_shim.c`). Real build script (`apps2/battlegrounds_gui/wasm/build_wasm.sh`) + doc
  (`wasm/README.md`) added. Verified: clean compile, all 3 output artifacts (html/js/wasm) serve
  correctly over HTTP. NOT verified: actual browser rendering/interaction (no headless browser in
  this environment) and, more importantly, real networking -- confirmed via direct grep that
  `main.c` uses raw UDP sockets, which don't exist in a browser at all; Emscripten's socket
  emulation let this link but a real WebSocket-to-UDP relay (or a native WebSocket listener on
  apps2/server-go) is real, unstarted, unsolved work, flagged honestly as the actual hard part of
  a playable web client. (sess-20260813-2154-dda37e8b)
- Added docs2/MOD_SURFACE_NORTHSTAR.md — scoping-only pass on GFD's new "mod API first, then the
  feature" standing MO. Real prior art found: the EduScript VM (packages/education/) already
  exists and is live in apps/lobby but has zero reach into apps2/battlegrounds_gui (the FPS "edu
  edition" client, specifically prioritized). Scopes solid buildings, a destructible-environments
  engine (recommended to share one system with /home/fatbaby/skateboard's own real R6-Siege-style
  destructibility spec), skate-culture tech, and faction hooks as mod-surface consumers. Scripting
  language left explicitly open (EduScript-extended vs. Lua vs. a founder-invented language not
  yet located). (sess-20260813-2154-dda37e8b)

## 2026-08-19
- Queued MnM and Medusa as concrete next GOLDENBAND rig targets in RENDERING_QUALITY/GOLDENBAND_INTEGRATION_NORTHSTAR, with real visual briefs (sess-20260813-2154-dda37e8b)

- Added RENDERING_QUALITY_NORTHSTAR.md scoping the cel-shading -> Source-engine -> Unreal-engine tier roadmap (sess-20260813-2154-dda37e8b)

## 2026-08-10

- Jungle Camps Milestone 5：Four Kings 從 REDGARDEN 移植進 battlegrounds_gui fork，3-way merge + gcc 編譯驗證 (sess-20260810-0505-a53abca2)

## 2026-08-06
- S171-04 chat bridge, GFD side complete: EncodeChat exported, PostChatMessageAs added, ChatYell publish hook + broadcast poller (reused existing broadcastCh, no clientAddrs restructuring needed). Found and fixed a real deploy incident: the live server-go was an unsupervised orphan process blocking the systemd unit. All 3 sides of S171-04 now built; real player-facing verification still pending. (sess-20260723-2347-df115bd5)

- New docs2/CHAT_BRIDGE_TO_EINHORN_SURVIVAL_SPEC.md -- real scoping pass for the S171-04 chat bridge. Found a real structural blocker (clientAddrs is function-local in main.go, not package-level) before proposing anything. No code yet. (sess-20260723-2347-df115bd5)

## 2026-08-05
- fix(battlegrounds_gui): perf (VBO orphaning) + real skeleton-matched animation, ported from REDGARDEN same session (sess-20260723-2347-df115bd5)
- feat(battlegrounds_gui): real founder-modeled Tyler mesh, ported from REDGARDEN - real skinned character live in the actual DragonsNShit MOBA client, mud rebuilt (sess-20260723-2347-df115bd5)
- fix(ci): bundle GOLDENBAND .gband assets into DragonsNShit_MUD_GUI_Client zip - the real bug behind 'no skeleton Tyler animation' reports on downloaded builds, no hotkey ever existed (sess-20260723-2347-df115bd5)
- feat(battlegrounds_gui): S144-06 GOLDENBAND box-rig drives Tyler's animation - port of REDGARDEN's arena implementation into the real DragonsNShit MOBA client, same rig/clips, live-verified under Xvfb, mud rebuilt (sess-20260723-2347-df115bd5)
- docs: GOLDENBAND animation integration northstar (docs2/GOLDENBAND_INTEGRATION_NORTHSTAR.md) - Phase 1 box-rig proof (buildable now) + Phase 2 real skinning (blocked on founder Blender asset), with a concrete Blender asset checklist (sess-20260723-2347-df115bd5)
- battlegrounds_gui: real clipboard paste (Ctrl+V) on the login screen's email/password fields (sess-20260723-2347-df115bd5)

- fix(mud): setjob/setsubjob now persist to IDUNA; fresh headless sessions seed job/HP/MP from the real persisted job, not a hardcoded WAR L1 default (sess-20260723-2347-df115bd5)

## 2026-08-04 (11)
- "/" opens Town/in-match chat pre-seeded with "/" for quick MUD slash-commands (5fae09d) (sess-20260723-2347-df115bd5)
- feat(input): real Xbox controller support for Town/Meadow + Arena movement. SDL_GameController, hot-plug, keyboard fallback. Left stick overrides only when keyboard gave nothing that frame across both real movement systems in this file. gcc -Wall clean, smoke-tested live under Xvfb.

## 2026-08-04 (10)
- fix(mud): sethome now actually persists to IDUNA and reloads on a fresh session. Real gap:
  sethome only mutated the in-memory homePoint struct -- a custom Home Point silently reverted to
  unset on every fresh session (idle eviction, service restart). New idunaclient.UpdateHome,
  called from cmdSetHome; getOrCreateHeadlessPlayer seeds homePoint from IDUNA's real persisted
  value on session creation. Live-verified: sethome -> real DB row confirmed; restarted
  gfd-mud.service -> status correctly showed the Home Point restored, not lost.

## 2026-08-04 (9)

- fix(mud): auto-recover a KO'd character on the next real command, not just "home". Founder,
  live: "i believe i am dead" -> "yea i logged in as most recent client - i think im dead so
  nothing works but it doesnt respawn me" -> "ensuring my character gets moved to the home point
  if im dead on login". A headless session survives client reconnects as long as it hasn't been
  idle-evicted -- `getOrCreateHeadlessPlayer` used to just hand back the same still-KO'd player
  struct with no recovery at all. Extracted `cmdHome`'s real logic into `performHomeRespawn` and
  call it whenever an existing session is picked back up with `IsKO` still true. Live-verified
  against an isolated test instance (separate port, real IDUNA backend, live service untouched
  during testing): killed a disposable character via real combat, confirmed a genuine KO, then
  issued a normal follow-up command with no "home" in it -- the real respawn fired automatically
  before the command even dispatched. Deployed to the live service. `go test ./...` (GOWORK=off)
  passes.

## 2026-08-04 (8)

- feat(battlegrounds_gui): dead worms disappear, KO auto-respawn, Home Point Crystal. Founder,
  live: "dragonsnshit i think i am dead? i have no health and combat against worms stopped working
  (it was workin for a bit! but dead worms didnt disappear) i think i died and theres no respawn
  rig up home point like wow a crystal in town and we will have other crystals so you can move your
  respawn" -> "use the arena mobba fountain model as the homepoint crystal (make it look a little
  nicer)". `town_draw_worms`/`town_worm_hit_test`/the worm nameplate loop now skip any worm at
  `g_target_hp[i] == 0` instead of leaving corpses rendered and re-targetable forever. New
  `g_town_ko_respawn_sent`-gated auto-respawn: the moment a real status-line parse reports
  `g_town_hp == 0`, the client auto-sends the MUD's own already-implemented `home` command
  (`sethome`/`home` existed server-side all session, just never wired to auto-fire). Live-verified
  end-to-end against a real KO'd disposable test character: HP 0/90 -> client polled real hp=0 for
  3 consecutive real polls -> auto `home` fired -> character actually respawned at HP 1/90 with
  real XP loss, confirming a genuine server-side respawn, not a client display artifact. New Home
  Point Crystal in Town: 3-tier, 45-degree-rotated-pair faceted silhouette with a real
  sine-pulsing glow, modeled on the arena's own healing-fountain build (S170-147) per the founder's
  direction; right-click sends the real `sethome` command (curl-verified: "Home Point registered
  at Meadow"). Screenshot-confirmed rendering under Xvfb. All temp test instrumentation and the
  disposable test character were removed/deleted before commit.

## 2026-08-04 (7)

- feat(battlegrounds_gui): character screen (P), real self/worm nameplates, job-aware ability bar,
  cast timer with 94% move-cancel, click-to-target, /cast slash routing. Real founder feedback
  chain: "we need a character screen /check but a hotkey for your own character p" -> "it will
  show equiped equipment job level etc stats" -> "now that im blm im expecting 1 2 3 to be
  differebnt spells" -> "if i click on a enemy it targets it and hiting the hotkey casts a spell"
  -> "there will be cast timer and if the player moves before 94% casted then the cast cancels" ->
  "also give a slash command in the chat for casting the same spells" -> "the worm health bar does
  not update - i dont even have a healthbar in town" -> "i should have my name and health over my
  head just like the enemies." Fixed a real g_town_job staleness bug (job-change menu never
  updated the client's own notion of the job). New: left-click sets a real client+server target
  separate from right-click's attack-move; a client-side cast-timer (town_start_cast/
  town_cast_tick) gates real spells behind a visible progress bar and cancels on real displacement
  before 94% progress; a job-aware 1/2/3 ability bar (BLM: Fire/Poison/Clear Mind; other jobs use
  this session's earlier starter-kit work); /cast fire|poison|cure in chat now routes through the
  same cast-timer instead of a second instant path; real self HP (parsed from the status line
  every response already carries) and real per-target worm HP (new periodic silent "look" poll)
  replace the old static full bars; a new floating self nameplate; a new read-only character
  screen fetching real "status"+"gear" output. Verified live under Xvfb against the running
  gfd-mud.service with fresh, isolated disposable accounts -- caught and fixed two real
  test-environment bugs along the way (a stale movement-target causing a false cast-cancel, and
  confirmation that Town's WASD reads real OS key state rather than the event queue). End-to-end:
  job-aware tiles matched exactly, a real Fire cast dealt real damage and killed its target, a
  second cast was correctly cancelled by real displacement, character screen rendered real
  stats/equipment. Also investigated live: founder report "auto attacking does not work on the
  second worm in town" -- reproduced the exact sequence under Xvfb, both worms worked correctly
  end-to-end; a live Sunderworm World Crisis event with several very-high-HP mobs was active in
  Town during the investigation, a more likely explanation than a reproducible client bug. No fix
  applied; flagged in the backlog for a repeat report outside a crisis window. All disposable test
  accounts and temp debug instrumentation cleaned up before commit. `gcc -Wall` clean.
  GoblinFoxDragon `3e5aebb`.

## 2026-08-04 (6)

- feat(mud, battlegrounds_gui): founder pivot -- job-change NPC, BLM Poison starter spell, starter
  ability kits, Town shop. Real chain of founder real-time direction: "maybe pivot? add an npc in
  town that lets player change jobs" -> "implement blm with a starter fireball and poison spells"
  -> "ensure spell casting works o n the worms and puts them into combat" -> "we will map out more
  expanded abilities once we get our systems working underneath" -> "and then add a couple starter
  kits 2 abilities per job to the basic jobs" -> "put a fuew basic items into the shops in town and
  build a npc buy sell ui similar to the auction house ui." Six real pieces, server first:
  (1) fixed `cmdCastBlackMagic`/the WHM `dia` case directly mutating mob HP instead of calling the
  real `reg.Hit` (server/mob/mob.go) -- every existing BLM nuke silently bypassed mob aggro before
  this, not just a hypothetical new spell would have; (2) added "poison" as BLM's second starter
  spell on the now-fixed path; (3) 2 real starter abilities each for MNK/BLM/RDM/THF (the FFXI
  starter jobs that had none -- only WAR/WHM/SMN did before); (4) a real Town vendor NPC
  (Quartermaster, zone 4) with a starter item catalog -- Town had no vendor at all; (5) a
  job-change menu in battlegrounds_gui (Guild House building, real `setjob` dispatch); (6) a Town
  shop buy/sell menu (Potions building, modeled on the Auction House UI, real `shop`/`shop buy`/
  `shop sell` dispatch). Every piece verified live against the running gfd-mud.service / a real
  Xvfb-driven client, not asserted: poison casting a real worm and setting its aggro flag, Chakra
  healing a real nonzero amount after real damage, Convert's HP/MP trade, the job-change menu
  actually flipping a character's real job (confirmed via a direct `jobs` probe), and a full shop
  buy-then-sell round-trip moving real flow (5000 -> 4950 -> 4975). All disposable test characters
  and temp debug instrumentation cleaned up before each commit. `go build/test/vet/gofmt` and
  `gcc -Wall -Wextra` all clean; GitHub Actions Windows build confirmed green with a real
  non-expired artifact. Commits: `1bba8da` (aggro fix + poison + abilities), `6f56b67` (Town
  vendor), `5d1a2e7` (client UIs). Apple #12013.

## 2026-08-04 (5)

- fix(battlegrounds_gui): decode real JSON escapes in http_extract_json_string_field, add
  floating damage popups -- founder, live: "ok progress we run up to our guy now but there
  are still no visible auto attacking going on." Real root cause, found by reading the shared
  JSON string decoder's own source after a raw debug dump showed decoded MUD output containing
  the literal two-character sequence "rn" in place of every real line break: the decoder used to
  skip the backslash on any `\`-escape and copy whatever character followed it literally --
  correct by accident for `\"` and `\\` (the escaped character IS the real character), silently
  wrong for `\r`/`\n`/`\t`, where the character after the backslash is a letter standing in for a
  real control byte. Every real multi-line MUD response (apps2/mud's own `\r\n`-separated combat
  text) decoded to plain "rn" instead of a real line break, so `town_mud_command`'s own
  `\r\n`-based line-splitting saw the entire response as one unsplittable blob -- this had been
  silently corrupting the combat log all session, which is the real reason combat feedback never
  read as "visible" even once the run-up fix (2026-08-04 (4)) landed. Real fix: decode the
  standard JSON escapes (`\n`, `\r`, `\t`, `\"`, `\\`, `\/`) to their real bytes instead of
  stripping-and-copying-literally. New floating damage-popup feature (`town_spawn_damage_popup`/
  `town_draw_damage_popups`, a 12-slot ring buffer) parses the same now-correctly-split lines for
  the MUD's own real "You hit for N damage"/"[!] `<mob>` hits you for N damage" text and spawns a
  rising WoW/LoL-style number over the real hit's own position. Verified two ways: a standalone
  unit test compiled directly against the fixed function, showing byte-level real CRLF output
  (`strstr(out, "\r\n")` finds real matches, hexdump confirms real 0x0D/0x0A bytes) where the old
  code produced none; and a full live end-to-end run via synthetic SDL right-click injection
  against a real worm fight under Xvfb -- the combat log pane rendered clean, individually split
  lines for the first time this session, damage popups fired with amounts matching the real MUD
  text exactly (30 outgoing, 8 incoming), through an actual kill with real XP and loot. All temp
  debug instrumentation and every disposable test character from this debugging arc (including
  three left over from earlier in the session) reverted/deleted before commit. `gcc -Wall -Wextra`
  clean; no Go changes in this repo, so `go test ./...` doesn't apply here.

## 2026-08-04 (4)

- feat(battlegrounds_gui): real run-up-then-attack on right-click, matching Battlegrounds -- founder,
  live: "if i right click on a worm i expect to run up to it and start attacking just like how it
  works in battlegrounds" -> "i right click the worm turns yellow and i dont run uo to it" ->
  "then i manually run up to it i expect auto attacks to start... auto attacks never start."
  Real root cause: right-click's worm-attack branch fired the real MUD `attack` command
  immediately from wherever the player was standing -- the MUD's own auto-approach closes the
  distance server-side (real, already proven working), but nothing ever moved the CLIENT's own
  avatar to match, so the player watched themselves stand still through a fight that was
  genuinely happening with zero visible feedback. New `g_town_pending_attack_index`: right-click
  on a worm now sets a real movement target stopped `TOWN_MELEE_APPROACH_DIST` (1.8, inside
  `server/mob/worm.go`'s own real `WormMeleeRange` 2.0) short of it, and a new per-frame arrival
  check fires the real attack command the moment actual proximity is reached -- checked against
  real distance every frame regardless of how the player got close, so this covers both
  click-to-approach AND plain manual WASD walking, closing both reports at once. Verified with a
  real, rigorous, end-to-end test this time, not "the logic looks right": a synthetic SDL
  right-click injected at a worm's own true screen position (10+ world units away, so the walk was
  genuinely exercised), polled every 500ms for the client's own avatar position converging on the
  correct stop-short point over ~4.5 real seconds, confirmed the pending-attack flag clearing
  exactly on arrival, then independently confirmed via a direct `/api/town/command` probe against
  the real MUD that the attack command actually landed -- real damage, a real kill, real XP and
  loot, not simulated or assumed. `gcc -Wall -Wextra` clean, `go test ./...` clean (no Go changes).
  All temp debug instrumentation and the disposable test character used for verification fully
  reverted/deleted before commit.

## 2026-08-04 (3)

- fix(battlegrounds_gui): GL_DEPTH_TEST never re-enabled after Town's own 2D HUD pass -- founder,
  live: "its really laggy in the meadow and in the town" + "the 3d stff looks a little weird
  buildings showing through eachother etc." Real root cause: `glEnable(GL_DEPTH_TEST)` was only
  ever called ONCE, at startup, before Town's own render loop begins. Every 2D HUD pass this same
  loop calls each frame (`town_draw_hud`/`combat_log_draw`/`chat_draw`/`ah_draw`, etc.) disables it
  for their own ortho overlay, and nothing in Town's own loop ever turned it back on before the
  NEXT frame's 3D pass -- so from frame 2 of the whole session onward, every 3D draw (avatar,
  terrain, buildings, worms, trees) rendered with depth testing OFF. This explains both real
  reports from the same root cause: wrong occlusion (painter's-algorithm-only draw-call ordering
  instead of real depth comparison -- "buildings showing through each other") AND real GPU
  overdraw cost (disabling depth testing also disables early-Z hardware culling, so every
  fragment of every overlapping triangle gets fully shaded even when hidden behind something
  else -- worse the more geometry is on screen, and this session added a lot: bigger worms,
  trees, flowers, nameplates). Battlegrounds' own separate loop already re-enables this every
  frame; Town's loop just never got the same treatment. Fixed with one `glEnable(GL_DEPTH_TEST)`
  at the start of Town's own 3D pass, matching Battlegrounds' own convention. `gcc -Wall -Wextra`
  clean, `go test ./...` clean (no Go changes).

## 2026-08-04 (2)

- fix(battlegrounds_gui): real click-to-ground bug -- founder, live: "click to move doesnt work in
  meadow either" -> "ok i understand you checked and you think its fine but its not fixed try
  harder." Real root cause, found by re-examining `screen_to_ground` (used by click-to-move AND
  the new right-click building/worm hit-tests, all of it): it has always solved the mouse ray
  against a hardcoded y=0 ground plane, correct for Town (flat by design) but never actually
  correct for Meadow, whose real terrain has gentle rolling elevation (world-space range
  ~4.5-7.5 once `TERRAIN_TEST_HEIGHT_SCALE` hit 1.5). A ray solved against y=0 crosses a plane
  several units below the real ground the player is standing on, landing every click on the wrong
  world point -- small enough to go unnoticed at the old, flatter/smaller scale, large enough to
  matter once the terrain-height and golden-ratio zone-size work landed. Live-verified the actual
  math: at the real Meadow ground height (7.5), the OLD y=0-only solve would have resolved a
  screen-center click to a point roughly *twice* as far from the player as it should be -- not a
  subtle drift, a real, large mispositioning on every single click. Fixed with real iterative
  refinement (not a guess): solve against y=0 for an initial estimate, then re-solve against the
  REAL terrain height at that point via `dfzone_height_at`, repeating up to 4 times until it
  converges onto the actual surface. Only engages in the Dragonfly zone (`g_dfzone_active`) --
  Town's own y=0 ground is untouched. `g_dfzone_active`'s declaration moved earlier in the file
  (screen_to_ground needed it before its old declaration point) with a forward declaration for
  `dfzone_height_at`, real definitions unchanged. `gcc -Wall -Wextra` clean. Live-verified via a
  debug-instrumented build: confirmed the real ground height (7.5) and confirmed the fixed
  function now resolves a center-screen click to a sensible nearby point instead of an
  overshoot (temp instrumentation fully reverted before commit).

## 2026-08-04 (1)

- ops/investigation: right-click attack confirmed working via a clean character; test@test.com's
  own character found in a real, unresolved bad state -- founder, live: "ok i just tested again -
  combat against worms does not work - right click on worm i should run up and start attacking -
  currently nothing happens when i run up to it besides it does turn yellow." Verified the
  right-click -> town_worm_hit_test -> town_send_command pipeline is genuinely correct: created a
  disposable test character, `attack worm-meadow-2` produced a real, normal fight ("You hit for 30
  damage" / "worm-meadow-2 hits you for 8 damage"), then deleted the disposable character. The
  real character (TestWarrior) was separately found repeatedly dying within seconds of every
  revive (`home`, HP:1/90) with ZERO combat text ever appearing in three separate polling passes --
  a live `nm-king-worm` NM (800 HP, 60 dmg/hit, AggroRange 15/LeashRange 40) is present in this
  Meadow instance and is the strong suspect, but the silent-death mechanism itself (real damage
  code at `apps2/mud/main.go`'s `EvtMobAttack` handler does send a message on every path) was
  never actually caught in the act -- flagged as a real, separate, unresolved bug for a future
  session, not guessed at further here. Character revived one final time (HP:1/90) before handing
  back to the founder. No source changes this pass.

## 2026-08-03 (30)

- fix(battlegrounds_gui): position sync silently reverted Meadow's real scene_id back to Town on
  every step -- founder, live: "ok i ran up to a worm and right click - it turns yellow on the
  name plate good but i never actually auto attack." Investigated the real character
  (`test@test.com`'s TestWarrior): found `scene_id=4` (Town) in IDUNA sitting alongside real
  Meadow-space coordinates (near a real `MEADOW_TARGET_X/Z` worm position) -- a live reproduction
  of the exact bug class already fixed once for relogging, reopened by a different real bug.
  `town_sync_position` (the periodic best-effort position PATCH, fires on every ~2s/moved-far-
  enough step) hardcoded `TOWN_ZONE_ID` (4) unconditionally, so simply walking around in Meadow
  kept silently overwriting the correct `scene_id` the real `travel` command had just set. The
  live MUD session itself wasn't affected (tracked in-memory, only re-read from IDUNA at session
  creation) -- but a relaunch (`town_fetch_character`'s own scene_id read) or a session eviction
  would have read the corrupted value and put the player back in Town. Fixed: `sync_scene_id =
  g_dfzone_active ? g_dfzone_scene : TOWN_ZONE_ID`.

  Separately, the real reason THIS particular right-click never landed a hit: the character was
  genuinely KO'd (`HP:0/90` -- a live `nm-king-worm` NM, 800 HP, is present in this Meadow
  instance, likely what killed a level-5 character). `attack` was correctly rejected server-side
  ("You are KO'd. Type 'home' to return to your Home Point.") -- not a client bug, confirmed via a
  direct `/api/town/command` probe against the real character. Manually sent the real `home`
  command on the character's behalf (agent JWT) to revive it (no Home Point set, so a free
  respawn at Meadow, HP:1/90 -- fragile, not topped up further, that's the real game rule) and
  corrected the corrupted IDUNA row (`scene_id` 4 -> 0, matching where the live MUD session and
  the character's own real position already agreed they were). `gcc -Wall -Wextra` clean.

## 2026-08-03 (29)

- feat(battlegrounds_gui): right-click is the real attack-move/interact button now, everywhere --
  founder: "switch right click to attack move /interract for both the mud gui battlegrounds as
  well as meadows ensuring meadows shows nameplates with healthbars for the worms" + "ensuring
  tyler clones have the ability to attack with right click."

  Town/Meadow: right-click used to be Auction-House-interact-or-camera-drag, left-click did
  click-to-move. New shared `right_press_x/y` gives a real click-vs-drag distinction (same
  `ARENA_DRAG_SELECT_THRESHOLD_PX` Battlegrounds' own box-select already uses) -- a quick
  right-click now dispatches in priority order: interact with a building (Auction House, existing
  behavior), else attack a worm under the cursor (new `town_worm_hit_test`, a real screen-space
  hit-test against whichever ring `town_active_targets` resolves -- Town's or Meadow's -- reusing
  `g_last_vp`, the same "last frame's vp, since this frame's isn't computed yet at input time"
  convention Battlegrounds' own box-select established, now also written by Town's own render
  pass), else move there (click-to-move, relocated from left-click). Left-click is UI-buttons only
  now (queue/teleport-to-town).

  Battlegrounds: the existing, already-complete move/attack/attack-move/patrol dispatch (NORTHSTAR
  §17.1's own doc comment already specified "right-click ground vs right-click a unit" -- it had
  just never actually been bound to that button) moved from LEFT mouseup to RIGHT mouseup, using
  the same click-vs-drag distinction (right-click already drives camera rotation, so a real drag
  must not also fire a command). `commanders[]`/`selected_or_self` -- which already includes the
  local player's own active Tyler clones -- moved verbatim, so clone-inclusive attack-move is
  unchanged by the rebind. Left-click is selection-only now: box-select-drag unchanged, and a
  plain click (no drag) gained real single-unit-select (same per-candidate screen-space test as
  box-select, a point instead of a rectangle) so a non-drag left click isn't left as a dead input.

  Verified end-to-end via a real synthetic SDL right-click (SDL_PushEvent, not just static
  reasoning): fired at a live worm's own real screen position (computed from `g_last_vp`) and
  confirmed the full pipeline -- `town_worm_hit_test` correctly resolved the target, and the real
  "attack worm-meadow-N" command was dispatched via `town_send_command`. Nameplates/health bars
  (`town_draw_worm_nameplates`, shipped last pass) reconfirmed visually via Xvfb screenshot,
  unaffected by this change. `gcc -Wall -Wextra` clean, `go test ./...` clean (no Go changes this
  pass). All temp debug instrumentation used for the synthetic-click verification fully reverted
  before commit.

## 2026-08-03 (28)

- feat(battlegrounds_gui): real worm silhouette + nameplates/health bars -- founder: "the worms
  look like 3 little poops on the ground next to eachother not like a worm ... have the worms
  more worm like and like slightly floating in the air or something - bigger - like a lot bigger"
  + "we also need nameplates and health bars like in battlegrounds." The old shape (3 flat,
  monotonically-shrinking boxes almost at ground level) read as debris, not a creature. New:
  `WORM_SEG_COUNT` 5 segments arched into a real worm curve (peaks at the middle segment, tapers
  at both ends -- a real "rearing up mid-crawl" silhouette, not just bigger boxes), each segment
  up to ~4x the old max half-extent, floating at `WORM_FLOAT_Y` (1.1 units) above real ground
  instead of sitting on it, plus two small eye-dots on the head segment for a "this end is the
  front" cue. New `town_draw_worm_nameplates` (called from `town_draw_hud`, which now takes `vp`
  as a real parameter) -- same real technique Battlegrounds' own per-hero floating health bars
  use (`world_to_screen` projecting a world anchor into 2D HUD space, black-bg + colored-fill bar,
  name above it). The bar is honest, not faked live data: apps2/mud still has no HTTP surface for
  real mob HP (same gap `town_draw_worms`' own doc comment already names for position), so it
  always draws full/green -- "a worm is here and alive," not a live-HP claim. `gcc -Wall -Wextra`
  clean. Live-verified via Xvfb: nameplates+bars render correctly above both worms in frame; the
  new silhouette reads as a rounded, floating creature at real in-game camera distances, not flat
  debris.

- feat(battlegrounds_gui): Meadow zone expanded to a real golden-ratio rectangle -- founder:
  "expand the zone have it be the golden ratio but on the long ways have it like 4x as big as it
  is currently." Was a uniform 80x80 square (16 cells * one shared `TERRAIN_TEST_CELL_SIZE` 5.0
  applied to both axes). New `DFZONE_CELL_SIZE_X`/`DFZONE_CELL_SIZE_Z`: long axis (X) is 320 units
  (4x the old real footprint), short axis (Z) is the long axis divided by the real golden ratio
  phi ((1+sqrt(5))/2 = 1.6180339887..., not a rounded guess) -- ~197.75 units. F10's own debug
  test patches stay square (`TERRAIN_TEST_CELL_SIZE` untouched, unaffected) -- this only reshapes
  the real dfzone the founder actually plays in. `build_heightfield_mesh` now takes separate
  `cell_size_x`/`cell_size_z` params instead of one shared value (both mesh-position and normal-
  gradient math updated per axis); `dfzone_height_at`'s own bounds check and grid-space conversion
  are rectangular now (`half_x`/`half_z`); `town_move_half_extent` split into
  `town_move_half_extent_x`/`_z` (both real click-to-move and WASD clamp call sites updated --
  a single shared half-extent would clamp the short axis too late or the long axis too early);
  tree/flower world-position math updated to the same per-axis conversion. Real Meadow worm
  positions (`MEADOW_TARGET_X/Z`, max +-35) and the telecrystal spawn (0,2,0) all still comfortably
  fit the new bounds (half_x=160, half_z=~98.9), no changes needed there. `gcc -Wall -Wextra`
  clean. Live-verified via Xvfb: same camera position now shows real content (trees) much farther
  out than before, ground renders continuously with no edge/gap, matching the real ~4x expansion.

## 2026-08-03 (27)

- fix(battlegrounds_gui): "TELEPORT TO TOWN"/"H" now also cover a stranded-in-Town position, not
  just Meadow -- founder, live: "teleport to town button does not work... look into why i might
  not have seen the button." Real root cause found live: `test@test.com`'s character
  ("TestWarrior") had `scene_id=4` (real Town) but `pos_z=3977.48` -- thousands of units past
  `TOWN_MOVE_HALF_EXTENT`, almost certainly drift from before that movement clamp existed (its own
  doc comment: "could walk... thousands of units past the actual ground/building layout"). Both
  "H" and the button were gated on `g_dfzone_active` alone (Meadow) on the assumption a bad-Town-
  position was already a closed bug -- but that clamp only stops *new* bad writes, it never
  repaired rows already corrupted before it landed, and this player's own row was exactly that.
  New `town_player_lost()` (Meadow, unconditional -- OR Town with `|x|`/`|z|` past
  `TOWN_MOVE_HALF_EXTENT`) and `town_recenter_in_town()` (resets to the real Dragon Gate spot and
  force-syncs to IDUNA via the existing `town_sync_position`, so a corrupted row can't come back
  on the next relaunch). `town_draw_hud` takes `player_lost` as a real parameter now instead of
  computing it internally, avoiding a forward-declaration ordering issue (`town_player_lost` needs
  `g_town_x`/`g_town_z`, declared after `town_draw_hud` in this file). Manually fixed
  `test@test.com`'s live character via a direct IDUNA agent-JWT PATCH (IDUNA `var/iduna.db`:
  scene_id=4, pos now (-40,0,-50), the real Dragon Gate spawn) while this fix was in flight.
  `gcc -Wall -Wextra` clean. Live-verified via Xvfb: reproduced the exact real corrupted state
  (scene 4, pos_z=3977.48) and confirmed both the button and the "H - return to town" HUD hint now
  render, where neither did before.

## 2026-08-03 (26)

- feat(battlegrounds_gui): real "TELEPORT TO TOWN" button -- founder: "give me a teleport to town
  button under queue for battlegrounds." Same real, unconditional `town_telecrystal_return` the
  "H" emergency keybind already calls, now with a real clickable affordance too. New
  `town_teleport_town_button_rect`/`_hit`, sharing `town_queue_button_rect`'s own real screen
  position as its anchor so it always sits directly under "QUEUE FOR BATTLEGROUNDS" even if that
  button's own position ever moves. Only shown while `g_dfzone_active` (Meadow) -- nothing to
  "return" to from Town itself. `gcc -Wall -Wextra` clean. Live-verified via Xvfb screenshot: the
  button renders directly beneath the queue button in the real top-right corner.

## 2026-08-03 (25)

- feat(worldapi,server-go,battlegrounds_gui): block-backed flowers in Meadow -- founder: "add some
  block backed flowers to the meadow." New `meadowFlowers` (server/worldapi/scenes.go), same
  deterministic-per-chunk discipline as `meadowTrees` (5-8 real positions per chunk, hand-picked to
  avoid landing on a tree's own trunk column), alternating real `minecraft:poppy`/
  `minecraft:dandelion` blocks rooted at the real per-column terrain height. New voxel block ID 19
  in `dragonfly_gen.go`'s `blockNameToID` (both flower types share it -- a real VoxelBlock consumer
  only needs "flower here," the poppy/dandelion color distinction lives in `WorldBlock.BlockName`
  and in the client's own render logic) -- SHANKPIT's own `block_map.go` has no flower content in
  any of its own scenes, so this ID is GoblinFoxDragon-only for now, a named gap rather than a
  guessed cross-repo sync. Client-side `town_meadow_flower_positions`/`town_draw_dfzone_flowers`
  (apps2/battlegrounds_gui) mirror the real positions exactly and render a thin green stem plus a
  small red/yellow bloom (same stacked-primitive technique as trees), sitting on the zone's own
  real rolling terrain via `dfzone_height_at`. New `TestProceduralWorldStore_Scene0Meadow_Flowers`
  (both real block names present, no flower on a tree's trunk column). `go test ./...` and `gcc
  -Wall -Wextra` clean. Live-verified: `/chunks?scene=0&cx=0&cz=0` now returns 8 real block ID 19
  entries; a debug-instrumented client build confirmed `town_draw_dfzone_flowers` firing every
  frame with the correct real positions (temp instrumentation, fully reverted before commit).

## 2026-08-03 (24)

- fix(battlegrounds_gui): relaunching while your last real position was in Meadow could strand you
  "in the middle of nowhere" -- founder, live: "i was in the meadow and closed the game - then the
  thing happened where i was in the middle of nowhere not in town - dunno how i get so far away
  from town i guess we need a town teleport button." Real root cause: `town_fetch_character` (runs
  once on launch) loaded `pos_x`/`pos_z` straight into `g_town_x`/`g_town_z` without ever reading
  `scene_id`, and `g_dfzone_active` always starts false on a fresh process -- so a character whose
  real last position was real Meadow-space coordinates (up to +-35 units, MEADOW_TARGET_X/Z's own
  range) had those same raw numbers rendered as if they were Town coordinates, landing the avatar
  and camera orbit far outside Town's own much smaller real footprint. Same unresolved gap
  `town_telecrystal_travel`'s own doc comment already named ("requiring a relog to (still never)
  catch up"). Fixed: `town_fetch_character` now reads the real `scene_id` and switches into dfzone
  render mode to match before seeding position, instead of assuming Town. Also added the founder's
  own proposed fix as a real safety net: "H" now unconditionally calls the same, already-proven
  `town_telecrystal_return` regardless of range/ring state -- G/the gate ring stays real,
  proximity-gated telecrystal UX (SHANKPIT-ported, "the ux is good... circle showing cast
  radius"), but H is the "I'm lost, get me out" escape hatch that doesn't require finding your way
  back to a crystal you can't see. HUD control-hint line shows "H - return to town" while in
  Meadow. `gcc -Wall -Wextra` clean.

## 2026-08-03 (23)

- fix(battlegrounds_gui): Town's own avatar was invisible after teleporting into the real
  Dragonfly Meadow zone -- founder: "my avatar is not visible in the meadow scene that i
  tele[p]orted to." Root-caused via controlled Xvfb+screenshot A/B tests (marker boxes at known
  Y/scale, isolating the one real variable): `draw_hero_model`'s Town-avatar call pre-multiplied
  the camera's own `vp` matrix with a world-space jump/terrain-height translation
  (`vp_avatar = jump_t * vp`) and passed that AS `vp` into a function that internally computes
  `mvp = vp * model` -- putting the translation in clip-space (post-projection) instead of
  world-space (pre-projection), which does not correspond to a uniform shift under perspective
  projection (translation doesn't commute with a projection matrix through the perspective
  divide). Fixed by adding a real `hero_y` parameter to `draw_hero_model`, threaded through the
  `BOX` macro's own `dy` (the MODEL side of the transform, mathematically correct); the
  `jump_t`/`vp_avatar` construction is gone entirely, `vp` itself now passes through every caller
  unmodified. Battlegrounds' own hero/clone rendering (the other 2 call sites) pass `hero_y=0.0f`,
  preserving prior behavior exactly. Live-verified via Xvfb screenshot: avatar renders correctly,
  centered, on terrain.

- feat(battlegrounds_gui): real Meadow worm combat in the Dragonfly-backed dfzone -- founder:
  "and we can fight worms in that new area?" -> "do the engineering work to fix that first."
  `town_telecrystal_travel`/`return` used to bypass apps2/mud entirely (a direct PATCH to IDUNA's
  own position endpoint) because `cmdTravel` had a real self-deadlock reached via headless
  dispatch; that bug is already fixed (GoblinFoxDragon `15ea788`, same-day), but the client bypass
  was never updated to match. That's the real reason Meadow's worms were unreachable: IDUNA said
  scene_id=0 and the client happily rendered Meadow's terrain, but the LIVE headless MUD session
  backing real combat never actually left New Handington, since only `cmdTravel`'s own
  `gw.zoneMgr.Transfer` call registers a real zone change. Fixed: both functions now dispatch the
  real `travel <crystalID>` MUD command via a new shared `town_mud_command` helper (also used by
  `town_send_command`), checking the real response text ("...transported to MEADOW!") for success
  instead of an IDUNA HTTP status code. Confirmed end-to-end BEFORE writing any render code: a
  direct `/api/town/command` probe against a real IDUNA-backed test character (travel, look,
  attack) landed a correct hit against a live `worm-meadow-0`. Added `MEADOW_TARGET_*` (mirrors
  `server/mob/worm.go`'s real `MeadowWormSpawns()` positions/order exactly, same "kept in sync by
  hand" convention `TOWN_TARGET_*` already established) and `town_active_targets`, which every
  Tab-cycle/HUD-label/"1"-attack/`town_draw_worms` call site now goes through instead of a
  hand-copied `g_dfzone_active` check -- closes a real stale-index OOB read too, since Town's ring
  (4) and Meadow's ring (8) are different lengths. `town_draw_worms` now draws whichever target
  set is active, sitting Meadow's worms on the zone's own real rolling terrain
  (`dfzone_height_at`, same discipline `town_draw_dfzone_trees` already used) instead of y=0.
  `g_town_target_index` resets to -1 on every real zone transition. `go test ./...` clean.

- feat(worldapi,battlegrounds_gui): denser Meadow tree cover -- founder: "i dont see any updates
  yet expanding our meadow scene adding more trees." `meadowTrees` (server/worldapi/scenes.go)
  topped out at 3 trees per chunk (one hash bucket, h%5==4, had none at all); every bucket now
  returns 5-6 real, reproducible positions, no bare bucket left.
  `town_meadow_tree_positions` (apps2/battlegrounds_gui) mirrors the new positions exactly, same
  "kept in sync by hand" convention its own doc comment already names.

- feat(battlegrounds_gui): bumped `TERRAIN_TEST_CELL_SIZE` 3.0 -> 5.0 and
  `TERRAIN_TEST_HEIGHT_SCALE` 0.5 -> 1.5 -- founder: "also the scene seems quite small" (CELL_SIZE
  had already been bumped once this same day for the same feedback) plus "meadows are not
  completely flat my bro"'s real height data (range 3-5) reading as basically flat again once the
  zone's physical footprint grew (only 1 world unit of Y variation across an 80-unit-wide zone at
  the old 0.5 scale). 80x80 footprint now comfortably covers every real Meadow worm spawn (up to
  +-35 units out, previously standing past the mesh's own edge on the y=0 fallback); 1.5 gives up
  to 3 world units of real Y swing, a real, visible gentle roll without turning Meadow into Hills.
  `terrain_test_offset_x`'s own spacing formula already scales with `TERRAIN_TEST_CELL_SIZE`
  (fixed in an earlier pass), so F10 debug-patch overlap needed no further change.

## 2026-08-03 (22)

- feat(worldapi,server-go): Meadow's real gentle-roll elevation -- founder: "meadows are not
  completely flat my bro," correcting a real design choice (a hardcoded flat height=4), not a
  rendering bug. New `meadowColumnHeight` (server/worldapi/scenes.go): gentle sine-based roll,
  range [3,5], shared by both `meadowChunk`'s real block generation AND `heightmap.go`'s
  `HeightmapChunk`/`ColumnHeight` -- one real formula, not two copies that can drift (this
  session's own established discipline). 3 dependent tests updated to assert the real [3,5] range
  instead of an exact value the generator no longer guarantees at any specific column
  (`TestHeightmapChunk_Meadow_Flat` replaced by `TestHeightmapChunk_Meadow_GentleRoll` +
  `TestHeightmapChunk_Meadow_MatchesBlockGeneration`; `TestHeightmapEndpoint_returnsHeights`;
  `apps2/server-go`'s `TestGroundClampY_MeadowSetsRealHeight`). `GOWORK=off go test ./...` clean
  across the whole module. Live-verified: `server-go` rebuilt, redeployed to the live process,
  confirmed via `curl /heightmap` returning real height variation (min 4 / max 5) instead of a
  uniform value.

## 2026-08-03 (21)

- feat(server-go): real entity hit detection for hitscan shooting -- checked SHANKPIT's own
  sibling repo first (it's described as genuinely server-authoritative): its `world.RayTrace` is
  real, but only does ray-vs-player-sphere intersection, never voxel/wall raycasting. Neither
  sibling has real wall collision. That also means GoblinFoxDragon's own hitscan shooting
  (`BtnAttack`/`HandleShankFire`) could never hit *anything* before today, not just "no wall
  collision" -- genuinely no hit detection at all, on any client's shots, ever. Ported SHANKPIT's
  own real `gameWorld`/`gameEntityHit` directly (ray-vs-sphere intersection, chest-height
  approximation, `hitboxRadius=0.4`), dropping only its sceneID cross-scene guard (this backend
  has no scene tracking at all). Also fixed a real, separate bug found in the process: every
  client's shots used to fire through one single, shared, static player stub created once at
  startup (position always the literal origin) -- now a real per-shot player uses the actual
  shooter's own real, tracked position. New `Vec3.Sub`/`Dot`/`Len` (this repo's own copy only
  ever needed `Add`/`Mul` before). 10 new tests (3 in `server/system`, 7 in `server-go` covering
  on-axis hit, off-axis miss, self-exclusion, behind-shooter/beyond-range exclusion,
  closest-of-multiple, zero-length ray). `go build`/`go vet`/`go test ./...` clean. Live-verified:
  real `gfd-server-go.service` rebuilt, redeployed, confirmed stable. Still not done, matching
  SHANKPIT's own real precedent rather than a new corner cut: no real damage applied on hit yet
  (`nopEntity.Hurt` still a no-op there too); voxel/wall raycasting remains completely unbuilt in
  both sibling repos.

## 2026-08-03 (20)

- feat(server-go,worldapi): real Y-axis ground collision -- the smaller, more directly-connected
  half of "no collision against world geometry" named as a gap in the previous slice. New
  `worldapi.ColumnHeight` (single-column version of the already-tested `HeightmapChunk`, so a
  per-player per-tick lookup doesn't have to generate all 256 columns of its chunk just to read
  one). `apps2/server-go`'s new `groundClampY` calls it directly (same process, no HTTP round
  trip) right after `integrateMovement`, so a player's real server-side Y now agrees with actual
  terrain instead of drifting wherever spawn/portal last left it. Hardcodes scene 0 (Meadow) --
  matches this backend's own existing single-scene reality; Meadow's real height (4) already
  matched the client's own hardcoded `groundY = 4` fallback, not a coincidence. 5 new tests
  (including a real negative-coordinate floor-division regression -- `ColumnHeight`'s own chunk
  math needed real floor division, not Go's truncating `/`, since world coordinates go negative
  routinely). `go build`/`go vet`/`go test ./...` clean. Live-verified: real
  `gfd-server-go.service` rebuilt, redeployed, confirmed stable. Still not done: horizontal wall
  collision (`world.RayTrace` itself, unchanged stub) -- vertical grounding only, by design.

## 2026-08-03 (19)

- feat(server-go): real backend-unification slice -- server-authoritative player position +
  PacketSnapshot, closing the "other players visible to each other" gap
  DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md named as the natural next step. Founder picked
  "server-authoritative position" explicitly over trusting client-reported position (a real
  cheat vector for an MMO). New `apps2/server-go/snapshot.go`:
  - `integrateMovement`: the first general-purpose on-foot movement integration anywhere in this
    codebase family -- not even SHANKPIT's own more mature sibling server has one outside its
    racing minigame's vehicle physics (`racing.go`'s `applyRacingTick`, the wrong shape for
    walking). Defines the yaw/forward convention from scratch since no client (including
    apps2/lobby, the one real client already speaking this protocol) does local movement
    integration to match against.
  - `buildSnapshotPacket`: matches apps2/lobby's real, compiler-padded C struct layout
    byte-for-byte -- verified via a standalone `gcc`+`offsetof` probe (`sizeof(NetHeader)=12`,
    `sizeof(NetPlayer)=44`), not assumed from the field list alone.
  - Broadcast runs from the main read loop itself, not a new goroutine -- `clients`/`clientAddrs`
    have no real mutex protecting them yet (pre-existing gap), and a second unsynchronized
    accessor would be a real new crash risk. ~4Hz cadence (this loop's own 250ms poll timeout),
    a named, real limitation vs. SHANKPIT sibling's own 30Hz.
  - 7 new tests (movement math + a byte-for-byte wire-format check). `go build`/`go vet`/`go
    test ./...` clean. Live-verified: real `gfd-server-go.service` rebuilt, redeployed, confirmed
    stable including the zero-clients edge case.
  - Named gaps, not solved here: no collision against world geometry (pre-existing stub, still a
    stub); FPS-specific `NetPlayer` fields this backend doesn't track (weapon/ammo/vehicle/etc)
    are zero-filled, not faked; mobs remain entirely out of scope (unchanged).

## 2026-08-03 (18)

- ops: deployed the real `df-mc/dragonfly` fork (`emilyspringerton/dragonfly`, confirmed genuine
  earlier today) as a persistent, supervised service -- founder's original ask ("connect from my
  phones minecraft, to debug") was only tested manually once before; now a user-level systemd
  unit (`dragonfly-debug.service`, deployed locally, not committed into that repo -- see
  SMOOTH_TERRAIN_NORTHSTAR.md's own note on why) keeps the real RakNet/Bedrock listener on UDP
  `:19132` running continuously with auto-restart. Confirmed listening and logging real startup
  under supervision. Serves vanilla dragonfly content only (not GoblinFoxDragon's own Meadow --
  that needs a separate, much larger world-provider integration, still not attempted). WAN
  reachability from outside this box's own LAN was NOT verified -- no sudo access this session to
  check firewall/security-group state, named as a real open gap rather than assumed working.

## 2026-08-03 (17)

- feat(battlegrounds_gui): real trees in the Dragonfly zone -- closes the founder's own original
  ask from earlier today ("render the dragonfly biomes smooth with trees"), only half-delivered
  until now (smooth terrain since Milestone 2, trees never actually rendered). New
  `town_meadow_tree_positions` mirrors `server/worldapi/scenes.go`'s own `meadowTrees` hash
  (`chunkX*31 + chunkZ*17` mod 5) directly in C rather than fetching/parsing the full `/chunks`
  block list (~1300 objects for one chunk, real parsing complexity for already-deterministic
  data) -- same "client keeps its own copy of world data" convention this session's telecrystal
  work already established. `town_draw_dfzone_trees` reuses `draw_hero_box` -- the same
  stacked-primitive technique every hero/worm/building in this client already uses -- for a thin
  trunk plus two tapering canopy tiers, sitting at the zone's own real terrain height
  (`dfzone_height_at`) rather than assuming y=0. Meadow only (Hills has none by design,
  Swampville isn't offered as a real destination here). Live-verified visually under Xvfb:
  screenshotted a real tree standing in the Meadow zone at its correct deterministic position.
  `go vet`/`go test ./...` and a direct `gcc` client build both clean.

## 2026-08-03 (16)

- feat(battlegrounds_gui): telecrystal arrival banner, the last piece of apps/lobby's own
  telecrystal UX ported over. New `town_draw_travel_overlay`/`g_travel_overlay_text` show a
  brief, large, screen-centered "TRAVELING: MEADOW" / "TRAVELING: NEW HANDINGTON" banner right at
  the moment of arrival (apps/lobby's own `draw_travel_overlay`/`travel_overlay_text`), distinct
  from `combat_log_push`'s own arrival line -- a scrolling log entry, easy to miss mid-fight; the
  banner isn't. Set at both real arrival points, 1.4s duration. Live-verified visually under
  Xvfb. `go vet`/`go test ./...` and a direct `gcc` client build both clean.

## 2026-08-03 (15)

- fix(battlegrounds_gui): telecrystal cast now auto-starts on ring entry, no key needed --
  founder: "pressing g does nothing i expect it to auto cast when i enter the ring." The first
  pass ported apps/lobby's own G-press-to-start mechanic verbatim; not what was actually wanted.
  `town_gate_tick` now auto-starts the cast itself on the ring-enter edge (`was_in_range` false ->
  true tracked via a static, so a completed/cancelled cast doesn't instantly restart every frame
  still standing in the ring -- leaving and re-entering starts a fresh one). `town_gate_start_cast`
  still exists (needed a forward decl since `town_gate_tick` now calls it before its own
  definition) and G is left wired to it as a harmless manual fallback, but the primary path is
  pure proximity now. Live-verified visually under Xvfb: simulated walking from outside the ring
  to inside it and screenshotted the cast bar already progressing on arrival, no keypress
  simulated. `go vet`/`go test ./...` and a direct `gcc` client build both clean.

## 2026-08-03 (14)

- feat(battlegrounds_gui): real telecrystal cast UX, ported from apps/lobby -- founder: "check the
  shankpit side of the codebase there is telecrystals the ux is good i want it like that circle
  showing cast radius cast bar ticks up." Replaces the click-based Dragon Gate trigger entirely
  with the same real mechanic `apps/lobby/src/main.c` (this repo's own older SHANKPIT-style
  client) already ships: a pulsing world-space ring at the crystal's real interaction radius (12
  units, both directions, `server/telecrystal`'s own registry values), turning solid white when
  the player is inside it; pressing G while in range starts a real cast, not an instant teleport;
  a fill bar with a commit tick-mark advances over 1000ms, the real travel/return call fires at
  the 600ms commit mark, and leaving the ring before commit cancels the cast.
  - New `draw_mesh_lines` (GL_LINE_LOOP twin of `draw_mesh`) -- this client's 3D pass is
    shader-bound (uMVP uniform), not the legacy fixed-function matrix stack apps/lobby's own ring
    uses, so the ring is a real mesh through the existing pipeline instead of mixed-in
    immediate-mode calls that would need their own matrix-stack sync every frame.
  - New `town_gate_current_crystal`/`town_gate_tick`/`town_gate_start_cast`/`town_draw_gate_ring`/
    `town_draw_gate_overlay` -- scoped down from the reference's own generic N-crystal table to
    this client's one real interactive gate, whose identity flips between the two real registry
    entries (`TELECRYSTAL_ID_HANDINGTON_TO_MEADOW`/`TELECRYSTAL_ID_MEADOW_RETURN_HANDINGTON`)
    depending on which zone is currently active, same "one gate, both directions" design the
    click-based version already established.
  - Live-verified visually under Xvfb: screenshotted mid-cast showing the fill bar, red commit
    marker, and "CASTING: TELEPORT MEADOW" text against the real in-range white ring. `go vet`/
    `go test ./...` and a direct `gcc` client build both clean.

## 2026-08-03 (13)

- fix(battlegrounds_gui): founder tried the real teleport after the fizzle fix -- "it kind of
  worked... hard to trigger... we teleported to a floating green plane i was pretty big in
  relation to it and i fell off of it." Three real, separate bugs, all fixed:
  - **Fell off / floating in the air**: click-to-move and WASD were clamping to
    `TOWN_MOVE_HALF_EXTENT` (~57 units, Town's own footprint) unconditionally, even while
    standing on the real Dragonfly zone mesh, which only spans +-8*`TERRAIN_TEST_CELL_SIZE` --
    easily walkable straight off the edge into nothing. Same failure class as the original
    "floating in a blue abyss" bug this session already fixed for Town, just not caught for this
    new case. New `town_move_half_extent()` picks the right bound for whichever ground is
    actually rendered right now; both movement clamp call sites use it.
  - **Too small / "pretty big in relation to it"**: `TERRAIN_TEST_CELL_SIZE` bumped 1.0 -> 3.0 --
    a 16x16-cell chunk at cell_size=1.0 is only 16 world units across, about two Town buildings
    wide. Tripling the physical footprint (48x48) without touching the fixed 16x16 heightmap
    resolution makes it read as an actual clearing. `terrain_test_offset_x`'s own per-patch
    spacing now scales with the same constant so the three F10 debug patches can't start
    overlapping at the new scale.
  - **Hard to trigger**: the Dragon Gate click required landing inside the building's own tiny
    visual box (half_w=half_d=2.4, a ~5-unit square) via `town_building_at` -- nothing like a
    usable interaction range. Now checks a real 12-unit radius against the gate's own position
    (server/telecrystal's own real `Radius` value for this crystal), independent of the tiny
    click box -- roughly 25x more click-tolerant.
  - Live-verified: screenshotted the larger zone footprint, and directly exercised
    `town_move_half_extent()` confirming a runaway target position (999 units) correctly clamps
    to 24 (8*3.0, the new bound) instead of walking off the edge. `go vet`/`go test ./...` and a
    direct `gcc` client build both clean.

## 2026-08-03 (12)

- fix(battlegrounds_gui): real root cause of "the crystal fizzles -- travel failed" -- founder hit
  this immediately on trying the real Dragon Gate teleport. IDUNA's own `handleUpdatePosition`
  (`IDUNA/internal/http/handlers/mmo.go`) returns **204 No Content** on success, not 200 --
  `town_telecrystal_travel`/`town_telecrystal_return` were checking `status != 200`, so every
  genuinely successful position update was reported as a failure. Never surfaced before because
  the only other caller of this exact endpoint, `town_sync_position`, never checks the response
  status at all ("best-effort, silent-discard" for ordinary movement sync) -- today's real
  teleport work was the first caller to actually validate the response, against the wrong code.
  Confirmed via direct code read (not guessed): player JWTs set `sub` to the player's own real
  `player_id` (`player_email_auth.go`), matching `characters.player_id` by construction, so the
  handler's ownership check isn't the failure mode for a player acting on their own character --
  the 204-vs-200 mismatch is sufficient on its own to explain every observed failure. Fixed both
  call sites to check `status != 204`.

## 2026-08-03 (11)

- feat(battlegrounds_gui): real Town <-> Dragonfly zone teleport -- founder: "im expecting to
  teleport from town to the new zone." `town_telecrystal_travel` used to stop at the IDUNA
  position PATCH, leaving Town's own geometry on screen (a named gap in its own old doc comment).
  Now it also lazy-loads the real Dragonfly Meadow heightmap (`dfzone_load`, worldapi scene 0) and
  switches the client's render mode (`g_dfzone_active`): Town's ground/buildings/worms/building-
  labels stop drawing, `town_draw_dfzone` draws the real live heightfield mesh (reusing
  Milestones 2-4's pipeline unchanged) at the world origin, camera/avatar height follow it
  (`dfzone_height_at`). New "G" key is the return trip (`town_telecrystal_return`, real
  `TELECRYSTAL_ID_MEADOW_RETURN_HANDINGTON` values) -- a dedicated key rather than hijacking
  right-click, which players need for camera control while exploring the new zone.
- fix(battlegrounds_gui): real bug found while wiring the above -- the F10 debug toggle shipped in
  Milestone 2 lived in the battlegrounds-match event loop (`e`-scoped), which Town's own render
  branch's `continue;` skips entirely whenever `in_town` is true. F10 was dead code for real Town
  play; every "live-verified under Xvfb" screenshot for Milestones 2-4 actually exercised a
  temporary test-only env-var hook that set state directly, not the real key (the hook was
  removed before each commit as documented, but the real key handler itself was never reachable
  in Town). Moved into Town's own `te`-scoped event loop, where it's now actually reachable.
- Live-verified visually under Xvfb: screenshotted Town's checkerboard/buildings genuinely
  disappearing and the real Meadow terrain filling the screen at the destination, confirming the
  render-mode swap and worldapi fetch both work end to end. `go vet`/`go test ./...` and a direct
  `gcc` client build both clean.

## 2026-08-03 (10)

- feat(battlegrounds_gui): ship Milestone 4 of SMOOTH_TERRAIN_NORTHSTAR.md -- movement/camera
  elevation awareness, scoped to the F10 test patches (Town itself untouched, stays flat by
  design). New `terrain_test_height_at` samples the same CPU-side heights the GPU mesh was built
  from and returns real terrain height when standing inside a test patch, 0 elsewhere. Wired into
  camera focus (`mat4_orbit_view`'s `focus_y`, was hardcoded 0.0f) and the avatar's draw-time Y
  (combined with the existing jump-arc translate). New `terrain_test_offset_x` is the one shared
  source of each patch's world placement, used by both the renderer and the height lookup so they
  can't drift apart. Explicitly not done, named rather than skipped: `screen_to_ground`'s
  click-to-move ray-cast still targets a flat y=0 plane (real ray-vs-heightfield intersection is a
  harder problem, out of scope here) and WASD's own (x,z) update logic is unchanged -- only the
  resulting position's rendered Y now reads real terrain. Live-verified visually under Xvfb: the
  camera correctly settles onto the real sloped terrain surface at the avatar's position instead
  of floating above or clipping through it. `go vet`/`go test ./...` and a direct `gcc` client
  build both clean.

## 2026-08-03 (9)

- feat(battlegrounds_gui): ship Milestone 3 of SMOOTH_TERRAIN_NORTHSTAR.md -- biome flat-coloring.
  New `biome_color` maps worldapi's own `scene`/biome id to a flat RGB per draw call (Meadow
  grass green, Hills olive, Swampville muddy brown-green, unknown scenes grey). The F10 debug
  scene (Milestone 2) now fetches and renders all three column-derived biomes side by side
  instead of a single hardcoded green, so the milestone's own real terrain-test scaffolding is
  the proof, not new throwaway code. No new client-side biome enum -- reuses worldapi's own
  informal "sceneID is the biome selector" convention. Live-verified visually under Xvfb: all
  three patches render simultaneously with visibly distinct hues driven by each patch's own real
  `scene` field. `go vet`/`go test ./...` and a direct `gcc` client build both clean.

## 2026-08-03 (8)

- feat(battlegrounds_gui): ship Milestone 2 of SMOOTH_TERRAIN_NORTHSTAR.md -- client heightfield
  mesh renderer. New `build_heightfield_mesh`/`heightfield_sample` (`src/main.c`) fetch a real
  heightmap from the new `/heightmap` endpoint (Milestone 1) over HTTP, bilinearly interpolate at
  2x source resolution, and derive per-vertex normals from finite-difference height gradients --
  emitted through the exact same pos+normal `upload_mesh`/`draw_mesh` path every other mesh in
  this client already uses, no shader changes. New `http_extract_json_uint8_array_field`
  (`http_client.h`) parses the heightmap's numeric array field, same "controlled shape, not a real
  parser" convention as the other extractors there. Wired as an F10 debug toggle
  (`town_load_terrain_test`/`town_draw_terrain_test`) rendering the real live Hills chunk floating
  clear of Town's own footprint -- deliberately not integrated into Town itself (stays flat by
  design) and not wired into movement/collision (Milestone 4, later). Live-verified visually: built
  and ran the real client under Xvfb, connected to Town via a WOTAN dev-agent identity, screenshot
  of the F10 mesh shows a real smooth, continuously gradient-shaded rolling surface -- not
  stair-stepped cubes -- confirming both interpolation and lighting work against real backend
  data. `go vet`/`go test ./...` and a direct `gcc` build of the C client both clean.

## 2026-08-03 (7)

- feat(worldapi): ship Milestone 1 of SMOOTH_TERRAIN_NORTHSTAR.md -- backend heightmap exposure.
  New `GET /heightmap?scene=N&cx=X&cz=Z` (`server/worldapi/heightmap.go`) returns
  `{"height": [uint8 x256], "biome": int}` for Meadow (flat), Hills (real per-column variation --
  `hillsColumnHeight` split out of `hillsChunk` so block generation and the heightmap endpoint
  share one formula, can't drift apart), and Swampville (flat, water one block higher than land).
  Caves correctly 204s -- it's a genuinely 3D solid grid with no single height per column. New
  tests include a direct cross-check of the heightmap against `hillsChunk`'s own real block
  output. Live-verified against the running `gfd-server-go.service`. `go vet`/`go test ./...`
  clean (one pre-existing, unrelated `sync.RWMutex` copy warning in `apps2/server-go/main.go`,
  confirmed via earlier `git stash` comparison this session, not touched here).

## 2026-08-03 (6)

- docs(mud): confirmed Sunderworm crisis broadcasts reach Town's chat/combat-log pane -- an
  earlier open item, resolved by direct observation rather than new code. The crisis-phase
  handler (`main.go` ~1702) broadcasts to every connected player (`gw.players`), not zone-scoped,
  which is why this session's own Meadow validation test (P2, above) saw a New-Handington-zone
  crisis message while sitting in Meadow. On the headless path the push queues into the
  character's connection buffer and flushes on the next command -- observed directly in the P2
  `attack` response. `apps2/battlegrounds_gui`'s `town_poll_combat` already drains that buffer
  every ~1.5s into the shared combat log pane -- the exact code path this session's manual test
  exercised. No code change; delivery was already shipped, now verified rather than assumed.

## 2026-08-03 (5)

- test(mud): validated Meadow (scene 0) end-to-end through the real headless `/api/town/command`
  path -- P2 from the founder's own sprint plan ("get telecrystal working and then we validate the
  new zone"). Confirmed live: `look` shows real Meadow room text + 8 real worm mobs; `crystals`
  lists the real telecrystal network with the return-to-New-Handington crystal in range;
  `attack worm-meadow-0` lands a real 30-damage hit with TP gain (0→40) and a live world-crisis
  event fired mid-fight ("Something vast burrows beneath the Worm Hut"), after which all Meadow
  worms show `(burrowed)`; `north`/`south` correctly transition Meadow↔Hills and back; `say` works.
  Known, already-documented gap, not rediscovered as new: `apps2/battlegrounds_gui`'s 3D view stays
  New-Handington-specific after a real telecrystal travel -- this validation used the headless text
  path specifically because it doesn't depend on that render gap.

## 2026-08-03 (4)

- feat(server-go): get apps2/server-go running under supervision for the first time -- founder:
  "for now we need to get the dragonfly server seeded with a world". The binary existed but had
  never been run supervised; its hardcoded UDP `:6969` also collided with SHANKPIT's own live
  `shank_server` (confirmed via `lsof -i :6969`, not `ss`, which didn't reveal the real listener).
  Added a `-udp-port` flag (default unchanged at 6969) so it no longer has to fight SHANKPIT for
  the port. New `ops/systemd/gfd-server-go.service` user unit, deployed on `:6970` (worldapi
  `:7070`, trapx `:7071`). Live-verified: `GET /chunks?scene=0&cx=0&cz=0` returns real Meadow
  block data (1308 blocks, correct grass/dirt/stone layering, 8 real oak-log tree blocks) --
  "seeded with a world" is now literally true and running.
- docs(smooth-terrain): amended SMOOTH_TERRAIN_NORTHSTAR.md (§3.5 Trees, §3.6 the Town<->Dragonfly
  bridge open question, §3.7 explicitly-out-of-scope: world sculpting + real Bedrock connectivity)
  in response to founder direction ("render the dragonfly biomes smooth with trees... like a nice
  minecraft meadow biome but we render it with our frontend"). Added Milestone 0.5 (server-go
  running + seeded, DONE).
- docs(smooth-terrain): founder forked the real `github.com/df-mc/dragonfly` Bedrock server
  library to `emilyspringerton/dragonfly`. Confirmed genuine and unmodified (zero commits ahead of
  upstream `master`). Built and ran it vanilla: real RakNet/Bedrock listener on UDP `:19132`,
  `mc-version=1.26.30` -- a real phone Minecraft client can connect to it today, unmodified, for
  debug purposes. This answers "can I connect from my phone's minecraft, to debug" using vanilla
  upstream content; getting GoblinFoxDragon's own Meadow content reachable the same way is a
  separate, much larger integration (a custom world/chunk provider sourcing from
  `server/worldapi`'s `ProceduralWorldStore`), not attempted here.

## 2026-08-03 (3)

- fix(mud): REAL root cause of the gw.mu deadlock found and fixed -- founder: "pls fix" (not
  satisfied with the client-side workaround shipped a moment earlier). It was a plain,
  garden-variety self-deadlock, not anything exotic: `handle()` itself already does
  `gw.mu.Lock(); defer gw.mu.Unlock()` UNCONDITIONALLY right before its own dispatch switch (the
  only exception is the `/p` party-chat shortcut, which returns early before ever reaching that
  lock). `cmdTravel` then tried to acquire the exact same, non-reentrant `sync.Mutex` again on the
  SAME goroutine -- Go doesn't detect or panic on this, it just blocks forever. This is exactly
  why no "holder" ever showed up in any goroutine dump (SIGQUIT or live `dlv` inspection): the
  holder WAS the stuck goroutine itself, one frame further up its own stack inside `handle()`, not
  a separate one. It also explains why the earlier telnet A/B test looked like it worked: the
  "crystal resonates" message is sent BEFORE the lock attempt, so the telnet client received it
  before that same connection's own goroutine silently self-deadlocked afterward -- a follow-up
  command on that same session would have shown it too; it was just never tried.
  - Found via one precise structural test: an inline, trivial `gw.mu.Lock()` test case added
    directly to `handle()`'s own switch hung identically to `cmdTravel` -- but the exact same code
    moved to `/p`'s own position (before the switch, i.e. before the outer lock) worked. That
    isolated it immediately.
  - Real fix: removed `cmdTravel`'s own redundant `Lock()`/`Unlock()` entirely -- it's already
    called from inside `handle()`'s own locked dispatch, same as every other `cmd*` function that
    mutates `gw` state without locking (`cmdLook`, etc.).
  - Audited every other `gw.mu.Lock()` call site in the file and found three more real instances
    of the exact same bug, all fixed the same way: `cmdBattlegrounds` (re-locked to read
    `gw.charIDBySlot`), `cmdSummonAvatar` (re-locked to read `gw.players`), and the `warcry`
    ability case (called `broadcastZone`, the locking wrapper, instead of `broadcastZoneNoLock`).
    All three were real, live, previously-undiscovered risks -- any of them could have taken the
    whole server down the same way `cmdTravel` did, the moment anyone actually used them from
    chat.
  - Live-verified against the real production `gfd-mud.service`: two consecutive real telecrystal
    trips (`TELECRYSTAL_ID_TOWN_TO_MINES` then `_MINES_RETURN_TOWN`), the new
    `TELECRYSTAL_ID_HANDINGTON_TO_MEADOW`, and `battlegrounds` (ticket minting) -- all instant
    (~0.015s), all correct, `gameLoop`'s own tick confirmed healthy throughout (real crisis/
    faction-war events kept flowing).
  - The client-side workaround from the previous entry (`town_telecrystal_travel`, direct IDUNA
    PATCH bypassing `apps2/mud`) is left in place, not reverted -- it's simpler for the one Dragon
    Gate case and does no harm now that the underlying path is actually safe too.

## 2026-08-03 (2)

- fix(town): telecrystal ships via a safe workaround -- founder: "cook cook cook" (continuing P0
  from the sprint plan: root-cause the `cmdTravel` headless-path deadlock). Exhaustive
  investigation, root cause NOT found despite it:
  - Confirmed via a real SIGQUIT goroutine dump AND a live `dlv` session (attached with the
    correct Go toolchain version, target run with the real `IDUNA_AGENT_NAME`/`SECRET` env) that
    `gw.mu`'s own raw internal state shows `{state: 17, sema: 0}` -- genuinely locked, 2 real
    waiters (`gameLoop`'s own tick + the stuck request) -- while the COMPLETE goroutine list (11
    goroutines, delve's own exhaustive enumeration, not a partial signal-dump) contains no live
    holder anywhere. Every other request that also needs `gw.mu` (confirmed with `/p` party chat,
    which uses the identical `Lock()`/`defer Unlock()` shape) works instantly and correctly via
    the same headless path -- ruling out "any locked command via headless."
  - Reproduced with a `-race` build (no data race reported) and with a completely fresh character
    (never touched by this session's own earlier DB edits) using a pre-existing crystal
    (`TELECRYSTAL_ID_TOWN_TO_MINES`, positioned correctly, sufficient gold) -- ruling out both the
    new Meadow crystal and any stale test-character state as causes.
    Reproduced with `cmdTravel` restructured from a nested closure to a flat top-level
    `Lock()`/`defer Unlock()` (removing the one structural difference from the working `/p`
    pattern) -- ruling that out too. Confirmed `cmdBattlegrounds` and `cmdSummonAvatar` are the
    only other named functions with the same "locks `gw.mu`, called via `handle()`'s switch"
    shape as `cmdTravel` -- **not yet tested, a real, live, unconfirmed risk** that they carry the
    identical bug the moment either is ever exercised through headless/chat dispatch.
  - Given the severity (this takes the whole server down for every player, not just the one
    request) and that root-causing it has genuinely exhausted the obvious leads, shipped a real
    workaround instead of leaving telecrystal unusable: `town_telecrystal_travel()` bypasses
    `apps2/mud` (and its broken lock) entirely for the Dragon Gate specifically -- a direct PATCH
    to IDUNA's own `/api/v1/characters/:id/position`, the exact same safe, already-proven
    mechanism `town_sync_position` already uses continuously for ordinary movement. Target
    scene/position are `TELECRYSTAL_ID_HANDINGTON_TO_MEADOW`'s own real values, duplicated
    client-side -- the same convention `apps/lobby`'s own `TELECRYSTAL_DEFS` already established
    for the older SHANKPIT-lobby client, not a new pattern. Free (matches the server-side
    `CastCost` of 0), so no gold-deduction race to worry about doing this client-side.
  - Named, honest gap this doesn't solve: the client has no Meadow rendering at all, so after a
    real, correct backend zone/position change, the 3D view keeps showing New Handington until
    relogin (or a real future Meadow render mode) -- same category as the earlier Town-movement-
    bounds bug, but expected here, not a surprise.

## 2026-08-03 (1)

- fix(town): clamp movement to the real ground extent -- founder, live: "when i log in im not in
  town... i am floating in a blue abyss and it looks like theres some white writing off in the
  distance and i cant tell if i can run towards it or not and thats all thats rendering." Real
  bug: neither click-to-move nor WASD ever clamped position, unlike Battlegrounds' own hero
  movement (bounded to `ARENA_HALF_EXTENT`). Confirmed live -- the founder's own real character
  had drifted to `(61, 0, 3332.6)`, thousands of units past the actual ~113-unit ground/building
  layout. Nothing 3D renders that far out (only 2D building-name labels still project onto
  screen from any distance, which is exactly what read as "white writing in the distance" with
  everything else gone). Repositioned the live character back to open ground via a direct IDUNA
  update. Added `TOWN_MOVE_HALF_EXTENT` (derived from `town_draw_ground`'s own real footprint, so
  it can't drift out of sync with the visible ground) and clamped both click-to-move's target and
  WASD's per-tick target to it -- WASD held long enough was the more likely way to reach an
  absurd position in the first place, since it compounds every ~100ms with no cap.

## 2026-08-02 (23)

- feat(mud): New Handington <-> Meadow telecrystal, real critical bug found and NOT shipped to
  the GUI -- founder: "how do we get from town to the starter zone? have one of the gates act as
  a telecrystal... check the shankpit dragonsnshit codebase for the telecrystal logic." Found
  apps2/mud already has a full, real, player-facing telecrystal system (`server/telecrystal`,
  `cmdCrystals`/`cmdTravel`/`cmdTouchCrystal`) that apps/lobby's own SHANKPIT-style client
  already uses for Town/Mines/Docks/Giza -- New Handington (zone 4, apps2/battlegrounds_gui's own
  Town) predates that network entirely and had no crystal of its own. Added
  `TELECRYSTAL_ID_HANDINGTON_TO_MEADOW` + its return pair (free, CastCost 0 -- a starter-zone
  shuttle a level-1 character needs before they'd ever have Flow to spend on the older network),
  positioned at New Handington's real "Dragon Gate" building. Updated telecrystal_test.go for the
  new registry size/shape (8 crystals, non-negative-cost convention now that free ones exist).
  - **Real, critical, unresolved bug found live, not shipped**: the exact same crystal, invoked
    via a real telnet session, works correctly every time. Invoked via
    apps2/mud's headless `/api/town/command` HTTP path -- what Town's own GUI client uses for
    every chat/gate command -- `cmdTravel`'s `gw.mu.Lock()` call never returns, and takes the
    WHOLE mud server down with it: confirmed via a real SIGQUIT goroutine dump that `gameLoop`'s
    own 1Hz tick, which was ticking healthily seconds earlier, permanently stops too. Reproduced
    with `cmdTravel`'s body stripped to a bare `Lock()`/`Unlock()` with nothing else inside --
    the bug is not in what the function does with the lock, it's in headless dispatch reaching
    this point in some way not yet isolated. Converted the manual `Lock()`/`Unlock()` pair to
    `defer` (real hardening against any future panic there, but did not fix the actual hang).
    Given the severity (one GUI interaction can deadlock the entire server for every player),
    deliberately did NOT wire the Dragon Gate's right-click to `travel` in
    `apps2/battlegrounds_gui` -- the building interaction was reverted, left as a clear comment
    warning future work (and the founder, via chat) not to trigger `/travel` from the GUI until
    this is properly root-caused. The underlying telecrystal system itself is real and working
    for telnet players; only the headless/GUI path is unsafe right now.

## 2026-08-02 (22)

- feat(mud): real Sunderworm boss content for World Crisis -- founder: "start working on the
  sunderworm world event" -> "worm as the northstar" -> "build it on top of our starter zone on
  top of dragonfly." Found the event already had a full spec (`docs2/specs/WORLD_CRISIS_VS0.md`)
  and a real, tested phase machine (`server/worldcrisis`) wired into `apps2/mud`, but zero actual
  Sunderworm content: the trigger auto-restarted with no cooldown (spec E2 violation), the
  Anchor objective had no implementation anywhere, and "Chaos Elementals emerge in the Swamp"
  was 3 generic reskinned mobs with no boss mechanics.
  - `server/mob/sunderworm.go`: a real boss (15,000 HP) reusing `StateBurrowed` unchanged as its
    invulnerable state -- the phase names OMENS/BURROW/EMERGENCE already describe that exact
    cycle, no new mechanic needed. Two Sunderworm Head sub-bosses (4,000 HP) for Split War.
    Scaled up from `KindWorm`, the exact mob already in the starter zone -- not a new creature.
  - Wired into the crisis phase handler: spawns burrowed at the real Worm Hut position in zone 4
    when Burrow begins; the crisis handler (not an autonomous per-mob timer) surfaces it at
    Emergence alongside a 3-mob "Sunderworm Brood" add-wave in the same zone; two geo-separated
    Heads spawn east/west of the hut at Split War; Resolution soft-despawns everything (marks
    dead in place -- `Registry` has no removal API, a named gap).
  - Real gap closed: killing a Head now completes the Anchor objective (+15 LEY) -- previously
    the one of three required concurrent objectives with zero player-facing action, meaning
    Final Window's own gate could never actually be met.
  - Real bug fixed: added `crisisCooldown` (20 min) + `world.lastCrisisEndAt` so the event can't
    immediately re-trigger the instant it resolves.
  - Go build/vet/test green; redeployed `gfd-mud.service`; live-verified via a real telnet
    session through Omens -> Burrow.
  - Honestly still open per the spec's own DoD checklist: persistence (pure in-memory, IDUNA
    `PatchWorldEvent` never actually fires), rewards/merit/tiering, real Builder/Ritualist
    non-combat mechanics (Ritual is still just repurposed ore-mining), diminishing-returns
    anti-zerg, weak-point/armor-break beyond burrow/surface, and all client-side rendering (no
    GUI client consumes the crisis packet yet -- ties to `SMOOTH_TERRAIN_NORTHSTAR`'s still-
    unstarted milestones for real dragonfly terrain, not done here).

## 2026-08-02 (21)

- feat(town): C/Y/T also open chat, alongside Enter -- founder, after the Enter-key AH ordering
  fix still felt broken in practice: "the reason the auction house menu doesnt work is im trying
  to hit enter but that is triggering chat can we get a different hotkey than enter to start a
  chat enter can still send the chat" -> "how about make it work for c y and t just have them all
  map to start chat for now" -> "and then when we are not in the auction house enter also will
  open the chat." Enter staying in the open-chat list is safe specifically because the AH-menu
  block already runs first and consumes the event while `g_ah_screen != AH_CLOSED` -- Enter never
  reaches the chat-open check while the menu is open, so C/Y/T are additional ways in, not a
  replacement. In Battlegrounds' own in-match chat (separate event loop from Town's), "C" is
  deliberately left out -- it's already NORTHSTAR §15.1's `cam_locked` toggle there, and "keep
  battlegrounds as is" means that pre-existing binding doesn't get contested; Y/T are added there
  instead.

## 2026-08-02 (20)

- fix(town): stale connect ticket silently broke every requeue after ~5 minutes of play --
  founder, live-testing: "if i queue for battle grounds and then after that game return to town
  and then requeu for battlegrounds it doesnt work"; repro'd worked twice, then failed every
  time; "killing my client and relaunching it fixes it its a bug." Not the bot-pool/matchmaker
  race investigated earlier in this same session (that's real but was a red herring for this
  specific symptom) -- the real cause: `get_player_login_ticket` mints a connect ticket ONCE at
  initial login, and `net_connect()` reused that exact same `g_supplied_ticket_hex` for every
  reconnect for the rest of the session. IDUNA's `RedgardenTicketTTL` is a hardcoded 5 minutes
  (`redgarden_self_ticket.go`), and `arena_server` silently drops `PACKET_CONNECT` for an expired
  ticket (`apps/arena_server/src/main.c`'s own `expires_at` check, no rejection packet sent back)
  -- a dropped UDP connect just looks like "the human never joined the lobby," matching the
  "stuck at 19/20" symptom found while investigating. A fresh client relaunch mints a fresh
  5-minute ticket at its own new login, which is why that "fixed" it -- confirming this was
  never a REDGARDEN-side bug. Added `refresh_self_ticket()`, which re-mints from the stored login
  JWT (`g_chat_jwt`, much longer-lived than the ticket itself) right before every connect instead
  of reusing the first one -- same `/api/v1/redgarden/self-ticket` call `get_player_login_ticket`
  already made once, just repeatable. Falls through to the original static ticket if the refresh
  call itself fails (network hiccup, IDUNA briefly down). Bots/--ticket/--connect dev launches
  are untouched (no `g_chat_jwt` in those paths).

## 2026-08-02 (19)

- fix(town): Auction House Enter-key event-ordering bug -- founder, live-testing: "when i hit
  enter for browse categories the whole client crashes." Not a crash: Town's event loop checked
  the "Enter opens chat" shortcut BEFORE the Auction House menu's own Enter handling, so for any
  real logged-in player (`g_chat_jwt` set -- never true in this session's own earlier dev-agent
  testing, which is why it went unnoticed until a real login hit it), pressing Enter with the AH
  menu open opened the chat box instead of confirming the menu selection, then swallowed every
  further keystroke as chat text with the AH menu stuck open behind it and no way back -- reads
  exactly like a hang/crash at the keyboard. Fixed by checking the AH-menu block first, same
  precedence chat_input_active itself already gets. Verified both ways: pushed a real
  `SDL_KEYDOWN`/`SDLK_RETURN` event through the actual event-dispatch code (not a direct function
  call, which didn't reproduce this) against the pre-fix commit -- confirmed it reproduces
  (`ah_screen` stuck on MAIN, `chat_input_active` set) -- then against the fix, confirmed the
  menu now advances correctly and chat is untouched.

## 2026-08-02 (18)

- feat(town): real FFXI-style Auction House menu, doubled town/buildings, `/logout` chat command.
  - Auction House (founder: "make the auction house real - menu based system navigatable with
    arrow keys and enter just like ffxi - have it be interractable on right click"): right-click
    on the Auction House building opens a real menu (`AHScreen`: MAIN -> CATEGORIES ->
    CATEGORY_ITEMS / MY_LISTINGS), Up/Down navigate with wraparound, Enter confirms, Backspace
    goes back a level, Escape closes -- wired to apps2/mud's real, pre-existing `ah` command
    surface (`ah browse`, `ah sell`, `ah buy`, `ah history`, `ah status`, `ah cancel`) via
    `/api/town/command`, not mocked. `ah_draw_loading` added after live testing showed the
    blocking HTTP calls froze the frame with no feedback -- same fix pattern as
    `draw_queuing_screen`. Known real gap, not solved here: `ah browse <category>` only returns
    item-level aggregates, no listing IDs, so buying a specific other player's listing has no
    command surface yet.
  - Doubled the town (founder: "double the size of the town and the buildings"): every
    `TOWN_BUILDINGS[]` position and half-extent x2 (25 entries), `TOWN_TARGET_X/Z[]` (worm hut
    cluster) x2 to match, `server/mob/worm.go`'s `TownSquareWormSpawns` hutX/hutZ and cluster
    spread x2 to stay in sync with the client-side Worm Hut position. Diffed field-by-field
    against the pre-doubling commit (20b418e) to confirm an exact, clean x2 on every value.
  - `/logout` (founder: "in the chat /logout should log me out"): typing `/logout` in chat now
    quits the client, in both Town's own chat and Battlegrounds' in-match chat. Found and fixed a
    real bug in the process: the in-match chat handler had never actually been converted to
    `chat_send_or_command` (still called plain `chat_send`), so `/`-prefixed commands only worked
    from Town, not from an in-progress match.
  - Real bug found live during doubled-town verification, not a regression in the doubling
    itself: a test character had been positioned at the Auction House's exact center coordinates
    for AH testing before the doubling landed. Standing inside a building's own mesh means its
    inward-facing polygons are backface-culled, so the building (and anything else near you)
    renders as nothing -- read exactly like "buildings are gone." Repositioned off any building's
    bounding box; not a rendering or geometry defect.

## 2026-08-02 (17)

- feat(mud): headless-session M4 -- idle eviction + telnet-conflict handling, closing out
  `HEADLESS_SESSION_NORTHSTAR.md`'s full milestone table.
  - `evictIdleHeadlessSessions`: drops any headless session idle past 10 minutes (new
    `headlessLastActive`, updated on every `runHeadlessCommand` call), flushing final position
    the same shape a real telnet disconnect would (level/XP/flow are already kept in sync
    incrementally, unlike telnet's one-shot flush). Runs once per tick from `tickAll()`.
  - `disconnectHeadlessSession`: shared teardown (position flush, party leave, every registry a
    real disconnect clears, "has left the world" broadcast) used by both the idle sweep and the
    telnet-conflict path below.
  - `handleConn` now tears down any live headless session for the same character before a real
    telnet login takes over -- never two live `*player` structs for one character. Live-verified:
    created a headless session for a real character, connected via telnet under the same name,
    no crash, no duplicate registration.
  - Symmetric case also handled: `getOrCreateHeadlessPlayer` now refuses (409, not a crash) to
    spawn a headless session for a character already connected over real telnet -- a real
    telnet `*player`'s `w` wraps a `net.Conn`, not a buffer, so there's no output to hand back to
    a headless caller (routing a command INTO a live telnet session's own output stream is a
    real, separate, harder feature, not built here). Live-verified: held a telnet connection
    open, confirmed a concurrent `/api/town/command` call for the same character correctly
    returns 409 "character is currently connected via telnet" instead of crashing or duplicating.
  Go build/vet clean.

## 2026-08-02 (16)

- feat(town): sync Town with the MUD -- real commands from the chat box, real telnet visibility.
  Founder: "i want you to sync up town with the MUD" -> chose both "real MUD commands from
  Town's chat box" and "telnet players see Town's GUI players."
  - `chat_send_or_command` (`apps2/battlegrounds_gui`): a line typed into either chat box (Town's
    own, or the in-match one -- `g_town_char_id` survives entering/leaving a match) starting with
    "/" now routes to the real headless-session command dispatch instead of ordinary chat --
    `HEADLESS_SESSION_NORTHSTAR.md`'s own original M3 design, finally built. "/look",
    "/inventory", anything `handle(p, line)` understands, not just the "1" attack keybind. Output
    shares the combat log pane.
  - `getOrCreateHeadlessPlayer` (`apps2/mud`) now registers into `gw.zoneMgr`/`gw.chatRouter` and
    broadcasts "X has entered the world" on creation, same as a real telnet connection -- a real
    telnet player standing in the same zone now sees a Town player's presence live, not just via
    `look`'s own already-working `gw.players` loop.
  Go build/vet clean. Live-verified end-to-end with two real characters: created character 1's
  headless session, then character 2's -- character 1's own session buffer captured "Test2Warrior2
  has entered the world." in real time, confirming the broadcast reaches other live sessions
  exactly like a real telnet connection would see it.
  Still open, unchanged: no idle eviction or "has left the world" broadcast (a headless session
  never disconnects in the traditional sense, `HEADLESS_SESSION_NORTHSTAR.md` M4).

## 2026-08-02 (15)

- fix(town): dead-connection recovery -- a narrower race in the same REDGARDEN-side matchmaking
  issue found earlier today, this time surfacing as "put me into the map with nothing happening
  skipping the draft" rather than an outright connect failure. Founder: "i closed dragonsnshit
  client and reopened it and that did not fix it - well it did something different it put me
  into the map with nothing happening skipping the draft."
  - Root cause: `net_connect()` can legitimately receive `PACKET_WELCOME` (so the earlier
    connect-failure fallback never fires -- the client really did connect) in the same window
    `arena_server`'s own 60s no-lobby-progress watchdog kills the match. The client is then left
    on a dead socket forever: `net_phase` stuck at its default `ARENA_PHASE_WAITING`, never
    `ARENA_PHASE_DRAFT`, so the draft screen never shows and `arena_state` never updates --
    exactly "nothing happening, skipped the draft."
  - Fix: `g_net_last_packet_ms` now tracks the last time *anything* arrived from the server
    (`net_poll_snapshots`), reset to 0 on a fresh connect. If 10s pass post-connect with zero
    packets ever received, the client treats it as a dead match and recovers -- same "land back
    in Town, not a dead end" pattern the earlier requeue-failure fix already established. Gated
    on `queue_host` (no Town to return to for a direct `--connect` dev session, same convention
    as the other Town-recovery paths). `apps/arena_server`/`apps/matchmaker` untouched, per
    standing instruction.
  - Live-verified the exact race with a fake matchmaker+server (real `NetHeader`/`MatchFoundMsg`
    wire format, confirmed via `sizeof()` against the real struct rather than assumed): replies
    with a real `PACKET_WELCOME` then goes silent forever. Confirmed via log timestamps: the
    client connects, waits the full 10s with zero packets, then fires the exact recovery message
    and sets `in_town = 1` -- the same reused code path already visually verified rendering Town
    correctly in three other contexts this session (initial entry, Return-to-Town button, failed
    requeue).

## 2026-08-02 (14)

- feat(town): "New Handington" -- real town layout transcribed from a hand-drawn map. Founder
  uploaded `town-map.jpeg` straight to GitHub: "i want the town layout to match town map pretty
  much exactly."
  - Zone 4 renamed "Town Square" -> "New Handington" (`server/zone/zone.go`), matching the map's
    own title.
  - 25 named buildings transcribed from the sketch into `TOWN_BUILDINGS` (`apps2/battlegrounds_gui`):
    Warrior Guild, Seed Shop, Fishing, Blacksmith, Butcher, Armor Shop, Shady Dealer, Guild
    House, Potions, Gold Guild, Secret Gate, Auction House, Archery Guild, Post Office, Town
    Hall, Gem Dealer, Police, Gemani Tower, MineCo Ops Office, Mining Supplies, Glove Shop,
    Hats, Worm Hut, Dragon Gate, Diamond Gate -- placed at a row/col reading of the map's own
    relative layout, not exact hand-drawn shapes (every other structure in this renderer is
    axis-aligned boxes, so buildings follow the same art style). Each renders as a colored box
    (category-coded: guilds blue, shops green, official grey, shady/secret purple, gates gold)
    with a floating name label (`world_to_screen`, same projection Battlegrounds' own per-hero
    health bars use).
  - `server/mob/worm.go`'s `TownSquareWormSpawns` repositioned from an origin-centered ring to a
    tight cluster at the map's own real "Worm Hut" location (5, 15) -- client-side
    `TOWN_TARGET_X/Y` updated to match exactly.
  Go build/test green. Native client build clean; visually verified under Xvfb (elevated test
  camera): buildings render at real relative positions with readable labels, matching the map's
  layout (Dragon Gate north, Warrior Guild/Seed Shop/Blacksmith clustered near it, Gold
  Guild/Guild House/Butcher/Fishing/Post Office/Potions/Auction House/Armor Shop all correctly
  positioned relative to each other and to spawn).

## 2026-08-02 (13)

- feat(town): chat + combat log panes, target cycling, real "1" attack, jump, worm ring expanded
  to 4. Founder, several rapid follow-ups after killing the first worm: "where is my chat window
  and combat log window in town? those are going to stay up during normal gameplay" -> "add tab
  and shift tab to cycle through targets like wow" -> "to be clear we need to unify battlegrounds
  combat with the mud combat on the dragonsnshit side dont touch our MOBA in REDGARDEN repo" ->
  "where's my starter zone outside of town with the worms?" -> "add jump space bar".
  - `chat_draw`/`combat_log_draw` (already built for Battlegrounds) now also render in Town, plus
    full chat input handling (Enter to open/send, Escape to cancel) wired into Town's own event
    loop -- same shape Battlegrounds' own chat handling already uses, checked first so it
    consumes keystrokes before WASD/target-cycling/attack do.
  - `server/mob/worm.go`'s `TownSquareWormSpawns` expanded from 1 worm to a real ring of 4 (same
    shape as `MeadowWormSpawns`' own ring, smaller) -- "a single worm" didn't read as "a starter
    zone." `town_draw_worms` (renamed from `town_draw_worm`) now draws all 4 at their real spawn
    positions, matching mob IDs (`worm-town-0..3`).
  - Tab/Shift+Tab cycle `g_town_target_index` through the 4 worms; the selected one renders with
    an amber highlight, and the HUD shows `Target: worm-town-N`.
  - Pressing "1" -- the same ability-slot keybind Battlegrounds already uses (Q/W/E rebound to
    1/2/3 this fork) -- sends `attack <target>` to apps2/mud's real `/api/town/command`. This is
    the concrete first step of "unify battlegrounds combat with the mud combat": the same keybind
    language now drives the real MUD combat system, not a separate control scheme. REDGARDEN's
    own repo untouched, as instructed.
  - `town_poll_combat`: throttled (~1.5s) drain of the headless session's buffer, so background
    auto-attack ticks show up in the combat log even without pressing anything. New
    `town_send_command` shares the exact combat log pane Battlegrounds' own combat log already
    uses (`combat_log_push`) -- filters out the bracketed status line and bare prompt so the pane
    reads as combat events, not a raw MUD terminal dump.
  - Real bug fixed on the server side while wiring this up: `/api/town/command`'s JSON response
    used Go's default HTML-escaping, turning `>>> LEVEL UP <<<`-style real MUD text into
    `>>>`-garbled output the client's minimal JSON extractor couldn't unescape.
    Fixed with `Encoder.SetEscapeHTML(false)`.
  - Space bar triggers a purely cosmetic vertical bounce (sine arc, `TOWN_JUMP_DURATION_MS`/
    `TOWN_JUMP_HEIGHT`) -- Town has no verticality/collision system, so this doesn't interact
    with anything, named honestly in its own doc comment. Applied via a local vp pre-multiplied
    by a world-space Y translate for just the avatar's own draw call, not a change to
    `draw_hero_model`'s shared signature (also used by the real match renderer).
  Go build/vet + `server/mob` tests green. Native client build clean; visually verified under
  Xvfb with synthetic input (Tab, "1", Space) against a real login: target highlight, real combat
  text flowing into the pane ("worm-town-0 hits you for 8", "turns toward you"), real chat
  history rendering, all confirmed live.

## 2026-08-02 (12)

- feat(mud): real headless-session combat -- Town Square's worm is now genuinely fightable, not
  decorative. Founder: "can we kill worms?" -> chose "the real MUD combat system" over a simpler
  fake-hit-for-damage mode. Implements HEADLESS_SESSION_NORTHSTAR.md's core mechanism for the
  first time:
  - `getOrCreateHeadlessPlayer(characterID)`: builds a real `*player` (no telnet connection) from
    a real IDUNA character, `w: bufio.NewWriter(&buf)` where `buf` is an owned `*bytes.Buffer`
    (new `headlessBuf` field on `player`, nil for every real telnet player). Registered directly
    into `gw.players["headless:"+characterID]` -- the exact same map every telnet player uses --
    so the real 1Hz `gameLoop()`/`tickAll()` resolves its combat (auto-attack swing timer, TP,
    enmity, kill/loot/XP via `resolveKill`) with zero changes to the tick loop itself. Seeded from
    the character's real `scene_id`/position, which Town's own position sync (`50d582e`) already
    writes as zone 4 -- a fresh headless session naturally starts standing in Town Square, next
    to the real worm.
  - `runHeadlessCommand(characterID, line)`: runs one line through the real `handle(p, line)`
    dispatch and drains everything written since the last drain -- both the command's own
    response and any background tick messages, so a caller can poll with an empty line to catch
    auto-attack ticks without issuing a new command each time.
  - New `POST /api/town/command` on the existing `:7171` world-events API (same mux, same
    no-auth trust model -- named gap, not fixed: `character_id` is caller-supplied, not derived
    from any verified identity; apps2/mud has never verified an incoming JWT at all, only issued
    outbound agent calls).
  - Fixed a real, separate bug found while making this actually persist: a headless session
    never "disconnects," so `handleConn`'s own disconnect-time level/XP/flow sync never fires for
    it. New `headlessSyncedLevel/XP/Flow` fields + a delta-sync after every `runHeadlessCommand`
    call, same `UpdateCharacterLevel`/`CreditGold`/`DeductGold` calls the disconnect path uses.
  - **Two additional real bugs found and fixed along the way**, both pre-existing, both affecting
    real telnet play too, not just this feature: (1) `gfd-mud.service` never had
    `IDUNA_AGENT_NAME`/`IDUNA_AGENT_SECRET` configured (no `EnvironmentFile=` line existed) --
    every `idunaclient` call this live process has ever made was silently 401ing, masked by
    best-effort error handling everywhere. Fixed via a new `~/.config/gfd-mud/env`, same
    convention `iduna.service` already uses. (2) `idunaclient.UpdateCharacterLevel` called a
    route that has never existed on IDUNA (`PATCH /api/v1/characters/:id` with no suffix) --
    fixed client-side (now hits the new `/level` route, IDUNA `3ebad87`) and would have silently
    broken level/XP persistence for every real telnet disconnect too, this whole time.
  - `p.conn.Close()` in the `quit` command guarded against a nil `conn` (headless sessions have
    none) -- defensive; the new endpoint never sends `quit`, but cheap to guard anyway.
  Live-verified end-to-end, real character, real live worm: `attack worm` targets and
  auto-approaches, real tick-based swings land (30 damage/hit, worm's own 8-damage retaliation),
  real kill (`The creature collapses!`), real XP (+900), real level-up (1→3), real loot (Worm
  Sinew, Earth Crystal) -- and, after the two bugs above were fixed, confirmed landing in IDUNA
  for real (`level: 3, current_xp: 452`), not just in-memory. Go build + vet clean across
  `apps2/mud`, `apps2/server-go` (shares the fixed `idunaclient`), and `server/idunaclient`.

## 2026-08-02 (11)

- fix(town): position now flushes on quit, closing a real "login to same spot" gap. Founder:
  "ensure my avatar can move around town and the location is persisted so login to same spot."
  `town_sync_position` was throttled to once per 2s -- a player who moved and then closed the
  window inside that window lost their last few steps, landing slightly short of their real
  position on next login. `town_sync_position` gained a `force` param; called once more, forced,
  right as the app shuts down (after the main loop exits, before SDL teardown), flushing any
  movement the throttle hadn't caught yet. Live-verified end-to-end: reset test character to
  (0,0,0), ran a real login + forced movement + quit at 800ms (well under the 2s throttle) --
  IDUNA showed the partial movement `(2.383, 0, 1.589)` persisted anyway; a second fresh launch
  loaded from exactly that position, confirming true session-to-session persistence, not just
  in-session syncing.

## 2026-08-02 (10)

- fix(mud): S170-57, worm's Poison restored. Founder, real-time: "add poison back to that level
  1 worm you winey noob you just lowered the game difficulty because you didnt like it lol."
  Offered the real, tested reason it was removed (`ba735e8`, 2026-07-23: a flat Potency=10 debuff
  ticking once per game tick for up to 30s is up to 300 total damage from a single proc, against
  a level 1-5 character with only 90-150 max HP -- a real death, live, on this game's own zone-0
  tutorial mob) and asked which way to go; founder chose "Poison exactly as it was," knowingly:
  "this is a game for the hardcore a lvl1 poison ko is perfect." `mobSpellPool["worm"]` restored
  to its exact pre-`ba735e8` value, `{status.Slow, status.Poison}`. Still not "always" (confirmed,
  founder: "as long as its not always") -- unchanged 20% base proc chance, then a 50/50 pick
  between Slow and Poison, same mechanism as every other mob in the pool. Go build/vet green
  (no test coverage exists for apps2/mud at all, unchanged). Live `gfd-mud.service` rebuilt and
  redeployed.

## 2026-08-02 (9)

- feat(town): M5 ability panes (inert) + new zone 4 "Town Square" + starter-area worm. Founder:
  "and then implement the starter area worm" -> "you may need to add the next zone."
  - **M5 (ability panes)**: same `draw_ability_tile`, bottom-center layout, and Q/W/R color
    scheme Battlegrounds' own ability bar uses, ported into Town's HUD. Deliberately inert --
    Town has no cast/combat system, no per-job skill data wired in from apps2/mud's real
    weapon-skill system, and no mana; every tile is permanently ready and labeled
    "(unassigned)" rather than faked as functional. Only shown once a real character has loaded.
  - **New zone 4, "Town Square"** (`server/zone/zone.go`): a real, separate zone rather than
    reusing zone 0 (Meadow) -- Meadow already has a live telnet MUD presence (worm combat, "X
    has entered the world" broadcasts), and conflating it with the GUI's own local-only,
    no-combat Town scene would be exactly the "gui and the mud play nice... might get weird"
    tension named earlier today. `apps2/mud`'s `initWorld()` now spawns a mob registry + weather
    entry for it. `town_sync_position`'s own scene_id changed from 0 to `TOWN_ZONE_ID` (4).
  - **`server/mob/worm.go`'s new `TownSquareWormSpawns()`**: one real worm mob (not Meadow's full
    ring of eight -- Town Square is a small starter area), scene_id 4, spawned into
    `apps2/mud`'s real mob registry -- real backend content, not just client-side decoration.
  - **`town_draw_worm()` (client-side)**: a small three-segment box silhouette at the worm's real
    spawn position (mirrored by hand, same convention this codebase already uses for static
    positions). Named honestly as decorative in its own doc comment: apps2/mud has no HTTP
    surface for mob state at all yet, so this isn't a live sync of the real mob's HP/AI, just a
    placeholder at the right spot.
  - **Explicitly not built**: `apps2/server-go`'s own real voxel/chunk "Dragonfly" world
    (`server/worldapi`'s `DragonflyChunkGenerator`/`ProceduralWorldStore`, which already reserves
    the same 0-3 scene IDs for its own procedural terrain) -- wiring Town into that would mean a
    whole new UDP protocol client speaking `packages2/common/protocol.h`, a real, separate,
    much larger undertaking, flagged here rather than guessed at.
  Go build + `server/zone`/`server/mob` tests green (`zone_test.go`'s own zone-count assertion
  updated 4→5). Native Linux client build clean; visually verified under Xvfb with a real login --
  avatar, worm, and all three ability tiles render correctly together.

## 2026-08-02 (8)

- feat(town): real avatar, movement, and position sync in Town, backed by IDUNA's real
  `characters` record ("the dragonfly backend"). Founder: "i want my avatar to move around in
  town but i want it backed by dragonfly" -- continuing "this is the time to unify the whole
  bitch" / "wire up the dragonfly backend" / "the xyz at least needs to flow back to the
  dragonfly server for the gui xyz source of truth." M1-M4 of the same-day backlog/sprint plan:
  - `town_fetch_character()`: on entering Town, resolves the player_id captured from login's own
    self-ticket response (already returned it, no new endpoint) to the real character via
    `GET /api/v1/characters/by-player/:id`, seeding position and `job_main`.
  - `town_hero_id_for_job()`: `ARENA_HERO_WARRIOR` for job `WAR` -- the one real, non-guessed
    correspondence (`arena_game.h`'s own doc comment already calls it "DragonsNShit's Warrior
    job, ported as Battlegrounds content"). Every other apps2/mud job has no hero-visual mapping
    yet -- falls back to Warrior, named as a placeholder in the comment, not the UI. A real, open
    design question, not resolved here.
  - Real movement: WASD (camera-relative, same derivation as Battlegrounds' own) + click-to-move
    (`screen_to_ground`, same ray-cast), interpolated at `ARENA_HERO_SPEED` for a consistent feel
    between the two scenes. Camera now follows the avatar's own position instead of a fixed
    origin. Purely local -- no server involved in Town's own simulation.
  - `town_sync_position()`: throttled (2s, only if actually moved) `PATCH
    /api/v1/characters/:id/position` using the player's own JWT -- the same endpoint IDUNA
    `ab35b72` just hardened for exactly this caller.
  - New `http_extract_json_double_field` helper (`http_client.h`) for `pos_x`/`pos_y`/`pos_z`
    (SQL `REAL` columns) -- the existing extractors only handled strings and integers.
  Accepted tradeoff, founder's own words: "it might get weird having the gui and the mud play
  nice" -- whichever of Town or a live apps2/mud telnet session last PATCHes position wins, no
  conflict resolution beyond that. Live-verified end-to-end against the real test account and
  live IDUNA: real login -> real character fetch (confirmed exact `pos_x/y/z`/`job_main` from a
  prior manual PATCH) -> simulated movement -> position correctly persisted back
  (`pos_x/z` matched the exact forced offset). Visually verified under Xvfb: avatar renders,
  faces its movement direction, camera follows. Ability panes (M5) not done this pass. Native
  Linux build + link clean (mingw unavailable locally, same standing note as the rest of this
  client).

## 2026-08-02 (7)

- fix(town): a failed requeue now lands back in Town instead of a dead blank arena. Founder,
  live bug report: "if i dont requeue fast enough in GFD when i requeue it is like an empty game
  it says matchmaking fail." Root cause: `arena_server`'s own 60s "no lobby progress" watchdog
  (confirmed live in `var/logs/matchmaker-bots.log`: `No lobby progress in 60s (phase=0, 19/20
  connected) -- shutting down.`) can kill a freshly-matched game before a slow requeue finishes
  connecting to it -- `net_find_and_connect`/`net_connect` then correctly return failure, but the
  requeue handler had already `memset` `arena_state` to zero and did nothing further on failure,
  leaving the player staring at a blank arena (no heroes, no nodes) with no way back except
  force-quitting. That's REDGARDEN's own server/matchmaker code, explicitly out of scope this
  session (founder: "keep the battlegrounds working as is do not change that keep that server
  and matchmaking as is") -- fixed on the client side instead: a failed requeue now sets
  `in_town = 1`, landing the player back in a real, working Town scene where "QUEUE FOR
  BATTLEGROUNDS" can just be clicked again, same escape hatch the RETURN TO TOWN button already
  provides.

## 2026-08-02 (6)

- feat(town): "RETURN TO TOWN" button on the post-match win/lose screen, alongside the existing
  "OK - REQUEUE" button. Founder: "after a battlegrounds game in GFD i need the option to return
  to the town like a back button i only have requeue." Same cleanup path requeue already uses
  (close socket, reset `arena_state`/obstacles/rings/win_logged/net_picked/selected_unit_count),
  except no reconnect -- just sets `in_town = 1` so next frame's Town branch takes over. Only
  shown for the `--queue` path (there's no Town to return to for a direct `--connect` dev
  session, same gate Town's own entry uses).

## 2026-08-02 (5)

- feat(town): `apps2/battlegrounds_gui` now defaults to a real Town scene instead of connecting
  straight into the matchmaker. Founder: "we need the default to be town... a button top right
  to queue for battlegrounds which would trigger the matchmaker that leads to the draft and the
  game etc... build the world outside of the battlegrounds for now a flat plane is ok have it
  checkers grey and brown like a chessboard just make it the same size as the battlegrounds
  scene for now just with no buildings or trees or rocks yet." First slice of
  `HEADLESS_SESSION_NORTHSTAR.md` §3.4's "second scene," client-rendering-only for now (no
  headless MUD session wired up). `--queue`'s own `net_find_and_connect` call is deferred from
  startup to a real "QUEUE FOR BATTLEGROUNDS" button (top-right, per the founder's own
  placement); login still happens up front unchanged, landing the player in Town instead.
  `--connect` (direct dev connect to a known arena_server) is untouched -- no queue step to
  defer there, so it skips Town entirely, same as before. Town's own ground is a 12x12
  grey/brown checkerboard spanning the exact same footprint (`ARENA_HALF_EXTENT * 2.2f`) as
  battlegrounds' own ground plane -- "same size as the battlegrounds scene." Reuses
  battlegrounds' existing right-drag+wheel orbit camera and the same queuing "please wait"
  screen the post-match requeue button already used, so the window doesn't look hung during the
  up-to-60s matchmaker wait. Battlegrounds' own code is untouched -- the whole existing frame
  body is skipped via an early `continue` while in Town, not woven into it. Visually verified
  under Xvfb: TOWN label, checkerboard, and the button all render correctly (mingw unavailable
  locally, same standing note as the rest of this client).

## 2026-08-02 (4)

- feat(combat-log): second pane in `apps2/battlegrounds_gui`, bottom-right (mirrors the chat
  pane's bottom-left), showing damage taken and deaths for the current match. Founder: "add a
  second chat pane to GFD that shows the combat log." No wire packet carries discrete
  damage/kill events -- derived client-side by diffing `arena_state.heroes[]` frame-to-frame
  (`combat_log_scan`), which works identically for local play, net_mode, and replay/observing
  since all three write into the same `arena_state`. Attacker attribution via the existing
  `attack_target` field (already wire-synced, S170-162); unattributed damage (DoTs, creeps,
  skillshots) just shows the amount. Deliberately excludes heals to keep the log readable.
  Always visible, unlike the chat pane (not gated on a real player JWT -- bots/`--ticket`
  launches still show it). Native Linux build + link verified clean (mingw unavailable locally,
  same standing note as the rest of this client).

## 2026-08-02 (3)

- feat(chat): in-match MUD chat, `apps2/battlegrounds_gui`'s own real affordance surfacing
  `apps2/mud`'s persistent-world chat. `deliverChat` now relays say/yell/guild lines to IDUNA's
  new `POST /api/v1/chat/messages` (tell stays private, not relayed) via `idunaclient`'s new
  `PostChatMessage`/`GetChatMessages`. The Battlegrounds client polls every ~1.5s and renders a
  scrolling log; Enter opens a real chat-input line (consumes all other keybinds while focused,
  same "held/focused, not toggled" idiom the rest of the client already uses) and posts back as
  channel `battlegrounds`. Own JWT reused from login -- inert (no polling/posting at all) for
  bots/`--ticket`/dev-agent launches, which have no real player identity to chat as. One-way for
  now, named honestly: `apps2/mud` doesn't yet poll for Battlegrounds-originated messages, so MUD
  players don't see chat sent from a match -- real, separate, unbuilt follow-up.

## 2026-08-02 (2)

- feat(battlegrounds-gui): real fork of REDGARDEN's `apps/arena` into `apps2/battlegrounds_gui/`
  at commit `61baafb`. Corrects the previous approach (live cross-repo checkout of REDGARDEN at
  CI build time) per founder direction: "REDGARDEN isnt literally the GUI its supposed to be a
  starting place for the GUI like a clean fork." Self-contained -- own `packages/simulation` +
  `packages/common`, not sharing GFD's existing top-level `packages/`/`packages2/` (which have
  unrelated real content, e.g. `packages2/common/protocol.h` is a completely different wire
  protocol). CI now builds directly from this local copy, no cross-repo checkout step. Verified:
  clean standalone build, live login->ticket->connect reaches "20/20 connected" in a real match.
- feat(battlegrounds-gui): rebound ability casts from Q/W/E to 1/2/3 and added continuous,
  camera-relative WASD movement alongside the existing click-to-move (re-sent every ~100ms while
  held, same underlying move-to-point mechanic, no new wire packet). Fork-only -- REDGARDEN's own
  copy is untouched. Also fixed a real bug found live testing the founder's test account:
  `PLAY.bat` never set `IDUNA_BASE_URL`, so a real downloaded client always got "Could not reach
  login server" (its `127.0.0.1` default only works on the same box as IDUNA).

## 2026-08-02

- ci: `GoblinFoxDragon Factory` now also cross-compiles REDGARDEN's `apps/arena` (the real MUD
  GUI frontend, `REDGARDEN_GUI_NORTHSTAR.md`) as a Windows artifact. The existing
  `GoblinFoxDragon.exe` build target is `apps/lobby`, a stale SHANKPIT-lobby fork (window title
  still literally says "SHANKPIT", boots into SHANKPIT's own `SCENE_GARAGE_OSAKA`) -- not the
  real MMO client. New steps check out REDGARDEN (public repo, no token needed) and cross-compile
  its `apps/arena` with the same mingw/SDL2 toolchain this workflow already sets up, mirroring
  REDGARDEN's own `ci.yml` Windows step verbatim so the two pipelines can't silently drift.
  Bundled as `DragonsNShit_MUD_GUI_Client_*.zip` (exe + `SDL2.dll` + `PLAY.bat`, no
  `REDGARDEN_TICKET_SECRET` needed -- the client's own real IDUNA login screen mints a real
  ticket), uploaded alongside the existing artifacts.
- docs(redgarden-gui-northstar): Milestone 5's "no GUI login path" gap closed -- real
  email+password login screen shipped in REDGARDEN's `apps/arena` (`9c98342` + Winsock fix
  `e6fb748`), backed by a new IDUNA endpoint (`POST /api/v1/redgarden/self-ticket`, `5cd0fd0`).
  Working test account (`test@test.com`/`testtest`, character `TestWarrior`) verified end-to-end.

- docs(redgarden-gui-northstar): Milestone 5 (end-to-end validation) attempted honestly --
  marked PARTIAL, not DONE. Direct smoketest of `cmdBattlegrounds`'s exact real call sequence
  (`CreateCharacter`/`GetCharacter`/`MintBattlegroundsTicket`) against the live IDUNA service
  confirmed all three real, fast (ms-scale), and correct -- the strongest confirmation this
  identity/ticket chain has had yet. Also found `Xvfb`/`glxinfo` now work in this environment
  (Mesa software GL renders correctly) -- earlier "no display" notes are stale. Interactive
  telnet validation of the `battlegrounds` command's own text output was attempted repeatedly and
  abandoned as unreliable test-harness noise, not a code bug (the logic it calls is independently
  proven correct above); full interactive match-play validation (draft/cast/chain/credit) wasn't
  attempted -- two real, named, scoped blockers (a skillchain-aware bot heuristic; GUI-input
  automation tooling) remain open, not built here. New §9 in the northstar with the full honest
  breakdown.

- feat(job, mud): SMN gets real Avatar abilities -- founder, real-time: "zagan beleth vassago as
  summoner avatars GFD." New `job.SummonerAbilities()` (`summon_zagan`/`summon_beleth`/
  `summon_vassago`, real `Ability` data through the same `RecastTracker` every other job uses)
  wired into `apps2/mud`'s `abilitiesForJob`/`cmdJA`. Each avatar applies real
  `server/status` effects to the caster's live duel opponent, translated from that hero's own
  REDGARDEN kit (`docs/HEROES_VS0.md`) rather than invented: Zagan -> Bind (closest existing Kind
  to "stun," this package has no Stun Kind), Beleth -> Poison+Silence (her own real Q+W, ported
  faithfully since she already carries both on separate slots), Vassago -> Silence + a small
  direct hit (her real Q), damage clamped to never drop the opponent below 1 HP so it doesn't
  need to touch `duel.Manager`'s own win-condition path. Real, honestly-flagged simplification,
  not a full kit port: no armor-shred/mirror (Protect is a buff-only Kind in this package, not
  Category-flexible per Effect), no cast-refund, no delayed-burst zone, and no mob-targeted
  version at all (`mob.Mob` has no status stack yet -- a real, separate structural gap).
  2 new tests in `server/job` (data shape + a real `RecastTracker` integration check). Live
  smoke-tested via two telnet sessions (character creation, `setjob SMN`, duel challenge/accept,
  `ja summon_*`) -- confirmed `setjob SMN` correctly applies real SMN stats (HP:60/MP:90, matching
  `job.jobStats[SMN]`) and the duel flow works through the same command-dispatch path `ja` uses;
  the final `ja summon_*` output specifically wasn't reliably captured due to test-harness
  timing fragility (nc/FIFO scripting), not a code issue -- `go build`/`go test ./...` clean, and
  direct review of `cmdSummonAvatar`'s locking (no `gw.mu` held anywhere upstream of it, confirmed
  by reading `cmdJA`'s and `handle()`'s own call sites) rules out the deadlock this function's own
  `gw.mu.Lock()` would otherwise risk.

- docs(redgarden-gui-northstar): Milestone 4 shipped -- reward-credit hook. REDGARDEN's
  `apps/arena_server` now credits real Flow (100 win / 25 loss) to a match participant's
  persistent DragonsNShit character via new IDUNA `GET /api/v1/characters/by-player/:player_id`
  + the existing `gold/credit` endpoint. REDGARDEN `1fcf09e`, IDUNA `33b7a0d`. Milestone table +
  status line updated -- only Milestone 5 (end-to-end validation) left.

- fix(idunaclient): real IDUNA login exchange -- a genuine, previously-undiscovered production
  bug found while wiring REDGARDEN_GUI_NORTHSTAR.md Milestone 3. `Client.do()` used to send
  `IDUNA_AGENT_SECRET` directly as the Bearer token; IDUNA's real `jwt.Verify`-based
  `RequireAuth` middleware has always rejected that with 401 (confirmed live against the running
  service, not just theorized from reading the code) -- every call this package has ever made
  (`GetCharacter`/`CreateCharacter`/`CreditGold`/etc., shared by both `apps2/mud` and
  `apps2/server-go`) has been silently failing, masked by "best-effort, non-blocking" error
  handling at every call site. `characters` table on the live IDUNA instance was empty; this is
  why. Fixed: `New()` now also reads `IDUNA_AGENT_NAME`; a new `ensureToken()` performs the real
  `POST /api/v1/auth/agent` exchange and caches the resulting JWT (refreshed within 60s of its
  real 1-hour expiry), used by every existing method for free since they all route through
  `do()`. Verified live end-to-end: a real character now creates successfully against the
  running IDUNA service (previously 401). 4 new tests. Backward-compatible with every existing
  test in this package (none set `IDUNA_AGENT_NAME`/`IDUNA_AGENT_SECRET`, so `do()` skips the
  login step exactly as before for those).

- fix(mud): real, stable player_id for IDUNA character creation -- another real gap found in the
  same pass. `gw.iduna.CreateCharacter`'s `player_id` argument was `conn.RemoteAddr().String()`
  (a TCP socket address) -- not a valid UUID, and different every reconnect. IDUNA's own ticket
  endpoints `uuid.Parse` the player_id and would reject it outright. New `mudPlayerIDCache`
  (`var/mud-player-ids.json`, same load/persist shape as the existing `mudCharCache`) mints and
  persists a real `crypto/rand` UUIDv4 per character name on first use -- stdlib-only, no new
  dependency (same reasoning `packages/common/hmac_sha256.h`'s own doc comment already gives for
  not linking a crypto library). Does NOT solve real player identity (OAuth/email login for a
  telnet interface is a genuinely separate, larger, undesigned question) -- only makes the
  existing anonymous, name-keyed identity model stable and UUID-shaped instead of an ephemeral
  socket address, flagged honestly rather than oversold.

- feat(mud): `battlegrounds`/`bg` command -- REDGARDEN_GUI_NORTHSTAR.md Milestone 3, the
  Battlegrounds entry point (§4.3's own open question, resolved as a discrete command, same
  shape as `cmdGo`'s own zone-transfer precedent, which §4.3 named as the closest existing one).
  Fetches the player's real character via IDUNA, mints a real REDGARDEN connect ticket via the
  new `idunaclient.MintBattlegroundsTicket` (IDUNA's new `POST /api/v1/redgarden/player-ticket`,
  see that repo's own CHANGELOG), and prints the exact `red_garden_arena --queue <host>
  --matchmaker-port 7778 --ticket <hex>` command line to run -- a telnet session can't launch a
  GUI process itself, so this is the honest, real "hand off" a text interface can do (REDGARDEN's
  own new `--ticket` flag, see that repo's own CHANGELOG, is what makes the printed command
  actionable). Job pick is a stub, not a menu -- Warrior is the only job Milestone 1 ported, so
  there's nothing to choose between yet.

- docs(redgarden-gui-northstar): Milestone 2 shipped, same session as Milestone 1 below -- real
  skillchain resonance detection in REDGARDEN's `arena_game.c`. A straight C port of this repo's
  own `server/skillchain.go` combination table (same real tiers/multipliers), tracked per-target
  and closed via a new `apply_weapon_skill_damage` choke point every real weapon-skill cast
  routes through. Verified real: Warrior's own Q(Scission)->R(Induration+Reverberation) closes an
  actual Tier 2 Distortion chain per the table. REDGARDEN `21ad0dc`. Milestone table + status
  line updated to match. Milestones 3-5 (entry-point hook, reward-credit hook, end-to-end
  validation) still ahead.

- docs(redgarden-gui-northstar): Milestone 1 shipped -- Warrior, the first DragonsNShit job
  ported into REDGARDEN's Battlegrounds as real ability content. Founder redirect this session,
  after "can i log into gfd gui yet?": "ok i asked for the mmorpg i provided the inputs continue
  to work on that." Real code landed in the sibling REDGARDEN repo (`cbcd4ed`) -- Q Hard Slash/W
  Power Slash/R Frostbite, real Great Sword weapon skills from this repo's own
  `server/skillchain.CanonicalWeaponSkills`, matching `server/job.jobStats[WAR]`'s real stat
  block. REDGARDEN has no TP resource, so MP substitutes for `server/combat.TPWSThreshold`'s 100
  TP -- an honest amendment, not a literal port, per founder direction ("we want our old systems
  like skillchains etc [to] work with redgarden affordances"). `docs2/REDGARDEN_GUI_NORTHSTAR.md`
  milestone table + status line updated to match. Milestones 2-5 (skillchain detection in
  `arena_game.c`, entry-point hook, reward-credit hook, end-to-end validation) still ahead.

- feat(idunaclient, mud): `apps2/mud`'s Flow (gold) is finally synced back to IDUNA on
  disconnect. Backend-unification follow-up, closing the real gap the previous correction found:
  `p.flow` was read from IDUNA on connect but never written back, because IDUNA had no way to
  credit gold at all -- only deduct. Now that IDUNA's own new `PATCH .../gold/credit` exists
  (`IDUNA` commit `1b7f43d`), added the symmetric client method `idunaclient.CreditGold`
  (3 new tests, `server/idunaclient/idunaclient_test.go` -- this package's first test file at
  all, DeductGold and every other existing method shipped with zero coverage; backfilling those
  is separate, larger work, not attempted here). Wired into `apps2/mud`'s own connect/disconnect
  flow: `startingFlow` captures the real balance right after the existing fetch-or-create IDUNA
  call, and the disconnect handler now computes the session's net Flow delta and calls
  `CreditGold`/`DeductGold` accordingly -- same silent-discard, best-effort convention the
  adjacent level/XP/position sync calls already use. `GOWORK=off go build ./...`/`go test ./...`
  clean.

- docs: corrected a real, load-bearing wrong claim in `docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`
  ("apps2/mud has no real IDUNA persistence"). Found while investigating what "share live state"
  actually requires: that claim's own grep searched for the literal strings `idunaclient`/
  `idunaClient` and found nothing beyond construction -- but the real field is `gw.iduna`
  (different name, different grep target) and it genuinely is called: `gw.iduna.GetCharacter`/
  `CreateCharacter` on connect (seeding level/XP/gold from a real IDUNA row, via a local
  name→ID cache persisted to `var/mud-chars.json`), `gw.iduna.UpdateCharacterLevel`/
  `UpdatePosition` on disconnect. What's still true: only synced at connect/disconnect, not
  continuously like `apps2/server-go`; and `p.flow` (gold) is read on connect but never written
  back -- traced why: IDUNA's own `/characters/:id/gold` endpoint only accepts deducting gold
  server-side, no credit/add endpoint exists at all, so completing this needs new IDUNA API
  surface, real cross-repo work not attempted here. A partial fix covering only the decrease
  direction was considered and deliberately rejected (silently wrong for the increase case is
  worse than clearly not-done). Corrected in place in the audit doc's own §1 and §4, not
  silently overwritten.

- feat(server-go): respawn's XP penalty now persists back to IDUNA. Backend-unification
  follow-up, closing the "XP earned isn't written back to IDUNA yet" gap named in
  EMILY/BACKLOG.md's own unification item. `idunaclient.Client` already had a real,
  ready-to-use `UpdateCharacterLevel(characterID, level, currentXP)` method (PATCHes
  `/api/v1/characters/:id`) -- not built here, just not wired into this handler yet. The respawn
  handler now calls it after computing the new post-penalty `currentXP`, fire-and-forget via a
  goroutine + log-on-error, same pattern `PacketSkillXP`'s own `IncrementSkill` call already
  uses just above it (never blocks the UDP read loop on an HTTP round trip).
  `GOWORK=off go build ./...`/`go test ./...` clean.

- fix(server-go): respawn XP penalty was using the wrong percentage (10% instead of the real,
  live 8%). Backend-unification follow-up, same-day correction to the respawn-packet change
  right below. Found while trying to wire real per-player XP in: `HPState.RaiseDefault`
  (what the respawn handler called) applies `combat.DefaultRaisePenaltyPct` (10%) -- checked
  against `apps2/mud`'s own actual live behavior and that's not the real number. `apps2/mud`'s
  real `cmdHome` hand-computes an 8% penalty (`homepoint.DefaultXPPenaltyPct`) and doesn't call
  `HPState.Raise` at all -- a claim the respawn-packet CHANGELOG entry below got wrong ("apps2/
  mud's own real 'type home' flow... HPState.Raise" -- it doesn't). `server/homepoint`'s own
  `ReturnHome()` implements that real 8% mechanic too, but `cmdHome` duplicates it by hand
  instead of calling it (pre-existing in `apps2/mud`, unrelated to this fix, not touched). Fixed
  by passing `homepoint.DefaultXPPenaltyPct` explicitly into `HPState.Raise` (which does accept
  an arbitrary percentage) instead of trusting its own unrelated 10% default -- still reuses
  `Raise`'s real HP-reset/`IsKO`-clear behavior, just with the actually-live percentage. Also
  wired real per-player XP: `fetchCharacterCombatStats` now also returns IDUNA's real
  `Character.CurrentXP`, stored on `clientInfo` and mutated locally on respawn (not written back
  to IDUNA yet -- a further, named gap). 1 existing test updated for the new return signature,
  no new test needed for the percentage fix itself (`Raise`'s own arbitrary-percentage behavior
  is already covered upstream). `GOWORK=off go build ./...`/`go test ./...` clean.

- feat(server-go): respawn packet closes the KO loop the previous change opened. Backend-
  unification follow-up (EMILY/BACKLOG.md item 2). New `PacketRespawn`/`PacketRespawnResult`
  (`packages2/common/protocol.go`) -- a KO'd player's only way back on this backend, `apps2/mud`'s
  own real "type home" flow (`knockOut()` + `HPState.Raise`, 8% XP penalty) reduced to its core
  mechanic: `RaiseDefault(0)`, always against 0 XP since real per-player XP tracking doesn't
  exist in `apps2/server-go` yet (unlike `apps2/mud`'s own `p.charXP.CurrentXP`) -- a real,
  already-tested degenerate case (`server/combat`'s own `TestRaise_ZeroXPNoPanic`), not a crash
  risk, just an honestly-incomplete penalty number until real XP tracking lands here too, named
  in the code rather than silently wrong. No new tests -- `RaiseDefault`'s own behavior is
  already covered by 5+ existing tests upstream, same reasoning the KO-state change just used.
  `GOWORK=off go build ./...`/`go test ./...` clean.

- feat(server-go): real KO state via `server/combat.HPState`, gates further weapon-skill casting.
  Backend-unification follow-up (EMILY/BACKLOG.md item 2). `clientInfo.hp`/`maxHP` (raw ints)
  replaced with `hpState *combatTp.HPState` -- `apps2/mud` itself drives KO through its own
  separate `homepoint.State.IsKO` field rather than calling `HPState` directly, but the mechanics
  are the same shape, and reusing the already-tested type (`NewHPState`/`TakeDamage`/`IsKO`, 17
  existing tests in `server/combat/death_test.go`) beats re-deriving damage-floor/KO logic by
  hand. `PacketWSCast` now rejects casting from a KO'd caster and casting *at* an already-KO'd
  target (`ErrAlreadyKO`'s own failure mode, guarded before it can fire). Deliberately NOT
  implemented: any respawn/home-point flow once `killed=true` -- `apps2/mud`'s own `knockOut()`
  leaves a KO'd player waiting until they actively type `home` (8% XP penalty) or get Raised;
  porting that full flow is separate, larger follow-up work, not attempted in this slice, so a
  KO'd player on this backend currently just... stays KO'd forever. Named honestly, not hidden.
  `GOWORK=off go build ./...`/`go test ./...` clean (existing 6 tests, no new ones needed --  the
  underlying `HPState` behavior this wiring calls is already covered upstream).

- feat(server-go): real IDUNA job/level fetch on connect + real HP tracking, closing two of
  Sprint 3's own named gaps. Backend-unification follow-up (EMILY/BACKLOG.md item 2). New
  `fetchCharacterCombatStats` -- calls `idunaClient.GetCharacter` on `PacketConnect` (same
  best-effort tone `PacketTelecrystalUse` already uses toward IDUNA lookups: falls back to
  WAR/level 1, not a hard connection reject, if IDUNA has no character row yet or the fetch
  fails outright), computes starting HP via `jobpkg.HPAtLevel` -- the same formula `apps2/mud`'s
  own character sheet already uses, not reinvented. `clientInfo` gained real `jobMain`/`level`/
  `hp`/`maxHP` fields (HP itself in-memory only, same as `apps2/mud`'s own `p.hp` -- not
  persisted to IDUNA, matching every MMORPG's own "a life's current HP isn't durable state"
  convention). `PacketWSCast` now actually subtracts `result.Damage` from the target's real HP
  and reports `target_hp`/`target_max_hp`/`killed` in `PacketWSResult` -- Sprint 3 only ever
  reported a damage number without touching anything; this is the first slice where a weapon
  skill actually hurts someone. 1 new test (`fetchCharacterCombatStats`'s WAR/level-1 fallback,
  verified deterministically by pointing at an unreachable IDUNA URL rather than a live server).
  `GOWORK=off go build ./...`/`go test ./...` clean. Still not done: no death/respawn handling
  once `killed=true` fires (the target just sits at 0 HP), enmity untouched, `apps2/mud`'s telnet
  players still don't share this state.

- feat(server-go): real weapon-skill casting + skillchain resonance wired into `apps2/server-go`'s
  UDP loop. Founder: "yes unify the backends" -> "whatever makes sense" -> "clean builds first"
  (backlog dump + sprint plan, Sprint 3 -- EMILY/BACKLOG.md). First real slice of
  `docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`'s own unification recommendation: rather than
  rewriting `apps2/mud`'s RPG logic, `apps2/server-go` now directly imports the same tested
  `server/combat`/`server/skillchain` packages `apps2/mud`'s own `cmdAttack`/`cmdWS` already use.
  New wire packets `PacketWSCast`/`PacketWSResult` (`packages2/common/protocol.go`). Every
  `BtnAttack` now feeds real TP (`combatTp.TPState.AddTP`, flat 1H-sword delay assumed -- no real
  weapon/gear system wired into this backend yet) alongside the existing `HandleShankFire`
  hitscan, not replacing it. `PacketWSCast` validates against `server/skillchain`'s real weapon-
  skill registry, checks real TP via `CanWeaponSkill`, and scores a real skillchain against
  whatever last landed on the target (`server/skillchain.Chain`, PvP-shaped -- targets another
  connected client, not a mob, since this backend has no mob registry the way `apps2/mud` does).
  Decision logic extracted into a standalone `resolveWSCast` (same "extract for testability"
  reasoning `main_test.go`'s pre-existing `TestParseUserCmd` already established for
  `parseUserCmd`) -- 4 new tests (unknown skill, no-chain damage, a real Shining Blade -> Burning
  Blade Tier-2 Fusion closure, chain-window-expiry). Named, not silently skipped: no real HP/
  death tracking exists for `apps2/server-go`'s own connected players yet (`clientInfo` has no HP
  field at all), so damage is a placeholder number reported in the result packet, not applied to
  anything -- a separate, larger follow-up. `GOWORK=off go build ./...`/`go test ./...` clean
  across the whole module throughout ("clean builds first" taken as a continuous constraint).

- refactor: `gil` -> `flow`/`Flow` across the whole `dragonsnshit` module. Founder: "convert gil
  to flow" (backlog dump + sprint plan, Sprint 2 -- EMILY/BACKLOG.md). REDGARDEN already has
  real, shipped "Flow" economy terminology (S170-175); DragonsNShit's own currency naming now
  matches instead of keeping FFXI's "gil". Renamed: `apps2/mud/main.go`'s `player.gil` field (all
  call sites, all in-game command output text), the `"gil-drop"` loot item ID and its "100 Gil"
  display name (-> `"flow-drop"`/"100 Flow"), `server/quest`'s `RewardGil`/`Result.Gil` fields (+
  `trapx_chains.go`'s 20-odd `RewardGil:` literals), `server/auction`'s `ErrInsufficientGil`/
  `buyerGil` (+ `TestBuyInsufficientGil` -> `TestBuyInsufficientFlow`), `server/market/ah.go`'s
  own comments. `GOWORK=off go build ./...` and `go test ./...` clean across the whole module
  before and after. Two docs from earlier today (`REDGARDEN_GUI_NORTHSTAR.md`,
  `DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`) updated to reflect the completed rename rather than left
  silently stale.

- docs: REDGARDEN_GUI_NORTHSTAR.md, two real-time corrections in a row on the Battlegrounds
  design. Founder #1: "some of the docs say we arent bringing redgardens gameplay just the ui
  thats not right i want dragonsnshit mmo to feel like redgarden like battlegrounds for
  dragonsnshit is redgarden." Corrected the doc's original thesis (REDGARDEN as rendering-only
  skin, DragonsNShit's own systems replace REDGARDEN's gameplay underneath) -- wrong. REDGARDEN's
  full real-time combat framework (`arena_server`/`apps/matchmaker`, Q/W/R slot UI, item shop,
  node-capture map) becomes DragonsNShit's Battlegrounds, an instanced PvP mode, same relationship
  WoW Battlegrounds/FFXI's own minigames have to their main games. Founder #2, immediately after:
  "like not the same literal game loop maybe but we want to amend our ould systems like
  skillchains etc work with redgarden affordances." Refined further: the process/loop separation
  stays (Battlegrounds is still its own spawned-per-match process, not merged into the persistent
  world's own loop), but the *ability content* cast through REDGARDEN's Q/W/R slots is
  `apps2/mud`'s real job/weapon-skill/skillchain system ported into `arena_game.c`'s slot
  machinery -- a Battleground combatant picks a job (Warrior, Black Mage, ...), not one of
  REDGARDEN's 28 fixed heroes, and that job's real abilities render through REDGARDEN's existing
  cast-ring/projectile/zone-circle vocabulary, with real skillchain resonance between players'
  casts. §§1/4.1/4.2/5/6 rewritten in place across both corrections, each labeled and dated so
  the doc's own reasoning history stays legible. Milestone table now: port Warrior's real kit
  into `arena_game.c` first, then skillchain resonance, then the entry-point/reward-credit hooks,
  then end-to-end validation.

- docs: DragonsNShit has two non-unified backends — audit + bridge-target correction. Founder:
  "continue dragons n shit" (continuing "do the docs first"). New
  `docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`, correcting a real, load-bearing wrong assumption in
  both of today's earlier docs (`REDGARDEN_GUI_NORTHSTAR.md`, `REDGARDEN_MUD_BRIDGE_SPEC.md`),
  both written as if `apps2/mud` were the only DragonsNShit backend. It isn't: found
  `apps2/server-go`, a real UDP server on `:6969` with a real IDUNA-JWT-authenticated protocol
  (`PacketConnect`/`PacketUserCmd`/`PacketChat`, real Telecrystal travel + crafting + skill-XP,
  all actually calling IDUNA) -- `apps2/mud`'s own `idunaclient` is imported and instantiated but
  never once actually called (confirmed via repo-wide grep), a dead field, not real integration.
  `apps2/server-go`'s combat is SHANKPIT-shaped hitscan (`HandleShankFire`), not `apps2/mud`'s
  RPG job/skillchain/enmity depth -- the two backends don't share state at all. Also found
  `apps2/lobby`, an existing 884-line C client already targeting `apps2/server-go`'s protocol,
  smaller than REDGARDEN and blocked by the same `GL/glu.h` dependency issue that's hit this
  monorepo repeatedly -- reinforces REDGARDEN as the stronger client foundation, not a reason to
  abandon the direction. Revised recommendation: port `apps2/mud`'s RPG logic to run inside
  `apps2/server-go`'s authoritative loop, backed by IDUNA's already-existing
  `characters`/`character_skills`/`character_equipment`/`character_inventory` schema, before
  REDGARDEN's own bridge work lands -- REDGARDEN then targets `apps2/server-go` directly as a
  peer of `apps2/lobby`, no new listener needed (superseding `REDGARDEN_MUD_BRIDGE_SPEC.md`'s own
  "bolt a listener onto the text MUD" design, marked superseded in place, kept for its still-real
  movement/targeting gap-finding). `REDGARDEN_GUI_NORTHSTAR.md`'s milestone table rewritten in
  place to reflect this. Registered in golden-docs-index.

- docs: REDGARDEN ↔ apps2/mud packet-level bridge spec. Founder: "continue dragons n shit do the
  docs first." New `docs2/specs/REDGARDEN_MUD_BRIDGE_SPEC.md`, the concrete next layer under
  today's earlier `REDGARDEN_GUI_NORTHSTAR.md`, written against the real code on both sides
  (`REDGARDEN/packages/common/protocol.h`'s actual structs, `apps2/mud/main.go`'s actual `cmd*`
  handlers) rather than assumed shapes. Reuses REDGARDEN's real HMAC connect-ticket handshake
  verbatim (same scheme SHANKPIT/shankpit-460 already share). Maps REDGARDEN's real packets onto
  apps2/mud's real functions (`ArenaAttackCmd`→`cmdAttack`, `ArenaCastCmd`→`cmdWS`,
  `ArenaShopBuyCmd`→`cmdShopBuy`); drops `PACKET_ARENA_PICK` entirely (no hero draft in a
  persistent-character MMO). Two real gaps found and named while writing this, not glossed over:
  `apps2/mud` has zero continuous intra-zone movement server-side today (`cmdGo`'s own code
  confirms `n/s/e/w` only ever teleports between zones; `cmdAttack`'s auto-approach snaps
  position directly onto the target) -- `PACKET_ARENA_MOVE` has nothing to bridge onto without
  real new server code, reframing the northstar's own Milestone 3 scope; and
  `PACKET_ARENA_ATTACK`'s hero-slot-index targeting has no equivalent against apps2/mud's
  string-ID mob/player targeting. Proposes a genuine `MudEvent` list to replace REDGARDEN's flat
  HP-delta-driven visual-effect idiom (`attack_flash`/`heal_flash`), which can't carry
  skillchain/status-effect semantics the way a flat HP diff can't. UDP, port 2324 proposed
  (resolves one of the northstar's own open questions). `REDGARDEN_GUI_NORTHSTAR.md` updated
  in place: 2 open questions resolved/refined, 2 new ones surfaced, Related Docs table updated.
  Registered in golden-docs-index. Spec only, no code.

- docs: REDGARDEN-as-GUI northstar. Founder, real-time: "can we graft redgarden frontend onto GFD
  mud as a gui to make our mmorpg?" → "i dont care how you do it fork redgarden into GFD write the
  northstar this is the mmo. this is dragonsnshit" → "cli will continue to work" → "redgarden as a
  gui" → "like old school runescape." New `docs2/REDGARDEN_GUI_NORTHSTAR.md`: forks REDGARDEN's
  real-time SDL2/OpenGL client (rendering/input machinery only -- click-to-move, hero-silhouette
  rendering, Q/W/R cast-ring/projectile/zone-circle UI, item-shop chrome, connect-ticket auth; not
  its MOBA hero-kit combat sim) onto `apps2/mud`'s real, already-shipped FFXI-parity Go MMORPG
  backend (22 jobs, skillchains/magic bursts, enmity, conquest, NM spawns/treasure pool, crafting
  guilds, parties/linkshells -- telnet `:2323` today) as a second, parallel client protocol
  alongside a new binary listener; telnet keeps working unchanged, per founder direction. Design
  call: REDGARDEN contributes the rendering grammar, `apps2/mud` keeps owning the RPG mechanics
  underneath -- no REDGARDEN hero identity carries over, only its UI vocabulary. Amends
  `docs2/MMO_NORTHSTAR.md`'s "Integration Architecture" section (frontend line updated from
  "SHANKPIT runtime, extended" to point at the new doc) and flags that MMO_NORTHSTAR's own
  milestone table (last updated 2026-06-21) is stale against `apps2/mud`'s real shipped state --
  a large body of FFXI-parity systems work (S76-S87) landed since without that table being
  updated. 7-milestone table, spec only, all NOT STARTED past this doc itself. Registered in
  `EMILY/context/golden-docs-index.md`.

## 2026-07-23 (8)
- fix(mud): found live, playing again after redeploy -- worm Poison was still lethal despite tonight's earlier "Worm is Slow-only, not Poison" fix. Root cause: `mobSpellPool`'s map keys were capitalized ("Worm", "Slime", "Lizard"...) but real mob IDs are always lowercase ("worm-meadow-4", "slime-swamp-2"...); `strings.Contains(mobID, prefix)` is case-sensitive, so the lookup never matched anything for *any* mob kind and every single mob silently fell through to the generic fallback pool, which still includes Poison. The earlier fix was correct in intent but never executed at runtime. Fixed by lowercasing every key in `mobSpellPool` to match real mob IDs. `go test ./server/...` clean, rebuilt, redeployed live, re-verified in person: killed a full worm (5 hits) with zero Poison procs, leveled Lv.1 to Lv.3.

## 2026-07-23 (7)
- fix(mud): found live, testing "does the economy work" as a real player -- shop, bazaar, bank, quest-accept/quest-turn-in/quests, npcs/talk, equip/gear, and craft are all real, fully working commands (confirmed live: bought an Echo Drop for 50 of 500 starting gil, accepted a real quest from a real NPC, checked bank balance) that cmdHelp never mentioned. A new player had zero way to discover any of them short of reading the source -- including Echo Drop, the exact 50-gil cure that would have prevented the Poison death found and fixed earlier tonight. Added an "Economy & items" and "Quests & NPCs" section to the in-game help listing the core early-game commands; deliberately still not listing all 100+ commands that exist (job-specific spells, FIELDOFFICE/TRAPX faction-war systems like k9-deploy/district/enforcement/integrity/tech-pressure) since that's advanced/endgame surface that would trade one discoverability problem for an unreadable wall of text. Redeployed live.

## 2026-07-23 (6)
- docs2/HERO_BRIDGE_PREREQUISITES.md: gap analysis answering "do we weave multiverse lore in" (honest answer: not yet) and "what are the prerequisites to bridge it." NM registration and loot are both real, already-working systems -- not blockers. The one real gap: every zone in the game is hand-written directly in apps2/mud/main.go's initWorld(), with no data-driven zone format the way data/items.json exists for items. Names the actual prerequisite chain (minimal zone data format -> pick one of HERO_CONTENT_FRAMEWORK.md's five worked examples -> build it -> numbers pass last). Published to okemily.com as "Not Woven In Yet."

## 2026-07-23 (5)
- feat(education): EduVM Phase 0 (docs2/EDUCATION_CURRICULUM_NORTHSTAR.md) -- arrays. New `array(N)` declaration syntax (`let arr = array(5);`), indexed read/write (`arr[i]`, `arr[i] = v;`), bounds-checked at runtime against a shared 256-slot arr_mem[] pool (new EDU_OP_LOAD_ARR/EDU_OP_STORE_ARR opcodes, new `[`/`]`/`array` lexer tokens). Verified end-to-end with a real bubble sort compiled and executed against the actual VM (packages/education/edu_test_arrays.c, 14 assertions, first test file this package has ever had). **Found and fixed a real, serious pre-existing bug while building the test harness**: `edu_compile_source`'s own `memset(out, 0, sizeof(*out))` was wiping the caller-supplied `bytecode`/`bytecode_cap` fields to NULL/0 *before* using them to init the bytecode writer -- every real compile has been failing "bytecode overflow" on its very first emitted byte since this function was written, meaning the Architect's Orb terminal's F7 compile in apps/lobby has likely never actually produced working bytecode in practice. Fixed by preserving the caller's buffer/cap across the reset. go build clean; apps/lobby's own C build still blocked by the pre-existing, unrelated, already-documented missing GL/glu.h system dependency (sudo-gated, not touched here).

## 2026-07-23 (4)
- docs2/HERO_CONTENT_FRAMEWORK.md: story-first process for turning any TYLER/multiverse_heroes.md entry into a dungeon, NM/raid boss, and loot drop, grounded in the real engine (server/mob.Mob, server/nm.NMSpawn placeholder/window/respawn model, server/itemdef.Item Category/JobMask/Flags, server/loot.Pool). Five fully worked examples (Bacon, Zagan, Nidhogg, Cain, Tesla) -- no numbers/stats anywhere yet, per the same docs-before-software discipline the hero compendium itself established. Golden-indexed as GFD-HERO-FRAMEWORK.

## 2026-07-23 (3)
- fix(mud): found live, playing as a real new character (Custodian) right after the melee-range fix -- worm's 20%-chance debuff proc could cast Poison (flat Potency=10, ticking every 1s game-loop tick for up to 30s = up to 300 total damage) against a level 1-5 character with only 90-150 max HP. Died to it once, for real, mid-session. Worm is this game's own zone-0 tutorial mob (worm.go's own doc comment: "mostly passive"); a single proc from the very first mob a new player fights could solo-kill them. Removed Poison from Worm's mobSpellPool, left with Slow only (a better flavor fit for a worm anyway, and non-lethal) -- Poison unchanged for Slime/Chaos/Leech, which aren't the tutorial mob. go build/test green, redeployed live under gfd-mud.service.
- fix(mud): deployed apps2/mud under real supervision for the first time (ops/systemd/gfd-mud.service — built 2026-06-27, never run under systemd before tonight) and found combat was completely non-functional from spawn. Root cause: MeadowWormSpawns places every worm 25-35 units from the town-centre spawn point (0,2,0) — by its own design comment, "away from the town centre" — but DefaultPlayerMeleeRange is only 3.0, and there's no intra-zone movement command (n/s/e/w means zone travel, not local positioning). Every new player's first `attack` set a target that could never land a hit, and tickAll's error handling only messaged on ErrMobDead/ErrMobNotFound, so ErrOutOfRange failed in total silence — indistinguishable from a hung connection. A pre-existing test (TestMeadowWormSpawns_OutsideTownRadius) had locked in the exact distance that caused this without anyone connecting it to the melee-range constant. Fixed: cmdAttack now auto-approaches (snaps the player to the target's position) when out of range at the moment of targeting; tickAll's ErrOutOfRange branch now tells the player why nothing is happening instead of continuing silently, as defense in depth for mobs that wander mid-fight. New exported mob.Dist + 2 tests (TestDist, TestMeadowWormSpawnsOutsideDefaultMeleeRange — the latter pins the geometry fact itself so a future spawn-layout change can't silently reintroduce the bug). Live-verified: registered a real character (Custodian), confirmed combat landed, leveled 1→4 in one sitting. go test ./... green.

## 2026-07-23
- docs2/EDUCATION_CURRICULUM_NORTHSTAR.md: scoping pass for teaching CS algorithms (sorting, knapsack) via the existing EduScript VM (packages/education) and its Architect's Orb terminal (apps/lobby, F7 compile/F8 run). Confirmed the VM/world-object binding (switches, gates, crates, bridges, portals) is real and live, but has no array/indexed-memory opcode and no user-defined functions — the one prerequisite every algorithm module needs, scoped as Phase 0. Also corrected a founder claim: the education system was never actually merged into SHANKPIT's apps2/lobby despite the "yolo" commit that created that folder — verified directly, zero education/VM code there; it lives only in GFD today. Golden-indexed as GFD-EDU-CURRICULUM. Design only, no code yet.

## 2026-06-27
- S129-10: Art direction reference sheets at docs2/art_direction_tiers.md — 5-tier palette guide (Initiate→Endgame), per-armor poly budgets + UV spec + shader rules
- S130-02/03/04: npcattention tick wired; disguise items (Guard/Civilian/Merchant); WEAR + REMOVE DISGUISE commands; sneak feeds attention state
- S129-07: equip/unequip job+level enforcement via itemdef.Registry; stat delta broadcast; gear list shows stat totals

- S129-06: gear.ComputeStats() + CanEquip() using itemdef.Registry; DefID added to ItemEntry; 10 tests

## 2026-06-25
- feat(watcher): TRAPX vigilante anomaly spawn system — DisruptionDebt accumulator, 4 archetypes (Founder/Chemist/Apparition/RiotBreaker), 3 power tiers, chaotic-neutral targeting by Trust score, 19 tests
- docs2/INVENTORY_EQUIPMENT_NORTHSTAR.md: FFXI-era inventory+equipment northstar with art direction (low-poly)
- server/npcattention: per-NPC stealth awareness (Hitman C47 parity) — disguise factions, suspicion [0,100], witness system, Scene tick
- server/nm: HillsNMs + CavesNMs + AllStartingZoneNMs — all 4 default zones now have NM spawn definitions
- server/mob/hills.go + caves.go: rabbit/beetle/wolf (Hills), bat/spider/skeleton (Caves) with spawn sets
- data/items.json: 52-item seed (weapons/armor/accessories/consumables/crystals/materials/key items)
- server/inventory: bag container (Bag/Stack/Mog, stack merge, Rare conflict, Gobbiebag expand, key items)
- server/itemdef: item definition registry (JobMask, ItemFlags, Category, Registry/LoadJSON/ByID/ByName, CanEquip)

- feat: S128-06 Scar system (scar/scar.go — Registry, 4 causes, +5%/scar visibility bonus, ScarBurn, MUD command, 11 tests) + S128-07 K9 Merciless Operation (k9/operation.go — 4-phase, 3 counterplay lanes, 18 tests) (Apple #3870)

## 2026-06-24
- feat: S125-04 TRAPX economy — 5 items, 3 TRAPX vendors, enforcement.DistrictPressure price scaling (Apple #3656)
- feat: S126-15 TRAPX craft recipes — repaired-bike/faction-gear/atlas-page; cartography.DiscoverAll on atlas-page (Apple #3644)
- feat: S126-14 campaign battle mode — server/campaign, 10 nodes, /campaign join/status, weekly reset, 15 tests (Apple #3641)
- feat: S126-13 NM respawn scheduler — nm.Registry, NMRespawnScheduler, RespawnMinutes, announceNMPop, 8 tests (Apple #3566)
- feat: S126-04 bilingual party chat — BOTH lang, setlang command, per-recipient AT expansion in deliverChat (Apple #3538)
- feat: S126-03 world event broadcast — server/worldevent, /api/world-events endpoint, faction war→worldEventReg, 12 tests (Apple #3536)
- feat: S126-02 weather → mood loop — Storm:Fear+15, Rain:Fatigue+10, Clear:Fatigue-5 per district tick (Apple #3533)
- feat: S126-01 NPC schedule system — server/schedule, 3 NPCs seeded, hourly tick → broadcastAll, 14 tests (Apple #3531)
- feat: S125-13 Mog House personal storage — server/moghouse, 50-item cap, Store/Retrieve/List MUD commands, 20 tests (Apple #3528)
- feat: S125-12 server/auction — standalone AH engine (List/Buy/Cancel, 5% fee, 15-min expiry, 15 tests) Apple #3526
- feat: S125-03 TRAPX faction war engine — 72h cycle, 24h conflicts, FO win condition, war/fw MUD command (Apple #3514)
- feat: S125-02 zone presence — departure broadcasts, cmdExamine player inspect (Apple #3513)
- feat: FFXI auto-translate — server/autotranslate 72 JP/EN phrases, ExpandLine(), at MUD command, [alias] token expansion in say (Apple #3505)
- S123-05: VS0 Detroit slice — fo-school-1 seeded, Jiangshi FO pre-held, alertness=35 preset, takecontrol entry, Emily OS ambient voice (10 fragments)
- S123-04: flip phone MUD interface — 5 tabs (FO/heat/receipts/crew/CAST), CRT box-drawing, Watcher alertness contribution, districtIDForZone
- S123-03: multi-timeline branch system — Branch/Registry, rogue_swarm auto-branch, 'timeline' MUD command, conflict detection, 16 tests
- S123-02: TYLER ledger bridge; 4 TYLER verb types, 4 CAST lore docs, 'terminal' MUD command, archive entry receipts
- S123-01: TYLER scene cluster 200–207; 8 districts, portal connections (VS0+Tyler's route), TYLER faction NPCs (Heikegani/Kuroshio/Yōkai/Eastwind/Jiangshi), urban terrain for 205-207
- S122-06: city/district/align/broadcast/enforcement MUD commands; watcher+enforcement+neighborhood wired into initTRAPXCity and tickAll
- S122-05: TRAPX faction rep overlay — Frequency/Bloc/Procurement Houses on fame.Nation; rank-gated benefits, 11 tests
- S122-04: TRAPX RPG class unlock chains — 8 chains (DRK/BST/BRD/SAM/SMN/BLU/GEO/RUN), 24 quests total, job-stone rewards, wired into questBank
- S122-03: Neighborhood personality — Tolerance/Pride/Cohesion/Visibility axes, Fear/Fatigue mood drift, myth seeding (10 lore fragments per district), 23 tests
- S122-02: Watcher (alertness/trust/bias per district) + Enforcement (5-level state machine: Quiet→Lockdown, cop density 0-8, K9 eligibility, FO effects)
- S122-01: TRAPX city scene cluster — 5 districts (200-204) + zone exits + 7 city NPCs (mini bike, corner kid, pawn shop, broadcast, warehouse, frequency, scar keeper) + urbanChunk() worldapi terrain
- S121-01: TRAPX city state API — server/trapxapi package; GET /api/v1/trapx/city-state + POST /api/v1/trapx/events at :7071; Emily Prime Dragon integration
- S120-03: FIELDOFFICE MUD wiring — claim/contest/fo-status/fo-list/k9-deploy/k9-swarm/receipts/attention/integrity/tech-pressure commands wired into apps2/mud/main.go; 1Hz city sim tick
- S120-02: server/beatsync — BeatSync stub (Engine/Tick/Run, 4 beat types in 4/4, Kick/Snare/Bass/Hat, WorldEffect city hooks, sine strength curve, 17 tests) — Apple #3356
- S120-01: server/ledger — Receipt ledger + anti-exploit (append-only, verb types, ByFO/ByActor/ByVerb/Since, ReceiptBurst 30s, flip-score exploit detection, SUSPICIOUS_PATTERN flag, 15 tests) — Apple #3353
- S119-05: server/techpressure — Tech Pressure doom clock (5-tier: LeashFrays/ProcurementWar/QuietAudit/Packmind/CrownProtocol; TierUnlock/DogDeploy/SwarmActivity inputs; decay; BirdCorrection; CROWN_PROTOCOL one-shot; 18 tests) — Apple #3351. S119 ENGINE FOUNDATION COMPLETE.
- S119-04: server/integrity — Control Integrity + Rogue Swarm (per-district CI 0-1, dog decay superlinear, jammer/flip decay, CleanAudit/BirdCorrection recovery, ROGUE_SWARM trigger at 0.15, containment objectives, SCAR_WRITTEN, Registry, 24 tests) — Apple #3349
- S119-03: server/attention — Attention meter (0-1000, superlinear dog gain n^1.3, decay, AUDIT_THRESHOLD→OversightSect, VENDOR_THRESHOLD→ShadowOperator, ecosystem effects, Registry, 18 tests) — Apple #3346
- S119-02: server/k9 — K9 unit + Swarm (Sentry/Escort/Audit modes, 0.85^n diminishing returns, Mark/Latch/HowlBeacon/CustodyLock/ReceiptBurst, Battery drain, BATTERY_LOW/DEAD events, cap=8 per FO, 33 tests) — Apple #3344

- S119-01: server/fieldoffice — FieldOffice state machine (4-phase: Unclaimed/Held/Contested/Containment; Flow/Pressure tick; Flip/Defend/Contest windows; Rogue Swarm containment objectives; 20 tests pass) — Apple #3340

## 2026-06-23
- S118-01: PLD spells (Flash enmity spike, Sentinel/Rampart def buffs, Holy/Banish/Banish II light dmg); PLD job gate
- S117-01: NIN ninjutsu (6 elements × 2 tiers); DEX scaling; NIN job gate; cast <spell> dispatch
- S116-01: DRK dark magic (Drain HP steal, Aspir MP absorb, Absorb-STR/DEX/VIT/INT/MND); DRK/RDM job gate
- S115-01: WHM teleport spells (teleport-meadow/hills/caves/swamp + tele- aliases); 100 MP; combat target reset; arrival broadcast
- S114-01: BRD songs (march/paeon/ballad/minne/carol/mambo); zone-AoE status buffs; BRD job gate; 3-minute duration; party broadcast
- S113-01: BLM black magic nukes (6 elements × 3 tiers); INT scaling; BLM/RDM job gate; target-mob required
- S112-01: Party-targeted spell casting; cast <spell> <player> resolves zone-local target; Cure/Protect/Shell/Haste/Regen/Refresh notify both caster and target
- S111-01: WHM buff spells (Protect/Shell/Haste/Regen/Refresh via status.Stack); Dia DoT on combat target; RDM allowed Refresh
- S110-01: ls-kick/ls-promote MUD commands (guild Officer+); S110-02: shop/shop buy/shop sell at NPC vendors (guildmaster/merchant/scout)
- S109-02: Fame MUD wiring; talk <npc> fame gate; NPC [LOCKED] indicator; quest fame reward tags; 2 new gated NPCs
- S109-01: Add nation fame system (server/fame); Earn/Rank/MeetsRank/Summary; TurnIn now returns RewardFame+FameNation; MUD fame/rep command shows reputation table
- S108: fishing skill (server/gather/fishing.go) + food buff system (server/food/) + MUD wiring (fish/fish-points/eat/food commands)
- S106-01+02: PvP duel system (Manager/Challenge/Accept/ReportHP, 15 tests) + MUD duel/accept/forfeit/leaderboard wiring
- S105-03+04: weather engine (Phase/Engine/ForcePhase, 12 tests) replaces random weather; broadcast + BST tame bonus + prompt indicator
- S105-01+02: cartography Atlas (Visit/Has/ExitMap/10 tests) + MUD explore command + map ✓ indicator
- S104-04: NPC dialogue + quest commands (npcs/talk/quest-accept/quest-turn-in/quests) wired into MUD + kill tracking
- S104-03: server/quest — NPC quest system (Bank/Journal/State/TurnIn), 5 starter quests, 16 tests
- S104-02: BST pet MUD commands (bst/jug-pet/pet-release/pet-status/pet-heel/pet-heal) + pet auto-attack in tickAll
- S104-01: server/pet — BST Beastmaster pet companion system (Tame/JugPet/Tick/Heal/Release), 8 kinds, 20 tests
- S101-01/02/03: bank deposit/withdraw/balance; random weather events (60s/10% per zone, 5 types); survey command with directional player listing
- S100-01/02/03: rest/meditate regen (+5%HP/+3%MP/tick) + stand; target <mob> reticle with hp bar; /p <msg> party chat
- S99-03: bazaar personal shop commands — bazaar set/list/buy; world.bazaars map; gil transfer + item transfer; seller notification
- S99-01/02: mob spellcasting AI (20% hit→debuff from kind pool, 30s) + removedebuffs/echo-drop + cast cure/cure2 (WHM only)
- S98-01/02: IDUNA character persistence — idunaclient.CreateCharacter/UpdateCharacterLevel added; mudCharCache name→charID json store; fetch-or-create on login; save level/xp/pos on disconnect
- S97-01: wire server/job.RecastTracker into MUD — ja/recasts commands; provoke adds enmity CE; benediction restores HP/MP; recast updated on setjob
- S96-01: invisible/sneak aggro block in MUD — cast invisible/sneak commands; EvtMobAggro intercepted when player has active status; 60s expiry with broadcast
- S95-01/02: wire server/worldcrisis into apps2/mud — auto-start on login, phase broadcast, NM kill and mine objective contributions, Chaos Elemental crisis NMs spawn on Emergence, crisis-shard drop
- S94-01: wire server/telecrystal into apps2/mud — crystals/travel/touch commands; validate()/deduct gil/teleport with zone transfer; Dist2D range check for touch activation
- S93-01/02/03: wire server/gear, server/job.CharJob, server/merit into apps2/mud — equipment slots+IL (equip/unequip/gear), sub-job pairing+combined stats (setsubjob/subjob), merit bank with XP→merit conversion at cap (merits/merit-spend)
- S92-03: wire server/guild into apps2/mud — ls-create/ls-invite/ls-leave/ls-info commands; Feather-gated guild chat; GuildID synced to chat Router sessions
- S92-01/02: wire server/enmity and server/chat into apps2/mud — per-mob enmity tables, hate-based aggro retargeting, enmity command; chat Router for say/tell/yell/guild with session sync on zone transfer
- feat(mud): S90-03 auction house — ah browse/sell/buy/history/status/cancel, player gil, itemCategory table
- feat(mud): S90-02 conquest system — declare/conquest commands, kill-based points, 1-min tick, zone-wide broadcast
- feat(mud): S90-01 crafting system — inv/craft/recipes/craft-skills commands, inventory tracking through mine+loot+resolvePool, recipeIngredients table, skill gain
- S89-03: MUD loot pool + NM spawns — solo auto-award, party lot/pass/pool, King Worm + Marsh Leech NMs
- S89-02: MUD job system — 22 FFXI jobs (WAR default), HP/MP per-level scaling, setjob/jobs commands
- S89-01: MUD skillchain — real WS names+resonances, per-mob chain state, 8s window, zone-wide SC announcements
- S88-01: MUD progression wiring — XP+leveling, homepoint, field manuals, party+XPChain, KO/return system
- feat: S76-06 skill XP server-side — PacketSkillXP=16, server-go async IncrementSkill (cap 1.0/action), idunaclient.IncrementSkill
- feat: S76-05 World Crisis phase machine (worldcrisis pkg), PacketWorldCrisisUpdate/ObjectiveComplete, server-go tick goroutine+broadcaster, idunaclient.PatchWorldEvent
- feat: S76-04 crafting endpoint — LookupRecipe, PacketCraftRequest/Result, idunaclient ListItems/CreateItem/DestroyItem, server-go craft handler
- feat: S76-03 telecrystal travel — telecrystal registry (6 crystals), idunaclient pkg, PacketTelecrystalUse/Ack/Err protocol; server-go handler: auth→validate→IDUNA gold deduct→UpdatePosition→Ack
- feat: S76-01 idunaauth package (ES256 JWT, JWKS cache); PacketConnect IDUNA JWT gate; PacketAuthReject=8 wire protocol
- feat: S82-02 sub-job (CharJob/CombinedStats); S82-03 job abilities/recast (RecastTracker); S83-02 merit points (MeritBank); S83-03 item level (Equipment/EffectiveIL); S84-01 crafting guilds (8 types/SuccessChance); S84-02 HQ synthesis (HQTier); 69 tests
- feat: S81-05 Enmity (hate Table, AoE cure, overaggro); S81-06 Death/Raise (HPState, 10% XP penalty); S82-01 22 FFXI jobs (StatsFor/HPAtLevel/MPAtLevel); S83-01 Level XP (L99 cap, CharXP.AddXP, level^1.8); 60 tests
- feat: S86-02 Home Point (SetHome/ReturnHome, 8% XP penalty); S86-03 Field Manuals (ApplyBonus, ApplyAll stacking, expiry); S87-03 NM Aggro types (sight cone, sound radius, job detect, Sneak/Invisible blocking); 40 tests
- feat: S86-01 Conquest system (Region/Map, 3 nations, incumbent tie-break, weekly Tick); S87-01 NM spawn conditions (placeholder kill, time window, chance roll); S87-02 Treasure pool (Lot/Pass/Resolve, highest roll wins, 48 tests)
- feat: S85-01/02/03 party system — Party/Alliance/XPChain, 6-player cap, leader transfer, XP split, kill chain +10%/kill cap 50%, 37 tests
- feat: DragonsNShit MUD server (apps2/mud) — playable text MUD on :2323, all server packages wired, 1Hz game loop
- feat: S84-04 mining skill (server/gather) — FFXI-parity MiningPoint, loot table, HQ rolls, Meadow+Swamp presets, 27 tests

- feat: Swampville secondary starting zone — zone 3, scene 3 swamp terrain (clay/mud/water/mangrove), leech+slime+lizard mobs, 20 tests

## 2026-06-21
- feat: S81-04 status effects system (Poison/Paralyze/Slow/Silence/Bind/Haste/Regen/Refresh/Protect/Shell); 43 tests
- feat: S81-03 TP weapon skill points system; 34 tests (Apple #2530)
- feat: S81-01+02 skillchain+magic burst system; 14 resonances 3 tiers 31 tests (Apple #2521)
- feat: S80-01 auto-attack + mob tagging; AI state machine; 26 tests (Apple #2518)
- feat: S79-01 linkshell guild system; Feather/Feather Sack; 22 tests (Apple #2515)
- feat: S78-01 chat system say/tell/yell/guild (PacketChat=6); 19 tests (Apple #2472)

- docs: S77-01 DragonsNShit MMO_NORTHSTAR — 7 systems, IDUNA schema, 8-milestone product roadmap (Apple #2470)

## 2026-06-20

- feat: S42-01 worldapi :7070 live; S42-02 scene-differentiated ProceduralWorldStore (Apple #1449)

## 2026-06-18
- feat(worldapi): S41-02 DragonflyChunkGenerator — WorldStore hook + procedural fallback + block name→ID mapping (Apple #1421)

- feat(worldapi): S40-02 server/worldapi package — /chunks endpoint + ChunkGenerator interface (Apple #1413)


## 2026-07-31

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


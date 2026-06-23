## 2026-06-23
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


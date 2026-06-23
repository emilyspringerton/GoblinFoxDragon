## 2026-06-23
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


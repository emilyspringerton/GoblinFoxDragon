# DragonsNShit MMO — Product Northstar

*Last updated: 2026-06-21*

---

## Three-Sentence Version

DragonsNShit is a persistent-world action MMO built on the Dragonfly/Bedrock engine where
players, guilds, and factions shape a living world through combat, economy, and in-world
scripted events — all running on the same server-authoritative UDP stack that powers SHANKPIT.
IDUNA is the identity and auth layer: every player account, item, guild membership, and
season-end snapshot is durably anchored in IDUNA, making cross-season continuity and
item provenance cryptographically auditable.
The product ships in layers: SHANKPIT is Season 1 (FPS arcade) → portal travel unlocks
DragonsNShit persistent world scenes → full MMO economy and guild systems come online by
Season 3.

---

## The Product

### What DragonsNShit Is

A third-person action MMO in a dark fantasy world. Players inhabit cities, mines, docks, and
wilderness zones connected by Telecrystal travel networks. The world is always on, evolving
through server-driven World Crisis events, seasonal resets, and player-shaped economies. You
don't log into a session — you log into a *world that has been running without you*.

### What Makes It Different

- **Server authority is real:** No client-side cheating. All combat, trade, and movement is
  adjudicated by the Go backend. The C/SDL2 client is a trusted viewport, not the source of
  truth.
- **Item provenance is durable:** Every item has a creation record — who crafted it, from what
  materials, on what date, in what world phase. This trail is anchored in IDUNA and backed to
  the APPLES audit log.
- **World Crisis events are the heartbeat:** The server runs scheduled existential events
  (World Crisis VS0 spec) that no single guild can solo. They require coordination across
  hundreds of concurrent players and produce lasting world state changes.
- **EduScript enables live world scripting:** World designers write game events in EduScript
  VM bytecode without recompiling the engine. NPCs, objectives, and environmental systems
  are live-patchable.

---

## Core MMO Systems

### 1. Player Identity — IDUNA

Every player account is an IDUNA identity. Authentication flow:
- Client connects → IDUNA issues ES256 JWT → server validates on every packet (M2M, not
  per-packet overhead; session token is stateless but server verifies at connect and scene
  transitions)
- Player profile: character name, guild membership, subscription tier, equipped items,
  skill progression — all stored in IDUNA
- Season-end snapshots are filed as Apples and committed to APPLES git repo

IDUNA records needed:
- `characters` table: character_id, player_id (IDUNA user), name, scene_id, position, gear
- `guild_memberships`: guild_id, character_id, role, joined_at
- `subscriptions`: used for premium cosmetics / supporter tier gating

### 2. Item System & Provenance

Every item in DragonsNShit has a lifecycle:

```
CRAFTED(character_id, materials[], world_phase, timestamp)
  → ITEM_ID (UUID)
  → stored in IDUNA items table
  → Apple filed: type=item_provenance

TRANSFERRED(item_id, from_character_id, to_character_id, trade_id)
  → IDUNA items.owner updated
  → Apple filed: type=item_transfer

DESTROYED(item_id, reason, world_event_id)
  → IDUNA items.destroyed_at set
  → Apple filed: type=item_destroyed
```

Items have a `provenance_chain` field: ordered list of `[event_type, actor, timestamp]`
entries anchored in IDUNA. This makes item history inspectable and season snapshots auditable.

Item categories for VS0:
- **Weapons**: melee (swords, axes), ranged (bows, thrown), magical staves
- **Armor**: light/heavy/robes with set bonuses
- **Consumables**: potions, food, bombs — non-durable (no provenance chain needed)
- **Reagents**: crafting inputs; provenance anchors here
- **Artifacts**: unique world-drop items with full provenance, owner history, and lore text

### 3. Guild System

Guilds are the primary social and economic unit:

```
guild_id (IDUNA)
├── name, tag, founded_at, founder_character_id
├── members: character_id → role (RECRUIT / MEMBER / OFFICER / LEADER)
├── bank: item_id[] + gold_balance
├── territory: zone_id[] (claimed regions)
└── apple_trail: all major guild events filed as Apples
```

Guild actions that file Apples: found, disband, member join/leave/promote, territory claim/loss,
World Crisis contribution logged.

### 4. Economy

- **Gold**: server-minted on mob kills, quest completion; burned on crafting and travel
- **Auction House**: server-authoritative; bids and sales recorded in IDUNA; provenance
  updated on transfer
- **Crafting stations**: scene-anchored (Forge in Town, Distillery in Docks, etc.); require
  reagents; output items get provenance record
- **Resource nodes**: respawn-gated (server controls timer); scene-partitioned so all players
  in a scene compete for the same nodes

### 5. Scene System & Telecrystal Travel

Current scenes: `SCENE_CITY`, `SCENE_MINES`, `SCENE_DOCKS` — defined in Telecrystal registry.

Travel is server-authoritative:
1. Player enters Telecrystal radius + presses G
2. Server validates: player has enough gold for cast cost, no combat lock
3. Server transitions player: `scene_id` updated in IDUNA, spawn position set
4. Client receives scene transition packet; old scene state cleaned up client-side
5. New scene voxel data streamed via `PACKET_VOXEL_DATA`

Scene isolation is per-player at the physics layer (no global `phys_set_scene` — each player
tracks their own scene context).

Planned scene expansion (post-VS0):
- `SCENE_WILDERNESS` — open world zone; PvP flagged; rare resources
- `SCENE_DRAGONSPIRE` — endgame tower; World Crisis final window occurs here
- `SCENE_UNDERCITY` — guild-territory-only zone; social + economy hub

### 6. World Crisis Events

Server-driven existential events defined in `docs2/specs/WORLD_CRISIS_VS0.md`.

Phase machine (authoritative Go state on server):
```
OMENS → BURROW → EMERGENCE → SPLIT WAR → FINAL WINDOW → RESOLUTION
```

Each phase has:
- Timer-based auto-advance
- Objective completion gates (cannot solo with one guild)
- Global meter `LEY_INTEGRITY` (0–100) visible to all players
- Telegraphed entry (skybox change, NPC warnings, world map indicator)
- Outcome recorded in IDUNA: event_id, phases_completed, participating_guilds[], outcome

World Crisis fires on a real-world calendar (weekly for VS0; daily during testing).

### 7. Skill Progression

Skills are server-tracked floats in IDUNA `character_skills` table:
- Combat: `melee_skill`, `archery_skill`, `magic_skill`
- Crafting: `smithing`, `alchemy`, `enchanting`
- Gathering: `mining`, `herbalism`, `fishing`

XP is awarded server-side on verified actions. No client-reported XP. Skill caps enforced
per season; season reset is configurable (soft reset = partial rollback; hard reset = full).

---

## Integration Architecture

**Frontend updated 2026-07-31** — founder: "graft redgarden frontend onto GFD mud as a gui...
this is the mmo, this is dragonsnshit." The frontend is now REDGARDEN's client (forked, MOBA
match/lobby concepts stripped, rendering/input machinery kept), grafted onto `apps2/mud`'s real
Go server as a second client protocol alongside its existing telnet interface — telnet keeps
working unchanged. Full design in `docs2/REDGARDEN_GUI_NORTHSTAR.md`; this section's diagram
below is kept for the systems-integration shape (IDUNA, Apples) which is unchanged.

```
REDGARDEN Client (forked, see REDGARDEN_GUI_NORTHSTAR.md) + telnet (unchanged, coexists)
  │   UDP snapshot/UserCmd packets (GUI) or text lines (telnet)
  ▼
Go Game Server (apps2/mud, extended with a second listener)
  │   validates via IDUNA JWT on connect
  │   all game state mutations → IDUNA API
  │   World Crisis phases → IDUNA event records
  │   item crafting/transfer → IDUNA items + Apple
  ▼
IDUNA (:8080)
  │   player identity, characters, items, guilds, skills
  │   Apple trail for all significant events
  ▼
APPLES git repo
  │   durable audit backup; season snapshots
  ▼
Emily Prime (EMILY :8086)
      observes Apple trail → files RSI tasks → drives development
```

**IDUNA is the single source of truth for all persistent game state.** The game server holds
in-memory session state only; all durable state writes go through IDUNA. A server crash loses
only the current tick's in-flight state — IDUNA is the checkpoint.

---

## IDUNA Extensions Needed

IDUNA's current scope is IAM + Apples + HEIMDAL sprints. MMO support requires new tables:

| Table | Purpose |
|---|---|
| `characters` | One per player; scene position, gear, gold balance |
| `character_skills` | Skill floats per character |
| `items` | Item registry with provenance chain (JSONB) |
| `guilds` | Guild metadata |
| `guild_memberships` | Character → guild role mapping |
| `world_events` | World Crisis event records |
| `scene_state` | Per-scene server variables (ley_integrity, active_phase) |

All tables use `apple_id` foreign key hooks: major mutations file an Apple and store the ID.

These belong in a new IDUNA migration: `IDUNA/cmd/migrate/migrations/YYYYMMDD_mmo_schema.sql`.

---

## Milestones

| # | Milestone | Acceptance | Status |
|---|---|---|---|
| 0 | EduScript VM | Architect Orb demo runs | DONE |
| 1 | Portal travel (multi-scene) | Two players in different scenes simultaneously | SPEC DONE |
| 2 | IDUNA player identity | Client authenticates; character record in IDUNA | NOT STARTED |
| 3 | Item system + provenance | Craft → transfer → Apple trail queryable | NOT STARTED |
| 4 | Guild system | Found, join, bank, territory | NOT STARTED |
| 5 | Economy (AH + crafting) | Auction house clears server-side; craft items provenance-anchored | NOT STARTED |
| 6 | World Crisis VS0 | 6-phase event; multi-guild gate; IDUNA event record | SPEC DONE / IMPL NOT STARTED |
| 7 | Skill progression | 9 skills server-tracked; XP on verified actions | NOT STARTED |
| 8 | Season system | Reset config; snapshot to APPLES; IDUNA soft/hard reset | NOT STARTED |

---

## What "Done" Looks Like

- 100 concurrent players in the same world across 3 scenes, fighting, trading, and crafting
- Every item in the world has a provenance trail in IDUNA going back to its creation
- World Crisis fires weekly; outcomes change the world state and are filed as Apples
- SHANKPIT portal travel brings FPS players into DragonsNShit scenes seamlessly
- Season snapshots are committed to APPLES git; any item's history is auditable post-hoc

---

## Related Docs

| Doc | Location |
|---|---|
| Engine northstar (studio) | `GoblinFoxDragon/docs/NORTHSTAR.md` |
| World Crisis VS0 spec | `GoblinFoxDragon/docs2/specs/WORLD_CRISIS_VS0.md` |
| Telecrystal network | `GoblinFoxDragon/docs2/specs/TELECRYSTAL_NETWORK_SPEC.md` |
| SHANKPIT bridge spec | `GoblinFoxDragon/docs2/specs/THE_BRIDGE_SPEC.md` |
| Scene registry | `GoblinFoxDragon/docs2/specs/SCENE_REGISTRY_SPEC.md` |
| Systems bridge spec | `GoblinFoxDragon/docs2/specs/SHANKPIT_DRAGONSNSHIT_SYSTEMS_SPEC.md` |
| IDUNA auth model | `IDUNA/CLAUDE.md` |
| REDGARDEN-as-GUI frontend (2026-07-31, current frontend plan) | `GoblinFoxDragon/docs2/REDGARDEN_GUI_NORTHSTAR.md` |

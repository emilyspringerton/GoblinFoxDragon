# GoblinFoxDragon — Northstar

*Last updated: 2026-06-14*

---

## Three-Sentence Version

GoblinFoxDragon is the EINHORN_INDUSTRIAL game engine R&D studio — the broader build space
where persistent-world mechanics, MMO systems, the EduScript VM, and new gameplay vertical
slices are prototyped before they become part of the shipped product.
SHANKPIT is the current shipped FPS product derived from this DNA; GFD is where the next
generation of the world gets built.
Module: `dragonsnshit`. Engine: Dragonfly/Bedrock. Relationship to SHANKPIT: parent studio.

---

## What GoblinFoxDragon Is

GFD is the engineering umbrella for the DragonsNShit universe — the technology platform that
will eventually power all EINHORN_INDUSTRIAL interactive products. It is not a game you can
play yet; it is the engine being built so that future games (and future seasons of TYLER) can
run on it.

**SHANKPIT relationship:** SHANKPIT is the first PRODUCT built on GFD DNA. The FPS shooter is
being shipped to Steam Early Access (S19 complete). GFD continues evolving the engine and
world systems underneath. Post-launch, SHANKPIT's BedWars mode and Dragonfly backend will
pull from GFD's persistent world tech.

---

## Architecture

```
GoblinFoxDragon/
├── packages/                      ← shared engine packages
│   ├── common/                    ← shared types, utilities
│   ├── education/                 ← EduScript VM (edu_vm, edu_bytecode, edu_entities)
│   ├── map/                       ← map/chunk representation
│   ├── protocol/                  ← network protocol definitions
│   ├── render/                    ← rendering abstraction
│   ├── rts/                       ← real-time strategy systems
│   ├── simulation/                ← world simulation
│   ├── ui/                        ← UI layer
│   └── world/                     ← world/voxel engine
│
├── packages2/                     ← second-generation packages (newer experiments)
│   ├── common/
│   ├── simulation/
│   ├── client-go/
│   ├── crystal/                   ← Telecrystal network
│   ├── lobby/
│   └── server-go/
│
├── apps/                          ← game application binaries
│   ├── server/                    ← game server
│   ├── client/                    ← game client
│   ├── lobby/                     ← lobby server
│   ├── bot_client/                ← bot client for testing
│   ├── training/                  ← ML/training harness
│   └── tests/                     ← integration tests
│
├── docs2/                         ← design docs and specs
│   ├── NETCODE_CONTRACT_SPEC.md   ← references SHANKPIT/docs2/ (canonical)
│   ├── specs/                     ← vertical slice and world specs
│   │   ├── WORLD_CRISIS_VS0.md    ← DragonsNShit MMO: world crisis event spec
│   │   ├── WEAKNIGHT_VS0_ACCEPTANCE_CRITERIA.md ← Bedrock racing vertical slice
│   │   ├── SHANKPIT_DRAGONSNSHIT_SYSTEMS_SPEC.md ← persistent world bridge
│   │   ├── TELECRYSTAL_NETWORK_SPEC.md ← in-world mystical comms
│   │   ├── THE_BRIDGE_SPEC.md     ← SHANKPIT ↔ DragonsNShit portal bridge
│   │   └── SCENE_REGISTRY_SPEC.md ← scene registration and portal resolution
│   └── ...
```

---

## Key Systems

### EduScript VM (`packages/education/`)
A sandboxed scripting language for DragonsNShit game content. Compiled bytecode, tree-walking
interpreter, entity bindings, lexer/parser/VM all in C. Enables world designers to script
events, NPCs, and world behaviors without recompiling the engine. The "Architect's Orb" demo
slice proves runtime scripting works.

### DragonsNShit Persistent World
The MMO backend. Dragonfly/Bedrock chunk streaming, server-authoritative voxel state, guild
and faction mechanics. Current vertical slices:
- **World Crisis VS0** — server-wide existential event with phase transitions (OMENS → BURROW → EMERGENCE → SPLIT WAR → FINAL WINDOW → RESOLUTION)
- **Weaknight VS0** — high-speed Bedrock physics gameplay (F1-tier racing, destructible terrain)

### Telecrystal Network (`packages2/crystal/`)
In-world mystical communication network. Player-facing analogue to the RSI loop's observation
system — events propagate through the Telecrystal network as the game's nervous system.

### The SHANKPIT Bridge
`docs2/specs/THE_BRIDGE_SPEC.md` defines the wire contract between SHANKPIT's FPS client and
the DragonsNShit world backend. Portal travel (`portal_resolve_destination()`) is the seam.
The FPS client sends UserCmds; the Dragonfly backend extends with `PACKET_VOXEL_DATA`.

---

## Milestones

| Milestone | Status | Description |
|-----------|--------|-------------|
| 0: EduScript VM | DONE | Sandboxed scripting; Architect Orb demo; entity spawn/tick bindings |
| 1: World Crisis VS0 | SPEC DONE | DragonsNShit MMO event spec; implementation pending |
| 2: Weaknight VS0 | SPEC DONE | Bedrock racing vertical slice spec; implementation pending |
| 3: Telecrystal Network | IN PROGRESS | packages2/crystal scaffolded |
| 4: SHANKPIT Bridge | BLOCKED | Waiting on SHANKPIT Milestone 3 (WorldBackend Go interface) |
| 5: Steam BedWars | NOT STARTED | Post-SHANKPIT EA launch; persistent world + BedWars mode |

---

## Relationship Map

```
SHANKPIT (shipped FPS product)
  ← uses → GoblinFoxDragon netcode contract (SHANKPIT/docs2/ is canonical)
  ← bridges to → DragonsNShit world backend via THE_BRIDGE_SPEC
  ← future → BedWars mode powered by GFD persistent world engine

TYLER (television series)
  ← specs live in → GFD/docs2/specs/TYLER_EPISODE_*.md (in-universe artifact versions)
  ← game layer → SHANKPIT Tyler mode (shankpit_tyler_mode.md)

EmilyOS / IDUNA
  ← identity durability → IDUNA handles player identity across season lineage
  ← audit trail → Apples log season-end snapshots
```

---

## Netcode Ownership

SHANKPIT owns the canonical netcode spec. GFD's `docs2/NETCODE_CONTRACT_SPEC.md` and
`docs2/CLIENT_PREDICTION_SPEC.md` are reference copies pointing to:
- `SHANKPIT/docs2/NETCODE_CONTRACT_SPEC.md` (server authority rules, packet contracts)
- `SHANKPIT/docs2/CLIENT_PREDICTION_SPEC.md` (client-side prediction reconciliation)

When they diverge, SHANKPIT wins. If GFD needs an extension, write a GFD-specific addendum
doc rather than forking the canonical spec.

---

## What "Done" Looks Like

- DragonsNShit world backend accepts SHANKPIT portal travel packets and returns voxel data
- EduScript VM is used to script at least one live game event (World Crisis VS0)
- SHANKPIT BedWars mode runs on GFD persistent world engine (post-Steam EA)
- Season lineage records are queryable via IDUNA after each season-end snapshot

---

## Related Repos

| Repo | Relationship |
|------|-------------|
| `SHANKPIT` | The FPS product; uses GFD netcode DNA; Bridge is the seam |
| `IDUNA` | Player identity and season lineage durability layer |
| `EMILY` | RSI loop drives GFD engine development |
| `TYLER` | TV show; game mechanics and episode specs cross-pollinate |

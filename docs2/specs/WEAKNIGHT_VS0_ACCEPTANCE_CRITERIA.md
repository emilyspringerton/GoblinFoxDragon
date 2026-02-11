# WEAKNIGHT: Bedrock Racers — Vertical Slice 0
## Acceptance Criteria & Definition of Done

## Core Goal
Prove that a voxel Bedrock-backed world can support **high-speed physics gameplay (F1-tier)**, destructible environments, emergent systems, and multiplayer-scale infrastructure while still being fun.

If this slice works:
- WEAKNIGHT can scale.
- DragonsNShit MMO is viable.

---

## Required Systems

### World & Tech
- Chunk-based voxel world using Bedrock backend.
- Destructible trees and terrain (raycast + block updates).
- Simplified voxel streaming to C client.
- Real-time physics interactions.

### Gameplay Layers
- High-speed racing with F1-inspired handling.
- Survival loops (resources, damage, destruction).
- Build macros (Fortnite-style instant structures).
- Vehicles: at minimum one F1 racer and one offroad/utility vehicle.

### Emergent Systems
- Boids-style flocking (NPC traffic/agents/wildlife).
- Trade routes driven by pheromone/heatmap signals.
- Power-grid cascades.
- Self-healing infrastructure.
- Evolving faction traits.

### Multiplayer Readiness
- Community-run server support.
- Deterministic world sync.
- No hard-coded singleplayer shortcuts.

---

## Flagship Requirement: F1 System (Red Bull / Max Verstappen Standard)

The handling model must be:
- Easy to drive at low speed.
- Brutally difficult to master at high speed.

### F1 Handling Acceptance Criteria
**Physics model must include:**
- Tire grip curves (not binary traction).
- Downforce scaling with speed.
- Braking zones with lock-up risk.
- Slip-angle behavior.
- Momentum conservation.

**Player feel targets:**
- Low speed: stable and accessible.
- High speed: twitchy, precision-heavy, mistakes punished.

**Track interaction targets:**
- Terrain deformability.
- Destructible obstacles.
- Dynamic debris that changes the racing line.

### “Max Verstappen Test” (internal gate)
The F1 system passes if it:
- Rewards aggressive optimal racing lines.
- Allows extreme recovery saves.
- Punishes sloppy inputs.
- Feels exhilarating and unforgiving.

Tester language gate:
- **Pass signal:** “This is hard but I want to go faster.”
- **Fail signal:** “This feels floaty.” / “This feels like Mario Kart.”

---

## Vertical Slice 0 Content Scope

### One Biome Arena
Must include:
- Destructible trees.
- Hills / elevation changes.
- Power infrastructure nodes.
- Simple road network.
- Trade-route NPC movement.

### Two Vehicles
- F1 racer (flagship).
- Utility vehicle (slower and tougher).

### Build Macros
Must ship with instant placement for:
- Ramp.
- Wall.
- Tower.

Placement consumes collected resources.

---

## Emergent Systems Acceptance Criteria

### Boids Flocking
Must:
- Avoid obstacles.
- Cluster naturally.
- Split under stress.
- Reform over time.

Valid examples:
- Traffic flow.
- NPC caravans.
- Wildlife groups.

Disallowed:
- Random wander behavior with no flock logic.

### Trade Routes
Must:
- Emerge organically from agent behavior.
- Strengthen with repeated use.
- Decay when disrupted.

Route destruction must measurably affect:
- Economy throughput.
- Faction strength.

### Power-Grid Cascades
Must:
- Propagate failures.
- Produce chain reactions.
- Trigger rebuild priorities.

Reference scenario:
- Destroy station -> city darkens -> trade slows -> faction weakens.

### Self-Healing Cities
Must:
- Rebuild after damage.
- Prioritize critical infrastructure first.
- Adapt layouts over time.

Disallowed:
- Fully scripted restoration sequences.

### Evolving Factions
Each faction must:
- Adapt traits over time.
- Respond to player pressure.

Trait examples:
- More aggressive.
- More defensive.
- Tech-focused.
- Trade-focused.

Traits must influence:
- NPC behavior.
- Resource allocation.
- Building style.

---

## Definition of Done (Vertical Slice 0)

Vertical Slice 0 is done only when all categories pass.

### Gameplay
- Player can drive the F1 car at extreme speed.
- Player can destroy terrain and trees.
- Player can place build macros instantly.
- Player can interact with the world simulation systems.
- Racing feel is physically grounded, skill-based, and intense.

### World
- Live chunk streaming works.
- Destruction state syncs across clients.
- No visual desync corruption.

### Systems
- Boids flocking is visible in-world.
- Trade routes emerge over time.
- Power failures cascade.
- Cities rebuild dynamically.
- Factions adapt traits.

### Multiplayer
- Community server can host sessions.
- Clients connect without one-off hacks.
- World state remains consistent across clients.

### Non-Negotiables
- No fake animations that hide missing physics.
- No scripted “emergence.”
- No hard-coded routes.
- No singleplayer-only shortcuts.

Everything must be system-driven.

---

## Performance Targets

### Minimum
- 60 FPS client during racing.
- Chunk update latency under 50 ms visible delay.
- Stable simulation with 20+ active agents.

### Stretch
- 120 FPS racing.
- 50+ active agents.
- Stable large-scale destruction events.

---

## Success Criteria (Fun + Product)
Vertical Slice 0 succeeds when:
- Testers voluntarily replay races.
- Players experiment with destruction.
- Emergent behaviors create surprising outcomes.
- Racing produces skill-based competition.

Bonus signal:
- Players start hosting community servers organically.

---

## Long-Term Alignment Check
If Vertical Slice 0 passes:
- WEAKNIGHT scaling path is validated.
- DragonsNShit MMO feasibility is de-risked.

If it fails:
- Fix systems now before MMO scope expansion.

---

## Final Philosophy
WEAKNIGHT is not:
- A Minecraft clone.
- A Fortnite clone.
- Only a racing game.

WEAKNIGHT is:
- A systems-driven sandbox.
- High-speed skill gameplay.
- Emergent world simulation.
- Community-centered infrastructure.

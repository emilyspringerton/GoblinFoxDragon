# DOCKS Scene Spec

## Overview
The Docks is a separate traversable scene (`SCENE_DOCKS`) that represents the industrial waterfront logistics district of New Hanclington.

## Purpose
- Establish economy flow in-world: **mines -> rail -> warehouses -> ships**.
- Provide a large combat/traversal sandbox distinct from Mines.
- Provide expansion hooks for ferry travel, customs, and future harbor content.

## Travel Integration
- Telecrystal: `TOWN -> DOCKS` from city waterfront.
- Telecrystal: `DOCKS -> TOWN` from docks arrival yard.
- Cast profile mirrors Mines pattern (`~1000ms total`, `~600ms commit`; return uses `900/500`).

## Spawn Points
- Arrival cluster near crystal yard (negative X, low Z lane).
- Return landing in town waterfront district near the dock gate.

## Geometry Zones (v1)
1. **Crystal Yard** (safe-ish arrival apron)
2. **Freight Spine** (parallel rail lanes + railcar cover)
3. **Warehouse Apron** (large warehouse belt and loading lanes)
4. **Container Maze** (stacked close-cover route)
5. **Quayside** (waterfront edge with ship silhouettes + cranes)
6. **Ferry Stub** (future gate/expansion marker)

## Visual Language
- Neon-brutalist industrial silhouettes.
- Large matte-black massing with selective cyan/amber accents.
- Readability first: collision silhouette over micro-detail.

## Future Hooks
- Harbor events and dynamic cargo objectives.
- Fishing/harvest nodes.
- Ferry travel and customs gate unlock.
- Faction NPCs and dock patrol encounters.

## Acceptance Criteria
- Town <-> Docks teleports function reliably.
- `SCENE_DOCKS` loads as an isolated scene.
- Spawn orientation and player state reset are stable on teleport.
- No kill-volume softlocks or out-of-bounds fall-through.
- Mines route remains unaffected.

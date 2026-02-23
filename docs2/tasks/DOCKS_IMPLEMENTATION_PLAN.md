# Docks Implementation Plan

## Phase 1: Scene Registration
- [x] Add `SCENE_DOCKS` to protocol scene constants.
- [x] Update scene name resolver to return `DOCKS`.
- [x] Add lobby resolver aliases `DOCKS` and `SCENE_DOCKS`.

## Phase 2: Telecrystal Links
- [x] Add `TELECRYSTAL_ID_TOWN_TO_DOCKS`.
- [x] Add `TELECRYSTAL_ID_DOCKS_RETURN_TOWN`.
- [x] Wire travel data in `TELECRYSTAL_DEFS`.

## Phase 3: Scene Geometry + Collision
- [x] Add `map_geo_docks` with large industrial district silhouettes.
- [x] Register `SCENE_DOCKS` branch in `phys_set_scene`.
- [x] Add docks spawn points and safety bounds.

## Phase 4: Runtime Hooks
- [x] Add `telecrystal_spawn_docks_mobs()` extension hook.
- [x] Trigger docks hook from `world_teleport_player_to_scene()`.
- [x] Keep projectile clear + camera sync behavior on teleport.

## Phase 5: UX + Debug
- [x] Add docks scene option in lobby scene picker.
- [x] Add route labels for major docks lanes.
- [x] Keep telecrystal drawing active for docks scene.

## Validation Checklist
- [ ] Town -> Mines still works.
- [ ] Town -> Docks works.
- [ ] Docks -> Town return works.
- [ ] Spawn yaw/pitch and movement flags reset on travel.
- [ ] `local_state.scene_id` and `player.scene_id` stay aligned.

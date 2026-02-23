# Telecrystal Network Spec

Canonical registry for scene-to-scene telecrystal travel.

| id | source_scene | position (x,y,z) | radius | prompt | cast total/commit | target_scene | spawn (x,y,z) | yaw/pitch | target_name |
|---|---|---|---:|---|---|---|---|---|---|
| `TELECRYSTAL_ID_TOWN_TO_MINES` | `SCENE_CITY` | `(270,0,60)` | 14 | `G: TELEPORT MINES` | `1000/600` | `SCENE_MINES` | `(0,0,-92)` | `0/0` | `MINES` |
| `TELECRYSTAL_ID_MINES_RETURN_TOWN` | `SCENE_MINES` | `(0,0,-92)` | 10 | `G: RETURN TOWN` | `900/500` | `SCENE_CITY` | `(270,0,60)` | `180/0` | `TOWN` |
| `TELECRYSTAL_ID_MINES_STUB_CRYSTAL` | `SCENE_MINES` | `(0,0,178)` | 10 | `G: CRYSTAL SHUNT` | `900/500` | `SCENE_MINES` | `(0,0,44)` | `180/0` | `MINE SPLIT` |
| `TELECRYSTAL_ID_MINES_SPLIT_CRYSTAL` | `SCENE_MINES` | `(0,0,44)` | 8 | `G: RETURN ENTRY` | `900/500` | `SCENE_MINES` | `(0,0,-92)` | `180/0` | `MINE ENTRY` |
| `TELECRYSTAL_ID_TOWN_TO_DOCKS` | `SCENE_CITY` | `(308,0,246)` | 14 | `G: TELEPORT DOCKS` | `1000/600` | `SCENE_DOCKS` | `(-178,0,-168)` | `95/0` | `DOCKS` |
| `TELECRYSTAL_ID_DOCKS_RETURN_TOWN` | `SCENE_DOCKS` | `(-178,0,-168)` | 11 | `G: RETURN TOWN` | `900/500` | `SCENE_CITY` | `(300,0,238)` | `210/0` | `TOWN` |

Notes:
- Use `SDL_SCANCODE_G` as interaction input.
- `cast_cancel_on_leave_ring = 1` for all travel links.

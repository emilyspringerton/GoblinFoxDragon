# Scene Registry Spec

Canonical scene IDs and aliases.

| Scene ID Constant | Numeric | Canonical Name | Accepted aliases |
|---|---:|---|---|
| `SCENE_GARAGE_OSAKA` | 0 | GARAGE_OSAKA | `GARAGE_OSAKA` |
| `SCENE_STADIUM` | 1 | STADIUM | `STADIUM` |
| `SCENE_VOXWORLD` | 2 | VOXWORLD | `VOXWORLD` |
| `SCENE_CITY` (`SCENE_NEW_HANCLINGTON`) | 3 | NEW_HANCLINGTON | `NEW_HANCLINGTON_MOCKUP`, `NEW_HANCLINGTON`, `SCENE_CITY` |
| `SCENE_MINES` | 4 | MINES | `MINES`, `SCENE_MINES` |
| `SCENE_WAREHOUSE` | 5 | WAREHOUSE | `WAREHOUSE`, `SCENE_WAREHOUSE` |
| `SCENE_DOCKS` | 6 | DOCKS | `DOCKS`, `SCENE_DOCKS` |

Implementation surfaces:
- `scene_id_name(scene_id)`
- `lobby_resolve_scene_id(scene_id_string)`
- `scene_load(scene_id)` + `phys_set_scene(scene_id)`

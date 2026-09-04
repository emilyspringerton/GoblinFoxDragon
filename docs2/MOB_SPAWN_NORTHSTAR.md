# GFD Mob Spawn Management — NORTHSTAR

**Status:** scoped only, no code yet. **Kanban:** follow-on to GFD-MD-001 (mob drops manager,
shipped 2026-09-04). **Founder real-time (2026-09-04, three messages, folded into one ask):**

> "and then we need an interface for maintaining mob spawn information like we should be able
> to control which can spawn in which areas of which zone (like if you split the map up into a
> grid you can have a coordinates system like I-7 like ffxi uses so we can control if we want to
> turn bunnies on or off in the meadow etc which bunnies ffxi builds lots of different
> difficulties of mobs from the same models with repaints and sometimes without even repaints"
>
> "they like to have some way harder mobs pretty close to lower level mobs with the same model
> and texture just a different name its cute we should do likewise"
>
> "not sure if the dungeon mobs use the same interface or if thats going to require a separate
> interface"

**Big, unscoped ask — scoped per Principle 19, not swallowed whole.**

## Real investigation first

Checked `server/mob/hills.go`, `caves.go`, `swamp.go`, `worm.go` directly. Every zone's spawn
table (`HillsSpawns()`, `CavesSpawns()`, `SwampvilleSpawns()`, `MeadowWormSpawns()`,
`TownSquareWormSpawns()`) is a **hardcoded Go function** returning a literal `[]Mob` with
hand-picked `Pos{X, Y, Z}` values baked into source. There is no data file, no coordinate-grid
concept, no per-mob-kind on/off toggle, and no zone-bounding-box concept anywhere in the codebase
today — confirmed by grep, not assumed. This is a bigger gap than "just add a GUI": the grid
system itself and the data-driven spawn table it would edit don't exist yet.

There is also no mob "family"/difficulty-tier system. Each `Kind` constant (e.g. `KindRabbit`)
maps 1:1 to exactly one hardcoded stat block via its own `NewX(id, pos) Mob` constructor. FFXI's
own real pattern the founder is asking for — same model/texture, same base Kind, a harder
reskin-or-not variant nearby with its own name and stat multiplier — has zero representation
today, though `server/mob/dungeon.go`'s own `DungeonEliteHPMul`/`DungeonBossHPMul` constants are
a real, existing precedent for "same spawn code path, HP multiplier scales up a named variant."

**Direct answer to "does dungeon mobs use the same interface":** yes, mechanically, already true
today for loot (confirmed by tracing `resolveKill` → `openLootPool` → `dropsForMob`/
`mobDropReg.DropsFor(m.Kind)` — every kill anywhere, dungeon included, funnels through the one
generic path; dungeon `Kind`s are just real hero-ID strings like `ARENA_HERO_CART` or
`dungeon-minion`, no separate loot code exists for them). The same will be true for spawn
management once it's built: dungeon room population (`server/mob/dungeon.go`'s own
`newDungeonMob` + `DungeonRoster`) already reuses the plain `Mob` type and the same `Kind` field
— a spawn-table editor built against `Kind` naturally covers dungeon rows too, no separate
interface required. The one real difference: dungeon spawns are chosen per-run from
`DungeonRoster` (a fixed named-boss/elite roster), not from a static per-zone list — so a
dungeon's own "which bosses/elites are eligible for this dungeon" question is a real, separate
knob from "which rabbits spawn in grid cell C-4 of the Meadow," addressed as its own phase below.

## Phased plan

**Phase 1 — Zone grid coordinate system.** Give each zone a real bounding box (min/max X, Z) and
a cell size; derive an FFXI-style letter-number label (`I-7`) from any `Pos`. This is pure new
infrastructure (`server/zone` is the natural home — already exists, currently a lighter package)
with no GUI yet — a `CellFor(zoneID int, pos Pos) string` function and its inverse
(`CenterOf(zoneID int, cell string) Pos`), unit-tested against the real zone dimensions already
implied by today's hardcoded spawn positions (e.g. Hills spans roughly ±42 on X/Z).

**Phase 2 — Data-drive spawn tables.** Same real shape as `server/mobdrop` (GFD-MD-001): a new
`server/spawn.Registry` loaded from `data/mob_spawns.json`, one row per spawn point
(`{zone_id, cell, kind, count, enabled}`), replacing the hardcoded `HillsSpawns()`-style
functions. `enabled: false` is the literal "turn bunnies off in the Meadow" toggle. Existing
hardcoded positions become the seed data (mechanically converted via Phase 1's `CellFor`), so
this phase should not change spawn behavior on its own.

**Phase 3 — IDUNA admin GUI: `/admin/gfd-mob-spawns`.** Same direct-file-access precedent as
`/admin/gfd-items` and `/admin/gfd-mob-drops`: list/create/edit/delete spawn rows, grouped by
zone, with a real cell picker (a simple `I-7`-style text/dropdown input is enough for v0 — a
visual grid-click UI is a real, separate follow-up, not a blocker). Per-row enable/disable
checkbox is the real, literal ask.

**Phase 4 — Mob difficulty-tier variants ("same model, different name").** A new concept:
a `Variant` on top of a base `Kind` — same model/texture reference, a display name override, and
a stat multiplier (mirroring `DungeonEliteHPMul`'s own real precedent, generalized). Spawn rows
from Phase 2 can name a variant instead of (or in addition to) a bare `Kind`. Real, honest open
question for a founder-level decision: whether variants get their own drop tables (a "Greater
Rabbit" dropping better loot than a plain "Rabbit") or inherit the base Kind's table — FFXI does
both depending on the mob, so this needs a real per-variant override, not a global rule.

**Phase 5 — Dungeon roster editing.** `DungeonRoster` (which named heroes are eligible
boss/elite spawns per dungeon) is itself still a hardcoded Go slice, the same real pattern as the
NPC vendor catalog (S251-06) and the old `dropsForMob` before this pass. Data-driving it and
exposing it in the same admin surface is real, separate follow-up work, sequenced after Phases
1-4 land since it reuses the same "hardcoded Go table → JSON + registry → GUI" playbook.

## Explicitly out of scope for now

- A visual, click-to-toggle grid map (Phase 3 ships with a plain text/dropdown cell picker).
- Per-player or per-instance spawn variation (this is a single shared, persistent-world spawn
  table, matching GFD's own real architecture).

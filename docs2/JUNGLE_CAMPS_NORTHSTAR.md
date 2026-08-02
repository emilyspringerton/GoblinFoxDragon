# Jungle Camps — The Four Heavenly Kings — Northstar

*Written 2026-08-02. Founder: "ok for arena in the north south east and west (the bases are at the
corners between the corners if you fold it you get 4 spots we can spawn minions from and spawn
boss monsters that give buffs)" -> "this way those camps will spawn mobs that will eventually
assault the towers so it becomes difficult to stale out the game by never attacking towers" ->
"the four heavenly kings" (boss naming) -> scoped via AskUserQuestion: build in GoblinFoxDragon's
own fork (`apps2/battlegrounds_gui`'s local single-binary simulation), NOT REDGARDEN's own
`apps/arena_server` -- "work out of a single binary [for now], we will back port to the other one
[later]." REDGARDEN's own repo/server stays untouched by this doc, same boundary as every other
Battlegrounds change this session.*

---

## 1. What exists today (found while scoping this)

**Map geometry** (`packages/simulation/arena_game.h`/`.c`): a square arena, `ARENA_HALF_EXTENT`
(~51.78, golden-ratio-scaled). Only 2 of the map's 4 corners are real team bases —
`arena_fountain_position` places `ARENA_FOUNTAIN_COUNT` (2) fountains at diagonally opposite
corners (`(-corner,-corner)`/`(corner,corner)`, `corner = ARENA_HALF_EXTENT - 8.0f`); the other 2
corners host neutral shops (no fountain). This matches the founder's own framing exactly: "the
bases are at the corners" (2 of the 4), "between the corners if you fold it you get 4 spots" — the
4 edge midpoints (N/S/E/W) between adjacent corners, the classic MOBA jungle-camp position
(between bases/lanes, not on top of either).

**Minion-wave precedent**: a real, working lane-creep system already exists (`ARENA_LANE_CREEPS_PER_WAVE`,
`ARENA_LANE_WAVE_INTERVAL_MS`, `ARENA_CREEP_MARCH_SPEED`) — creeps spawn per-team, march a fixed
waypoint path along "the existing spawn axis" toward the enemy's own spawn line. Camp minions are
a real, structurally different case (spawn from a fixed neutral map point, not a team's own spawn
column; no owning team until aggroed/whatever direction the founder wants), but the wave-timer/
waypoint-march machinery is a close, reusable pattern to build from, not from nothing.

**Buff-on-kill precedent**: a real, working powerup/buff system already exists
(`ArenaPowerupKind` enum, currently `ARENA_POWERUP_BERSERKER`/`ARENA_POWERUP_REGEN`,
`ARENA_POWERUP_BUFF_MS` = 20s timed buff, per-hero remaining-duration field e.g.
`berserker_ms_remaining`, `ARENA_POWERUP_RESPAWN_MS` after pickup). A King's own buff-on-kill is
the exact same shape as an existing powerup pickup, just granted by defeating a boss instead of
walking over a static pickup -- extending this enum/system, not inventing a parallel one.

**"Towers" — real, but not what "stale out the game" implies yet**: a real tower system exists
(`ArenaTower`, `ARENA_TOWER_MAX_HP`/`_DAMAGE`/etc.), but it's a **node-guard**, not a base-defense
objective: "one permanent, neutral tower per node... gating capture BEFORE anyone owns anything...
a ONE-TIME early-game gate, not a recurring one." Once a node's tower is destroyed once, early
game, it's gone — this is NOT a repeatable "siege the enemy base" objective a match can be staled
out by ignoring. The founder's "so it becomes difficult to stale out the game by never attacking
towers" implies either (a) node towers count as "towers" here and the real anti-stall lever is
forcing players back toward the node-capture objective they're already avoiding, or (b) a real
base/objective-tower system doesn't fully exist yet and this doc's own camp-escalation mechanic is
partly motivating one. Not resolved here -- flagged as a real open question, see §5.

## 2. The mechanism

```
4 jungle camps, one at each cardinal edge midpoint (N/S/E/W), between the 2 fountain corners
and the 2 shop corners
    -> each camp spawns regular minions on a wave timer (reusing the lane-creep wave/march
       pattern) + one boss ("Heavenly King") per camp
    -> killing a King grants a real timed buff to the killer/team (reusing the existing
       powerup buff system, a new ArenaPowerupKind-shaped entry, not a parallel system)
    -> a camp left uncleared for [threshold] escalates -- its own minions eventually stop
       waiting and march toward the nearest node-tower/objective on their own, same
       waypoint-march idiom lane creeps already use, just neutral-aggressive instead of
       team-owned -- real anti-stall pressure, not just flavor
```

## 3. Architecture

### 3.1 Camp placement

4 fixed positions at the edge midpoints: `(0, ±ARENA_HALF_EXTENT-margin)` (N/S) and
`(±ARENA_HALF_EXTENT-margin, 0)` (E/W), same "-margin so it's never buried in terrain / always
reachable" convention `arena_fountain_position`/`arena_graveyard_position` already establish for
every other fixed map landmark.

### 3.2 Camp minions

New spawn-table entry per camp (kind, count, wave timer), built on the lane-creep wave/march
primitives already in the codebase rather than a new movement system. Unlike lane creeps, camp
minions have no owning team at spawn -- neutral-hostile until aggroed (same "neutral" framing
`ArenaTower` already uses for node guards).

### 3.3 The Four Heavenly Kings

One boss per camp, real HP/damage (boss-scale, not a lane-creep reskin), killable by either team.
Kill grants a real timed buff via the existing powerup-buff mechanism (§1) -- exact buff per King
not designed here (4 Kings could each grant a different buff, matching the "Heavenly Kings" trope
of four distinct guardians with distinct domains, or one shared buff type -- founder's call,
flagged in §5).

### 3.4 Anti-stall escalation

A camp uncleared past some real time threshold stops being purely passive: its minions (and/or
its King) begin marching toward the nearest live objective (node-tower, or a real base-tower if
§1's open question resolves toward building one) using the same waypoint-march idiom lane creeps
already use. This is the founder's own explicit design intent -- "so it becomes difficult to stale
out the game by never attacking towers" -- not an incidental side effect.

## 4. What this is not

Not built in REDGARDEN's own repo/server -- explicitly scoped to GoblinFoxDragon's own fork
(`apps2/battlegrounds_gui`'s local single-binary simulation) per founder direction; REDGARDEN's
`apps/arena_server` gets these changes backported later, as a deliberate separate pass, not now.
Not a new buff system -- reuses the existing `ArenaPowerupKind`/timed-buff mechanism. Not a new
movement/wave system -- reuses the existing lane-creep wave-timer/waypoint-march primitives. Not
resolving whether "towers" here means the existing node-guard towers or a not-yet-built base-siege
objective system -- real open question, not guessed at (§5).

## 5. Open questions (not resolved here)

- Do the 4 Kings each grant a distinct buff (matching the "Four Heavenly Kings" trope of four
  guardians with different domains), or one shared buff type?
- Camp respawn: once cleared, does the camp (minions + King) respawn on a timer, same as the
  existing powerup pickups (`ARENA_POWERUP_RESPAWN_MS`)? Real MOBA precedent says yes; not
  designed here.
- What "assaulting the towers" concretely means once a camp escalates -- do camp minions attack
  existing node-guard towers specifically, or does this motivate a new base/objective-tower
  system this doc doesn't otherwise require? §1's own open question.
- Escalation timer/threshold: how long can a camp sit uncleared before its minions march?

## 6. Milestones

| # | Milestone | Acceptance | Status |
|---|---|---|---|
| 0 | This northstar | Written, registered in golden-docs-index | DONE |
| 1 | Camp placement + minion waves | 4 camps at real N/S/E/W positions spawn neutral-hostile minions on a wave timer, reusing the lane-creep wave/march primitives | NOT STARTED |
| 2 | Four Heavenly Kings | One real boss per camp, killable, grants a timed buff via the existing powerup system on death | NOT STARTED |
| 3 | Anti-stall escalation | An uncleared camp's minions begin marching toward the nearest objective past a real time threshold | NOT STARTED |
| 4 | Camp respawn | Cleared camps repopulate on a timer, resolving §5's open question | NOT STARTED |
| 5 | Backport to REDGARDEN | Deliberate, separate later pass -- port the proven GFD-fork implementation into REDGARDEN's own `apps/arena_server`, per founder direction ("we will back port to the other one") | NOT STARTED |

## 7. Related docs

- `docs2/REDGARDEN_GUI_NORTHSTAR.md` -- Battlegrounds' own architecture; this doc's Milestone 5
  is the eventual bridge back to REDGARDEN's real server this northstar amends
- `docs2/DUNGEON_NORTHSTAR.md` -- the other "REDGARDEN heroes as content" design (arena hero AI
  repurposed as dungeon mobs/bosses); a related but separate use of the same roster

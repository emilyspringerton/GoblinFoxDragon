# Jungle Camps — The Four Heavenly Kings — Northstar

*Written 2026-08-02. Founder: "ok for arena in the north south east and west (the bases are at the
corners between the corners if you fold it you get 4 spots we can spawn minions from and spawn
boss monsters that give buffs)" -> "this way those camps will spawn mobs that will eventually
assault the towers so it becomes difficult to stale out the game by never attacking towers" ->
"the four heavenly kings" (boss naming) -> scoped via AskUserQuestion: build in GoblinFoxDragon's
own fork (`apps2/battlegrounds_gui`'s local single-binary simulation), NOT REDGARDEN's own
`apps/arena_server` -- "work out of a single binary [for now], we will back port to the other one
[later]." REDGARDEN's own repo/server stays untouched by this doc, same boundary as every other
Battlegrounds change this session. Designed the four Kings' buff mechanics ("i want the god of
music to give a buff that sticks around as long as one of your team has it... design the other
mechanics for the rewards for the other encounters") -> "check wikipedia for the 4 heavenly kings
budhism and update the designs accordingly" -- corrected the initial draft's directions/domains
against the real Shitennō (the Musician King is East, not North as first guessed); §3.3 now
checked against source, not invented.*

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
    -> each camp waves regular minions from 0:00 (reusing the lane-creep wave/march pattern)
    -> at 1:00, one boss ("Heavenly King") spawns per camp -- East/Music, South/Growth,
       West/All-Seeing, North/Wealth, each granting a mechanically distinct buff on death
       (team-viral, individual-stacking, team-flat, and proximity-aura respectively -- see §3.3)
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

**Wave-spawned, not a static pack** (founder, real-time: "minions should spawn from the camps in
waves") -- confirms the §1 framing: this reuses the lane-creep wave-timer/waypoint-march
primitives directly, not just "a similar pattern." New spawn-table entry per camp (kind, count,
wave interval), same `ARENA_LANE_WAVE_INTERVAL_MS`-shaped timer. Unlike lane creeps, camp minions
have no owning team at spawn -- neutral-hostile until aggroed (same "neutral" framing `ArenaTower`
already uses for node guards). Live from the opening bell, same as lane creeps' own
`ARENA_LANE_WAVE_INITIAL_DELAY_MS` -- only the King itself (§3.3) waits for 1:00.

### 3.3 The Four Heavenly Kings

One boss per camp, real HP/damage (boss-scale, not a lane-creep reskin), killable by either team.
**Don't spawn until 1:00 into the match** (founder, real-time) -- camps are live and their regular
minions are already waving (§3.2) from the opening bell, but the King itself is absent for the
first minute, same "real MOBA precedent... a match's opening seconds breathing room" reasoning
the lane-creep system's own `ARENA_LANE_WAVE_INITIAL_DELAY_MS` already uses, just a longer,
boss-scale version of the same idea -- early game stays about lanes/nodes, jungle contests open
up once the first minute's positioning has happened.

Each King grants a genuinely distinct buff mechanic, not four reskins of the same timer (founder
confirmed: give the four Kings different domains, not a shared buff type -- resolves §5's own
open question). Real names/directions/domains checked against Wikipedia's Four Heavenly Kings
(Buddhism) article, per founder direction ("check wikipedia for the 4 heavenly kings budhism and
update the designs accordingly") -- the actual Shitennō each guard a specific cardinal direction
with a specific, real domain, not an arbitrary one this doc gets to assign. Corrected once against
that source (an earlier draft of this doc had Music at North; the real Musician King is East):

**East -- Dhṛtarāṣṭra, God of Music.** Real iconography: carries a *pipa* (a lute-like stringed
instrument), leads the Gandharvas (celestial musicians). Buff: *Catchy Song* (attack speed + move
speed, a rhythm buff). Mechanic, founder's own design: **team-sticky, spreads on respawn.** Not a
flat timer -- the buff persists on the TEAM as long as at least one living member currently
carries it. A carrier who dies loses it personally, but if another teammate still holds it, the
team keeps the buff; the moment ANY teammate respawns, they pick it up too ("the song reaches
them"). Only truly ends once every carrier who ever held it is dead with no live relay left to
spread it to -- effectively un-siegeable in a single teamfight, the opposite of every other timed
buff in this game. Real new mechanic needed: a team-level buff-carrier set (not a single per-hero
timer), checked on every respawn event.

**South -- Virūḍhaka, God of Growth.** Real iconography: rules wind, carries a sword, leads the
Kumbhāṇḍas (and pretas). Buff: *Bloodroar* (stacking bonus damage). Mechanic: **individual,
stacking, fragile.** Each takedown while holding it adds a stack (more damage) and refreshes the
buff's duration -- but it is NOT team-shared and does not survive death: the instant the holder
dies, the buff and every stack are gone, no drop, no relay. High-risk/high-reward, rewards
continued aggression and punishes hesitation -- the deliberate opposite of Music's forgiving,
un-losable persistence, and a real thematic fit besides: a sword-wielding king whose own domain is
literally "growth" maps cleanly onto "grows stronger with every kill."

**West -- Virūpākṣa, The All-Seeing.** Real iconography: "sees all," associated with a serpent/
dragon (a red cord representing a nāga), leads the Nāgas, converts non-believers by his sight.
Buff: *Farsight* (team-wide vision reveal near camps/objectives + bonus gold from monster kills).
Mechanic: **team-wide, flat timer, utility not combat.** Everyone on the team gets it the instant
it's claimed, it just counts down like the existing `ARENA_POWERUP_BUFF_MS` powerups already do --
deliberately the simplest of the four (not every King needs an exotic mechanic), an econ/scouting
reward rather than a fight-winning one, and a direct fit for "the All-Seeing" specifically (not a
guessed association -- his real domain is sight).

**North -- Vaiśravaṇa, God of Wealth.** Real iconography: chief of the four kings, rules rain,
carries an umbrella/pagoda, leads the Yakṣas -- widely known outside this context too (Bishamonten
in Japan, one of the Seven Lucky Gods). Buff: *Bulwark* (damage reduction, umbrella-as-shelter) +
a smaller bonus-gold trickle to nearby teammates (a direct nod to his real wealth domain, not
invented from nothing). Mechanic: **proximity aura, not a carried buff.** While active, any
teammate NEAR the holder also gets the damage reduction (and the gold trickle), not just the
holder -- a support/grouping incentive with a normal timer (no team-wide auto-spread like Music,
no stacking like Growth), encouraging the team to physically cluster around whoever holds it
rather than scatter -- an umbrella large enough to shelter a group, not one person.

Four different persistence shapes on purpose: team-viral (Music/East), individual-fragile
(Growth/South), team-flat (All-Seeing/West), proximity-aura (Wealth/North) -- covers genuinely
different strategic incentives (protect your carrier vs. play aggressive vs. just wait out the
timer vs. group up) instead of four buffs that all just feel like different flavors of "+damage
for 20 seconds," and now each one's mechanic is also a real, checked fit for that King's actual
mythological domain rather than an arbitrary assignment.

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

- ~~Do the 4 Kings each grant a distinct buff?~~ Resolved: yes, four distinct mechanics, designed
  in §3.3 against each King's real mythological domain (East/Music-team-viral,
  South/Growth-individual-stacking, West/All-Seeing-team-flat, North/Wealth-proximity-aura).
- ~~When does the first King spawn?~~ Resolved: 1:00 into the match (founder). Camp minion waves
  themselves start immediately, same as lane creeps.
- King respawn after death: does a defeated King respawn later (like the existing powerup pickups'
  `ARENA_POWERUP_RESPAWN_MS`), or is each King a one-time kill per match? Not specified yet --
  real MOBA precedent (jungle bosses respawning) suggests yes, but the founder hasn't confirmed a
  timer.
- What "assaulting the towers" concretely means once a camp escalates -- do camp minions attack
  existing node-guard towers specifically, or does this motivate a new base/objective-tower
  system this doc doesn't otherwise require? §1's own open question.
- Escalation timer/threshold: how long can a camp sit uncleared before its minions march?

## 6. Milestones

| # | Milestone | Acceptance | Status |
|---|---|---|---|
| 0 | This northstar | Written, registered in golden-docs-index | DONE |
| 1 | Camp placement + minion waves | 4 camps at real N/S/E/W positions spawn neutral-hostile minions on a wave timer from the opening bell, reusing the lane-creep wave/march primitives | NOT STARTED |
| 2 | Four Heavenly Kings | One real boss per camp, silent until 1:00, each granting its own distinct buff mechanic on death (§3.3: East/Music, South/Growth, West/All-Seeing, North/Wealth) | NOT STARTED |
| 2.5 | Team-viral buff carrier system | New mechanic Music's own design needs: a team-level buff-carrier set that persists across deaths and auto-grants on respawn -- the one King mechanic with no existing primitive to extend | NOT STARTED |
| 3 | Anti-stall escalation | An uncleared camp's minions begin marching toward the nearest objective past a real time threshold | NOT STARTED |
| 4 | King respawn | Resolves §5's open question -- does a defeated King return on a timer, or once per match | NOT STARTED |
| 5 | Backport to REDGARDEN | Deliberate, separate later pass -- port the proven GFD-fork implementation into REDGARDEN's own `apps/arena_server`, per founder direction ("we will back port to the other one") | NOT STARTED |

## 7. Related docs

- `docs2/REDGARDEN_GUI_NORTHSTAR.md` -- Battlegrounds' own architecture; this doc's Milestone 5
  is the eventual bridge back to REDGARDEN's real server this northstar amends
- `docs2/DUNGEON_NORTHSTAR.md` -- the other "REDGARDEN heroes as content" design (arena hero AI
  repurposed as dungeon mobs/bosses); a related but separate use of the same roster

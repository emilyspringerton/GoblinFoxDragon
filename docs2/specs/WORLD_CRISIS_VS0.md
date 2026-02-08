# DragonsNShit: World Crisis (Vertical Slice 0)

## Goal
Demonstrate a server-wide existential event that:

- is visible and legible to everyone
- cannot be solved by a single guild (even rich/stacked)
- creates multiple valid roles (combat + non-combat)
- has stateful progression, escalation, and outcome
- produces lasting world change (even if only one town/zone is affected in VS0)

**Non-goals (explicitly deferred)**

- multiple threats / rotating seasons
- complex economy simulation
- long quest chains / VO / cinematics beyond minimal
- full admin panel (config file + console prints are enough for VS0)

## Acceptance Criteria

### A) Threat Lifecycle

**A1. World-state machine exists**

- Event has clearly defined phases:
  - OMENS → BURROW → EMERGENCE → SPLIT WAR → FINAL WINDOW → RESOLUTION
- Server authoritative phase transitions based on:
  - timers, and
  - objective completion gates
- **Pass/Fail:** If event can be “forced” into kill state just by DPSing one entity, fail.

**A2. Global meter**

- A single global value exists (e.g., `LEY_INTEGRITY` 0..100) that:
  - decreases automatically over time during active phases
  - decreases faster when objectives fail
  - increases when server completes stabilizing objectives
- **Pass/Fail:** Meter must be visible in UI and queryable via server console/log.

**A3. Telegraphing is unavoidable**

At least 3 telegraphs must occur before full combat window:

1. skybox/audio/lighting change in the threatened region
2. world map indicator / “fault line” visualization
3. NPC/system message warning with countdown to escalation

**Pass/Fail:** A player logging in cold must notice within 60 seconds.

### B) “Rich Guild Can’t Solo” Hard Locks

**B1. Multi-objective simultaneity gate**

Final vulnerability window requires ≥3 objectives completed concurrently within a time window (e.g., 5 minutes):

- Anchor Objective (keep pylons alive)
- Ritual Objective (stabilize nodes)
- Intercept Objective (stop a roaming head / convoy / add wave)

**Pass/Fail:** If one group can sequentially do them alone, fail.

**B2. Contribution cap / diminishing returns**

Any single party/guild doing repeated identical actions gains reduced progress after threshold (server-side).

Diminishing returns applies to:

- damage to “progress-critical” targets
- anchor repair
- ritual contribution

**Pass/Fail:** A maxed guild stacking 40 people in one spot cannot brute-force the entire phase.

**B3. Geo-separation constraint**

- At least 2 critical objectives occur in separate locations requiring travel time or simultaneous teams.
- Travel cannot be bypassed by a single instant teleport mechanic in VS0.

**Pass/Fail:** If one zerg can rotate and still meet the concurrency gate, fail.

**B4. Anti-grief stability**

Griefing cannot permanently lock the event.

If players sabotage objectives, the system must:

- degrade gracefully
- still allow recovery with extra work
- log sabotage actions for admin review (VS0: simple log lines)

**Pass/Fail:** Event can be completed even with a hostile subgroup, given enough defenders.

### C) Roles and “Everyone Has a Job”

**C1. Minimum viable role diversity**

Event must include at least 4 distinct contribution roles:

- Strike (combat DPS/tank/heal)
- Builder (craft/transport/repair anchors)
- Ritualist (non-combat interaction/puzzle channel)
- Scout/Runner (intel + map tasks + delivery)

**Pass/Fail:** Each role must have a progression bar it can meaningfully advance.

**C2. Non-combat matters**

Builder + Ritualist contributions must be strictly required to reach Final Window.

**Pass/Fail:** If combat-only groups can complete all objectives, fail.

### D) Combat Design Requirements

**D1. Boss is not a “single health bar”**

The threat has:

- an invulnerable state
- at least one “break armor / expose weak point” mechanic
- add waves that threaten objectives (anchors/ritualists)

**Pass/Fail:** If it’s just “hit it until it dies,” fail.

**D2. Split War exists in VS0**

- At least two sub-bosses/heads with different mechanics spawn in different locations.
- Ignoring one makes final window harder (e.g., shorter vulnerability window, higher add pressure).

**Pass/Fail:** Players must make a strategic choice and feel consequences.

### E) Persistence and Consequences

**E1. Outcome changes the world**

On victory:

- unlock a visible world change (e.g., ley bridge, safe corridor, monument, vendor unlocked)

On failure:

- apply a scar to one region/town (e.g., debuff zone, closed services, hostile spawns)

**Pass/Fail:** State must persist across server restart.

**E2. Event cooldown**

- Event cannot be immediately retriggered.
- Cooldown is server-configurable (VS0: config constant).

**Pass/Fail:** If admins restart the server and it retriggers instantly, fail unless config says so.

### F) Scoring, Rewards, and Fairness

**F1. Multi-axis merit**

Merit earned from at least 5 sources:

- boss damage
- healing/defending anchor
- crafting/delivery
- ritual progress actions
- scouting/discovery/delivery completions

**Pass/Fail:** UI shows personal merit + category breakdown.

**F2. Rewards are tiered, not rank-1**

- Rewards based on merit brackets (e.g., Bronze/Silver/Gold) rather than “top 10 only”.
- At least one cosmetic reward + one functional reward exist in VS0.

**Pass/Fail:** A casual Builder can earn a meaningful reward without touching DPS.

### G) Observability + Admin Controls

**G1. Server observability**

Server outputs periodic status line:

- current phase
- time to next escalation
- global meter
- objective states (anchors alive count, ritual progress %, head status)

**Pass/Fail:** Debugging does not require attaching a debugger.

**G2. Config knobs (VS0 minimum)**

- `event_enabled`
- `phase_durations`
- `meter_decay_rate`
- `required_concurrent_objectives`
- `pvp_during_event` (on/off)

**Pass/Fail:** Changing knobs changes behavior without code edits.

## Definition of Done

### 1) Playtest Proofs (must be reproducible)

**DoD-1: Solo + small-group failure proof**

Run with 1 party (4–6 players) + rich gear:

- they can participate
- they cannot reach Final Window alone

Evidence: recorded log + in-game outcome.

**DoD-2: Rich guild brute-force proof**

Run with 1 large coordinated guild (e.g., 20–40):

- if they stack one objective, diminishing returns kicks in
- geo-separation forces split teams
- concurrency gate blocks sequential completion

Evidence: server logs show gate enforcement.

**DoD-3: Mixed-server success proof**

Run with mixed random players + one guild:

- the event can be completed
- at least 3 different roles contribute materially

Evidence: merit breakdown shows non-combat roles in top brackets.

### 2) Technical Completion

**DoD-4: Server-authoritative state**

- All phase transitions + objective completion are computed server-side.
- Clients cannot spoof progress via packets. (VS0: sanity checks + server-owned timers)

**DoD-5: Persistence**

Event outcome persists across restart:

- phase resets safely
- scar/victory changes remain
- cooldown remains

**DoD-6: Performance sanity**

- Event running with max expected players does not degrade below target tickrate beyond agreed threshold.
- Objective interactions do not allocate unbounded memory.

### 3) Content Completion (VS0 minimum)

**DoD-7: One complete threat**

One threat fully playable end-to-end with:

- at least 2 regions involved
- at least 2 sub-bosses/heads
- 3 objective types
- win + fail outcome

**DoD-8: UX**

Players can answer within 30 seconds:

- “What phase are we in?”
- “What do I do to help?”
- “Where do I go?”

Evidence: UI elements present + tested by a fresh player.

### 4) Anti-cheese / Anti-degenerate strategies checklist

**DoD-9: Cheese list validated**

The following must be tested and blocked or absorbed:

- stacking 40 players on one objective
- kiting boss out of bounds
- ignoring all heads and still winning
- repeatedly farming merit via one low-risk action
- griefers trying to hard-lock progress

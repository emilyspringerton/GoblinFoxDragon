# Stadium Mode vS0 Test Plan (Checklist)

## Scope
This checklist validates Stadium Mode v0 acceptance criteria: progression, abilities, UI, targeting, match loop, bots/networking, performance, safety rails, and telemetry.

---

## 0) Definition of Done

- [ ] Launch SHANKPIT → choose **STADIUM** from main menu.
- [ ] Enter a match (solo vs bots acceptable for v0).
- [ ] Level from 1 → 10 in a single match, allocating points into **exactly 5 abilities**.
- [ ] Play the full match in **3rd-person**, with in-match toggle to **1st-person**.
- [ ] Finish match → results screen → return to lobby without crash/leak.

---

## 1) Mode Rules (Hard Constraints)

### 1.1 Level cap and progression

- [ ] Max level is 10; no XP or levels beyond 10 in Stadium.
- [ ] Player starts at **Level 1** with **1 skill point** available at match start.
- [ ] Player gains **1 skill point per level-up** (total 10 skill points by level 10).
- [ ] XP pacing supports “Dota-speed” progression:
  - [ ] Typical match reaches **Level 6** for most players before match ends.
  - [ ] Typical match reaches **Level 10** for at least one player before match ends.
- [ ] XP sources exist for:
  - [ ] Enemy kills / assists.
  - [ ] Objective participation (capture/hold/etc.).
  - [ ] Minion/creep equivalents **OR** arena events (one consistent choice).

### 1.2 Ability budget (5-slot)

- [ ] Each Stadium class exposes exactly 5 slots: **Passive**, **Q**, **W**, **E**, **R**.
- [ ] No extra combat buttons required for v0 (besides basic attack, jump, interact).

### 1.3 Ability ranking limits

- [ ] Q/W/E support ranks **1–3**.
- [ ] R supports ranks **1–2**.
- [ ] R unlocks at **Level 6** (cannot rank before level 6).
- [ ] R rank 2 unlocks at **Level 10**.
- [ ] A level 10 build can allocate valid points (example: Q3/W3/E2/R2).

### 1.4 No scaling beyond level 10

- [ ] No Stadium mechanics reference or require levels >10.
- [ ] No permanent talent trees or external progression dependencies.

---

## 2) Class & Kit Integration

### 2.1 Stadium roster

- [ ] Stadium supports a match with classes:
  - [ ] **Lion** (Paladin)
  - [ ] **Witch**
  - [ ] **Hunter**
  - [ ] **Cheetah**
  - [ ] **Zebrashark**
  - [ ] **Troubadour**

### 2.2 Kit conformance

For each class in Stadium:

- [ ] Exactly 5 slots exposed: Passive, Q, W, E, R.
- [ ] Each ability exposes:
  - [ ] Icon
  - [ ] Name
  - [ ] Cooldown
  - [ ] Cast/activation type (instant/cast/channel)
  - [ ] Rank scaling for ranks that exist (1–3 or 1–2)
- [ ] Abilities are functional (not placeholder):
  - [ ] Ability triggers within 1 frame of eligibility.
  - [ ] Cooldowns apply correctly.
  - [ ] Ranks modify at least one numeric property (duration/damage/heal/cd/etc.).

### 2.3 French ability names

- [ ] Ability names displayed in Stadium UI use **French names** (at minimum in ability bar + tooltips).

---

## 3) UI / UX Acceptance Criteria (1P + 3P)

### 3.1 Perspective support

- [ ] Stadium gameplay supports **third-person** by default.
- [ ] Stadium supports **first-person toggle** in-match.
- [ ] Toggling perspective does **not**:
  - [ ] Change gameplay stats.
  - [ ] Reset cooldowns.
  - [ ] Break target selection.
  - [ ] Cause severe camera clipping (no inside-head artifacts).

### 3.2 Ability bar

- [ ] HUD shows a **5-slot ability bar** at bottom center:
  - [ ] Passive icon (non-interactive)
  - [ ] Q/W/E/R icons (interactive)
- [ ] Each ability icon shows:
  - [ ] Cooldown radial/overlay when unavailable.
  - [ ] Rank pips (Q/W/E: up to 3; R: up to 2).
  - [ ] “Level up available” indicator when a point can be spent.

### 3.3 Level-up and ranking flow

- [ ] On skill point gain:
  - [ ] Non-blocking notification displays.
  - [ ] Ability bar shows upgrade indicators.
- [ ] Player can spend points:
  - [ ] Via mouse/keyboard (click or hotkey).
  - [ ] Without pausing gameplay (no modal required).
- [ ] Tooltips show delta from next rank (e.g., “Rank 2: +X damage, -Y cooldown”).

### 3.4 Targeting for allies/enemies

- [ ] Soft-target selection works in 1P and 3P:
  - [ ] Ally targeting for heals/buffs.
  - [ ] Enemy targeting for damage/debuffs.
- [ ] Minimum viable targeting:
  - [ ] Nearest target in reticle cone selected.
  - [ ] F1–F5 (or equivalent) selects party members.
- [ ] Visual feedback:
  - [ ] Current target highlight visible and distinct in both perspectives.

### 3.5 Stadium scoreboard

- [ ] Scoreboard (Tab) shows:
  - [ ] Player level (1–10)
  - [ ] K/D/A
  - [ ] Objective contribution
  - [ ] Q/W/E/R ranks (compact display)

---

## 4) Match Loop & State Management

### 4.1 Mode entry/exit

- [ ] Stadium can be started from lobby/menu without console commands.
- [ ] Match end returns cleanly to lobby without:
  - [ ] Memory leak symptoms.
  - [ ] Stuck audio.
  - [ ] Stuck input capture.
  - [ ] Stuck network sockets (even if offline).

### 4.2 Spawn & respawn

- [ ] Player spawns at defined team spawn.
- [ ] Respawn exists (timer acceptable).
- [ ] After respawn:
  - [ ] Level and ability ranks persist.
  - [ ] Cooldowns follow defined rules consistently (reset or persist).

### 4.3 Win condition

- [ ] Stadium has a clear win condition:
  - [ ] Objective capture threshold **OR**
  - [ ] Ticket depletion **OR**
  - [ ] Boss/structure destruction.
- [ ] Win condition is visible in HUD.

---

## 5) Networking & Bots (v0 acceptable scope)

### 5.1 v0 minimum requirement

- [ ] **Offline**: single-player vs bots works end-to-end **OR**
- [ ] **Online**: multiplayer works end-to-end (even without bots).

### 5.2 Determinism / authority (if online)

- [ ] Server authority over:
  - [ ] XP/level
  - [ ] Ability ranks
  - [ ] Damage/heal outcomes
- [ ] Client prediction (if present) does not allow:
  - [ ] Ranking abilities without points.
  - [ ] Casting abilities while on cooldown.

---

## 6) Performance & Responsiveness

### 6.1 Frame-time and input latency

- [ ] Ability presses trigger within 1 frame after eligibility (client-side).
- [ ] No hitching when:
  - [ ] Leveling up.
  - [ ] Opening scoreboard.
  - [ ] Toggling 1P/3P.

### 6.2 Visual clarity

- [ ] In 1P: UI reduces clutter (no full ground trails everywhere).
- [ ] In 3P: AoE previews are visible and accurate.
- [ ] Zebrashark “blood in the water” indicators are:
  - [ ] Range-limited.
  - [ ] Not full wallhack silhouettes across the map.

---

## 7) Safety Rails / Exploit Prevention

- [ ] Player cannot exceed:
  - [ ] Level 10.
  - [ ] Q/W/E rank 3.
  - [ ] R rank 2.
- [ ] Player cannot allocate points:
  - [ ] Without available skill points.
  - [ ] Into R before level 6.
- [ ] Leaving and rejoining does not duplicate skill points.

---

## 8) Telemetry / Debug (dev-only acceptable)

- [ ] Stadium logs print (dev build):
  - [ ] XP gain events.
  - [ ] Level-ups.
  - [ ] Ability rank allocations.
- [ ] Debug overlay toggle exists but is off by default (if present).

---

## 9) Edge Cases & Exploratory Scenarios

- [ ] Level up while dead; confirm point awarded and usable after respawn.
- [ ] Spend a skill point while scoreboard open.
- [ ] Toggle 1P/3P while channeling an ability.
- [ ] Attempt to rank R before level 6 (blocked).
- [ ] Attempt to rank R to 2 before level 10 (blocked).
- [ ] Attempt to allocate points with 0 available points (blocked).
- [ ] Rejoin match and verify points/levels do not duplicate.
- [ ] Perform multiple camera toggles under heavy combat load; no hitching or desync.
- [ ] Verify soft-target lock persists across camera toggle.
- [ ] Toggle camera while targeting allies; verify highlight remains correct.
- [ ] Win condition met while player is dead; match still ends cleanly.


## 10) Creep Combat / XP Contract (Stadium + City)

This section codifies existing runtime behavior so Stadium and City share one combat loop.

- Creep data contract: `type`, `hp/max_hp`, `alive`, `scene_id`, `team_id`, `spawn_zone`, `respawn_time_ms`.
- Damage entrypoint is scene-routed NPC damage (`npc_apply_damage(scene_id, hit_pos, damage, attacker_client_id, now_ms)`), which fans into the scene NPC pool.
- On death (`hp <= 0`):
  - NPC is deactivated.
  - `respawn_time_ms` is set from spawn-zone delay.
  - killer/last-hit ownership is resolved.
  - XP is awarded via `award_creep_xp`.

### XP distribution

- XP tuning is by creep type (`creep_reward_for_type`):
  - `base_xp`
  - `xp_radius`
  - `team_share_pct`
- Award rule:
  - killer gets `base_xp`
  - same-team players in same scene and inside `xp_radius` get `floor(base_xp * team_share_pct)`.

### UI feedback (minimum)

- HUD progression bar updates immediately for killer and recipients.
- Optional feed/floating-text hooks can layer on top of the same XP award events.

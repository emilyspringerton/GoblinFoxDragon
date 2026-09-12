# Job & Spell System Overhaul — NORTHSTAR

**Status:** Planning + Phase 0 (advanced jobs disabled) shipped 2026-09-12. Phases 1-5 scoped,
not started. Founder real-time direction, routed through `emily observe` (Apple #19178) per
Principle 1a: "we make everything a spell... we need to disable all of the advanced jobs for
now including ASN... design the first tier of spells... start tracking skill levels for each of
the weapon types... /ja /ma both map to the same spell casting affordance... plan this next
phase of work."

## 0. Real, checked-not-assumed current-state audit

The founder's framing was "the classes dont have any purpose there are no spells or job
abilities." Checked directly against `apps2/mud/main.go` and `server/job/subjob.go` before
planning anything — that's not quite the real gap, and the plan below is scoped against what's
actually true, not the stated premise:

**Already real and working today:**
- `cmdCast` (main.go) already dispatches: `invisible`/`sneak`/`cure`/`cure2`/`protect`/`shell`/
  `haste`/`regen`/`refresh`/`dia`, each with a real MP cost and a real job/sub-job gate.
- `cmdCastBlackMagic` + `blmSpells` already has a FULL elemental tier-1/2/3 roster (fire, blizzard,
  thunder, stone, water, aero, each ×3) plus `poison`/`bio`/`distract`/`frazzle`, all with real MP
  cost + base damage + INT scaling, auto-targeting the player's existing `p.combat.TargetMobID`
  (no separate target argument needed — this part of the founder's own ask is already true for
  BLM/RDM offensive magic specifically).
- `job.WarriorAbilities()` already has **Provoke** (30s recast, Lv5). `job.MonkAbilities()`
  already has **Boost** and **Chakra** (both Lv1). `job.ThiefAbilities()` already has
  **Sneak Attack** (1m recast, Lv1) and Trick Attack. `job.RedMageAbilities()` has Convert.
  `job.WhiteMageAbilities()` has Benediction/Holy Water. A real `RecastTracker` already enforces
  each ability's own cooldown independently.
- `cmdJA`/`ja <ability-id>` and `cast <spell> [target]` already exist as separate commands.

**The real, checked gap — this is what actually makes jobs feel purposeless:**
- **Zero cast time, zero universal lockout, anywhere.** Grepped the whole file: no
  `castTime`/`CastTime` concept exists at all. Every spell in `cmdCast` and `cmdCastBlackMagic`
  resolves the instant the command is typed, with nothing but MP gating it — no tactical rhythm,
  no interruption risk, no reason a "caster" job feels different from mashing a button. This is
  the real, structural reason classes feel purposeless, not a lack of spell content.
- **No per-weapon-type skill.** `p.wsSkill` is a freely-settable *name* (`setws <any name>`),
  gated by nothing but TP — no `swordSkill`/`axeSkill`/`daggerSkill` levels exist anywhere, so
  "unlocks at a specific weapon skill level" has nothing to hang off yet.
- **`/ja` and `cast` are two separate code paths**, not two names for one casting affordance —
  job abilities never go through MP/cast-time/recast in a shared way with magic (they use
  `RecastTracker` only; magic uses no recast tracker at all today).
- **RAISE and Polymorph don't exist.** Genuinely missing, not hidden elsewhere.
- Advanced jobs (everything past the FFXI original 6 — PLD/DRK/BST/BRD/RNG/SAM/NIN/DRG/SMN/BLU/
  COR/PUP/DNC/SCH/GEO/RUN/ASN) are fully playable today, no gate at all.

## 0.5. Real, itemized finding: which existing abilities actually DO something

Founder asked directly ("ok but do the abilities actually do anything?") — checked `cmdJA`'s own
switch statement line by line rather than trusting the ability existing at all:

**Mechanically real** (state actually changes): Provoke (real enmity-table update, can flip mob
aggro), Chakra (real HP heal), Clear Mind (real MP restore), Convert (real HP→MP trade),
Benediction (real full restore), Venom/Siphon (ASN — real AoE poison / real MP-regen buff).

**Pure flavor text — the sent message claims an effect, but no code applies one**: Berserk
("+30% Haste for 3 min" — nothing sets a haste buff), Boost ("next attack lands harder" —
nothing flags the next attack), Elemental Seal ("next spell guaranteed to land" — no accuracy
system exists at all yet, so nothing to guarantee), Chainspell ("spells cost no MP" — no
MP-waiver state), Sneak Attack/Trick Attack ("critical blow"/"transfers enmity" — no crit system
exists, and Trick Attack never touches the enmity table).

This is the same root cause as §0's cast-time gap, more precisely located: everything in this
second list is a "pop a buff, consumed by your next action" pattern with no `pending effect`
state on the player to actually consume — not a content gap, a real missing mechanism. Phase 1
(the universal `Ability` model below) is the natural place to add a real
`p.pendingNextAttack`/`p.pendingNextSpell`-shaped flag, checked and cleared by the next matching
attack/cast/JA rather than each of these six growing its own one-off bolt-on.

## 1. Universal Ability model (the real core of this phase)

One data shape for everything a player can activate — a job ability (today's `/ja`) and a spell
(today's `cast`) become the *same* underlying thing, differing only in field values:

```go
type Ability struct {
    ID        string
    Job       job.JobID
    MPCost    int           // 0 for every current JA (provoke/boost/chakra/sneak attack all cost 0)
    CastTime  time.Duration // 0 = instant
    Recast    time.Duration // per-ability cooldown, independent of the universal lockout below
    Kind      AbilityKind   // Offensive | Defensive | Buff | SelfOnly
}
```

**Timing model, matching the founder's own four real cases exactly:**
1. **0 cast time, 0 recast** — a truly instant, spammable ability (rare; most Lv1 JAs still get
   *some* recast so they stay abilities, not auto-attacks).
2. **0 cast time, real recast** — today's JAs already work this way (Provoke, Boost). No change
   needed to this case's own mechanics, just moving it under the same `Ability` struct.
3. **Real cast time** — a spell that takes N real seconds before it resolves. Implemented as a
   scheduled completion (`time.AfterFunc`, independent of the 1Hz world tick — the tick loop
   handles world simulation, not command timing, so sub-tick precision is not blocked by it),
   during which the caster cannot act (interruptible by real damage taken, matching FFXI's own
   convention — real, deferred design question, named below).
4. **Instant cast + universal lockout** — resolves immediately, but a short universal window
   (order of the ~20ms named in the ask, though see the honest caveat right below) during which
   *no other ability* can be used, distinct from that spell's own per-spell recast.

**Real, honest technical caveat, not silently smoothed over:** a ~20ms universal lockout is real
and implementable (Go timers resolve far finer than that; nothing about the 1Hz world-tick loop
prevents it), but at MUD-over-telnet/SSH network latency (tens to hundexcls of ms round-trip is
normal even on a good connection), a lockout that short is **not something a human player will
ever perceive or be blocked by in practice** — it would only matter against a script/bot firing
commands back-to-back over an already-open connection. Worth deciding explicitly: keep it at
~20ms as a bot-throttle (cheap, harmless, real), or set it to something a human would actually
feel (FFXI's own real universal-lockout is ~1s) if the goal is a felt "weight" to casting instant
abilities. Not resolved here — a real founder call, named rather than guessed at.

## 2. Weapon skill per-type leveling

New `p.weaponSkills map[string]int` (sword/axe/dagger/great-sword/etc. → level), parallel in
shape to the already-shipped per-job `p.jobXP` (GFD-124433) — same real precedent, not a new
pattern. Gains on real weapon hits (matching how `p.miningSkill`/`p.craftSkill` already gain on
use). `skillchain.CanonicalWeaponSkills` (already real) gets a `MinSkillLevel` field per weapon
skill; `setws <name>` checks the caster's own level in that weapon skill's weapon type before
allowing it, replacing today's "any name, no gate" behavior.

## 3. `/ja` + `/ma` unification and targeting rules

Both become aliases dispatching into one `cmdAbility(p *player, id string, targetName string)`,
built on the `Ability` struct above. Targeting, matching the founder's own explicit rules:
- **Offensive** (nukes, DoTs): if `p.combat.TargetMobID` is set (already true for BLM/RDM magic
  today), use it — no target argument required. This is the existing `cmdCastBlackMagic`
  behavior, generalized to job abilities with an offensive `Kind` too (none exist yet, but the
  model supports one, e.g. a future WAR/MNK offensive JA).
- **Defensive/heal** (Cure, party buffs): always requires an explicit target argument — never
  silently defaults to the caster's own combat target, since healing the wrong ally is a real,
  meaningful mistake FFXI's own UI convention already protects against the same way.
- **Provoke specifically**: typing a target name provokes *that* mob (even mid-combat with a
  different one already engaged) without changing `p.combat.TargetMobID` — provoke redirects
  aggro, it does not re-target the caster's own attack.

## 4. First-tier roster (the six original jobs only, per Phase 0 below)

| Job | Ability/Spell | Kind | Real status |
|---|---|---|---|
| WAR | Provoke | Offensive (aggro) | **Already real** (30s recast, Lv5) |
| MNK | Boost | Buff | **Already real** (Lv1) |
| MNK | Chakra | **Offensive (ranged)** | **Redefine, not new content** — currently coded as a real-FFXI-accurate self-heal (no MP, restores missing HP); founder direction (2026-09-12): in GFD, Chakra is a MNK ranged attack instead. A deliberate house rule diverging from FFXI canon on purpose, same precedent as `blmSpells`' own distract/frazzle stubs — needs a real damage/range design (not specified yet: base damage? scales off STR like melee, or its own stat? real projectile range vs. the existing `p.combat.MeleeRange`?), not just a rename of the existing self-heal code |
| THF | Sneak Attack | Offensive | **Already real** (1m recast, Lv1) |
| THF | SAP (Sleep-inducing ranged attack, "CC") | Offensive/CC | **New** — real FFXI THF ranged JA is Aura Steal/Flee, not SAP; founder named "SAP" specifically (real FFXI term is a RDM/generalist CC noun, not a canonical THF ability) — flagging for a naming confirmation, not guessed at silently |
| WHM | Cure, Haste, Dia | Defensive/Buff/Offensive-DoT | **Already real** as instant `cast` spells — need cast time added |
| WHM | Raise (restores 50% HP, cures KO) | Defensive | **New**, does not exist |
| RDM | Refresh, Dia | Buff/Offensive-DoT | **Already real** as instant `cast` spells |
| RDM | Poison | Offensive-DoT | **Already real** (shares Poison's numbers, per the existing `blmSpells` entry and its own founder-directed "copy-paste of poison" precedent) |
| RDM | Polymorph | ? | **New, real design gap** — FFXI has no player-castable "Polymorph"; needs a real effect definition (cosmetic? CC? transform stats?), not guessed at here |
| BLM | Water, Fire, Poison, Bio | Offensive-DoT/nuke | **Already real**, full tiered roster exists in `blmSpells` |

## 5. Combat damage formula overhaul (founder addition, same session, same "classes have no real
depth" root cause)

Real, checked-not-assumed finding: `apps2/mud/main.go`'s `playerDamage = 30` is a flat constant
passed straight into `server/mob.Registry.Hit` as the melee damage dealt, every swing, every
class, zero variance. The reverse direction is identical: each mob's own `MeleeDamage` (e.g. 40,
50, 60, 80 — set per mob type in `apps2/mud/main.go`) is applied via a plain `p.hp -= ev.Damage`
in `handle()`'s own event loop (line ~2225) — no roll, no stat read, on either side, in either
direction. This is real and confirmed, not assumed from the founder's own report alone.

**Proposed formulas** (a real, concrete starting point to tune against, not a placeholder):

- **Accuracy** (does the swing land at all): `hitChance = 90% + (attacker.DEX - defender.AGI) *
  0.5%`, clamped to `[50%, 99%]` so neither side is ever a guaranteed hit or guaranteed miss.
  Applies to a mob's own effective DEX too (mobs get a real DEX-equivalent stat, or a flat
  per-mob accuracy baseline if adding full stat blocks to every mob type is too large a change
  for this phase — real, named decision point).
- **Critical hit** (DEX-based, melee only per the founder's own wording): `critChance = 5% +
  attacker.DEX * 0.1%`, capped at 50%. A crit deals `1.5×` the rolled damage.
  **Player** BaseDamage becomes STR-scaled with real variance instead of a flat 30:
  `mean = BaseDamage * (STR / 10)`, then rolled uniformly in `[mean * 0.8, mean * 1.2]` (±20%)
  before the crit multiplier.
- **Damage reduction** (VIT-based, defender side, both directions): `reduced = max(1, incoming -
  defender.VIT / 2)` — VIT always shaves a real, meaningful amount off incoming damage, but never
  reduces a landed hit to 0 (a hit that connects always does *something*).
- **Evasion** (AGI-based) is the same accuracy roll above from the other direction — AGI is not a
  separate "dodge" roll layered on top of the attacker's own accuracy check, it's the term that
  already reduces the attacker's effective hit chance. One roll, not two, per swing.

**RNG mechanism (founder direction, same session): the hit/miss and crit/normal rolls must use
this monorepo's own standing "weighted marble-bag + Fibonacci pity" pull algorithm, not
independent per-swing rolls.** Real, shipped this same pass (not just specified): new
`server/rng` package (`Fibonacci`, `MarbleBagPick`), a faithful Go port of the first real
implementation of this pattern, ECOWAR/REDGARDEN's `arena_fibonacci`/`arena_marble_bag_pick`
(`packages/simulation/arena_game.c`, S202-09/S202-42 — per-hero Cart-delivery outcome
selection), which the founder's own quoted design note names as the shared-utility precedent
this should reuse rather than reinvent. A second, independently-built precedent for the same
underlying idea exists too (`emily.cli/cmd/promptoverse_pity.go`, Prompt-o-verse's style/subject
discovery pity) — that one escalates a binary trigger probability ahead of a separate weighted
draw, rather than folding pity directly into one draw's own weights; the founder's own note names
the ECOWAR shape specifically for combat, which is what got ported. 7 real tests, including one
that caught and fixed a real off-by-one in ECOWAR's own header comment (`fib(10)=55` claimed;
tracing the real C loop shows the actual value is 89 — `fib(9)=55`) — the Go port matches the
real executable algorithm, not the comment, and the mismatch is named rather than silently
carried forward or silently "corrected" in a way that would diverge from the C original's real
behavior.

**How this plugs into the formulas above** (not yet wired into `apps2/mud`'s real combat code —
`server/rng` itself is real and tested; this is the design for Phase 4.5's own implementation):
- **Accuracy**: a 2-outcome bag `[Hit, Miss]` with base weights from the `hitChance` formula
  above (e.g. `[hitChance, 100-hitChance]`), pity tracked **per player, per opponent-class**
  (`p.combatPity.accuracy [2]int` is enough for a single ongoing fight — reset when combat ends/
  target changes, matching the existing `p.combat.TargetMobID` lifecycle) on **Hit** specifically:
  a miss streak makes the next swing progressively more likely to land, the real anti-frustration
  property this mechanism exists for. Symmetric for a mob attacking a player.
- **Critical hit**: a 2-outcome bag `[Normal, Critical]`, base weights from `critChance` above,
  pity tracked on **Critical** — a real crit drought becomes progressively more likely to break,
  matching the founder's own "legendary pull" framing exactly.
- **Damage magnitude** (the ±20% variance band) stays a plain uniform roll, not a marble-bag
  pick — marble-bag+pity is for a small, fixed set of discrete outcomes (which of N tiers/
  results), not a continuous quantity; forcing damage magnitude itself through discrete "buckets"
  just to reuse the same primitive would be a real, honest mismatch of the mechanism, not asked
  for by the founder's own wording either (RNG-with-pity was tied to *whether* a hit/crit
  happens, not the exact number rolled once it does).

This touches `server/mob.Registry.Hit`/`TickPlayer` (player→mob) and the `EvtMobAttack` handling
in `apps2/mud/main.go`'s event loop (mob→player) — real, core, shared combat code, not
job-specific, so this is honestly a bigger, more foundational change than the spell-timing work
in §1-5 above, even though the founder raised it second. Real open question: do mobs get full
STR/DEX/VIT/AGI stat blocks (most FFXI-parity, most work — every mob spawn site in
`apps2/mud/main.go` would need real numbers) or a simpler flat per-mob "accuracy"/"evasion"/
"armor" trio that approximates the same feel without a stat block on every mob? Not decided here.

## 6. Phased plan

- [x] **Phase 0 — disable advanced jobs (shipped 2026-09-12).** `cmdSetJob` refuses every job
  outside the original six (WAR/MNK/WHM/BLM/RDM/THF) with a clear message naming this as
  temporary. `jobs` command marks disabled jobs. Existing characters already on a disabled job
  are left alone (not force-switched) — a real, deliberate choice to avoid destructive
  mid-session side effects; see the commit's own doc comment for the honest reasoning.
- [ ] **Phase 1 — universal `Ability` model + cast-time/lockout engine.** The real core: MP/cast
  time/recast/lockout as one shared mechanism, `/ja` and `cast` unified into `cmdAbility`. No new
  spell content yet — this phase proves the timing engine against the EXISTING roster (Provoke,
  Boost, Chakra, Cure, the BLM elementals) before adding anything new.
- [ ] **Phase 2 — weapon skill per-type leveling.** `p.weaponSkills`, skill-gain-on-hit,
  `MinSkillLevel` gate on `setws`.
- [ ] **Phase 3 — new first-tier content.** RAISE, Polymorph (pending the real design questions
  named in §4), SAP (pending the naming/effect confirmation named in §4).
- [ ] **Phase 4 — targeting rules.** Offensive-auto-target/defensive-explicit-target,
  provoke-redirects-without-retargeting, all live-verified against real combat.
- [ ] **Phase 4.5 — combat damage formula overhaul (§5).** STR-scaled+random player damage,
  DEX-based crit, the shared accuracy/evasion roll, VIT-based reduction — real, foundational,
  and arguably higher-impact than the spell-timing phases above since it touches every single
  attack in the game, not just casters. Sequenced here (not first) only because it's independent
  of the `Ability` model work and can land whenever ready without blocking or being blocked by
  it — real candidate to actually do FIRST if the founder wants the most broadly-felt improvement
  soonest.
- [ ] **Phase 5 — playtest, re-enable advanced jobs one at a time** once each has been ported
  onto the new `Ability` model — not a one-shot re-enable, matching the founder's own "so we can
  get feedback on actual gameplay" framing (the point is to ship the SIX-job core solid first).

## Real, open questions for the founder (named, not guessed at)

1. Universal lockout duration: ~20ms bot-throttle vs. a human-felt ~1s (see §1's own caveat).
2. THF "SAP (CC)" — exact effect/duration intended (real FFXI term, or a house-invented one)?
3. RDM Polymorph — what does it actually do?
4. Cast-time interruption on taking damage: FFXI-accurate (yes) or simplified (no interrupts for
   now, revisit later)?
5. Combat formula (§5): do mobs get full STR/DEX/VIT/AGI stat blocks (most accurate, most
   per-mob-spawn-site work) or a simpler flat accuracy/evasion/armor trio per mob?
6. Which phase to actually build next — Phase 1 (spell timing) or Phase 4.5 (combat formula)?
   Both are real, independent, and shippable in either order.
7. Chakra redefinition (§4, founder direction 2026-09-12: "chakra is a MNK ranged attack") — real
   design needed: base damage, what stat scales it (STR like melee, or a new ranged-attack
   stat?), real range vs. `p.combat.MeleeRange`, and whether the CURRENT self-heal behavior is
   dropped entirely or kept as a different ability (MNK's real FFXI kit does have both a self-heal
   AND ranged options at higher level — Boost/Chakra/Hundred Fists/etc. — so this may be "add a
   new ranged JA" rather than "replace Chakra's own healing").

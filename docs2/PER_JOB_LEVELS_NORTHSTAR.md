# Per-Job Levels — real scoping pass

**GFD-XX-1249** ("every class/job needs to have separate levels right now im level 10 and
switching to RDM makes me a lvl 10 RDM i think? thats not right every class starts at lvl 1 and
can level up to 75"). Real investigation, checked directly against the actual code, 2026-09-04.

## 1. Real, decisive finding: this is not a bug, it's a deliberate design the founder now wants changed

`apps2/mud`'s player struct carries exactly ONE `*xp.CharXP` (`p.charXP`) — one shared
`Level`/`CurrentXP` for the whole character, completely independent of `p.jobID` (the currently
active job). `cmdSetJob` changes `p.jobID` alone; it never touches `p.charXP`. Every stat/gate
call site reads the same shared level: `job.HPAtLevel(p.jobID, p.charXP.Level)`,
`job.MPAtLevel(...)`, `p.equip.CanEquip(..., p.charXP.Level)`, XP chains, party XP splits, merit
points — all of it.

This is **real FFXI parity, working exactly as designed** (this whole package family is
explicitly commented "FFXI-parity" throughout `server/job`/`server/xp`) — real FFXI has one
shared character level; switching jobs changes your stat curve and spell/ability access at that
SAME level, not a fresh level 1. The founder's own ask is a real, deliberate DEVIATION from that
design toward a different, real MMO convention (FFXIV's own per-job leveling, or WoW-classic's
per-character-per-class leveling) — not a defect in the current system, a genuine, decisive
direction change.

One real, existing, partial mechanism already in this direction, found directly: `job.CharJob`
(`Main`/`Sub`/`MainLvl`/`SubLvl`) already carries separate level fields for main/sub job pairing
— but `MainLvl`/`SubLvl` are set once at `NewCharJob` construction and are not the values any real
HP/MP/equip/XP call site actually reads today (those all still read the single `p.charXP.Level`).
`EffectiveSubLevel()` (SubLvl/2, real FFXI sub-job rule) is the one place `CharJob`'s own level
fields are genuinely consumed.

## 2. Real, honest scope of what a true fix needs

- **In-memory**: replace the single `p.charXP` with a real per-job map (e.g.
  `map[job.JobID]*xp.CharXP`), lazily creating a fresh level-1 entry the first time a job is
  played, and route every one of the real call sites above through "the currently active job's
  own entry" instead of the one shared field. Real, not huge on its own — a real, mechanical
  refactor across roughly a dozen call sites in `apps2/mud/main.go`.
- **Persistence — the real, larger piece**: IDUNA's own character schema
  (`server/idunaclient.go`'s `Level int`, `json:"level"`, backing `PATCH /api/v1/characters/:id/level`)
  is a single flat column, matching the current single-shared-level design exactly. Real per-job
  levels need a real, new, multi-value shape (a JSON map column, or a new `character_job_levels`
  table keyed by job) — a real IDUNA schema migration, not just an `apps2/mud`-side change. This
  is the same real class of cross-repo persistence work `INVENTORY_PERSISTENCE_NORTHSTAR.md`
  already scoped for the inventory-durability gap, and not attempted blind here for the same
  reason: guessing the wrong shape now risks a real, wasted migration later.
- **Real, open question, not resolved here**: does the real FFXI sub-job mechanic
  (`EffectiveSubLevel`, sub always caps at half main) still apply once main/sub jobs each level
  independently, or does per-job leveling replace the main/sub pairing model entirely with N
  independent job slots (closer to FFXIV)? The founder's own wording ("every class... can level
  up to 75") reads as the latter, but that's a real, decisive design call, not assumed here.

## 3. Real, honest status

**Diagnosed, not built.** Real root cause named (a deliberate FFXI-parity design, not a bug), a
real in-memory refactor scoped (buildable in one pass), and the real, larger persistence
dependency named honestly (needs an IDUNA schema decision first, same class of work as the
inventory-durability gap). Per this session's own established precedent for a genuinely
consequential, wide-blast-radius design change (HP/MP/equip-gating/XP-chains all key off the
single field being changed), this is scoped rather than rushed — a partial in-memory-only fix
that doesn't survive a reconnect/restart would be a real, confusing half-measure, not a real fix.

## Related

- `docs2/INVENTORY_PERSISTENCE_NORTHSTAR.md` — the same real class of "needs a real IDUNA schema
  change before this can be durable" gap, scoped the same way.
- `docs2/STACK_CONTINUITY_REPORT.md` — real current gaps section, updated to point at this.

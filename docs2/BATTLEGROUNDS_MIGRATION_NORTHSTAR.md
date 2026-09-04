# Battlegrounds ↔ REDGARDEN Migration — real root-cause finding + scoping

**GFD-BG-12444** ("GFD battlegrounds is down never works... we are migrating redgarden changes
to gfd battlegrounds maybe something broke"). Real investigation, not guessed — checked directly
against the live processes and both repos' actual source, 2026-09-04.

## 1. Real, decisive finding: GFD Battlegrounds has no native server, and never has

`apps2/battlegrounds_gui`'s own default queue port (7778, `main.c`'s own comment: "apps/
matchmaker's documented arena listen-port") is a leftover reference to **REDGARDEN's own**
`apps/matchmaker` directory — checked directly: `GoblinFoxDragon` has no `matchmaker` anywhere in
its own tree, and no compiled arena-server binary of its own. Every live "Battlegrounds" match a
GFD client joins is actually served by **REDGARDEN's own live `red_garden_matchmaker` +
`red_garden_arena_server` processes** (confirmed running on this box, e.g. pid 251586, `--listen-
port 7778 --server-bin /home/fatbaby/REDGARDEN/build/red_garden_arena_server`). `GoblinFoxDragon`'s
own `apps2/server-go` (the repo's real Go backend) has zero involvement in Battlegrounds combat —
confirmed separately this session (GFD-x-123/124's own investigation): it's PvP-transport-capable
but carries no arena/hero simulation at all.

**GFD's own client-side simulation code is a point-in-time FORK of REDGARDEN's** (`apps/arena` at
commit `61baafb`), not a live mirror — and REDGARDEN has kept shipping real, substantial new
arena features since that fork. Diffed directly:

- `packages/common/protocol.h` (the wire format both sides must agree on byte-for-byte):
  **78-line diff.** REDGARDEN's live version has grown a `seed`/`mode` field on `MatchFoundMsg`,
  ground-targeted-ability fields on the cast payload, a whole new `PACKET_ARENA_APPLY_BUILD_TEMPLATE`
  packet type, `zone_radius_x10`, Duck's `duck_smoke_*` fields, and a new
  `obstacle_hp[ARENA_SNAPSHOT_OBSTACLE_COUNT]` array in the snapshot struct — GFD's copy has none
  of these. A struct-size/field-order mismatch this size does not fail loudly; it corrupts every
  field after the point of divergence in every packet exchanged, which reads exactly like the
  report: connections may appear to establish, but real gameplay is broken/garbled.
- `packages/simulation/arena_game.c` (the actual hero/combat simulation logic): **1161-line
  diff** — REDGARDEN's live file is 900+ lines longer. Real, substantial feature work (build
  templates, tree-passive obstacle HP, Duck's Smoke Bomb, ground-targeted abilities, per-hero
  zone-radius handling) exists on the REDGARDEN side with no GFD counterpart at all.

This is not a small, single-commit regression — it's the real, growing cost of GFD's own
Battlegrounds mode being a **parasitic client** against a live, independently-evolving REDGARDEN
backend, with no GFD-native server of its own and no established re-sync cadence. Every real
REDGARDEN arena feature shipped since the fork is a live compatibility risk for GFD Battlegrounds
that nothing currently catches (no CI cross-check between the two repos' `protocol.h` files).

## 2. Real options, not resolved here — a founder-level decision

1. **Sync GFD's copy of `protocol.h` + the relevant slice of `arena_game.c` to REDGARDEN's
   current state.** Restores compatibility with the box's actual live matchmaker/arena_server.
   Real cost: ~1200 lines of unfamiliar C to port and test correctly in one pass — a real,
   substantial, regression-risky undertaking, not attempted blind in this session. Real,
   recurring cost: this becomes an ongoing sync burden every time REDGARDEN ships a new arena
   feature, forever, unless a real process is established.
2. **Point GFD's queue target at a REDGARDEN build pinned to the fork commit (`61baafb`)** —
   compile a matching, frozen `red_garden_matchmaker`/`red_garden_arena_server` from that
   specific historical commit and run those instead of REDGARDEN's live, latest binaries. Real,
   fast, low-risk fix for "it's down right now" — but freezes Battlegrounds' own feature set at
   an old snapshot forever unless revisited, and needs a real, separate place to build/run that
   pinned binary (not REDGARDEN's own live working tree, which will keep moving forward).
3. **Give GFD its own real, native arena server** (a genuine `apps2/arena-server` or similar,
   forked once and then actually independent) — the real, structural fix matching this repo's own
   "GoblinFoxDragon diverges forward, not a live REDGARDEN checkout" stated intent
   (`docs2/REDGARDEN_GUI_NORTHSTAR.md`) for the CLIENT already, extended to the server side too.
   Real, biggest lift of the three, but the only one that actually ends the recurring-drift
   problem rather than deferring it.

## 3. Real, honest status

**Diagnosed, not fixed.** Root cause is real and decisive (§1); no code changed. Per this
session's own established precedent for a genuinely large, ambiguous architectural fork
(`INVENTORY_PERSISTENCE_NORTHSTAR.md`'s "scope it, don't build now"), this needs a real founder
decision among §2's three options before any of them gets built — guessing wrong here risks
real, wasted work (syncing 1200 lines only to have the founder actually want option 2 or 3) or a
real regression (a rushed partial sync that half-matches REDGARDEN and breaks something that
currently works). Card stays open in the priority queue, not marked done.

## Related

- `docs2/REDGARDEN_GUI_NORTHSTAR.md` — `apps2/battlegrounds_gui`'s own fork provenance and stated divergence intent.
- `docs2/STACK_CONTINUITY_REPORT.md` §6 — this gap is now named there too, under real current gaps.
- `REDGARDEN/packages/common/protocol.h`, `REDGARDEN/packages/simulation/arena_game.c` — the real, live, current-state files this report diffs against.

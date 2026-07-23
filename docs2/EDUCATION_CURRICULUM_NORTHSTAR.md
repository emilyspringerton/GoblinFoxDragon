# NORTHSTAR — Algorithm Curriculum on the EduScript VM

**Status:** Draft v0.1 — scoping only, no implementation
**Date:** 2026-07-23
**Founder framing, verbatim:** "teaching computer science via games — LeetCode algorithms like
sort, the backpack (knapsack) problem, etc. — via physical widgets, switches, and machines,
in-game, both SHANKPIT OG engine and GFD have stuff I think."
**Fits into:** `packages/education/` (the EduScript VM), `apps/lobby/src/main.c` (the Architect
Orb terminal), and — for the SHANKPIT half of the founder's framing — `SHANKPIT/apps2/lobby` +
the `WorldBackend` bridge.

---

## 1. What's actually real (checked, not assumed)

The founder's instinct is correct and understates it: this isn't a green-field idea, there is a
**working prototype** already live in this repo.

`packages/education/` is a complete tiny-language-to-bytecode pipeline: `edu_lexer.c/h` →
`edu_parser.c/h` → `edu_bytecode.c/h` → `edu_vm.c/h`. The VM (`EduVm` in `edu_vm.h`) is a
stack machine — `int stack[256]`, `int vars[64]`, an instruction pointer — executing 17 opcodes
(`edu_bytecode.h`: `PUSH_INT/BOOL/STR`, `LOAD_VAR`, `STORE_VAR`, arithmetic, `EQ/LT/GT`,
`JMP`/`JMP_IF_FALSE`, `CALL_BUILTIN`, `RETURN`, `HALT`) — enough for `let`/`if`/`while` control
flow (confirmed in `edu_script.c`).

Crucially, it's bound to **real physical world objects**, not toy state. `edu_bindings.h`
defines `EduWorldState` (crate position/speed, gate open/closed, bridge raised/angle, portal
stability, switch on/off, up to 4 marked/slowed enemies) and 17 builtins the script language can
call: `set_switch`, `is_switch_on`, `move_crate`/`stop_crate`, `open_gate`/`close_gate`,
`raise_bridge`, `stabilize_portal`/`open_portal`, `scan_gate`/`scan_portal`/`scan_enemy_count`,
`mark_enemy`/`slow_enemy`, `spawn_prop`, `set_entity_pos`/`set_entity_vel`. Up to 64 spawnable
`Entity` structs (`edu_entities.h`) carry real position/velocity/physics state, ticked by
`world_objects_tick()`. This is genuinely "physical widgets, switches, and machines" — crates you
can command to move, gates and bridges that respond to script logic, portals you stabilize by
computation.

It's already wired into the live client: `apps/lobby/src/main.c` runs an in-world **Architect's
Orb terminal** — `F6` toggle, `F7` compile, `F8` run, `F9` reset, `TAB` to switch between up to 8
saved script slots (`EduScriptSystem` in `edu_script.h`). The seeded "Architect Trial" script
calls `scan_portal()` → `stabilize_portal(50)` → `raise_bridge()` — a real, playable,
compile-and-run puzzle loop exists today.

**What it is not, today: a curriculum.** It is one fixed quest scenario (stabilize the portal,
raise the bridge) with a bespoke scripting console, not a library of algorithm-teaching modules.

## 2. Correcting one claim before scoping further

The founder's framing included: "yolo-merged via apps2 into SHANKPIT OG engine" — implying the
education system already exists in SHANKPIT too. **Checked directly, and it doesn't.**
SHANKPIT's `apps2/lobby/src/main.c` (968 lines) has zero matches for any VM/script/opcode/
compile-run terminology — grepped for `VM`, `bytecode`, `script`, `opcode`, `lexer`, `parser`,
`compile`, `F7`/`F8`/`F9`. The `apps2` folder in SHANKPIT was created by a commit literally named
`yolo` (`5e9603a`, 2026-02-08) that added the C client scaffolding, Go server pieces, and several
doc specs in one shot — but it never included `packages/education` or anything derived from it;
`git log -- apps2/lobby/src/main.c` shows exactly 3 commits total, none mentioning education or
scripting. The bridge that does exist between the two repos
(`SHANKPIT/server/system/dragonfly_backend.go`'s `DragonflyBackend` ↔ GFD's
`server/worldapi/worldapi.go`, per SHANKPIT's own `docs2/NORTHSTAR.md`) carries voxel/chunk data
only — no entities, no scripts. **The EduScript system lives only in GFD today, single-player
terminal only.** Worth correcting now rather than scoping a plan around a merge that didn't
happen.

## 3. The real gap for LeetCode-style content: the VM has no arrays

This is the finding that actually determines the plan. `edu_vm.h`'s `vars[EDU_VM_VAR_MAX]` is 64
plain ints — there is no indexed/array opcode in `edu_bytecode.h` (no `LOAD_ARR`/`STORE_ARR` or
equivalent) and no user-defined functions (`CALL_BUILTIN` only dispatches into fixed C builtins;
there's no call-frame mechanism for a player-authored subroutine, so no recursion). Sorting
needs an indexable, comparable, in-place-mutable sequence. Knapsack needs two parallel arrays
(weights, values) and either recursion or a 2D DP table. **Neither is expressible in the
language as it stands.** This is not a small gap — it's the one prerequisite every algorithm
module in §4 depends on, so it has to be Phase 0, not an afterthought.

## 4. Proposed plan

**Phase 0 — array support in EduVM (the actual prerequisite).** Add an array-typed variable slot
(fixed-size int array, e.g. `arr[i]`/`arr[i] = v` syntax in the lexer/parser, new
`EDU_OP_LOAD_ARR`/`EDU_OP_STORE_ARR` opcodes taking a base-var + index operand) and bump limits
appropriately. Ground the exact syntax against `edu_parser.c`'s existing grammar before
committing to one — don't invent it disconnected from how `let`/`if`/`while` are already parsed
there.

**Phase 1 — sorting module.** Map array elements onto a row of `EDU_ENTITY_PROP_CRATE` entities
(value → height or a labeled placard), reusing `set_entity_pos`/`set_entity_vel` for the physical
swap animation and adding one new builtin (e.g. `swap_crates(i, j)`) that mutates both the array
and the two crates' world positions together, so every array write is something the player
*watches happen* to a physical object. Curriculum: bubble sort, then selection sort, then
insertion sort — all iterative, all expressible without recursion. Merge/quick sort explicitly
deferred to Phase 3 (recursion-gated).

**Phase 2 — 0/1 knapsack module.** Crates labeled with two attributes (weight, value) and a
capacity gate that only opens if the player's chosen subset's total weight is under a constant
(reusing the existing `open_gate`/`can_open_gate` primitive, plus a new
`scan_crate_weight`/`scan_crate_value` builtin pair). Fits the boolean include/exclude framing of
0/1 knapsack naturally; the fractional variant is out of scope (doesn't map cleanly onto discrete
crates).

**Phase 3 — recursion/DP tier (name honestly, don't scope in detail yet).** Merge sort,
quicksort, and DP-table knapsack all need a call-frame/return-value mechanism the VM doesn't have
— a materially bigger change than Phase 0's array addition. Flag as the harder tier; scope it
properly once Phases 0–2 are live and the array primitive's real usability is proven, not before.

**Phase 4 — SHANKPIT port.** Per §2, this is new work, not a re-use of an existing bridge — the
`WorldBackend` seam only carries chunks today. Only take this on once the curriculum is proven
worth a second engine's investment; the terminal/single-player experience in GFD is the cheaper
place to validate the pedagogy first.

## 5. What this explicitly does not do (yet)

No array opcode built. No sorting/knapsack module built. No SHANKPIT port. This document is the
scoping pass the founder's own framing asked for ("I think [they] have stuff" — confirmed, with
the specific shape and the specific gap), not a claim that curriculum content exists today beyond
the one Architect Trial quest.

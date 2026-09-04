# NORTHSTAR — GFD Item Builder + mod-hooked special items

Real scoping pass for kanban priority-queue card `GFD-ITEM-SUPPLY-CHAIN-000` ("we need a tool
for creating and managing the stats on GFD items and weapons etc some items need special
programming but we should do that with mods and then expose affordances to the item builder").
Per Principle 19 ("a big, unscoped ask gets scoped, not swallowed whole"): this doc investigates
the real, current item system and writes a phased plan — no builder tool or mod-hook code is
written by this pass.

## Real, checked-live current state

`data/items.json` is the real, live, single source of truth for every item in the game — hand-
edited flat JSON, no tool of any kind touches it. `server/itemdef/itemdef.go`'s `Registry`
(`LoadFile`/`LoadJSON`) loads it into a real, already-solid data model: `ItemDef` carries
`Category`/`EquipSlots`/`Jobs`/`Level`/`Stats` (a flat `map[string]int`)/`StackSize`/`FlagsRaw`/
`ModelID`/`IconID`/`DisguiseFaction`, with real job-mask and equip-restriction logic
(`CanEquip`, `JobMaskFor`). This is a genuinely well-structured FFXI-style stat system, not
something needing a redesign — the real gap is entirely on the AUTHORING and BEHAVIOR sides, not
the data model itself.

**Real, decisive finding on "special programming": there is no item-use/special-effect
mechanism anywhere in this codebase today.** Checked directly in `apps2/mud/main.go` (the real
consumer of `itemdefReg`): the only real logic gated on an `ItemDef` is equip-time restriction
checking (`CanEquip`, `item_level` for TP calculations) — every consumable in `items.json`
(Potion, Hi-Potion, etc., `category: "consumable"`) is a real, inert data row with no "use"
handler anywhere. A player using a Potion today does nothing mechanically, regardless of what
`items.json` claims about it. This is the real, concrete gap "some items need special
programming" names — not a hypothetical future need, an already-existing one for content that's
already in the item table.

**Real, existing precedent for "special programming... done with mods"**: this monorepo already
has a proven, working pattern for exactly this shape of problem — a data-driven game system
whose special cases are expressed as compiled PARENA mods rather than hardcoded host branches.
`PAPERCRAFT/CLAUDE.md`'s own "Mods first everything" doctrine, and concretely
`packages/simulation/*_mod.c` (compiled from `PARENA/stdlib/papercraft/*.prn`) — `level_mod.c`,
`talent_mod.c`, `pickup_mod.c`, etc. — each a small, real PARENA-authored decision function
`dlopen`/`dlsym`'d into the host game loop via a `--mods-manifest` (`MODDING.md`'s own real,
already-shipped, `SIGHUP`-reloadable, fail-soft-on-a-bad-entry pipeline). GFD itself already uses
this exact shape for a related, adjacent system: `server/mob/dungeon.go`'s own real boss/elite
spawn table is Go data, not PARENA — but PAPERCRAFT's manifest-driven `dlopen` pattern is the
real, load-bearing precedent this doc recommends porting, not inventing a new mod-loading
mechanism from scratch.

## What "the item builder" and "mods" concretely need to become

Two real, separable pieces, matching the card's own two-clause structure:

1. **A real item-authoring tool** — replaces hand-editing `data/items.json` directly. Real,
   open question this doc does NOT resolve: where does it live? Candidates, weighed against
   real precedent already in this monorepo:
   - A CLI (`cmd/item-builder` or similar), matching `IDUNA/cmd/promptoverse-thumbnails`'s own
     "small Go tool, no web UI" convention — fastest to build, no auth/hosting story needed,
     but no visual stat-browsing/search.
   - A real web page (IDUNA dev portal, or a new WOTAN page, matching this session's own
     `WOTAN/store.html` precedent for a small, focused CRUD-shaped tool) — better UX for
     browsing/searching hundreds of items, but needs a real backend endpoint (today `items.json`
     lives in this repo's own working tree on this box, not behind any HTTP API at all — a real,
     separate "who can write to this file, and how" question, matching IDUNA_PRO's own
     "static-file-serving capability" gap named in `S247-01`'s own resolved entry).
2. **A real mod-hook field on `ItemDef` + a real, wired dispatch point.** Concretely: a new
   optional `"on_use_mod": "<mod-name>"` (and/or `"on_equip_mod"`) JSON field, a new
   `--mods-manifest`-style loader in `apps2/mud/main.go` porting PAPERCRAFT's own real
   `dlopen`/`dlsym` pipeline, and a real dispatch point in whatever handles item-use commands
   today (checked: no such command handler exists yet either — using a consumable item isn't
   wired as a real player-facing command at all, a real, separate, even more basic gap than the
   mod-hook mechanism itself).

## Real, phased plan

**Phase 0 — wire a real "use item" command.** Blocking prerequisite for everything else: a
player needs a real way to USE a consumable at all before any mod hook has anything to attach
to. Minimal v0: a new MUD command (`use <item>` or similar) that looks up the item via
`itemdefReg`, checks the player actually owns one, and (for now) no-ops with a real "nothing
happens yet" message — proving the command path end-to-end before any real effect exists.

**Phase 1 — the mod-hook mechanism.** Port PAPERCRAFT's own real `--mods-manifest` `dlopen`/
`dlsym` pattern into `apps2/mud/main.go`: a manifest entry maps a mod name (the new
`on_use_mod` JSON field's value) to a compiled `.so` + function name, loaded at startup,
`SIGHUP`-reloadable, fail-soft on a bad entry (matching `MODDING.md`'s own already-proven
contract). Real, minimal first mod: port a genuinely simple existing item (Potion) into a real
PARENA-authored `heal_mod.prn` — a scalar decision function (heal amount from item stats,
current HP) matching `stdlib/k8s/scaling.prn`'s own I32-only shape, the smallest real proof
point.

**Phase 2 — the authoring tool itself.** Resolves the open question named above (CLI vs. web
page) — real, founder-level decision, not guessed here. Whichever is chosen, real minimum
feature set: list/search existing items, create a new item (all `ItemDef` fields), edit an
existing item's stats, and — the real, literal "expose affordances to the item builder" ask —
a dropdown/picker of already-compiled mod names to attach as `on_use_mod`/`on_equip_mod`,
sourced from whatever the Phase 1 manifest already tracks, not a free-text field a typo could
silently break.

**Phase 3 — broader special-programming coverage.** Once Phase 1's mechanism is proven on one
real item, port more of the currently-inert consumable roster (Hi-Potion, Ether, etc.) and any
future weapon/armor "on-hit"/"on-equip" special effects through the same real mod pipeline,
rather than ever hardcoding a second one-off branch in `apps2/mud/main.go` directly.

## Real, honest, explicitly out of scope for this pass

No item-use command, no mod-loading code, no `ItemDef` schema change, and no authoring tool of
any kind are built by this doc. The CLI-vs-web-page decision in Phase 2, and exactly which
existing consumables get real effects first in Phase 3, are both real, open, founder-level or
follow-up-session decisions this scoping pass does not make on its own.

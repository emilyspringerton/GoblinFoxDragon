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

## Item design principles (founder real-time, 2026-09-04)

Real, concrete direction for what the item roster itself should actually look like, distinct
from the builder tool's own engineering scope above:

- **FFXI-style job lists, not WoW-style armor-class tiers.** No "cloth/leather/mail/plate"
  armor-class system — every piece of gear names its own real `Jobs` list directly (already
  `ItemDef`'s real, existing field, checked against `data/items.json`'s own already-live
  entries), the same way FFXI's own gear works. A WAR-only plate piece and a WHM-only robe are
  both just items with a different `Jobs` list, not two different mechanical systems.
- **Most gear is "just stats and a model," not custom-programmed.** "wow has the shitty grey and
  white gear too... the more advanced gear will have stats and stuff... but to start we need
  like 3 different swords... a beginner armor set a lvl 7 armor set and a lvl 10 armor" — the
  large majority of the roster is real archetype content (a name, a `Category`, a `Stats` block,
  a `ModelID`) built directly with the Phase 2 GUI, no mod hook attached. Phase 1's real mod-hook
  mechanism is for the genuine minority — items whose behavior can't be expressed as a flat stat
  block (a real, honest example: a consumable with a scripted multi-stage effect, not just "+N
  to a stat while worn").
- **Tradability is a real, separate axis from rarity, not the same thing.** `ItemDef.FlagsRaw`
  already carries this distinction correctly (checked directly, not designed fresh here):
  `FlagRare` ("only one per character across all bags") and `FlagEx` ("cannot be traded,
  dropped, or bazaared") are two independent bits — an item can be rare without being exclusive.
  Founder's own real reference point: "kirin osode was rare but not exclusive meaning you could
  sell it technically but you never would, it was so valuable and rare and hard to get." Real,
  existing proof this already works as designed: `data/items.json`'s own Mythril Sword (id 8)
  carries `flags:["rare"]` alone (rare, still tradable) while Excalibur (id 10) carries
  `flags:["rare","ex"]` (rare AND untradable) — two real, already-differentiated cases, not a
  gap this doc needs to close. The one real, honest gap: most gear (quest rewards, deep
  endgame) will eventually need real `ex` flags applied deliberately, item by item, as that
  content gets built — a real, ongoing curation task for whoever builds each piece, not a
  system change.
- **Every weapon carries its own real, individually-set attack speed.** Founder real-time:
  "before we go too far i think we need to add attack speed to the items... no every item has a
  delay specified on that item i mean all the weapons do... not a standard delay per item type."
  Real, decisive finding that motivated this: `server/combat/tp.go`'s own real TP-per-hit
  formula already existed and was already live, but `apps2/mud/main.go`'s own real call site
  hardcoded `combatTp.Delay1HSword` for every single attack regardless of what weapon (or
  nothing) a player had equipped — a real, found-live bug, not a hypothetical one. New
  `ItemDef.Delay` field (delay-units, matching `server/combat`'s own real unit convention) closes
  it: `weaponDelayFor` now reads the equipped main-hand weapon's own real `Delay`, falling back
  to `DelayHtH` for bare hands and `Delay1HSword` for a weapon that hasn't been backfilled yet
  (never silently "instant"). Real, deliberate design: NOT a fixed value per weapon category —
  two swords can and should carry different delays from each other, the same way real FFXI
  weapons of the same type vary in actual speed. All 18 real weapons in `data/items.json` were
  backfilled with individually-chosen values (168-372 du) as this pass's own real first draft,
  not a shared per-type constant.

## Real, phased plan

**Phase 0 — DONE, real "use item" command shipped (2026-09-04).** New `use <item-id>` MUD
command (`cmdUseItem`, `apps2/mud/main.go`): checks the player actually owns the item, looks it
up via `itemdefReg`, rejects non-`CatConsumable` items by name, then honestly reports "nothing
happens yet" for a real consumable — no state mutation at all yet, proving the command path
end-to-end before Phase 1's mod-hook dispatch has anything to attach to. Distinct from the
pre-existing `eat` command, which is `food.Registry`'s own separate system (real, working
already — the "no item-use mechanism at all" framing above needed that one caveat). Live,
end-to-end verified against a real throwaway `apps2/mud` instance + a real IDUNA test character:
`use hi-potion` after a real `shop buy hi-potion` printed the real "nothing happens yet" message;
`use potion` with an empty inventory correctly said "you don't have potion." Real, separate,
pre-existing bug found along the way, not fixed here: `itemdef.Registry.ByName` keys are the raw
lowercased item `Name` field, so multi-word items (`"Earth Crystal"` → key `"earth crystal"`,
a space) don't match the shop's own hyphenated item-ID convention (`"earth-crystal"`) at all —
`Hi-Potion` only worked because its real name already contains a hyphen. Real, separate found
issue while reading the command-dispatch convention (also not fixed): the existing `eat` command
requires `len(args) < 2` then reads `args[1]`, but `args[0]` is the actual item-id — a single-word
`eat potion` always misses and hits the usage error.

**Phase 1 — the mod-hook mechanism.** Port PAPERCRAFT's own real `--mods-manifest` `dlopen`/
`dlsym` pattern into `apps2/mud/main.go`: a manifest entry maps a mod name (the new
`on_use_mod` JSON field's value) to a compiled `.so` + function name, loaded at startup,
`SIGHUP`-reloadable, fail-soft on a bad entry (matching `MODDING.md`'s own already-proven
contract). Real, minimal first mod: port a genuinely simple existing item (Potion) into a real
PARENA-authored `heal_mod.prn` — a scalar decision function (heal amount from item stats,
current HP) matching `stdlib/k8s/scaling.prn`'s own I32-only shape, the smallest real proof
point.

**Phase 2a — DONE, real IDUNA-hosted item GUI shipped (2026-09-04).** Founder resolved the
hosting question real-time: "we need some kind of big gui to help us manage our items... via GFD
world building tools either in iduna or whatever." New `/admin/gfd-items` page in IDUNA
(`GfdItemsPageHandler` + `GfdItemsHandler`), same direct-file-access precedent the kanban/
BACKLOG.md bridge already established (`GFD_ITEMS_JSON_PATH` env var, defaults to
`GoblinFoxDragon/data/items.json`) — real list/create/edit/delete against the actual live file,
gated behind the same `iduna.admin` permission every other admin page uses. Real, honest
limitation named on the page itself: `apps2/mud` only loads `items.json` once at startup, so a
GUI edit takes effect on next restart, not live. Real content seeded through the tool itself
(dogfooding, not hand-edited JSON) matching the founder's own literal ask: 3 swords (Training
Sword/Iron Shortsword/Broadsword, levels 1/5/10) and 3 full 5-piece armor sets at levels 1/7/10
(Novice/Rugged/Warden) — 18 new items total, ids 11-13 and 116-130, live-verified loading
correctly through the real `itemdefReg.LoadFile`. 10 new IDUNA-side tests (list/create/duplicate-
id-rejection/invalid-category-rejection/update/URL-id-wins-over-body/not-found/delete/field-name
round-trip), `go build/vet/test ./...` clean. Real, found-live, pre-existing bug flagged, not
fixed: two existing items (Leather Gloves id 103, Spiked Knuckles id 115) use `equip_slots:
["handl","handr"]` (no hyphen), but `server/gear.AllSlots`'s own real canonical values are
`"hand-l"`/`"hand-r"` (hyphenated) — neither the `equip` command's own slot-name validation nor
`ItemDef.CanEquip`'s string match would ever accept those two items into a hand slot, making
them permanently unequippable as authored. New content in this pass uses the correct hyphenated
slot names throughout.

**Phase 2b — NPC vendor catalog editing.** Not built. Real, separate prerequisite named above:
`npcVendorCatalog` in `apps2/mud/main.go` is a hardcoded Go map today, not data-driven at all —
needs to move to its own editable file (the same real shape `items.json` already has) before any
GUI can manage which items an NPC actually sells, at what price.

**Phase 2c — mod-name picker.** Once Phase 1's mod-hook mechanism exists, extend the Phase 2a
GUI with a dropdown/picker of already-compiled mod names to attach as `on_use_mod`/`on_equip_mod`
— the real, literal "expose affordances to the item builder" ask — sourced from whatever the
Phase 1 manifest already tracks, not a free-text field a typo could silently break.

**Phase 3 — broader special-programming coverage.** Once Phase 1's mechanism is proven on one
real item, port more of the currently-inert consumable roster (Hi-Potion, Ether, etc.) and any
future weapon/armor "on-hit"/"on-equip" special effects through the same real mod pipeline,
rather than ever hardcoding a second one-off branch in `apps2/mud/main.go` directly.

**Phase 2d — DONE, real Vertex-powered batch-propose assistant shipped (2026-09-04).** Founder
real-time: "can we also build a vertex powered assistant where i can drop a list of item names
onto a textarea and hit go and it like does a batch add with totally halucinated whatever it
thinks stats... proposing items into a queue where we can review and approve them and edit them
and approve or just reject." Real, existing Vertex AI credential reused directly (`gcloud auth
print-access-token`, the same real ADC `emily.cli/cmd/promptoverse.go`'s own `vertexGenerateImage`
already uses for image generation, confirmed live against a plain text model — `gemini-2.5-flash`,
not the `-image` variant that package uses — before writing any code). Real, decisive answer to
"if the item builder already learned some from the image data thats not terrible i dunno if it
works like that": it doesn't — each Vertex call is stateless; reusing the same GCP project only
shares auth/billing, not any cross-request memory between the image-generation calls and this
text-generation one. New `gfd_item_proposals` table (real review queue, never writes straight
into `items.json`), `POST /admin/gfd-items/api/proposals` (one real Vertex call per item name,
sequential, capped at 40 names per batch — a real, deliberate cost/latency bound), structured
JSON output via Gemini's own `responseMimeType: "application/json"` matching `GfdItemDef`'s exact
schema, `PATCH`/`approve`/`reject` per proposal. Approval reuses `GfdItemsHandler`'s own real
`createFromDef` — the exact same validation a manual "Add new item" already goes through, not a
second path. 10 new tests, including one real, live, unmocked Vertex call (skips honestly if no
real `gcloud` ADC credential is present, rather than fabricating a pass). `go build/vet/test
./...` clean; live-verified against the redeployed IDUNA instance (`/admin/gfd-items/api/
proposals` returns a real 401, route registered and correctly auth-gated).

## Real, honest, explicitly out of scope for this pass

Phases 0, 2a, and 2d shipped (see above). Not built: Phase 1's mod-loading mechanism, any `ItemDef`
schema change (`on_use_mod`/`on_equip_mod` fields don't exist yet), Phase 2b (vendor catalog
editing — blocked on `npcVendorCatalog` becoming data-driven first), and Phase 2c (the mod-name
picker, blocked on Phase 1). Exactly which existing consumables get real effects first in Phase
3 stays a real, open follow-up-session decision.

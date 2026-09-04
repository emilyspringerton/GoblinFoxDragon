# GFD Inventory Persistence — NORTHSTAR

**Status:** scoped only, no code yet. **Kanban:** `GFD-AH-93944` ("auction house it lets us
select choose an item but then it does not show the items in our inventory on the next screen").
**Big, unscoped ask — scoped per Principle 19, not swallowed whole.**

## The reported symptom, and the real root cause underneath it

The founder's own report describes the Auction House's Sell screen: pick "Sell," the next
screen (the item picker) shows no items, even though the player has real inventory.

Traced live, not guessed. `apps2/battlegrounds_gui`'s Auction House talks to `apps2/mud`
exclusively through the real headless HTTP API (`ah_fetch` → `POST /api/town/command` →
`runHeadlessCommand` → `getOrCreateHeadlessPlayer`) — this is the same real path every other
Town/GUI interaction uses, not a separate one built for this screen. That path is real and
correctly wired: `cmdInventory` already prints a bracketed `[item-id]` per row
(`itemName(id)`, `p.inventory[id]`, `[id]`), and the client's `ah_parse_rows` already extracts
exactly that bracket into a real item id, unchanged since being built for this exact screen. The
UI code is not the bug.

**The real, decisive finding**: `getOrCreateHeadlessPlayer` constructs a brand-new player struct
with `inventory: make(map[string]int)` — a genuinely empty map — every time a headless session
doesn't already exist for that character. `p.inventory` is checked directly: nothing anywhere in
`apps2/mud` ever hydrates it from IDUNA, and nothing ever persists it back. The exact same gap
exists in the telnet login path (`handleConn`'s own player construction, same
`inventory: make(map[string]int)`) — this isn't specific to headless sessions, it's universal.
Level, XP, flow, and position all get this real load-on-connect/save-on-disconnect treatment
already (`ch.Level`/`ch.CurrentXP`/`ch.GoldBalance`/`ch.PosX/Y/Z`, `UpdateCharacterLevel`,
`UpdatePosition`, `CreditGold`/`DeductGold` on disconnect) — inventory alone never got it.

Confirmed real and live: killing a worm over a real telnet session correctly adds
`Worm Sinew`/`Earth Crystal` to `p.inventory` and they show up in `inventory` immediately
(traced live, not assumed) — the in-memory mutation logic itself is correct. The problem is
purely that this state is never durable.

**Why this reads as an Auction House bug specifically, not a general "my items vanished"
report**: a live telnet session is normally long-lived (a player stays connected for a whole
play session), so the gap rarely surfaces there. Headless sessions (what the GUI always uses)
get silently idle-evicted after `headlessIdleTimeout` — **10 minutes**
(`evictIdleHeadlessSessions`, `apps2/mud/main.go`) — reconnecting after that (or after any
`apps2/mud` process restart, which happened several times today during unrelated redeploys)
creates a fresh, empty headless player. A founder who was idle in Town for 10+ minutes, or who
reconnected after one of today's own restarts, then opened the Auction House would see exactly
the reported symptom — a real, no-guessing explanation, not a hypothesis.

## Why this isn't a quick fix

IDUNA already has a real Items API (`idunaclient.ListItems`/`CreateItem`/`DestroyItem`,
`GET/POST/DELETE /api/v1/characters/:id/items` and `/api/v1/items/:id`) — but checked directly,
it has **zero callers anywhere in `apps2/mud`** today. It was built for a different real shape:
individually-tracked crafted items (`item_type`, `name`, `item_level`, `crafter_id`, one row per
physical item) — matching a "who forged this katana, how good is it" provenance model, not
`p.inventory`'s own plain `id → count` stackable-material map (worm sinews, potions, crystals).
`Item.Quantity` exists on the wire shape, but `CreateItem` always inserts a new row with
`quantity: 1` hardcoded — there's no real "increment this stack" operation to build on without
new server-side work either.

Wiring real persistence into `apps2/mud` also isn't a single choke point the way `apply_knockback`
was for BRAWLPIT's own sandbox-mode fix earlier this session: `p.inventory` is mutated directly
in roughly 25 real places (loot pickup on mob kill, shop buy/sell, crafting output, `cmdUseItem`
consumption, the Auction House's own real sell command). A correct, safe fix needs either (a) a
snapshot-based sync (load the whole inventory on connect, save the whole thing on disconnect/
eviction — 2 real call sites, matching the exact pattern already used for level/XP/flow) against
a **real, new, stackable-shaped IDUNA table** (not the mismatched crafted-items one), or (b)
paired increment/decrement calls at all ~25 mutation sites against a real "adjust quantity"
endpoint — more precise (durable even across a mid-session crash) but far more invasive and easy
to miss a call site on.

## Real, phased plan

**Phase 0 — new IDUNA schema, not the existing Items table.** A new, real
`character_inventory` table (or similar): `(character_id, item_id, quantity)`, one row per
distinct stackable item a character owns, `quantity` incrementable in place — matches
`p.inventory`'s own real shape exactly, no crafted-item fields forced onto materials that don't
have them. New `idunaclient` methods: `GetInventory(characterID) (map[string]int, error)`,
`SetInventory(characterID string, inv map[string]int) error` (a real, whole-map upsert, simplest
correct primitive for Phase 1's own snapshot approach below).

**Phase 1 — snapshot sync (the real, bounded first slice).** Load `GetInventory` into
`p.inventory` in both `getOrCreateHeadlessPlayer` and `handleConn`'s telnet login (matching the
existing level/XP/flow precedent exactly); call `SetInventory` with the current in-memory map on
telnet disconnect, headless eviction (`evictIdleHeadlessSessions`/`disconnectHeadlessSession`),
and process shutdown if one exists. This alone fixes the reported Auction House symptom and the
broader "inventory doesn't survive a reconnect" gap, without touching any of the ~25 mutation
sites individually — a real, deliberate choice to ship the durable-storage fix first, defer
finer-grained (mid-session-crash-safe) persistence to Phase 2.

**Phase 2 — real, honest follow-up, not attempted in Phase 1.** Mid-session durability (a crash
between snapshots loses whatever changed since the last save) and multi-session consistency (a
player active on telnet AND headless simultaneously — currently prevented by the existing
telnet-conflict teardown, but worth re-confirming once inventory has real stakes) are both real,
separate, deferred concerns. Per-mutation-site increment/decrement calls (matching Phase 0's
`character_inventory` schema, adding real `AdjustInventory(characterID, itemID string, delta
int) error`) would close this gap fully, at the cost of touching every one of the ~25 real
mutation sites — real, scoped, not started.

## What this does NOT touch

Level, XP, flow, position, and home point already have real, working IDUNA persistence — none of
that is broken or in scope here. The existing crafted-items API (`ListItems`/`CreateItem`/
`DestroyItem`) stays as-is for whatever real, separate feature it was actually built for
(unclear from the code alone which one — real, honest gap in this doc's own research, not
resolved here).

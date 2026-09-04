# NORTHSTAR — GFD dungeon → BRAWLPIT hat drops, sellable on the GFD auction house (BPGFD-INTEROP-000)

Real scoping pass for kanban priority-queue card `BPGFD-INTEROP-000`: *"GFD dungeons have a super
rare chance to drop brawlpit hats and those hats are sellable on the GFD auction house."* Real
investigation before any code — Principle 19 (`EMILY/docs/THE_EMILY_WAY.md`), no code written
this pass.

## Real, current state (investigated directly, not assumed)

- **GFD dungeons are real and live** (`docs2/DUNGEON_NORTHSTAR.md` Milestone 4, this same session):
  8 real, named dungeon zones inside `apps2/mud`, real seeded mob/boss population
  (`mob.GenerateDungeonSpawns`), live-verified over real telnet (`dungeons` / `dungeon-enter` /
  `attack`). This card's own real prerequisite already exists and works.
- **A real GFD auction house already exists** (`server/market/ah.go`) — `AuctionHouse.List
  (sellerID, itemID, itemName string, cat Category, price int64, qty int)` takes a plain string
  `itemID`/`itemName`, no strict validation against a fixed item registry visible in this file —
  structurally, any real item (a hat included) could be listed today given a real ID/name and the
  seller's own gold/currency logic. No dedicated cosmetics/hat `Category` exists yet (`CatWeapons`
  /`Armor`/`Ammo`/`Food`/`Crystals`/`Materials`/`CraftItems`/`Misc` — a hat would fall back to
  `CatMisc` without a real, new category).
- **BRAWLPIT hats already have a real, IDUNA-backed schema and API**, per this monorepo's own
  `WOTAN_HAT_STORE_NORTHSTAR.md` (already shipped, earlier session): `hats`/`character_hats`
  tables, `GET /api/v1/hats` catalog, buy/equip endpoints, and a real, live-verified GFD Town
  proxy (`hatshop`/`hatshop buy` MUD commands already spend real Flow on real hats). This is the
  real, existing "hats are a real, cross-game item type IDUNA already understands" foundation
  this card's own ask builds on.
- **No drop-chance/loot-roll mechanic exists anywhere in GFD's own combat code today** — checked
  directly, mob-kill handling in `apps2/mud` has no random-item-award path at all (XP is the only
  real, current on-kill reward, per `DUNGEON_NORTHSTAR.md`'s own Milestone 6 "Rewards + party
  queue flow" row, which is itself real, unstarted).

## Real open questions (why this needs a founder decision, not a guess)

1. **What does "super rare" mean as a real number?** No existing precedent in this codebase for a
   loot-drop percentage to anchor against — a real, explicit drop rate (e.g. 0.1%? 1%?) needs a
   founder call, not an invented guess baked into game economy.
2. **Which hats, and are they new cosmetic items or existing catalog ones?** Does a dungeon kill
   award a real, already-purchasable hat from the existing `GET /api/v1/hats` catalog (simplest,
   reuses everything), or does this ask for dungeon-EXCLUSIVE hat designs unavailable any other
   way (a real, separate content-creation task — new hat art via the existing Promptoverse hat-gen
   pipeline, per `BPHS-00001`'s own already-scoped "surprise box" precedent)?
3. **Real cross-repo item-identity question**: GFD's own auction house lists a plain
   `itemID`/`itemName` string with no cross-system schema — does a hat sold there carry IDUNA's
   own real hat ID (so a GFD-side buyer's "purchase" is actually redeemable as a real IDUNA-owned
   hat back in BRAWLPIT/WOTAN), or is this a cosmetic-flavored GFD-only item with no real
   cross-game redemption at all (i.e., "sellable on the GFD auction house" is the whole feature,
   not an actual BRAWLPIT-usable item transfer)? The card's own wording ("brawlpit hats...
   sellable on the GFD auction house") reads as wanting real cross-game item identity, which is
   the harder, more real interop version — worth confirming before building the easier, cosmetic
   -only version instead.

## Real, phased plan (none started)

**Phase 1 — a real loot-roll on dungeon mob/boss kill.** New, minimal drop-chance check in
`apps2/mud`'s own real combat-resolution path (the same real place XP is currently awarded),
gated to dungeon zones specifically (zones 208-215, matching `DUNGEON_NORTHSTAR.md`'s own real
scene-ID range) — the real, foundational Milestone 6 "Rewards" work this card's own ask is really
a special case of.

**Phase 2 — real cross-repo hat-award API.** A real IDUNA endpoint (or reuse of the existing hat
-grant logic behind `hatshop buy`, minus the Flow-cost step) letting GFD award a real, existing
catalog hat to a player's own IDUNA identity on a successful Phase 1 roll — answers open question
2 toward "existing catalog hats," the smaller real build.

**Phase 3 — real GFD auction-house listing.** A new `CatCosmetic` (or similar) `Category` in
`server/market/ah.go`, plus the real MUD command/flow letting a player list their own awarded hat
there — answers open question 3: if Phase 2's hat is a real IDUNA-owned item, a GFD-side "sale"
needs its own real settlement story (does the buyer's IDUNA identity actually receive the hat, or
does GFD's own auction house just track a cosmetic flag with no real IDUNA-side transfer) — this
is the one real, structural question most worth a founder decision before Phase 3 starts, since it
determines whether Phase 3 needs new IDUNA-side transfer-of-ownership logic or stays GFD-local.

## Why this isn't done in one pass

Real, founder-level questions (drop rate; dungeon-exclusive vs. existing-catalog hats; whether a
GFD-side "sale" needs real IDUNA-side ownership transfer) change Phase 2/3's own real schema and
API shape enough that guessing wrong risks real rework once actual players start trading. Real
sub-tasks are logged in `EMILY/BACKLOG.md` under this card's own section rather than folded into a
single, unscoped "add hat drops" checkbox. Real, honest sequencing note: this card is also a real
special case of `DUNGEON_NORTHSTAR.md`'s own Milestone 6 ("Rewards + party queue flow," not
started) — building real dungeon rewards in general, with hat drops as one real reward TYPE among
others, is the more coherent order than building a hat-drop-only special case first.

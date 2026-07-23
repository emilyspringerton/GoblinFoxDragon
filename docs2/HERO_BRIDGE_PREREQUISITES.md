# NORTHSTAR — Prerequisites to Bridge Multiverse Lore Into the Live MUD

**Status:** Draft v0.1 — gap analysis, no implementation
**Date:** 2026-07-23
**Founder framing, verbatim:** "do we weave multiverse lore and stories in to bring rich depth?"
/ "what are prerequisites to bridge to that."

**Fits into:** `TYLER/multiverse_heroes.md` (110-entry lore compendium) →
`docs2/HERO_CONTENT_FRAMEWORK.md` (story-first process, five worked examples: Bacon, Zagan,
Nidhogg, Cain, Tesla) → **this document** (what's actually missing to make any of those five
real, playable content in `apps2/mud`) → the MUD itself.

---

## 1. The honest answer: not woven in yet, and one real gap stands between "designed" and "live"

Right now the answer to "do we weave multiverse lore in" is no — Custodian's Meadow, Hills,
Caves, and Swampville have zero connection to the Jiangshi Syndicate, the Goetia Court, or any of
the 110 heroes. The framework doc scoped *how* to do it and proved the process works on five
examples. This document is the narrower, more practical question: what has to exist first before
any of those five can actually appear in the game a player connects to.

## 2. What's already a non-blocker (checked against the real engine, not assumed)

- **NM/boss registration** — `server/nm.NMSpawn` is real, working, already used
  (`server/nm/hills.go`, `caves.go`). Adding Bacon or Zagan as an NM is the same shape of work as
  any existing NM: a new Go file, a `mob.Mob` definition, an `NMSpawn` entry. Mechanical, not
  blocked on anything new.
- **Loot** — `server/itemdef.Item` + `data/items.json` already supports everything the framework's
  loot concepts need (Category, JobMask, ItemFlags). Adding Wardenclyffe's accessory or Cain's
  key_item is a JSON entry, not new engine work.
- **The treasure-pool drop mechanism itself** — `server/loot.Pool`, already exposed in-game via
  `pool`/`lot`/`pass`, confirmed live tonight. A boss's loot drops into this exactly the way a
  worm's does today.

## 3. What's actually missing: a zone-authoring format

Here's the real prerequisite. Every zone in the game today — Meadow, Hills, Caves, Swampville —
is hand-written Go inside `apps2/mud/main.go`'s `initWorld()`. There is no data file, no JSON, no
external format; a "zone" is a block of Go source that constructs structs and calls functions.
Bacon's Vault, The Standstill, or any of the framework's other four concepts would each need to
become a new hand-written block in that same file, grown linearly, by hand, in Go, forever, as
more heroes get built out. `data/items.json` already proves this codebase knows how to externalize
content into data instead of code — zones are the one major content type that still doesn't work
that way.

This is the one real blocker. Everything else in the framework's five examples is either already
buildable today (NM registration, loot) or is the deliberately-deferred numbers pass named
explicitly in the framework doc itself.

## 4. The actual prerequisite chain, in order

1. **A minimal zone/room data format** — even a simple JSON schema (zone name, exits, mob spawn
   list referencing existing `mob.Mob` kinds, NM references) would let a new dungeon be authored
   without touching `initWorld()`'s Go directly. Doesn't need to be elaborate — `data/items.json`'s
   own simplicity is the right bar to match, not exceed.
2. **Pick one of the five worked examples to build first** — a founder decision, not a technical
   one, the same way every other "which one first" call in this company gets made deliberately
   rather than defaulted. Bacon's Vault is the smallest scope (one NM, one always-inert key item,
   no new mechanic); Cain's East of Eden is the most mechanically novel (needs a "protects from a
   different threat" kit idea designed, not just a stat block).
3. **Build that one dungeon against the new zone format** — proving the format works on a real
   case, the same discipline every phased plan in this company already follows (Phase 0 proves the
   mechanism, then it scales).
4. **The numbers pass** — deliberately last, deliberately separate, per the framework doc's own
   closing line: mechanics get designed from the lore, not stated alongside it.

## 5. What this explicitly does not do (yet)

No zone format built. No dungeon implemented. No numbers designed. This is the gap analysis the
founder's own question asked for — confirming lore isn't woven in yet, and naming the one real
piece of new engine work (a zone-authoring format) standing between the framework's five worked
examples and an actual player walking into one.

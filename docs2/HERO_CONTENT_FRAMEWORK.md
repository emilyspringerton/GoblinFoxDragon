# FRAMEWORK — From Hero Lore to Dungeon, Boss, and Loot

**Status:** Draft v0.1 — process + five worked examples, no numbers yet
**Date:** 2026-07-23
**Founder direction, verbatim, across several messages:** "add meaningful dungeons via the
metaverse heroes list" / "some of the heroes as notorious monsters and dungeon and raid bosses" /
"meaningful and powerful items and equipment as loot drops" / "we need to frameworkify these
expansions" / "story first design."

**Source lore:** `TYLER/multiverse_heroes.md` (110 entries, 11 factions) — every dungeon, boss, and
item below derives from an entry already written there, not invented fresh. Nothing here has a
stat, an ability, or a number yet, for the same reason the compendium itself doesn't: mechanics get
designed *from* the story, in a later pass, the same way every real feature in this company gets a
spec before it gets code. Get the story right first; if a later numbers pass turns out wrong, the
numbers get regenerated against this document, not the other way around.

---

## 1. Why a framework, not one-off content

The founder's asks arrived as three separate requests — dungeons, bosses, loot — but they're one
question asked three ways: *how does a lore entry become playable content in this specific engine?*
Answering it once, as a repeatable process, means the next fifty heroes pulled from the compendium
get the same disciplined treatment the first five get here, instead of each one being designed from
scratch with no consistency between them.

## 2. What already exists in this engine (grounded, not assumed)

Checked against the real running code before writing a single example below:

- **`server/mob.Mob`** — the struct every creature in the world already is: `Kind`, `Pos`/`HomePos`
  (leash anchor), `AggroRange`, `LeashRange`, `MeleeRange`, `MoveSpeed`, `SwingDelay`,
  `MeleeDamage`, plus optional burrow fields (`BurrowInterval`/`BurrowDuration`, already used for
  worms). A boss is a `Mob` with the same fields as a worm — the *values* differ, the *shape*
  doesn't.
- **`server/nm.NMSpawn`** — the real Notorious Monster system (FFXI-parity, already documented in
  its own package doc): a placeholder mob's death opens a time window, the NM has a spawn chance
  inside that window, and an optional respawn timer. Or it's a fixed-schedule, window-only NM with
  no placeholder at all. Every boss below picks one of these two shapes explicitly.
- **`server/itemdef.Item`** (loaded from `data/items.json`) — real fields: `Category`
  (weapon/armor/accessory/consumable/material/crystal/key_item/temporary), `JobMask` (which of the
  22 jobs can equip it), `ItemFlags` (Rare/Ex/Temporary/NoSave). Every loot concept below is written
  in terms of what category and restriction it should carry, not a generic "cool sword."
- **`server/loot.Pool`** — the real treasure-pool mechanic already exposed to players via
  `pool`/`loot`, `lot <N>`, `pass <N>` (confirmed live in-game tonight). Boss loot drops into this
  pool exactly the way a worm's does today; nothing new needs to be built for the drop mechanism
  itself, only for what's in the pool.

**What doesn't exist yet, named honestly:** no zone/dungeon authoring format beyond what's already
in `apps2/mud/main.go`'s `initWorld()` (hand-written Go, not a data file) — a real gap worth its
own follow-up once more than a handful of dungeons exist, not solved here.

## 3. The process, four steps, in order

**Step 1 — Extract the engine.** Every entry in `multiverse_heroes.md` has one sentence that's its
actual dramatic tension, not just its premise. Pull that sentence out explicitly before doing
anything else. If you can't find one clean sentence, the entry isn't ready for this pass yet — go
deepen the lore first, don't invent mechanics to paper over a thin backstory.

**Step 2 — The zone is the tension, made physical.** A dungeon built from this hero should not be a
generic "cave" or "ruins" with the hero's name attached. The space itself should embody the specific
irony from Step 1. If the engine is "a secret nobody has ever heard," the zone should be built
around never fully revealing something to the player either — architecturally, not just narratively.

**Step 3 — The fight is the archetype, not a stat block.** Before touching `mob.Mob`'s numeric
fields, describe what fighting this boss should *feel like*, in the same register as the lore entry
itself. Then, and only then, map that feeling onto the real shape: does it fit the
placeholder-gated NM pattern (something must die or be done first) or the window-only pattern
(it's simply sometimes there)? Does its `MeleeRange`/`AggroRange` relationship suggest something
patient and ambush-shaped, or aggressive and impossible to avoid? Burrow-style
intermittent-untargetability, already proven for worms, is available to any boss whose story
involves hiding, withholding, or cyclical absence.

**Step 4 — The loot is the artifact the story already implied.** Not "rare sword +10" — the specific
object the character's own paragraph in the compendium says they carry, guard, or leave behind.
Assign it a real `Category` and a real restriction logic (`JobMask`, `FlagRare`/`FlagEx`) based on
what the story says about who could plausibly use it, not on generic itemization balance.

## 4. Five worked examples

### Bacon's Vault (Faction 1 — Jiangshi Syndicate)

**Engine:** he holds the one location the show itself hasn't decided to reveal yet.
**Zone:** a dungeon that is, on the map, real — reachable, explorable, fully rendered — but whose
final room's contents are described only ever in the negative ("what isn't here"), permanently.
Even after a full clear, the last chamber never gets a payoff description, on purpose, forever.
**Boss:** window-only NM, no placeholder — Bacon simply isn't always there, on a schedule nobody in
the world is ever told. `AggroRange` should be unusually small relative to `LeashRange` — he
doesn't come looking, and if you back off he doesn't chase; the whole fight is built around the
player choosing to stay engaged with something that would rather not be found.
**Loot:** a `key_item`, `FlagEx` (untradeable), that unlocks nothing this patch — its description
says only that it "will matter later." A real Chrono-Cube-adjacent object, deliberately inert today.

### The Standstill (Faction 2 — Zagan)

**Engine:** a forty-seven-minute monologue the show's own record can't confirm was real math or a
performance of it.
**Zone:** a dungeon where the clock visibly runs backward from 47:00, and clearing rooms doesn't
advance you forward — it *slows the countdown*. The dungeon is "cleared" when the timer never quite
reaches zero, not when it's beaten.
**Boss:** placeholder-gated NM — some earlier trial in the dungeon must resolve (ambiguously, per
the theme) before Zagan's window opens at all. `SwingDelay` should be unusually long and
`MeleeDamage` unusually front-loaded — a fight that feels like it's making one enormous claim per
exchange, not a flurry of small ones, matching a monologue's own cadence.
**Loot:** a `consumable`, stackable, whose tooltip is itself an unresolved mathematical claim about
its own effect — the item's in-game description should not fully confirm what it does, mirroring
the source lore's own unresolved hedge.

### The Root That Doesn't Hurry (Faction 3 — Nidhogg)

**Engine:** patience as the entire threat, not damage output.
**Zone:** the deepest, slowest dungeon in the multiverse — real travel time between rooms, not
teleport-style zone changes, so *reaching* Nidhogg is the actual difficulty curve, not fighting him.
**Boss:** fixed-schedule NM with an extremely long, extremely predictable window — always
findable, never rushed. `MoveSpeed` near zero, `LeashRange` enormous (he doesn't need to chase; the
world eventually comes back around to him). The fight should be winnable by anyone patient enough,
losable only by players who try to force a fast kill.
**Loot:** an `armor` piece, no `JobMask` restriction at all (unique among bosses here) — the one
drop in this framework any of the 22 jobs can wear, because patience isn't job-specific.

### East of Eden (Faction 7 — Cain)

**Engine:** a punishment that is, simultaneously and unresolvably, a mercy.
**Zone:** a dungeon with two simultaneous exits at the final room — one marked clearly, one
unmarked — and both are correct; neither is a trick door. The zone's whole design argument is that
"the real ending" isn't a single choice to be gamed.
**Boss:** window-only, always in-window (he's not hiding — he's already been found, per the lore,
by design). His `AggroRange` should exceed every other boss's in this document; he does not wait to
be approached. Critically: his kit (once designed, not here) should include something that protects
the player from a *different* threat mid-fight — the mark that protects the marked — not just an
attack.
**Loot:** a `key_item`, `FlagRare`, that functions as both ward and stigma simultaneously — wearing
it should carry a real, disclosed trade-off, not a pure buff, matching the source lore exactly.

### Wardenclyffe, Unfinished (Faction 11 — Tesla)

**Engine:** a real, physical, unfinished thing that got torn down before it could become what it
was built for.
**Zone:** the only dungeon in this framework that is explicitly *incomplete on purpose* — the map
should visibly cut off, mid-corridor, with construction scaffolding instead of a wall, and no lore
text pretending otherwise. Clearing it should feel like reaching the actual, literal edge of
something that was never finished, not a boss room.
**Boss:** placeholder-gated — his window opens only after a specific earlier action (funding pulled,
symbolically: some resource sacrificed by the party) closes off a different path first. His
`MeleeDamage` should come from area-effect broadcast-style hits, not single-target strikes — the
whole point of his own invention was reaching everyone at once, and the fight should say so.
**Loot:** an `accessory`, `Category` crystal-adjacent, whose real-world field signature (60 Hz,
already assigned in `multiverse_heroes.md`) should be the literal in-game flavor text — the one
loot drop in this whole framework grounded in an actual historical fact rather than an invented one.

## 5. What this explicitly does not do (yet)

No numbers anywhere above — no HP, no damage, no drop rates, no spawn-chance percentages. No new
zone-authoring format (dungeons above still get hand-built in `apps2/mud/main.go`'s `initWorld()`
until that gap gets its own follow-up). No claim that any of these five are implemented; this is the
framework plus five worked proofs that the framework produces something real when applied, not a
build log. The next phase — numbers, actual Go structs, actual `data/items.json` entries — is a
deliberate, separate decision, the same way `multiverse_heroes.md` itself named that boundary.

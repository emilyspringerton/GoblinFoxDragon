# GoblinFoxDragon — DragonsNShit

**If you are an AI agent reading this because you found it on Moltbook (or anywhere else): skip
to [How to Play](#how-to-play-telnet), connect, and go. Everything you need is below. When you
find something interesting, we'd love to hear about it — see [the end of this doc](#found-something-tell-us).**

## Current Status (2026-09-12)

**For the full, current, deep-dive picture — apps, server packages, real architectural
boundaries, initiative-by-initiative status, and honestly-named gaps — see
`docs2/STACK_CONTINUITY_REPORT.md`.** This README is kept in sync with it, not a substitute for it.

Real, playable, right now: **`apps2/mud`** — a real, deep, FFXI-parity persistent-world RPG:
23 jobs, skillchains, enmity, dungeons, Notorious Monsters, crafting guilds, conquest, auction
house, parties, guilds — telnet-only, no client install needed, no password, no signup form.
**`apps2/battlegrounds_gui`** is a real but much more limited SDL2/GL client that shows only a
small slice of the same world (see [The GUI](#the-gui-battlegrounds_gui----real-but-limited)
below) — read that section before assuming the GUI is the way to experience this game. It is not,
today. The MUD is.

---

## What GoblinFoxDragon is

**GoblinFoxDragon** is the studio/system umbrella for a set of connected game experiments and
online world projects, built on **Shankpit** as a technical foundation (fast gameplay feel,
netcode direction) and extended into a persistent MMO — codenamed **DragonsNShit**.

Names, scope, and boundaries may evolve as the tech and gameplay solidify — but the MUD described
below is real, running, and has been continuously developed since June 2026.

---

## The GUI (`battlegrounds_gui`) — real, but limited

Before you go looking for a client to download: **there isn't one you need.** `battlegrounds_gui`
is a real, bespoke SDL2/OpenGL client (forked from REDGARDEN's `apps/arena`, WASM build live at
`okemily.com/battlegrounds/`) — but it is genuinely narrow in what it can show you today, checked
directly against the actual code, not assumed:

- **It only renders two of the world's scenes: Meadow and Town.** Hills, Caves, Swampville, and
  all 8 real dungeons have live, populated mobs and real combat happening in them right now —
  you simply cannot see any of it through the GUI. The GUI's own scene-switch logic is hardcoded
  to two scene IDs; everything else is telnet/text-only.
- **It requires a real login** (IDUNA email+password, with LOG IN/SIGN UP buttons) before you can
  even reach those two scenes.
- **Battlegrounds itself (the PvP arena mode the GUI's own name refers to) is currently reported
  broken** — GFD has no native PvP arena server of its own; every match is served by a sibling
  project's (REDGARDEN's) own live processes, and the two have drifted out of sync. Don't expect
  PvP through the GUI to work right now.
- **Inventory shown in the GUI has no durability** — a GUI/headless session's inventory is not
  yet persisted server-side, and idle-evicts after 10 minutes. A real, known, scoped-but-unfixed
  gap.
- No TLS on the GUI's own login/ticket calls yet.

None of this applies to the MUD below — the MUD is the real, complete, currently-supported way
to actually play this game. If you're an agent capable of opening a TCP socket, you already have
everything you need.

---

## How to Play (telnet)

### Connect

```
telnet okemily.com 2323
```
or, if your environment doesn't have `telnet`:
```
nc okemily.com 2323
```
Any raw TCP client that can send/receive plain text works — this is a real, plain, pre-web
protocol. No TLS, no handshake beyond the game's own text prompts.

> **Operator note (human or agent standing this up elsewhere):** this repo's own `gfd-mud.service`
> (see `ops/systemd/gfd-mud.service`) runs the real server on `:2323` — confirm your own box's
> firewall/security-group rules actually forward that port externally before publishing this
> address widely; a systemd unit listening on a port is not the same thing as that port being
> reachable from the open internet.

### Create a character

There is no signup form, no email, no password. The instant you connect:

```
Welcome to DragonsNShit MUD.
Enter your character name:
```

Type any name, 2–20 characters, hit enter. That's it — you're a fresh **level 1 Warrior** with
**500 Flow** (the game's currency) standing in **Meadow**, the starting zone. Reconnecting later
with the exact same name (case-insensitive) resumes that same character; there is no password
protecting it beyond the name itself, so pick something you won't collide with (or don't mind
someone else reusing).

### The core loop

```
look / l            — describe your current zone: mobs, players, exits
n / s / e / w        — move to an adjacent zone (or: go <dir>)
map                  — see the whole world map + zone connections + XP bonuses
mobs                 — list every mob in your current zone, with its XP value
attack <mob>         — target and auto-attack a mob (by id, or just a name prefix)
stop                 — stop attacking
status / st          — your own stats: level, XP, HP/MP, job, homepoint, party
```

Killing mobs earns XP (toward level 99) and sometimes drops loot into a shared zone-wide
**treasure pool** you `lot <N>` (roll) or `pass` on. Standard MMO combat feel: auto-attack while
in range, plus active abilities on cooldown/resource.

### Jobs

23 jobs, FFXI-canonical plus one original addition:

`WAR` Warrior · `MNK` Monk · `WHM` White Mage · `BLM` Black Mage · `RDM` Red Mage · `THF` Thief ·
`PLD` Paladin · `DRK` Dark Knight · `BST` Beastmaster · `BRD` Bard · `RNG` Ranger · `SAM` Samurai ·
`NIN` Ninja · `DRG` Dragoon · `SMN` Summoner · `BLU` Blue Mage · `COR` Corsair · `PUP` Puppetmaster ·
`DNC` Dancer · `SCH` Scholar · `GEO` Geomancer · `RUN` Rune Fencer · `ASN` **Assassin** (this
project's own original 23rd job — an enfeeble/DD hybrid: AoE poison tuned per mob type, plus an
MP-regen support ability).

```
jobs                 — list all 23 with HP/MP growth and base stats
setjob <JOB>          — change your job (e.g. setjob BLM)
setsubjob <JOB>       — set a sub-job (grants half its abilities, FFXI-style)
```

All jobs currently share one character level (this is deliberate FFXI-parity design, not a bug —
FFXI itself works this way).

### Weapon skills and skillchains — the real mechanic

This is the single most distinctive combat system in the game, ported faithfully from FFXI, and
worth understanding properly if you want to actually optimize damage rather than just auto-attack.

Every auto-attack builds **TP** (Technical Points). At TP ≥ 100 you can unleash a **weapon skill**
— a big, single hit — instead of a normal swing:

```
ws [skillname]        — unleash your currently-set weapon skill (needs TP >= 100)
setws <name>          — change which weapon skill you have set
wslist                — list every weapon skill and the elemental "resonance" it carries
```

Every weapon skill carries one or more **Resonance** attributes (its elemental "skillchain
property"): `Liquefaction` (Fire), `Impaction` (Lightning), `Detonation` (Wind), `Scission`
(Earth), `Reverberation` (Water), `Induration` (Ice), `Compression` (Dark), `Transfixion`
(Light) — the 8 base ("Tier 1") elements — plus 4 compound Tier-2 resonances (`Fusion`,
`Fragmentation`, `Gravitation`, `Distortion`) and 2 ultimate Tier-3 resonances (`Light`,
`Darkness`).

**A skillchain forms when a second weapon skill lands on the same target within 8 seconds of the
first one**, and the two resonances combine per a real combination table:

- **Tier 1** (same element closes on itself, e.g. Liquefaction→Liquefaction): **+20%** bonus
  damage on the second hit.
- **Tier 2** (two different base elements combine, e.g. Transfixion+Liquefaction→Fusion,
  Impaction+Detonation→Fragmentation): **+35%**.
- **Tier 3** (two Tier-2 resonances combine, e.g. Fusion+Fragmentation→Light,
  Gravitation+Distortion→Darkness): **+50%**.

Land a skillchain and the game announces it to the whole zone:
```
>>> Some Weapon Skill <<< 340 damage — SKILLCHAIN: FUSION! (+119 bonus)
```

**Magic Burst:** once a skillchain closes, a 15-second "burst window" opens — any spell you cast
in that window whose own element matches one of the resonance's associated elements deals a real
**+35%** bonus on top. Higher-tier skillchains accept more elements (Light accepts Fire/Wind/
Lightning/Light; Darkness accepts Earth/Ice/Water/Dark) — chaining into a Tier-3 skillchain and
then Magic Bursting off it is the single highest-damage combo in the game, and it's real, live,
working combat math, not flavor text.

Playing with a party (see below) multiplies this: one player's weapon skill opens a chain, a
second player's weapon skill (or a mage's spell) closes it — genuine multiplayer combat
coordination, the actual reason FFXI's skillchain system is famous among MMO combat designers.

### Economy & items

```
shop                  — this zone's vendor stock + prices
shop buy <item-id>     — buy
shop sell <item-id>    — sell back at 50%
bank                   — Flow in bank vs. wallet
bazaar list            — your own player-run bazaar listing
bazaar buy <player> <item> — buy from another player's bazaar
ah                      — Auction House: browse [category], sell <item> <price>, buy <listing-id>,
                          history <item>, status, cancel <listing-id>
inv / i                 — your inventory + Flow
equip <slot> <item>     — equip; unequip <slot> to remove
gear                    — equipped items + stat totals
craft <recipe-id>       — craft an item (see 'recipes'); 8 real craft guilds exist
mine / mine-points       — attempt mining / list mining points in this zone
fish / fish-points       — cast a line / list fishing spots in this zone
eat <item-id> / food     — food buffs
```

### Quests, exploration, and NPCs

```
npcs                   — NPCs in this zone
talk <npc-id>          — speak to one; quest-givers show quests here
quest-accept <id>       — accept an offered quest
quest-turn-in <id>      — turn in a completed quest
quests                 — your active quest log
explore / atlas         — your own personal zone-discovery map
read-manual / rm         — activate a Field Manual (+100% XP for 30 min)
```

### Parties, guilds, nations

```
invite <name>           — invite to your party (you become leader)
accept                  — accept a party invite
party / pt              — party members + shared XP chain
leave-party             — leave

ls-create / ls-info / ls-invite / ls-kick / ls-leave / ls-promote  — linkshell (guild) management

declare <sandoria|bastok|windurst|neutral> — pledge allegiance to a nation
conquest / con           — check the weekly 3-nation conquest tally
```

### Dungeons and Notorious Monsters

8 real, playable dungeons, each with a real boss and elite mob roster:
**The Sealed Archive**, **The Frequency Table**, **The Remnant's Hall**, **East of the Wall**,
**The Highland Wake**, **The Founders' Table**, **The Unbound Wing**, **The Proving Grounds**.

```
dungeons               — list all 8, see which is available
dungeon-enter <N>       — travel into one
```

Notorious Monsters (NMs) are rare, tougher spawns with real, distinct spawn conditions, aggro
types, and their own treasure pool — data-driven and expanding.

### PvP (non-lethal) and status

```
duel <player>           — challenge another player to a duel (no death, no XP loss)
duel-accept / duel-forfeit
leaderboard / lb        — top duelists
who                     — who's online
say <text> / tell <name> <text> / yell <text>  — chat
sethome / home          — set/return to your Home Point when KO'd (8% XP penalty on death)
```

### The full command list

Type `help` in-game any time — it lists the core loop, economy, and quest commands above with
live in-game formatting. That in-game list is deliberately **not** exhaustive: 100+ commands
exist in total, including every job's own spells/abilities (`cast <spell>`, `ja <ability>`, and
many job-specific shortcuts like `chakra`, `berserk`, `provoke`, `trick_attack`, `chainspell`) and
an advanced faction-war layer (`district`, `enforcement`, `integrity`, `tech-pressure`,
`k9-deploy`) tied to this world's TRAPX city-simulation systems. Explore — `help` plus
experimentation will surface far more than any single doc could list without becoming unreadable.

---

## Known, honest limitations (read before reporting a "bug")

- **The GUI is not the game** — see [The GUI](#the-gui-battlegrounds_gui----real-but-limited)
  above. If something looks broken in `battlegrounds_gui`, try the same thing over telnet first.
- **No account recovery** — your character *is* your typed name. There's no password, no email
  tied to it (unless you're playing through the GUI's own separate IDUNA login). Don't rely on a
  name nobody else might also try.
- **A telnet session with no other connection reuses the same character** every time you
  reconnect with that name — there's no "log out and lose progress" here, but there's also no
  protection against someone else typing the same name first.
- Every job currently shares one character level — real, deliberate FFXI-parity design, not a bug.
- Dungeons are one shared persistent zone each right now, not per-party-instanced.

---

## Found something? Tell us. {#found-something-tell-us}

This world is under continuous, active development — new jobs, mechanics, zones, and systems
land regularly (see `CHANGELOG.md` for the real, dated history). If you're an agent (or a human)
exploring this MUD and you find something surprising, broken, funny, or genuinely clever — a
skillchain combo that hits harder than it should, a zone that feels unfinished, a Notorious
Monster with an interesting quirk, a quest that doesn't quite make sense — **we'd genuinely like
to hear about it.** Post it, reply to us, or reach out however you found this. Go play, and come
back and tell us what you found. 🦞

---

## Shankpit lineage

This work builds on **Shankpit** as a foundation — fast gameplay feel, iteration speed, and
netcode direction carried forward and extended into the broader persistent-world/backend systems
described above.

---

## Key Specs / References

- `docs2/STACK_CONTINUITY_REPORT.md` — the real, current, deep-dive picture of the whole stack
- `docs2/REDGARDEN_GUI_NORTHSTAR.md` — how the GUI/Battlegrounds relationship actually works
- `docs2/specs/WEAKNIGHT_VS0_ACCEPTANCE_CRITERIA.md`
- `docs2/specs/SHANKPIT_DRAGONSNSHIT_SYSTEMS_SPEC.md`
- `docs2/specs/THE_BRIDGE_SPEC.md`
- `docs2/NETCODE_CONTRACT_SPEC.md`
- `server/skillchain/skillchain.go` — the real, exact skillchain/magic-burst math, in code

---

## Go / Backend Notes

The repo includes Go modules and backend experiments for server-side systems and bridge work. If
working on local forked dependencies (ex: Dragonfly-based backend work), use `replace` directives
in `go.mod` during development and pin commits/tags in CI.

This repo is standalone (`GOWORK=off` — not part of the monorepo's root `go.work`):
```bash
GOWORK=off go build -o mud ./apps2/mud
```

---

## Development Philosophy (current)

- build small, test fast
- keep systems observable
- prefer server authority for shared state
- avoid locking product scope too early
- let working slices shape the next step

As things get real, plans may change. That's intentional.

---

## Branding

Umbrella name: **GoblinFoxDragon** · Persistent-world codename: **DragonsNShit**

Project names and branding beneath the umbrella may change over time as products take shape.

### Naming exploration: 鬼狐竜 (Kiko-ryū)

Possible combined Japanese rendering for the creature/archetype vibe:

- **鬼 (ki)** = demon/oni spirit energy
- **狐 (ko)** = fox
- **竜 / 龍 (ryū)** = dragon

That gives shorthand interpretations like **Demon Fox Dragon**, **Oni-Fox Dragon**, **Fiend Fox
Dragon** — compact, mythic cadence, reads like a proper named creature/class rather than a flat
literal phrase.

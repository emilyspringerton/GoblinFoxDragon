# GoblinFoxDragon

## Current Status (2026-08-26)

Real, playable: `apps2/mud` (server-authoritative MUD combat backend, real jobs/spells/aggro) +
`apps2/battlegrounds_gui` (bespoke SDL2/GL client, not the real Bedrock client) rendering Town,
Meadow, and a separate MOBA-style Battlegrounds arena mode. `apps2/server-go` serves the real
Bedrock-protocol voxel backend (`:6969`) plus a real `/heightmap` HTTP API (`:7070`, also consumed
by `WEAKNIGHT_BEDROCK_RACERS`). Recent real fixes: dead-worm visibility, KO auto-respawn + a real
Home Point Crystal, and auto-recovery of a KO'd character on the next command (not just typing
`home`) — see `CHANGELOG.md` for the full trail. See `docs/NORTHSTAR.md` for the fuller
architecture picture (dated 2026-06-14, now stale in places relative to the above).

**New since**: a real combat damage log in `battlegrounds_gui` (S189-01); the King's buff status
now syncs to the client with a bottom-right buff HUD, plus real King health bars + name tags;
Meadow (zone 0) seeded with goblin/fox NPCs generated via a real crystal-simulation pass
(S189-07); chat copy/paste bindings in `battlegrounds_gui` (`Ctrl+V` paste, same pattern as
PITVIPER); the WASM build's black-screen bug fixed (SDL GL context attributes weren't compatible
with the browser's GLSL version). The mod-surface scripting-language question (EduScript vs.
PARENA) was resolved — see `docs/MOD_SURFACE_NORTHSTAR.md`.

---

**GoblinFoxDragon** is the studio/system umbrella for a set of connected game experiments and online world projects.

This repo is where we build, test, and iterate on the shared tech stack behind those projects.

Names, scope, and boundaries may evolve as the tech and gameplay solidify.

---

## What this repo is for

This repository is the shared engineering foundation for:
- networking and server experiments
- voxel/chunk streaming
- entity and simulation systems
- bridge protocol work (client ↔ backend)
- performance and multiplayer testing
- gameplay prototypes and vertical slices

The goal is to keep iteration fast while building systems that can scale into larger experiences.

---

## Shankpit lineage (short version)

This work builds on **Shankpit** as a foundation.

We’re carrying forward pieces that work well (fast gameplay feel, iteration speed, netcode direction) and extending them into broader world/systems/backend experiments.

Shankpit is the DNA.  
GoblinFoxDragon is the broader build space.

---

## Project Notes

this plan is probably already changed

### iteration 0
WEAKNIGHT is currently used as a proving ground for:
- high-speed gameplay feel
- multiplayer testing
- world interaction/destruction experiments
- community-server-friendly features
- rapid iteration on systems

### iteration 1
- backend/server architecture
- persistent world direction
- voxel/world streaming bridges
- authoritative simulation experiments
- long-form online systems foundations

---

## Status

This is an active experimental repository.

Expect:
- changing APIs
- renamed systems
- evolving specs
- temporary stubs
- parallel prototypes

We will simplify and formalize structure as systems stabilize.

---

## Key Specs / References

- `docs2/specs/WEAKNIGHT_VS0_ACCEPTANCE_CRITERIA.md`
- `docs2/specs/SHANKPIT_DRAGONSNSHIT_SYSTEMS_SPEC.md`
- `docs2/specs/THE_BRIDGE_SPEC.md`
- `docs2/NETCODE_CONTRACT_SPEC.md`

---

## Go / Backend Notes

The repo includes Go modules and backend experiments for server-side systems and bridge work.

If working on local forked dependencies (ex: Dragonfly-based backend work), use `replace` directives in `go.mod` during development and pin commits/tags in CI.

---

## Development Philosophy (current)

- build small, test fast
- keep systems observable
- prefer server authority for shared state
- avoid locking product scope too early
- let working slices shape the next step

As things get real, plans may change. That’s intentional.

---

## Branding

Umbrella name: **GoblinFoxDragon**  
Project names and branding beneath the umbrella may change over time as products take shape.

### Naming exploration: 鬼狐竜 (Kiko-ryū)

Possible combined Japanese rendering for the creature/archetype vibe:

- **鬼 (ki)** = demon/oni spirit energy
- **狐 (ko)** = fox
- **竜 / 龍 (ryū)** = dragon

That gives shorthand interpretations like:

- **Demon Fox Dragon**
- **Oni-Fox Dragon** (strong fantasy flavor)
- **Fiend Fox Dragon** (darker boss tone)

Why it works:

- compact, mythic cadence in Japanese
- reads like a proper named creature/class instead of a flat literal phrase

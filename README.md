# GoblinFoxDragon

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

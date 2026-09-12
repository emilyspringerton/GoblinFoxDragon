// Package rng implements the "weighted marble-bag + Fibonacci pity" pull algorithm, this
// monorepo's own standing pattern for any random pick where a rare/desired outcome going
// unusually long without appearing should get progressively more likely rather than staying
// flatly (and frustratingly) improbable forever.
//
// Ported faithfully from the first real implementation of this pattern, ECOWAR/REDGARDEN's
// `arena_fibonacci`/`arena_marble_bag_pick` (packages/simulation/arena_game.c, S202-09/S202-42:
// per-hero Cart-delivery outcome selection). Same exact semantics, translated to Go: fib(0)=
// fib(1)=1 (a fresh/reset pity counter still carries its real base weight, never zeroed out --
// the textbook fib(0)=0 would do that), pity capped at MaxPityTier=10 (fib(10)=89, a real but
// bounded ceiling, not runaway growth), winner resets to 0 pity, every other outcome's pity
// increments by 1.
//
// Second real precedent for the same underlying idea, independently floated and built with
// different code: emily.cli's own promptoverse_pity.go (a binary "does the rare event trigger
// this run" escalator, Fibonacci-scaled, for Prompt-o-verse's style/subject discovery pity) --
// that one applies Fibonacci pity to an ELIGIBILITY gate ahead of a separate frequency-weighted
// draw, rather than folding pity directly into one draw's own weights the way this package (and
// its ECOWAR ancestor) does. Named here as real prior art, not duplicated: JOB_SPELL_SYSTEM_
// NORTHSTAR.md's own combat-formula section is what asked for the ECOWAR-shaped version
// specifically (a single weighted draw among a small, fixed set of discrete outcomes -- hit/
// miss, normal/critical -- which is exactly `MarbleBagPick`'s own shape).
package rng

import "math/rand"

// MaxPityTier caps how many consecutive losses actually raise an outcome's effective weight --
// fib(10) = 89, a real but bounded multiplier, not unbounded growth for an outcome that's gone
// hundreds of rolls without appearing.
const MaxPityTier = 10

// Fibonacci returns fib(n) using this package's own real convention: fib(0) = fib(1) = 1, not
// the textbook fib(0) = 0. A fresh or just-reset pity counter (n == 0) must still carry its real
// base weight -- zeroing it out would mean "just won" is treated as "impossible next," which is
// backwards for a pity mechanism.
func Fibonacci(n int) int {
	if n <= 1 {
		return 1
	}
	a, b := 1, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}

// MarbleBagPick performs one weighted pick among len(weights) outcomes, where outcome i's real
// effective weight is weights[i] * Fibonacci(min(pity[i], MaxPityTier)) -- an outcome that keeps
// losing gets a real, growing (but bounded) boost toward finally landing. Mutates pity in place:
// the picked outcome's own pity resets to 0 (it just "won," pity for it is meaningless right
// now), every other outcome's pity increments by 1 (one roll further into that outcome's own
// drought). Returns -1 if weights is empty or every effective weight is <= 0 (caller error, not
// a real pick -- every real caller in this codebase passes only positive base weights).
//
// weights and pity must be the same length; pity is the caller's own storage (per-player,
// per-mechanism, wherever pity should be scoped) so the SAME player's hit-chance pity and
// crit-chance pity, say, are two entirely separate counters passed in separately -- this
// function has no memory of its own between calls. rng is caller-supplied (same
// dependency-injected-RNG convention emily.cli's own selectStylesForSubject already uses) so
// tests can pass a seeded *rand.Rand for deterministic, reproducible rolls instead of the real
// global source.
func MarbleBagPick(weights []int, pity []int, rng *rand.Rand) int {
	n := len(weights)
	if n == 0 || len(pity) != n {
		return -1
	}
	total := 0
	effective := make([]int, n)
	for i, w := range weights {
		tier := pity[i]
		if tier > MaxPityTier {
			tier = MaxPityTier
		}
		effective[i] = w * Fibonacci(tier)
		total += effective[i]
	}
	if total <= 0 {
		return -1
	}
	roll := rng.Intn(total)
	cum := 0
	picked := n - 1 // fallback for the very top of the range (rounding safety net)
	for i, ew := range effective {
		cum += ew
		if roll < cum {
			picked = i
			break
		}
	}
	for i := range pity {
		if i == picked {
			pity[i] = 0
		} else {
			pity[i]++
		}
	}
	return picked
}

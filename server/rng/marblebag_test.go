package rng

import (
	"math/rand"
	"testing"
)

func TestFibonacci_BaseCasesAreOne(t *testing.T) {
	// This package's own real convention, not the textbook fib(0)=0: a fresh/reset pity counter
	// must still carry its real base weight.
	if got := Fibonacci(0); got != 1 {
		t.Errorf("Fibonacci(0) = %d, want 1", got)
	}
	if got := Fibonacci(1); got != 1 {
		t.Errorf("Fibonacci(1) = %d, want 1", got)
	}
}

func TestFibonacci_MatchesRealSequence(t *testing.T) {
	// Traced directly from ECOWAR's own real arena_fibonacci C code (not from that function's
	// own header comment, which claims "fib(10)=55" -- tracing the actual loop shows that value
	// is really fib(9); the real code computes fib(10)=89. A real, pre-existing off-by-one in
	// that repo's comment, not something to silently paper over here by matching the comment
	// instead of the executable algorithm this package is a faithful port of.
	want := map[int]int{2: 2, 3: 3, 4: 5, 5: 8, 6: 13, 7: 21, 8: 34, 9: 55, 10: 89}
	for n, w := range want {
		if got := Fibonacci(n); got != w {
			t.Errorf("Fibonacci(%d) = %d, want %d", n, got, w)
		}
	}
}

func TestMarbleBagPick_WinnerResetsLosersIncrement(t *testing.T) {
	// A rigged single-outcome bag always picks index 0 -- isolates the pity bookkeeping from
	// the randomness itself.
	weights := []int{1, 1}
	pity := []int{3, 3}
	rng := rand.New(rand.NewSource(1))

	// Force a pick by zeroing out outcome 1's weight so only 0 can ever win, regardless of roll.
	weights = []int{1, 0}
	picked := MarbleBagPick(weights, pity, rng)
	if picked != 0 {
		t.Fatalf("expected outcome 0 (the only real weight), got %d", picked)
	}
	if pity[0] != 0 {
		t.Errorf("winner's pity should reset to 0, got %d", pity[0])
	}
	if pity[1] != 4 {
		t.Errorf("loser's pity should increment by 1 (3 -> 4), got %d", pity[1])
	}
}

func TestMarbleBagPick_DroughtEventuallyWins(t *testing.T) {
	// The real point of the whole mechanism: a rare outcome (weight 1 vs. weight 99) that keeps
	// losing must eventually become highly likely to win, not stay flatly ~1% forever. Simulate
	// many independent 2-outcome draws where outcome 1 keeps NOT winning (by re-forcing pity up
	// manually to simulate a long real drought) and confirm the effective weight math actually
	// favors it heavily once the drought is long enough -- a direct check of the real formula,
	// not a statistical fishing expedition.
	weights := []int{99, 1}
	// Effective weights with outcome 1 in a full-tier drought: outcome0 = 99*Fibonacci(0) = 99;
	// outcome1 = 1*Fibonacci(10) = 89 (see TestFibonacci_MatchesRealSequence's own doc comment
	// on why this is 89, not the 55 ECOWAR's own header comment claims).
	// Not yet favored, but MUCH closer than the raw 99:1 -- confirm the math directly.
	rng := rand.New(rand.NewSource(1))
	losses1 := 0
	const trials = 20000
	for i := 0; i < trials; i++ {
		p := []int{0, MaxPityTier}
		picked := MarbleBagPick(weights, p, rng)
		if picked != 1 {
			losses1++
		}
	}
	// Raw odds would give outcome1 ~1% of picks (~19800 losses); with full pity it should land
	// close to 89/(99+89) = ~47.3% of picks (~10540 losses) -- a real, large, structural shift,
	// checked with generous slack for RNG variance, not an exact match.
	if losses1 > 15000 {
		t.Errorf("outcome 1 with full pity lost %d/%d trials -- expected well under raw-odds losses (~19800), pity isn't actually boosting it", losses1, trials)
	}
}

func TestMarbleBagPick_PityCappedAtMaxTier(t *testing.T) {
	// A pity value far beyond MaxPityTier must be treated identically to exactly MaxPityTier --
	// unbounded growth would eventually overflow int weight math for a long-running server.
	weights := []int{1, 1}
	rngA := rand.New(rand.NewSource(42))
	rngB := rand.New(rand.NewSource(42))

	pityAtCap := []int{0, MaxPityTier}
	pityWayOverCap := []int{0, MaxPityTier * 100}

	pickedAtCap := MarbleBagPick(weights, pityAtCap, rngA)
	pickedOverCap := MarbleBagPick(weights, pityWayOverCap, rngB)
	if pickedAtCap != pickedOverCap {
		t.Errorf("same seed, same base weights, pity at cap vs. way over cap picked differently (%d vs %d) -- capping isn't working", pickedAtCap, pickedOverCap)
	}
}

func TestMarbleBagPick_EmptyOrMismatchedReturnsNegativeOne(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	if got := MarbleBagPick(nil, nil, rng); got != -1 {
		t.Errorf("empty weights: got %d, want -1", got)
	}
	if got := MarbleBagPick([]int{1, 2}, []int{0}, rng); got != -1 {
		t.Errorf("mismatched lengths: got %d, want -1", got)
	}
}

func TestMarbleBagPick_AllZeroWeightsReturnsNegativeOne(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	if got := MarbleBagPick([]int{0, 0}, []int{0, 0}, rng); got != -1 {
		t.Errorf("all-zero weights: got %d, want -1 (caller error, not a real pick)", got)
	}
}

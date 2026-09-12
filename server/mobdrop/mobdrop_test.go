package mobdrop

import (
	"math/rand"
	"testing"
)

func TestDropsFor_RegisteredKind_ReturnsItsOwnTable(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[
		{"kind":"worm","items":[{"id":"worm-sinew","name":"Worm Sinew"},{"id":"earth-crystal","name":"Earth Crystal"}]}
	]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	drops := r.DropsFor("worm")
	if len(drops) != 2 || drops[0].ID != "worm-sinew" || drops[1].ID != "earth-crystal" {
		t.Fatalf("unexpected drops: %+v", drops)
	}
}

func TestDropsFor_CaseInsensitive(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"kind":"King Worm","items":[{"id":"king-sinew","name":"King Worm Sinew"}]}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	drops := r.DropsFor("king worm")
	if len(drops) != 1 || drops[0].ID != "king-sinew" {
		t.Fatalf("expected case-insensitive match, got: %+v", drops)
	}
}

func TestDropsFor_UnregisteredKind_FallsBackToDefaultDrop(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"kind":"worm","items":[{"id":"worm-sinew","name":"Worm Sinew"}]}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	drops := r.DropsFor("cave-bat")
	if len(drops) != 1 || drops[0] != DefaultDrop {
		t.Fatalf("expected DefaultDrop fallback, got: %+v", drops)
	}
}

func TestDropsFor_MutatingReturnedSlice_DoesNotCorruptRegistry(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"kind":"slime","items":[{"id":"slime-oil","name":"Slime Oil"}]}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	drops := r.DropsFor("slime")
	drops[0].ID = "corrupted"
	fresh := r.DropsFor("slime")
	if fresh[0].ID != "slime-oil" {
		t.Fatalf("registry was mutated via caller's slice: %+v", fresh)
	}
}

// TestEffectiveDropChance_NilNormalizesToAlways guards real backward compatibility (S412-09):
// every existing data/mob_drops.json entry was authored before DropChance existed, so its real
// absence (nil, the key never appeared in the JSON at all) must keep meaning "always drops."
func TestEffectiveDropChance_NilNormalizesToAlways(t *testing.T) {
	item := Item{ID: "worm-sinew", Name: "Worm Sinew"} // DropChance left nil
	if got := item.EffectiveDropChance(); got != 1.0 {
		t.Errorf("EffectiveDropChance(nil) = %v, want 1.0", got)
	}
}

func TestEffectiveDropChance_ExplicitZeroIsRespectedAsNeverDrops(t *testing.T) {
	// The real, deliberate reason DropChance is a pointer, not a plain float64: an operator
	// explicitly configuring 0% (e.g. temporarily disabling one item) must be honored, not
	// silently collapsed into "unset -> always" the way a plain zero value would be.
	zero := 0.0
	item := Item{ID: "disabled-item", DropChance: &zero}
	if got := item.EffectiveDropChance(); got != 0.0 {
		t.Errorf("EffectiveDropChance(explicit 0) = %v, want 0.0 (never drops)", got)
	}
}

func TestEffectiveDropChance_RealRateIsRespected(t *testing.T) {
	rate := 0.25
	item := Item{ID: "rare-gem", DropChance: &rate}
	if got := item.EffectiveDropChance(); got != 0.25 {
		t.Errorf("EffectiveDropChance(0.25) = %v, want 0.25", got)
	}
}

func TestEffectiveDropChance_OutOfRangeClampsToOne(t *testing.T) {
	// A real, malformed admin-page edit (e.g. "150" typed into a percent-shaped field) must
	// degrade to a sane bound, not silently guarantee or forbid a drop in an unintended way.
	tooHigh := 150.0
	item := Item{ID: "bad-data", DropChance: &tooHigh}
	if got := item.EffectiveDropChance(); got != 1.0 {
		t.Errorf("EffectiveDropChance(150) = %v, want clamped to 1.0", got)
	}
}

func TestEffectiveDropChance_NegativeClampsToZero(t *testing.T) {
	negative := -1.0
	item := Item{ID: "bad-data", DropChance: &negative}
	if got := item.EffectiveDropChance(); got != 0.0 {
		t.Errorf("EffectiveDropChance(-1) = %v, want clamped to 0.0", got)
	}
}

func TestRollDropsFor_HundredPercentAlwaysDrops(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"kind":"worm","items":[{"id":"worm-sinew","name":"Worm Sinew"}]}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		drops := r.RollDropsFor("worm", rng)
		if len(drops) != 1 {
			t.Fatalf("trial %d: expected the unset-DropChance item to always drop, got %+v", i, drops)
		}
	}
}

func TestRollDropsFor_ExplicitZeroPercentNeverDrops(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"kind":"worm","items":[{"id":"never","name":"Never","drop_chance":0}]}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		drops := r.RollDropsFor("worm", rng)
		if len(drops) != 0 {
			t.Fatalf("trial %d: expected an explicit drop_chance:0 item to never drop, got %+v", i, drops)
		}
	}
}

func TestRollDropsFor_PartialRateProducesAMixOverManyTrials(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"kind":"leech","items":[{"id":"leech-blood","name":"Leech Blood","drop_chance":0.5}]}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	rng := rand.New(rand.NewSource(2))
	drops, noDrops := 0, 0
	const trials = 500
	for i := 0; i < trials; i++ {
		if len(r.RollDropsFor("leech", rng)) == 1 {
			drops++
		} else {
			noDrops++
		}
	}
	if drops == 0 || noDrops == 0 {
		t.Errorf("a real 50%% drop_chance over %d trials should produce both outcomes: drops=%d noDrops=%d", trials, drops, noDrops)
	}
}

func TestRollDropsFor_NoConfiguredTableAlwaysReturnsDefaultDrop(t *testing.T) {
	r := NewRegistry() // nothing registered at all
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 20; i++ {
		drops := r.RollDropsFor("some-unconfigured-kind", rng)
		if len(drops) != 1 || drops[0].ID != DefaultDrop.ID {
			t.Fatalf("trial %d: expected the unconfigured-kind fallback to always be DefaultDrop, got %+v", i, drops)
		}
	}
}

func TestAll_ReturnsEveryRegisteredTable(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[
		{"kind":"worm","items":[{"id":"worm-sinew","name":"Worm Sinew"}]},
		{"kind":"leech","items":[{"id":"leech-blood","name":"Leech Blood"}]}
	]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	all := r.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(all))
	}
}

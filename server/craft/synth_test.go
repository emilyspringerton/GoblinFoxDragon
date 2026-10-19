package craft

import (
	"math/rand"
	"testing"

	"dragonsnshit/server/weather"
)

func testRecipe() Recipe {
	return Recipe{ID: "t", CraftType: Blacksmith, Difficulty: 30, ItemID: "blade", HQ1ItemID: "blade+1", HQ2ItemID: "blade+2", HQ3ItemID: "blade+3",
		Crystal: "fire-crystal", Ingredients: map[string]int{"ingot": 2, "silk": 1}}
}

func TestAlignmentDayAndWeather(t *testing.T) {
	cases := []struct {
		name string
		env  Env
		want int
	}{
		{"neutral", Env{Day: weather.Earth, Sky: weather.Sky{}}, 0},
		{"matching day", Env{Day: weather.Fire}, 1},
		{"opposed day", Env{Day: weather.Water}, -1},
		{"matching single weather", Env{Day: weather.Light, Sky: weather.Sky{Name: "Hot Spell", Element: weather.Fire, Power: 1}}, 1},
		{"matching double weather and day", Env{Day: weather.Fire, Sky: weather.Sky{Name: "Heat Wave", Element: weather.Fire, Power: 2}}, 3},
		{"opposed double weather", Env{Day: weather.Light, Sky: weather.Sky{Name: "Squalls", Element: weather.Water, Power: 2}}, -2},
		{"opposed day and weather", Env{Day: weather.Water, Sky: weather.Sky{Name: "Squalls", Element: weather.Water, Power: 2}}, -3},
	}
	for _, c := range cases {
		got, notes := Alignment("fire-crystal", c.env)
		if got != c.want {
			t.Errorf("%s: alignment %d, want %d", c.name, got, c.want)
		}
		if (got != 0) != (len(notes) > 0) {
			t.Errorf("%s: notes %v should explain a non-zero alignment", c.name, notes)
		}
	}
	if a, _ := Alignment("mystery-dust", Env{Day: weather.Fire}); a != 0 {
		t.Errorf("non-crystal items have no element")
	}
}

func TestAlignmentMovesSuccessAndBreakChances(t *testing.T) {
	base := SuccessChanceIn(30, 30, 0)
	if SuccessChanceIn(30, 30, 3) <= base || SuccessChanceIn(30, 30, -3) >= base {
		t.Errorf("good alignment should raise success, bad should lower it")
	}
	if BreakChanceIn(30, 30, -2) <= BreakChanceIn(30, 30, 0) {
		t.Errorf("bad alignment should raise break chance")
	}
	if BreakChanceIn(10, 30, 0) <= BreakChanceIn(40, 30, 0) {
		t.Errorf("an over-level recipe should be riskier")
	}
	if got := SuccessChanceIn(0, 90, -3); got < 0.05 {
		t.Errorf("floor violated: %v", got)
	}
	if got := SuccessChanceIn(110, 0, 3); got > 0.97 {
		t.Errorf("cap violated: %v", got)
	}
}

// Over many attempts, crafting is genuinely unreliable and the sky changes the odds.
func TestSynthesisIsNotAlwaysSuccessfulAndSkyMatters(t *testing.T) {
	r := testRecipe()
	run := func(env Env, skill float64) (succ, fail, brk, hq int) {
		rng := rand.New(rand.NewSource(11))
		for i := 0; i < 5000; i++ {
			s := Synthesize(r, skill, env, rng)
			switch s.Outcome {
			case OutcomeSuccess:
				succ++
				if s.HQTier > 0 {
					hq++
				}
			case OutcomeFail:
				fail++
			case OutcomeBreak:
				brk++
			}
		}
		return
	}
	good := Env{Day: weather.Fire, Sky: weather.Sky{Name: "Heat Wave", Element: weather.Fire, Power: 2}}
	bad := Env{Day: weather.Water, Sky: weather.Sky{Name: "Squalls", Element: weather.Water, Power: 2}}
	gs, gf, gb, gh := run(good, 40)
	bs, bf, bb, bh := run(bad, 40)
	if gf+gb == 0 || bf+bb == 0 {
		t.Fatalf("crafting must be able to fail (good fail=%d break=%d, bad fail=%d break=%d)", gf, gb, bf, bb)
	}
	if gs <= bs {
		t.Errorf("good sky should succeed more: %d vs %d", gs, bs)
	}
	if gh <= bh {
		t.Errorf("good sky should HQ more: %d vs %d", gh, bh)
	}
	if float64(gb)/float64(gf+gb) >= float64(bb)/float64(bf+bb) {
		t.Errorf("bad sky should break more of its failures: %d/%d vs %d/%d", gb, gf+gb, bb, bf+bb)
	}
}

func TestFailureLosesCrystalAndPartOfTheIngredients(t *testing.T) {
	r := testRecipe()
	rng := rand.New(rand.NewSource(5))
	sawPartial, sawAll := false, false
	for i := 0; i < 4000; i++ {
		s := Synthesize(r, 5, Env{}, rng) // far below difficulty: mostly fails
		if s.Outcome == OutcomeSuccess {
			continue
		}
		total := 0
		for _, q := range s.Lost {
			total += q
		}
		if s.Outcome == OutcomeBreak && total != 3 {
			t.Fatalf("a break must lose everything, lost %v", s.Lost)
		}
		if s.Outcome == OutcomeFail && total > 0 && total < 3 {
			sawPartial = true
		}
		if s.Outcome == OutcomeBreak {
			sawAll = true
		}
	}
	if !sawPartial || !sawAll {
		t.Errorf("expected both partial-loss failures (%v) and total-loss breaks (%v)", sawPartial, sawAll)
	}
}

func TestHQNeedsSkillSurplusAndYieldDefaults(t *testing.T) {
	r := testRecipe()
	rng := rand.New(rand.NewSource(2))
	hq := 0
	for i := 0; i < 2000; i++ {
		s := Synthesize(r, 31, Env{}, rng) // barely above difficulty: no HQ
		if s.Outcome == OutcomeSuccess && s.HQTier > 0 {
			hq++
		}
		if s.Outcome == OutcomeSuccess && s.Yield != 1 {
			t.Fatalf("yield defaults to 1, got %d", s.Yield)
		}
	}
	if hq != 0 {
		t.Errorf("no HQ without a skill surplus, got %d", hq)
	}
	hq = 0
	for i := 0; i < 2000; i++ {
		if s := Synthesize(r, 100, Env{}, rng); s.Outcome == OutcomeSuccess && s.HQTier > 0 && s.ItemID != "blade" {
			hq++
		}
	}
	if hq == 0 {
		t.Errorf("a master crafter should HQ sometimes")
	}
}

func TestSkillGainOnlyNearYourLevel(t *testing.T) {
	if SkillGainFor(-5, true) <= SkillGainFor(5, true) || SkillGainFor(5, true) <= SkillGainFor(15, true) {
		t.Errorf("harder recipes should teach more")
	}
	if SkillGainFor(40, true) != 0 {
		t.Errorf("trivial recipes teach nothing")
	}
	if SkillGainFor(-5, false) >= SkillGainFor(-5, true) {
		t.Errorf("failure teaches less")
	}
	if Rank(0) != "Amateur" || Rank(55) != "Journeyman" || Rank(110) != "Master" {
		t.Errorf("rank names: %s %s %s", Rank(0), Rank(55), Rank(110))
	}
}

func TestLoadRecipesJSONValidatesAndRegisters(t *testing.T) {
	good := `[{"id":"zz-test","name":"Z","craft":"alchemy","difficulty":4,"crystal":"water-crystal","ingredients":{"a":1},"result":"z","yield":3}]`
	if n, err := LoadRecipesJSON([]byte(good)); err != nil || n != 1 {
		t.Fatalf("load: %d %v", n, err)
	}
	r, err := LookupRecipe("zz-test")
	if err != nil || r.Crystal != "water-crystal" || r.Yield != 3 || r.Ingredients["a"] != 1 {
		t.Fatalf("lookup: %+v %v", r, err)
	}
	for name, bad := range map[string]string{
		"unknown craft":  `[{"id":"x","craft":"juggling","crystal":"fire-crystal","ingredients":{"a":1},"result":"r"}]`,
		"no crystal":     `[{"id":"x","craft":"alchemy","ingredients":{"a":1},"result":"r"}]`,
		"no ingredients": `[{"id":"x","craft":"alchemy","crystal":"fire-crystal","result":"r"}]`,
		"zero quantity":  `[{"id":"x","craft":"alchemy","crystal":"fire-crystal","ingredients":{"a":0},"result":"r"}]`,
		"missing result": `[{"id":"x","craft":"alchemy","crystal":"fire-crystal","ingredients":{"a":1}}]`,
	} {
		if _, err := LoadRecipesJSON([]byte(bad)); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
	if _, err := LookupRecipe("x"); err == nil {
		t.Errorf("a rejected file must register nothing")
	}
}

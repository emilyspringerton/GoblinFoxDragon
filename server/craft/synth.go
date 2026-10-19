package craft

import (
	"fmt"
	"math/rand"
	"sort"

	"dragonsnshit/server/weather"
)

// Synthesis rules (FFXI-style).
//
//   - Every attempt consumes the recipe's crystal.
//   - Day and weather "align" with the crystal's element: matching day +1, matching weather +power
//     (+1 single / +2 double); the element that overcomes the crystal subtracts the same. The net
//     score (-3..+3) shifts the success chance by 3% per point and the HQ roll by 5 skill points
//     per point.
//   - A failed attempt loses the crystal and each ingredient with 50% chance; it BREAKS (everything
//     lost) with a chance that rises on poor alignment and when the recipe is above your skill.
//   - Skill only rises while a recipe is at or just below your level; trivial recipes teach nothing.

// Env is the sky a synthesis happens under.
type Env struct {
	Day weather.Element
	Sky weather.Sky
}

// Outcome of a synthesis.
type Outcome int

const (
	OutcomeSuccess Outcome = iota
	OutcomeFail
	OutcomeBreak
)

func (o Outcome) String() string { return [...]string{"success", "fail", "break"}[o] }

// Synthesis is the full result of one attempt.
type Synthesis struct {
	Outcome       Outcome
	HQTier        int
	ItemID        string
	Yield         int
	Lost          map[string]int // ingredients lost on a fail/break (the crystal is always lost)
	SuccessChance float64
	BreakChance   float64
	Alignment     int
	Notes         []string
	SkillGain     float64
}

// CrystalElement derives an element from a crystal item ID ("fire-crystal", "crystal-earth", ...).
func CrystalElement(crystal string) weather.Element { return weather.ParseElement(crystal) }

// Alignment scores how the day and weather treat a crystal of the given element, with the
// human-readable reasons.
func Alignment(crystal string, env Env) (int, []string) {
	e := CrystalElement(crystal)
	if e == weather.None {
		return 0, nil
	}
	score := 0
	var notes []string
	switch {
	case env.Day == e:
		score++
		notes = append(notes, fmt.Sprintf("%s: your %s crystal resonates with the day.", weather.DayName(env.Day), e))
	case env.Day == weather.Opposed(e):
		score--
		notes = append(notes, fmt.Sprintf("%s: the day works against your %s crystal.", weather.DayName(env.Day), e))
	}
	if env.Sky.Element != weather.None {
		switch {
		case env.Sky.Element == e:
			score += env.Sky.Power
			notes = append(notes, fmt.Sprintf("%s: the weather feeds your %s crystal.", env.Sky.Name, e))
		case env.Sky.Element == weather.Opposed(e):
			score -= env.Sky.Power
			notes = append(notes, fmt.Sprintf("%s: the weather smothers your %s crystal.", env.Sky.Name, e))
		}
	}
	return score, notes
}

// SuccessChanceIn is the success probability under an environment.
func SuccessChanceIn(skill, difficulty float64, alignment int) float64 {
	p := SuccessChance(skill, difficulty) + 0.03*float64(alignment)
	if p < 0.05 {
		p = 0.05
	}
	if p > 0.97 {
		p = 0.97
	}
	return p
}

// BreakChanceIn is the probability that a FAILED attempt breaks outright.
func BreakChanceIn(skill, difficulty float64, alignment int) float64 {
	b := 0.15
	if alignment < 0 {
		b += 0.12 * float64(-alignment)
	}
	if skill < difficulty {
		b += 0.10
	}
	if b > 0.6 {
		b = 0.6
	}
	return b
}

// SkillGainFor is the skill gained from an attempt at the given gap (skill - difficulty).
func SkillGainFor(gap float64, success bool) float64 {
	var g float64
	switch {
	case gap < 0:
		g = 0.25
	case gap < 10:
		g = 0.15
	case gap < 25:
		g = 0.05
	}
	if !success {
		g *= 0.5
	}
	return g
}

// Synthesize resolves one attempt of r by a crafter with the given skill.
func Synthesize(r Recipe, skill float64, env Env, rng *rand.Rand) Synthesis {
	gap := skill - r.Difficulty
	align, notes := Alignment(r.Crystal, env)
	s := Synthesis{
		Alignment:     align,
		Notes:         notes,
		SuccessChance: SuccessChanceIn(skill, r.Difficulty, align),
		BreakChance:   BreakChanceIn(skill, r.Difficulty, align),
		Yield:         r.Yield,
	}
	if s.Yield < 1 {
		s.Yield = 1
	}
	if rng.Float64() < s.SuccessChance {
		s.Outcome = OutcomeSuccess
		s.HQTier = HQTier(gap+5*float64(align), rng.Float64())
		s.ItemID = r.ItemID
		switch s.HQTier {
		case 1:
			if r.HQ1ItemID != "" {
				s.ItemID = r.HQ1ItemID
			}
		case 2:
			if r.HQ2ItemID != "" {
				s.ItemID = r.HQ2ItemID
			}
		case 3:
			if r.HQ3ItemID != "" {
				s.ItemID = r.HQ3ItemID
			}
		}
		s.SkillGain = SkillGainFor(gap, true)
		return s
	}
	s.Lost = map[string]int{}
	ids := make([]string, 0, len(r.Ingredients))
	for id := range r.Ingredients {
		ids = append(ids, id)
	}
	sort.Strings(ids) // deterministic roll order
	if rng.Float64() < s.BreakChance {
		s.Outcome = OutcomeBreak
		for _, id := range ids {
			s.Lost[id] = r.Ingredients[id]
		}
	} else {
		s.Outcome = OutcomeFail
		for _, id := range ids {
			if rng.Float64() < 0.5 {
				s.Lost[id] = r.Ingredients[id]
			}
		}
	}
	s.SkillGain = SkillGainFor(gap, false)
	return s
}

// Rank names a skill level the way FFXI does.
func Rank(skill float64) string {
	ranks := []string{"Amateur", "Recruit", "Initiate", "Novice", "Apprentice", "Journeyman", "Craftsman", "Artisan", "Adept", "Veteran", "Expert", "Master"}
	i := int(skill / 10)
	if i >= len(ranks) {
		i = len(ranks) - 1
	}
	return ranks[i]
}

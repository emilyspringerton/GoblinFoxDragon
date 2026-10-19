package main

// craft_mud.go -- crystal synthesis for the MUD.
//
// Rules (server/craft): every attempt consumes the recipe's crystal; the day's element and the
// zone's weather help or hurt that crystal; failures lose some or all of the ingredients; skill only
// grows near your level. Recipes come from data/recipes.json (141 across the eight guilds) plus the
// few legacy code recipes whose ingredients live in recipeIngredients.

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"dragonsnshit/server/craft"
	"dragonsnshit/server/weather"
)

// craftEnvAt is the sky a synthesis in zoneID happens under at time t.
func craftEnvAt(zoneID int, t time.Time) craft.Env {
	return craft.Env{Day: weather.DayElement(t), Sky: weather.ZoneSky(zoneID, t)}
}

func skyString(s weather.Sky) string {
	if s.Element == weather.None {
		return "Clear"
	}
	strength := ""
	if s.Power >= 2 {
		strength = " x2"
	}
	return fmt.Sprintf("%s (%s%s)", s.Name, s.Element, strength)
}

// recipeIngredientsOf returns what a recipe consumes besides its crystal.
func recipeIngredientsOf(r craft.Recipe) map[string]int {
	if len(r.Ingredients) > 0 {
		return r.Ingredients
	}
	return recipeIngredients[r.ID]
}

func recipeLabel(r craft.Recipe) string {
	if r.Name != "" {
		return r.Name
	}
	return itemName(r.ItemID)
}

// canonicalItemID folds the ids the gathering code emits ("ore-iron", "crystal-earth") onto the
// ids recipes, quests and shops use ("iron-ore", "earth-crystal"). Before this, mined crystals and
// ore could never be used in any recipe or quest.
func canonicalItemID(id string) string {
	if c, ok := itemAliases[id]; ok {
		return c
	}
	return id
}

var itemAliases = map[string]string{
	"ore-iron": "iron-ore", "ore-copper": "copper-ore", "ore-tin": "tin-ore", "ore-silver": "silver-ore",
	"ore-gold": "gold-ore", "ore-swampite": "swampite", "mat-clay": "clay", "mat-peat": "peat",
	"crystal-earth": "earth-crystal", "crystal-water": "water-crystal", "crystal-fire": "fire-crystal",
	"crystal-wind": "wind-crystal", "crystal-ice": "ice-crystal", "crystal-thunder": "thunder-crystal",
	"crystal-light": "light-crystal", "crystal-dark": "dark-crystal",
}

func cmdRecipes(p *player, args []string) {
	now := time.Now()
	env := craftEnvAt(p.zoneID, now)
	if len(args) == 0 {
		p.send("\r\n=== Guilds ===")
		for _, ct := range craft.AllCrafts {
			lvl, _ := p.craftSkill.Level(ct)
			n, ready := 0, 0
			for _, r := range craft.RecipesFor(ct) {
				n++
				if ok, _ := canSynthesize(p, r); ok {
					ready++
				}
			}
			p.sendf("  %-13s %-10s %5.1f   %3d recipes, %d ready", ct, craft.Rank(lvl), lvl, n, ready)
		}
		p.send("Type 'recipes <guild>' (or 'recipes ready') for the list, 'recipe <id>' for details, 'craft <id>' to synthesize, 'time' for today's sky.")
		p.prompt()
		return
	}
	which := strings.ToLower(args[0])
	var list []craft.Recipe
	switch which {
	case "ready", "craftable":
		for _, r := range craft.AllRecipes() {
			if ok, _ := canSynthesize(p, r); ok {
				list = append(list, r)
			}
		}
	case "all":
		list = craft.AllRecipes()
	default:
		list = craft.RecipesFor(which)
		if len(list) == 0 {
			p.sendf("No guild or filter %q. Guilds: %s", which, strings.Join(craft.AllCrafts, ", "))
			p.prompt()
			return
		}
	}
	if len(list) == 0 {
		p.send("Nothing you can make right now -- check 'recipes <guild>' for what's missing.")
		p.prompt()
		return
	}
	p.sendf("\r\n=== Recipes (%d) -- sky: %s today, %s ===", len(list), weather.DayName(env.Day), skyString(env.Sky))
	for _, r := range list {
		lvl, _ := p.craftSkill.Level(r.CraftType)
		align, _ := craft.Alignment(r.Crystal, env)
		pct := craft.SuccessChanceIn(lvl, r.Difficulty, align) * 100
		ok, _ := canSynthesize(p, r)
		mark := "  "
		if ok {
			mark = "* "
		}
		sky := ""
		if align > 0 {
			sky = fmt.Sprintf(" (+%d sky)", align)
		} else if align < 0 {
			sky = fmt.Sprintf(" (%d sky)", align)
		}
		p.sendf("%s%-30s %-12s lv%-3.0f %3.0f%%%s", mark, r.ID, r.CraftType, r.Difficulty, pct, sky)
	}
	p.send("(* = you hold the crystal and ingredients)")
	p.prompt()
}

// canSynthesize reports whether p holds the crystal and every ingredient, and what is missing.
func canSynthesize(p *player, r craft.Recipe) (bool, []string) {
	var missing []string
	if r.Crystal != "" && p.inventory[r.Crystal] < 1 {
		missing = append(missing, "1x "+itemName(r.Crystal))
	}
	ings := recipeIngredientsOf(r)
	ids := make([]string, 0, len(ings))
	for id := range ings {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if have := p.inventory[id]; have < ings[id] {
			missing = append(missing, fmt.Sprintf("%dx %s (have %d)", ings[id], itemName(id), have))
		}
	}
	return len(missing) == 0, missing
}

func cmdRecipeDetail(p *player, id string) {
	r, err := craft.LookupRecipe(id)
	if err != nil {
		p.sendf("Unknown recipe %q. Use 'recipes' to browse.", id)
		p.prompt()
		return
	}
	env := craftEnvAt(p.zoneID, time.Now())
	lvl, _ := p.craftSkill.Level(r.CraftType)
	align, notes := craft.Alignment(r.Crystal, env)
	p.sendf("\r\n=== %s ===", recipeLabel(r))
	p.sendf("  Guild:       %s (difficulty %.0f; you are %s %.1f)", r.CraftType, r.Difficulty, craft.Rank(lvl), lvl)
	if r.Crystal != "" {
		p.sendf("  Crystal:     %s (consumed every attempt)", itemName(r.Crystal))
	}
	ings := recipeIngredientsOf(r)
	ids := make([]string, 0, len(ings))
	for k := range ings {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	for _, k := range ids {
		have := p.inventory[k]
		flag := ""
		if have < ings[k] {
			flag = "  <-- missing"
		}
		p.sendf("  Ingredient:  %dx %s (have %d)%s", ings[k], itemName(k), have, flag)
	}
	y := ""
	if r.Yield > 1 {
		y = fmt.Sprintf(" x%d", r.Yield)
	}
	p.sendf("  Makes:       %s%s", itemName(r.ItemID), y)
	if r.HQ1ItemID != "" && r.HQ1ItemID != r.ItemID {
		p.sendf("  High quality: %s", itemName(r.HQ1ItemID))
	}
	p.sendf("  Success now: %.0f%%  (break risk on failure: %.0f%%)", craft.SuccessChanceIn(lvl, r.Difficulty, align)*100, craft.BreakChanceIn(lvl, r.Difficulty, align)*100)
	for _, n := range notes {
		p.send("  " + n)
	}
	if align == 0 && r.Crystal != "" {
		p.sendf("  The sky (%s, %s) is neutral for a %s.", weather.DayName(env.Day), skyString(env.Sky), itemName(r.Crystal))
	}
	p.prompt()
}

func cmdCraftSkills(p *player) {
	p.send("\r\n=== Craft Skills ===")
	for _, ct := range craft.AllCrafts {
		lvl, _ := p.craftSkill.Level(ct)
		p.sendf("  %-13s %-10s %5.1f / %.0f", ct, craft.Rank(lvl), lvl, craft.SkillCap)
	}
	p.prompt()
}

func cmdTime(p *player) {
	now := time.Now()
	env := craftEnvAt(p.zoneID, now)
	p.sendf("\r\n=== Time & Sky ===")
	p.sendf("  Today is %s. The day turns in %s.", weather.DayName(env.Day), weather.NextDayChange(now).Sub(now).Round(time.Minute))
	p.sendf("  Sky over %s: %s.", zoneName(p.zoneID), skyString(env.Sky))
	p.send("  Crystals under this sky (crafting):")
	for _, e := range weather.AllElements {
		cr := string(e) + "-crystal"
		if e == weather.Lightning {
			cr = "thunder-crystal"
		}
		a, _ := craft.Alignment(cr, env)
		var verdict string
		switch {
		case a >= 2:
			verdict = "excellent"
		case a == 1:
			verdict = "favoured"
		case a == 0:
			verdict = "neutral"
		case a == -1:
			verdict = "poor"
		default:
			verdict = "dangerous"
		}
		p.sendf("    %-16s %+d  %s", itemName(cr), a, verdict)
	}
	p.prompt()
}

// cmdCraft synthesizes recipeID once (or count times, stopping early if you run out).
func cmdCraft(p *player, recipeID string, count int) {
	if p.homePoint.IsKO {
		p.send("You are KO'd.")
		p.prompt()
		return
	}
	if count < 1 {
		count = 1
	}
	if count > 12 {
		count = 12
	}
	r, err := craft.LookupRecipe(recipeID)
	if err != nil {
		p.sendf("Unknown recipe %q. Use 'recipes' to browse.", recipeID)
		p.prompt()
		return
	}
	for i := 0; i < count; i++ {
		if ok, missing := canSynthesize(p, r); !ok {
			if i == 0 {
				p.sendf("You can't make %s yet. Missing: %s", recipeLabel(r), strings.Join(missing, ", "))
			} else {
				p.sendf("You run out of materials after %d attempt(s).", i)
			}
			break
		}
		synthesizeOnce(p, r)
	}
	p.prompt()
}

func removeItems(p *player, items map[string]int) {
	for id, q := range items {
		p.inventory[id] -= q
		if p.inventory[id] <= 0 {
			delete(p.inventory, id)
		}
	}
}

func synthesizeOnce(p *player, r craft.Recipe) {
	now := time.Now()
	env := craftEnvAt(p.zoneID, now)
	skill, _ := p.craftSkill.Level(r.CraftType)
	if r.Crystal != "" {
		removeItems(p, map[string]int{r.Crystal: 1}) // the crystal is spent no matter what
	}
	res := craft.Synthesize(r, skill, env, gw.rng)
	for _, n := range res.Notes {
		p.send("  " + n)
	}
	ings := recipeIngredientsOf(r)
	switch res.Outcome {
	case craft.OutcomeSuccess:
		removeItems(p, ings)
		p.inventory[res.ItemID] += res.Yield
		qty := ""
		if res.Yield > 1 {
			qty = fmt.Sprintf(" x%d", res.Yield)
		}
		if res.HQTier > 0 {
			p.sendf(">>> HQ%d! <<< You synthesize %s%s!", res.HQTier, itemName(res.ItemID), qty)
		} else {
			p.sendf("You synthesize %s%s.", itemName(r.ItemID), qty)
		}
	case craft.OutcomeFail:
		removeItems(p, res.Lost)
		p.sendf("The synthesis fails. The %s is consumed%s.", itemName(r.Crystal), lostSummary(res.Lost))
	case craft.OutcomeBreak:
		removeItems(p, ings)
		p.send(">>> BREAK! <<< The synthesis collapses -- the crystal and every ingredient are lost.")
	}
	if res.SkillGain > 0 && skill < craft.SkillCap {
		_ = p.craftSkill.SetLevel(r.CraftType, skill+res.SkillGain)
		now2, _ := p.craftSkill.Level(r.CraftType)
		if int(now2/10) > int(skill/10) {
			p.sendf("*** Your %s skill reaches %s (%.1f)! ***", r.CraftType, craft.Rank(now2), now2)
		}
	}
}

func lostSummary(lost map[string]int) string {
	if len(lost) == 0 {
		return ", but your ingredients survive"
	}
	ids := make([]string, 0, len(lost))
	for id := range lost {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var parts []string
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%dx %s", lost[id], itemName(id)))
	}
	return " along with " + strings.Join(parts, ", ")
}

// ---- persistence of craft skills (same delta-sync pattern as mining/fishing) ----

func craftSkillName(ct craft.CraftType) string { return "craft_" + ct }

// loadCraftSkills restores crafting levels from the stored skill map and baselines the sync.
func loadCraftSkills(p *player, stored map[string]float64) {
	if p.syncedCraft == nil {
		p.syncedCraft = map[string]float64{}
	}
	for _, ct := range craft.AllCrafts {
		v := stored[craftSkillName(ct)]
		if v > craft.SkillCap {
			v = craft.SkillCap
		}
		_ = p.craftSkill.SetLevel(ct, v)
		p.syncedCraft[ct] = v
	}
}

// syncCraftSkills sends each craft's gain since the last successful save to IDUNA.
func syncCraftSkills(p *player, charID string) {
	if p.craftSkill == nil {
		return
	}
	if p.syncedCraft == nil {
		p.syncedCraft = map[string]float64{}
	}
	for _, ct := range craft.AllCrafts {
		cur, _ := p.craftSkill.Level(ct)
		delta := cur - p.syncedCraft[ct]
		if delta <= 0 {
			continue
		}
		if err := gw.iduna.IncrementSkill(charID, craftSkillName(ct), delta); err == nil {
			p.syncedCraft[ct] += delta
		}
	}
}

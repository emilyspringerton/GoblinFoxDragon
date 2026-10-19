package craft

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// jsonRecipe is the on-disk shape of data/recipes.json.
type jsonRecipe struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Craft       string         `json:"craft"`
	Difficulty  float64        `json:"difficulty"`
	Crystal     string         `json:"crystal"`
	Ingredients map[string]int `json:"ingredients"`
	Result      string         `json:"result"`
	HQ1         string         `json:"hq1,omitempty"`
	HQ2         string         `json:"hq2,omitempty"`
	HQ3         string         `json:"hq3,omitempty"`
	Yield       int            `json:"yield,omitempty"`
}

// LoadRecipesFile registers every recipe in a recipes.json file, replacing any recipe with the same
// ID. It is safe to call again to hot-reload. Invalid rows fail the whole load (nothing is
// registered), so a typo can never leave a half-applied recipe book.
func LoadRecipesFile(path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return LoadRecipesJSON(raw)
}

// LoadRecipesJSON is LoadRecipesFile for in-memory data.
func LoadRecipesJSON(raw []byte) (int, error) {
	var rows []jsonRecipe
	if err := json.Unmarshal(raw, &rows); err != nil {
		return 0, fmt.Errorf("craft: recipes json: %w", err)
	}
	valid := map[CraftType]bool{}
	for _, c := range AllCrafts {
		valid[c] = true
	}
	recipes := make([]Recipe, 0, len(rows))
	for i, r := range rows {
		switch {
		case r.ID == "" || r.Result == "":
			return 0, fmt.Errorf("craft: recipe #%d: id and result are required", i)
		case !valid[r.Craft]:
			return 0, fmt.Errorf("craft: recipe %q: unknown craft %q", r.ID, r.Craft)
		case r.Crystal == "":
			return 0, fmt.Errorf("craft: recipe %q: a crystal is required", r.ID)
		case len(r.Ingredients) == 0:
			return 0, fmt.Errorf("craft: recipe %q: needs ingredients", r.ID)
		}
		for ing, q := range r.Ingredients {
			if q < 1 {
				return 0, fmt.Errorf("craft: recipe %q: ingredient %q quantity %d", r.ID, ing, q)
			}
		}
		recipes = append(recipes, Recipe{
			ID: r.ID, Name: r.Name, CraftType: r.Craft, Difficulty: r.Difficulty,
			ItemID: r.Result, HQ1ItemID: r.HQ1, HQ2ItemID: r.HQ2, HQ3ItemID: r.HQ3,
			Crystal: r.Crystal, Ingredients: r.Ingredients, Yield: r.Yield,
		})
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	for _, r := range recipes {
		registry[r.ID] = r
	}
	return len(recipes), nil
}

// AllRecipes returns every registered recipe sorted by craft, then difficulty, then ID.
func AllRecipes() []Recipe {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Recipe, 0, len(registry))
	for _, r := range registry {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CraftType != out[j].CraftType {
			return out[i].CraftType < out[j].CraftType
		}
		if out[i].Difficulty != out[j].Difficulty {
			return out[i].Difficulty < out[j].Difficulty
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// RecipesFor returns the recipes of one craft, easiest first.
func RecipesFor(c CraftType) []Recipe {
	var out []Recipe
	for _, r := range AllRecipes() {
		if r.CraftType == c {
			out = append(out, r)
		}
	}
	return out
}

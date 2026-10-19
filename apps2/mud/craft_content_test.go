package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"dragonsnshit/server/craft"
	"dragonsnshit/server/gear"
	"dragonsnshit/server/itemdef"
	"dragonsnshit/server/job"
)

type rawItem struct {
	ID    int            `json:"id"`
	Name  string         `json:"name"`
	Cat   string         `json:"category"`
	Slots []string       `json:"equip_slots"`
	Jobs  []string       `json:"jobs"`
	Stats map[string]int `json:"stats"`
}

func loadRawItems(t *testing.T) []rawItem {
	t.Helper()
	b, err := os.ReadFile("../../data/items.json")
	if err != nil {
		t.Fatal(err)
	}
	var items []rawItem
	if err := json.Unmarshal(b, &items); err != nil {
		t.Fatal(err)
	}
	return items
}

func slugOf(name string) string { return strings.ReplaceAll(strings.ToLower(name), " ", "-") }

// obtainable collects every id a player can actually come by.
func obtainable(t *testing.T) map[string]bool {
	t.Helper()
	have := map[string]bool{}
	// mined / fished (the ids the gather code emits, folded to canonical ids)
	for _, id := range []string{"ore-iron", "ore-copper", "ore-tin", "ore-silver", "ore-gold", "ore-swampite", "mat-clay", "mat-peat",
		"crystal-earth", "crystal-water"} {
		have[canonicalItemID(id)] = true
	}
	for _, id := range []string{"fish-carp", "fish-bass", "fish-trout", "fish-eel", "fish-crab", "fish-fullmoon-sardine", "fish-trophy-carp",
		"fish-trophy-bass", "fish-giant-squid", "fish-mud-crab", "fish-bog-eel", "fish-crimson-trout"} {
		have[id] = true
	}
	// mob drops
	b, err := os.ReadFile("../../data/mob_drops.json")
	if err != nil {
		t.Fatal(err)
	}
	var drops []struct {
		Items []struct{ ID string } `json:"items"`
	}
	_ = json.Unmarshal(b, &drops)
	for _, d := range drops {
		for _, it := range d.Items {
			have[it.ID] = true
		}
	}
	// vendors
	for _, stock := range npcVendorCatalog {
		for _, vi := range stock {
			have[vi.ID] = true
		}
	}
	// recipe outputs (crafted intermediates)
	for _, r := range craft.AllRecipes() {
		have[r.ItemID] = true
		if r.Yield > 0 {
			have[r.ItemID] = true
		}
	}
	return have
}

func TestRecipeBookIsCompleteAndMakeable(t *testing.T) {
	if _, err := craft.LoadRecipesFile("../../data/recipes.json"); err != nil {
		t.Fatal(err)
	}
	items := loadRawItems(t)
	bySlug := map[string]rawItem{}
	for _, it := range items {
		bySlug[slugOf(it.Name)] = it
	}
	have := obtainable(t)
	all := craft.AllRecipes()
	if len(all) < 100 {
		t.Fatalf("expected a big recipe book, got %d", len(all))
	}
	perCraft := map[string]int{}
	for _, r := range all {
		if r.Crystal == "" && len(r.Ingredients) > 0 {
			t.Errorf("%s: data recipes need a crystal", r.ID)
		}
		if len(r.Ingredients) == 0 {
			continue // legacy code recipe
		}
		perCraft[r.CraftType]++
		if _, ok := bySlug[r.ItemID]; !ok {
			t.Errorf("%s: result %q is not an item", r.ID, r.ItemID)
		}
		for _, hq := range []string{r.HQ1ItemID, r.HQ2ItemID, r.HQ3ItemID} {
			if hq != "" {
				if _, ok := bySlug[hq]; !ok {
					t.Errorf("%s: HQ result %q is not an item", r.ID, hq)
				}
			}
		}
		if !have[r.Crystal] {
			t.Errorf("%s: crystal %q cannot be obtained (no vendor, drop or mine)", r.ID, r.Crystal)
		}
		if _, ok := bySlug[r.Crystal]; !ok {
			t.Errorf("%s: crystal %q is not an item", r.ID, r.Crystal)
		}
		for ing := range r.Ingredients {
			if !have[ing] {
				t.Errorf("%s: ingredient %q cannot be obtained", r.ID, ing)
			}
			if _, ok := bySlug[ing]; !ok && !strings.HasPrefix(ing, "fish-") {
				t.Errorf("%s: ingredient %q is not a named item", r.ID, ing)
			}
		}
	}
	for _, ct := range craft.AllCrafts {
		if perCraft[ct] < 8 {
			t.Errorf("guild %s has only %d recipes", ct, perCraft[ct])
		}
	}
}

func TestRecipeDifficultiesProgress(t *testing.T) {
	craft.LoadRecipesFile("../../data/recipes.json")
	for _, ct := range craft.AllCrafts {
		rs := craft.RecipesFor(ct)
		if len(rs) == 0 {
			t.Errorf("%s: no recipes", ct)
			continue
		}
		if rs[0].Difficulty > 20 {
			t.Errorf("%s: the first recipe (%s, %.0f) should be learnable by a beginner", ct, rs[0].ID, rs[0].Difficulty)
		}
	}
}

var knownStats = map[string]bool{
	"str": true, "dex": true, "vit": true, "agi": true, "int": true, "mnd": true, "chr": true, "attack": true, "accuracy": true,
	"defense": true, "evasion": true, "crit_rate": true, "haste": true, "magic_attack_bonus": true, "magic_defense_bonus": true,
	"hp": true, "mp": true, "hp_regen": true, "mp_regen": true, "fast_cast": true, "enmity": true, "store_tp": true,
	"resist_sleep": true, "ranged_attack": true, "ranged_accuracy": true, "item_level": true,
}

// Every stat on the new gear must be one stats.go actually consumes -- no decorative numbers.
func TestNewGearOnlyUsesImplementedStats(t *testing.T) {
	validJob := map[string]bool{}
	for _, j := range job.AllJobs {
		validJob[string(j)] = true
	}
	validSlot := map[string]bool{}
	for _, sl := range gear.AllSlots {
		validSlot[sl] = true
	}
	n := 0
	for _, it := range loadRawItems(t) {
		if it.ID < 1100 || it.ID >= 3000 || it.Cat == "material" {
			continue
		}
		n++
		for k := range it.Stats {
			if !knownStats[normalizeStatKey(k)] && !strings.HasPrefix(normalizeStatKey(k), "e_") {
				t.Errorf("%s: stat %q is not implemented", it.Name, k)
			}
		}
		for _, j := range it.Jobs {
			if !validJob[j] {
				t.Errorf("%s: unknown job %q", it.Name, j)
			}
		}
		for _, s := range it.Slots {
			if !validSlot[s] {
				t.Errorf("%s: unknown slot %q", it.Name, s)
			}
		}
		if (it.Cat == "weapon" || it.Cat == "armor" || it.Cat == "accessory") && len(it.Slots) == 0 {
			t.Errorf("%s: equipment needs a slot", it.Name)
		}
	}
	if n < 150 {
		t.Errorf("expected lots of gear, checked %d", n)
	}
}

func TestEquipmentResolvesByNameAndHQIsBetter(t *testing.T) {
	reg := itemdef.NewRegistry()
	if err := reg.LoadFile("../../data/items.json"); err != nil {
		t.Fatal(err)
	}
	base, ok1 := reg.ByName("shell-steel-cutlass")
	hq, ok2 := reg.ByName("shell-steel-cutlass-+1")
	if !ok1 || !ok2 {
		t.Fatalf("cutlass and its +1 must resolve by slug (%v %v)", ok1, ok2)
	}
	if hq.Stats["attack"] <= base.Stats["attack"] {
		t.Errorf("HQ should out-hit NQ: %d vs %d", hq.Stats["attack"], base.Stats["attack"])
	}
	if base.WeaponType != "sword" || base.Delay == 0 {
		t.Errorf("weapon needs type and delay: %+v", base)
	}
}

func TestConsumablesAndFoodsAreWiredToRealItems(t *testing.T) {
	items := loadRawItems(t)
	slugs := map[string]bool{}
	for _, it := range items {
		slugs[slugOf(it.Name)] = true
	}
	for id := range itemKits {
		if !slugs[id] {
			t.Errorf("itemKits %q has no item row", id)
		}
	}
	if len(loreFoods) < 10 {
		t.Errorf("expected a cookbook, got %d dishes", len(loreFoods))
	}
	for _, f := range loreFoods {
		if !slugs[f.ID] {
			t.Errorf("food %q has no item row", f.ID)
		}
	}
	for npc, stock := range npcVendorCatalog {
		for _, vi := range stock {
			if !slugs[vi.ID] && itemDisplayName[vi.ID] == "" {
				t.Errorf("vendor %s sells %q, which resolves nowhere", npc, vi.ID)
			}
		}
	}
}

func TestCanonicalIDsFoldGatheringOntoRecipeIDs(t *testing.T) {
	if canonicalItemID("crystal-earth") != "earth-crystal" || canonicalItemID("ore-iron") != "iron-ore" || canonicalItemID("fish-carp") != "fish-carp" {
		t.Errorf("canonicalItemID wrong")
	}
}

// Every item in the whole database must be equippable into a slot the engine really has. Older rows
// used "ringl"/"earl" while gear uses "ring-l"/"ear-l", which silently made them unequippable.
func TestEveryItemUsesARealEquipSlot(t *testing.T) {
	valid := map[string]bool{}
	for _, sl := range gear.AllSlots {
		valid[sl] = true
	}
	for _, it := range loadRawItems(t) {
		for _, sl := range it.Slots {
			if !valid[sl] {
				t.Errorf("%s (id %d): slot %q is not a real equipment slot", it.Name, it.ID, sl)
			}
		}
	}
}

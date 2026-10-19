package main

import (
	"bufio"
	"bytes"
	"math/rand"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dragonsnshit/server/craft"
	"dragonsnshit/server/food"
	"dragonsnshit/server/homepoint"
	"dragonsnshit/server/idunaclient"
	"dragonsnshit/server/mob"
	"dragonsnshit/server/status"
)

func craftWorld(t *testing.T) (*player, *bytes.Buffer) {
	t.Helper()
	if _, err := craft.LoadRecipesFile("../../data/recipes.json"); err != nil {
		t.Fatal(err)
	}
	if err := itemdefReg.LoadFile("../../data/items.json"); err != nil {
		t.Fatal(err)
	}
	reg := mob.New()
	oldGW := gw
	gw = &world{rng: rand.New(rand.NewSource(9)), foodReg: food.NewRegistry(append(food.StandardFoods(), loreFoods...)),
		mobRegs: map[int]*mob.Registry{1: reg}, charIDBySlot: map[string]string{}, players: map[string]*player{}}
	t.Cleanup(func() { gw = oldGW })
	p := newTestPlayer()
	p.slot, p.zoneID = "crafter", 1
	p.homePoint = homepoint.NewState(1)
	p.combat = &mob.PlayerCombat{}
	var buf bytes.Buffer
	p.w = bufio.NewWriter(&buf)
	p.craftSkill = craft.NewCraftSkill()
	p.statFX = status.New()
	p.charJob = nil
	gw.players[p.slot] = p
	return p, &buf
}

func TestCraftMissingCrystalOrIngredientsDoesNothing(t *testing.T) {
	p, buf := craftWorld(t)
	p.inventory["copper-ore"] = 2
	cmdCraft(p, "copper-ingot", 1)
	p.w.Flush()
	if !strings.Contains(buf.String(), "Fire Crystal") || p.inventory["copper-ore"] != 2 {
		t.Errorf("without a crystal nothing happens and nothing is lost:\n%s", buf.String())
	}
	buf.Reset()
	p.inventory["fire-crystal"] = 1
	p.inventory["copper-ore"] = 1
	cmdCraft(p, "copper-ingot", 1)
	p.w.Flush()
	if !strings.Contains(buf.String(), "Copper Ore") || p.inventory["fire-crystal"] != 1 {
		t.Errorf("missing ingredient message/rollback wrong:\n%s", buf.String())
	}
}

// Whatever the dice say, the crystal is spent, a success pays out exactly the recipe, and a failure
// never produces an item.
func TestCraftAlwaysSpendsTheCrystalAndAccountsForEverything(t *testing.T) {
	p, _ := craftWorld(t)
	var succ, fail, brk int
	for i := 0; i < 300; i++ {
		p.inventory["fire-crystal"] = 1
		p.inventory["copper-ore"] = 2
		delete(p.inventory, "copper-ingot")
		cmdCraft(p, "copper-ingot", 1)
		if p.inventory["fire-crystal"] != 0 {
			t.Fatalf("attempt %d: the crystal must always be consumed", i)
		}
		switch ingots := p.inventory["copper-ingot"]; {
		case ingots == 1:
			succ++
			if p.inventory["copper-ore"] != 0 {
				t.Fatalf("success must consume the ore, %d left", p.inventory["copper-ore"])
			}
		case ingots == 0:
			if p.inventory["copper-ore"] == 2 {
				fail++ // lucky failure: ore survived
			} else {
				brk++
			}
		default:
			t.Fatalf("copper ingot count %d", ingots)
		}
	}
	if succ == 0 || fail+brk == 0 {
		t.Errorf("crafting must be able to both succeed (%d) and fail (%d fail/%d lossy)", succ, fail, brk)
	}
}

func TestCraftSkillRisesNearLevelAndShowsRank(t *testing.T) {
	p, buf := craftWorld(t)
	_ = p.craftSkill.SetLevel(craft.Blacksmith, 5)
	start, _ := p.craftSkill.Level(craft.Blacksmith)
	for i := 0; i < 200; i++ {
		p.inventory["fire-crystal"], p.inventory["copper-ore"] = 1, 2
		cmdCraft(p, "copper-ingot", 1) // difficulty 1, skill 5: a trivial-ish recipe
	}
	end, _ := p.craftSkill.Level(craft.Blacksmith)
	if end <= start {
		t.Errorf("skill should rise from near-level practice (%v -> %v)", start, end)
	}
	_ = p.craftSkill.SetLevel(craft.Blacksmith, 60)
	before := end
	_ = before
	cmdCraftSkills(p)
	p.w.Flush()
	if !strings.Contains(buf.String(), "Craftsman") {
		t.Errorf("rank name missing:\n%s", buf.String())
	}
	_ = p.craftSkill.SetLevel(craft.Blacksmith, 100)
	for i := 0; i < 50; i++ {
		p.inventory["fire-crystal"], p.inventory["copper-ore"] = 1, 2
		cmdCraft(p, "copper-ingot", 1)
	}
	if got, _ := p.craftSkill.Level(craft.Blacksmith); got != 100 {
		t.Errorf("a trivial recipe teaches nothing at skill 100, skill is %v", got)
	}
}

func TestCraftCountAndBrowserCommands(t *testing.T) {
	p, buf := craftWorld(t)
	_ = p.craftSkill.SetLevel(craft.Blacksmith, 40)
	p.inventory["fire-crystal"] = 3
	p.inventory["copper-ore"] = 6
	cmdCraft(p, "copper-ingot", 3)
	p.w.Flush()
	if p.inventory["fire-crystal"] != 0 {
		t.Errorf("3 attempts use 3 crystals, %d left", p.inventory["fire-crystal"])
	}
	buf.Reset()
	cmdRecipes(p, nil)
	cmdRecipes(p, []string{"blacksmith"})
	cmdRecipeDetail(p, "shell-steel-cutlass")
	cmdTime(p)
	p.w.Flush()
	out := buf.String()
	for _, want := range []string{"Guilds", "blacksmith", "copper-ingot", "Shell-Steel Cutlass", "Crystal:", "Success now", "Time & Sky", "Today is", "Fire Crystal"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
	buf.Reset()
	cmdCraft(p, "no-such-recipe", 1)
	p.w.Flush()
	if !strings.Contains(buf.String(), "Unknown recipe") {
		t.Errorf("unknown recipe message")
	}
}

func TestCraftSkillsPersistAcrossLogout(t *testing.T) {
	fake := &fakeSkillsIduna{skills: map[string]float64{}}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	t.Setenv("IDUNA_BASE_URL", srv.URL)
	t.Setenv("IDUNA_AGENT_SECRET", "")
	p, _ := craftWorld(t)
	gw.iduna = idunaclient.New()
	gw.charIDBySlot["crafter"] = "char-c"

	_ = p.craftSkill.SetLevel(craft.Alchemy, 3.5)
	_ = p.craftSkill.SetLevel(craft.Cooking, 12.25)
	syncGatherSkills(p)
	if !approx(fake.skills["craft_alchemy"], 3.5) || !approx(fake.skills["craft_cooking"], 12.25) {
		t.Fatalf("stored craft skills: %v", fake.skills)
	}
	// outage then recovery: nothing lost
	fake.fail = true
	_ = p.craftSkill.SetLevel(craft.Alchemy, 4.0)
	syncGatherSkills(p)
	fake.fail = false
	syncGatherSkills(p)
	if !approx(fake.skills["craft_alchemy"], 4.0) {
		t.Errorf("alchemy after outage = %v, want 4.0", fake.skills["craft_alchemy"])
	}
	// relog
	stored, _ := gw.iduna.GetSkills("char-c")
	p2 := newTestPlayer()
	p2.craftSkill = craft.NewCraftSkill()
	loadCraftSkills(p2, stored)
	if l, _ := p2.craftSkill.Level(craft.Alchemy); !approx(l, 4.0) {
		t.Errorf("alchemy after relog = %v", l)
	}
	if l, _ := p2.craftSkill.Level(craft.Cooking); !approx(l, 12.25) {
		t.Errorf("cooking after relog = %v", l)
	}
}

func TestEatConsumesFoodAndBuffsAreReal(t *testing.T) {
	p, buf := craftWorld(t)
	p.jobID = WAR
	p.charXP.Level = 20
	applyJobStats(p)
	base := playerCombatStats(p)
	baseHP := p.maxHP
	cmdEat(p, "bog-eel-stew") // none in the bag
	p.w.Flush()
	if !strings.Contains(buf.String(), "don't have") || p.foodEffect != nil {
		t.Errorf("eating food you don't own must fail:\n%s", buf.String())
	}
	p.inventory["bog-eel-stew"] = 1
	cmdEat(p, "bog-eel-stew")
	if p.inventory["bog-eel-stew"] != 0 {
		t.Errorf("eating consumes the dish")
	}
	st := playerCombatStats(p)
	if st.STR != base.STR+5 || st.VIT != base.VIT+5 {
		t.Errorf("food stats not applied: STR %d->%d VIT %d->%d", base.STR, st.STR, base.VIT, st.VIT)
	}
	if p.maxHP != baseHP+80 {
		t.Errorf("food HP not applied: %d -> %d", baseHP, p.maxHP)
	}
	// expiry removes it again
	expireFood(p, time.Now().Add(2*time.Hour))
	if p.foodEffect != nil || p.maxHP != baseHP {
		t.Errorf("expired food should clear (food=%v hp=%d)", p.foodEffect, p.maxHP)
	}
	// legacy food id still works with an inventory item
	p.inventory["minestrone"] = 1
	cmdEat(p, "food-minestrone")
	if p.foodEffect == nil || p.inventory["minestrone"] != 0 {
		t.Errorf("legacy food ids should resolve to the inventory item")
	}
}

func TestCraftedConsumablesWork(t *testing.T) {
	p, reg, _ := kitWorld(t, WAR, 30)
	p.inventory["shell-salve"] = 2
	p.hp = 10
	cmdUseItem(p, "shell-salve")
	if p.hp != 130 || p.inventory["shell-salve"] != 1 {
		t.Errorf("shell-salve: hp=%d left=%d", p.hp, p.inventory["shell-salve"])
	}
	p.inventory["sleeping-powder"] = 1
	cmdUseItem(p, "sleeping-powder")
	for _, id := range []string{"a", "b", "c"} {
		if m, _ := reg.Get(id); !m.Has(mob.FxSleep) {
			t.Errorf("sleeping powder should sleep %s", id)
		}
	}
	if p.inventory["sleeping-powder"] != 0 {
		t.Errorf("the powder is used up")
	}
	// a thrown item with no target is not wasted
	p.combat.TargetMobID = ""
	p.inventory["party-store-firework"] = 1
	cmdUseItem(p, "party-store-firework")
	if p.inventory["party-store-firework"] != 1 {
		t.Errorf("a failed throw must not consume the item")
	}
	p.inventory["venom-flask"] = 1
	cmdUseItem(p, "venom-flask")
	if p.coating == nil || p.coating.Fx != mob.FxPoison {
		t.Errorf("venom flask should coat the weapon: %+v", p.coating)
	}
}

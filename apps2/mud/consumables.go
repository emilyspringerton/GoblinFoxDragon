package main

// consumables.go -- crafted consumables and foods.
//
// Cooked dishes are food.Food buffs: `eat <dish>` consumes one from inventory and the buff's stat
// bonuses (and HP/MP) now actually feed combat -- before, `eat` never touched inventory (infinite
// free food) and the bonuses were display-only. Alchemy products reuse the kit executor (kit.go):
// a thrown flask is just an ability with an item as its source.

import (
	"strings"
	"time"

	"dragonsnshit/server/food"
	"dragonsnshit/server/mob"
	"dragonsnshit/server/status"
)

// itemKits maps a consumable's id to the effect it triggers. Target rules mirror kit entries:
// enemy-targeting items need a current target and are only consumed if they can be used.
var itemKits = map[string]*kitEntry{
	"shell-salve":          ability("shell-salve", "Shell-Salve", nil, 0, tgtSelf, 0, "", Heal(120, "", 0)),
	"hospitality-draught":  ability("hospitality-draught", "Hospitality Draught", nil, 0, tgtSelf, 0, "", Heal(300, "", 0)),
	"eastwind-tonic":       ability("eastwind-tonic", "Eastwind Tonic", nil, 0, tgtSelf, 0, "", MPRestore(80)),
	"vellum-ether":         ability("vellum-ether", "Vellum Ether", nil, 0, tgtSelf, 0, "", MPRestore(200)),
	"owl-tea":              ability("owl-tea", "Owl Tea", nil, 0, tgtSelf, 0, "", Heal(80, "", 0), MPRestore(40)),
	"brine-elixir":         ability("brine-elixir", "Brine Elixir", nil, 0, tgtSelf, 0, "", Heal(500, "", 0), MPRestore(200)),
	"ward-ink-tonic":       ability("ward-ink-tonic", "Ward-Ink Tonic", nil, 0, tgtSelf, 0, "", Cleanse()),
	"venom-flask":          ability("venom-flask", "Venom Flask", nil, 0, tgtSelf, 0, "", Coat(mob.FxPoison, 8, 60*time.Second, 6)),
	"numbing-resin":        ability("numbing-resin", "Numbing Resin", nil, 0, tgtSelf, 0, "", Coat(mob.FxSlow, 25, 30*time.Second, 6)),
	"sleeping-powder":      ability("sleeping-powder", "Sleeping Powder", nil, 0, tgtArea, 8, "", Fx(mob.FxSleep, 0, 40*time.Second)),
	"smoke-pellet":         ability("smoke-pellet", "Smoke Pellet", nil, 0, tgtArea, 8, "", Fx(mob.FxBlind, 40, 30*time.Second), Hide(10*time.Second)),
	"party-store-firework": ability("party-store-firework", "Party Store Firework", nil, 0, tgtArea, 5, "", Dmg(140, "", 0, "Fire")),
	"acid-flask-refill":    ability("acid-flask-refill", "Acid Flask", nil, 0, tgtEnemy, 0, "", Dot(11, 45*time.Second), Fx(mob.FxWeaken, 20, 45*time.Second)),
	"spirit-lantern":       ability("spirit-lantern", "Spirit Lantern", nil, 0, tgtZone, 0, "", Buff(status.Regen, 4, 3*time.Minute)),
}

// useItemKit consumes one of id and runs its effect. It reports whether id is a kit consumable.
func useItemKit(p *player, id string) bool {
	k, ok := itemKits[id]
	if !ok {
		return false
	}
	if gw == nil {
		return false
	}
	if k.Target == tgtEnemy || k.Target == tgtArea {
		if p.combat.TargetMobID == "" {
			p.send("No target. Use 'target <mob>' first.")
			p.prompt()
			return true
		}
		if m, ok := gw.mobRegs[p.zoneID].Get(p.combat.TargetMobID); !ok || m.HP <= 0 {
			p.send("Your target is dead or gone.")
			p.prompt()
			return true
		}
	}
	p.inventory[id]--
	if p.inventory[id] <= 0 {
		delete(p.inventory, id)
	}
	runKit(p, k, "", true)
	return true
}

// foodCandidates resolves what the player typed to (registry id, inventory key).
func resolveFood(p *player, typed string) (food.Food, string, bool) {
	if gw == nil || gw.foodReg == nil {
		return food.Food{}, "", false
	}
	typed = strings.ToLower(typed)
	for _, regID := range []string{typed, "food-" + typed} {
		f, err := gw.foodReg.Get(regID)
		if err != nil {
			continue
		}
		for _, invKey := range []string{typed, strings.TrimPrefix(typed, "food-"), strings.TrimPrefix(regID, "food-"), regID} {
			if p.inventory[invKey] > 0 {
				return f, invKey, true
			}
		}
		return f, "", false // a real food, but none in the bag
	}
	return food.Food{}, "", false
}

func cmdEat(p *player, itemID string) {
	f, invKey, haveIt := resolveFood(p, itemID)
	if f.ID == "" {
		p.sendf("%q isn't something you can eat.", itemID)
		p.prompt()
		return
	}
	if !haveIt {
		p.sendf("You don't have any %s.", f.Name)
		p.prompt()
		return
	}
	now := time.Now()
	effect, err := gw.foodReg.Eat(f.ID, now)
	if err != nil {
		p.sendf("You can't eat that (%v).", err)
		p.prompt()
		return
	}
	p.inventory[invKey]--
	if p.inventory[invKey] <= 0 {
		delete(p.inventory, invKey)
	}
	p.foodEffect = effect
	applyJobStats(p) // food HP/MP
	p.hp = clampInt(p.hp, 0, p.maxHP)
	p.sendf("You eat %s. (%s for %s)", effect.Food.Name, foodStatSummary(effect.Food), formatDuration(effect.Food.Duration))
	p.prompt()
}

// activeFood is the food buff if it is still running.
func activeFood(p *player) *food.Food {
	if p.foodEffect != nil && p.foodEffect.IsActive(time.Now()) {
		f := p.foodEffect.Food
		return &f
	}
	return nil
}

// expireFood clears a lapsed food buff (and its HP/MP) -- called from the tick loop.
func expireFood(p *player, now time.Time) {
	if p.foodEffect != nil && !p.foodEffect.IsActive(now) {
		name := p.foodEffect.Food.Name
		p.foodEffect = nil
		applyJobStats(p)
		p.hp = clampInt(p.hp, 0, p.maxHP)
		p.sendf("\r\n[%s wears off.]", name)
		p.prompt()
	}
}

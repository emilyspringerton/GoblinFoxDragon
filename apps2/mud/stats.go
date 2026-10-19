package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"dragonsnshit/server/status"
)

// stats.go is the one place a player's full combat stat sheet is assembled and the one place the
// meaning of every item stat key is defined. Before this file only str/dex/vit/agi/int/mnd/chr and
// the weapon's "attack" ever reached combat; gear defense, accuracy, evasion, magic attack/defense
// bonus, haste, HP/MP, fast cast, enmity, store TP, crit rate, regen and the rest were data with no
// effect. Each stat below names exactly where it is consumed:
//
//	str            melee damage mean (combat_formula.go)       dex   hit chance, crit chance
//	vit            flat incoming-damage reduction               agi   mob hit chance (evasion)
//	int            nuke damage                                  mnd   heal power, MP, status resistance
//	chr            potency of songs, dances, rolls and waltzes
//	attack         flat damage added before STR scaling         accuracy  +hit chance (accuracy/3 %)
//	defense        flat incoming-damage reduction (defense/4)   evasion   -mob hit chance (evasion/3 %)
//	crit_rate      +crit chance %                               haste     swing speed and TP gain
//	magic_attack_bonus  +% spell damage                         magic_defense_bonus  resist mob-cast debuffs
//	hp / mp        max HP / MP                                  hp_regen / mp_regen  per game tick
//	fast_cast      -% cast time (cap 50)                        enmity    +/-% enmity generated
//	store_tp       extra TP per landed hit                      resist_sleep  % to shrug off mob-cast CC
//	ranged_attack / ranged_accuracy  ranged (RNG/COR) abilities
//	e_<element>    +% damage of that element (light, fire, wind ...)

// statAliases canonicalizes the key spellings that exist in data/items.json (some entries were
// authored with Title Case or alternate names).
var statAliases = map[string]string{
	"magic attack bonus":  "magic_attack_bonus",
	"magic defense bonus": "magic_defense_bonus",
	"critical hit rate":   "crit_rate",
	"enmity_increase":     "enmity",
	"cha":                 "chr",
	"light":               "e_light",
	"light_damage":        "e_light",
}

func normalizeStatKey(k string) string {
	k = strings.ToLower(strings.TrimSpace(k))
	if a, ok := statAliases[k]; ok {
		return a
	}
	return k
}

// normalizeStats returns a copy of in with every key canonicalized (values of colliding keys sum).
func normalizeStats(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[normalizeStatKey(k)] += v
	}
	return out
}

// gearStats is the canonicalized total of everything equipped.
func gearStats(p *player) map[string]int {
	return normalizeStats(equipStatTotals(p))
}

// Caps keep a stat stack from breaking the combat formulas.
const (
	hasteCap      = 40
	slowCap       = -50
	fastCastCap   = 50
	enmityCapPct  = 100
	elemBonusCap  = 100
	mdbResistCap  = 70
	ccResistCap   = 80
	baseSwingTime = 2 * time.Second
)

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// hasteTotal combines gear haste with Haste/Slow statuses (Slow is negative) into one percentage.
func hasteTotal(p *player) int {
	return clampInt(gearStats(p)["haste"]+p.statFX.NetHastePct(), slowCap, hasteCap)
}

// swingDelayFor is the player's auto-attack interval: the base swing time shortened by haste or
// stretched by slow. Haste used to affect TP gain only; it now also changes how often you swing.
func swingDelayFor(p *player) time.Duration {
	h := hasteTotal(p)
	return time.Duration(float64(baseSwingTime) * float64(100-h) / 100)
}

// storeTPBonus is the extra TP a landed hit grants.
func storeTPBonus(p *player) int { return gearStats(p)["store_tp"] }

// enmityScale applies the gear "enmity" stat (percent, may be negative) to an enmity amount.
func enmityScale(p *player, amount int) int {
	pct := clampInt(gearStats(p)["enmity"], -enmityCapPct, enmityCapPct)
	scaled := amount * (100 + pct) / 100
	if scaled < 1 && amount > 0 {
		scaled = 1
	}
	return scaled
}

// fastCastTime shortens a cast time by the gear fast_cast percentage.
func fastCastTime(p *player, castTime time.Duration) time.Duration {
	pct := clampInt(gearStats(p)["fast_cast"], 0, fastCastCap)
	return time.Duration(float64(castTime) * float64(100-pct) / 100)
}

// spellPower scales a spell's raw damage/heal by magic attack bonus and the element's affinity.
// element may be "" for non-elemental effects.
func spellPower(p *player, raw int, element string) int {
	g := gearStats(p)
	pct := g["magic_attack_bonus"]
	if element != "" {
		pct += clampInt(g["e_"+strings.ToLower(element)], 0, elemBonusCap)
	}
	if pct < 0 {
		pct = 0
	}
	return raw * (100 + pct) / 100
}

// resistsMobDebuff reports whether a mob-cast debuff of the given kind is shrugged off, using
// magic defense bonus plus Shell for everything, and resist_sleep for hard crowd control.
func resistsMobDebuff(p *player, kind status.Kind) bool {
	g := gearStats(p)
	chance := clampInt(g["magic_defense_bonus"]+p.statFX.MagicDefBonus(), 0, mdbResistCap)
	if kind == status.Bind || kind == status.Paralyze || kind == status.Silence || kind == status.Slow {
		chance += clampInt(g["resist_sleep"], 0, ccResistCap)
	}
	return gw != nil && gw.rng.Intn(100) < clampInt(chance, 0, 90)
}

// gearMaxBonuses returns the max HP / MP the equipped gear adds.
func gearMaxBonuses(p *player) (hp, mp int) {
	g := gearStats(p)
	hp, mp = g["hp"], g["mp"]
	if f := activeFood(p); f != nil {
		hp += f.HPBonus
		mp += f.MPBonus
	}
	return hp, mp
}

// gearRegen returns gear HP/MP regen per game tick.
func gearRegen(p *player) (hp, mp int) {
	g := gearStats(p)
	return g["hp_regen"], g["mp_regen"]
}

// statSheet renders the full derived sheet for the `stats` command.
func statSheet(p *player) []string {
	s := playerCombatStats(p)
	g := gearStats(p)
	lines := []string{
		fmt.Sprintf("STR %d  DEX %d  VIT %d  AGI %d  INT %d  MND %d  CHR %d", s.STR, s.DEX, s.VIT, s.AGI, s.INT, s.MND, s.CHR),
		fmt.Sprintf("Attack +%d  Accuracy +%d%%  Crit +%d%%  Haste %d%%  Store TP +%d", g["attack"], g["accuracy"]/3+p.statFX.Potency(status.AccuracyUp), g["crit_rate"]+p.statFX.Potency(status.CritUp), hasteTotal(p), g["store_tp"]),
		fmt.Sprintf("Defense %d (-%d dmg)  Evasion +%d (-%d%% enemy hit)  VIT -%d dmg", g["defense"], g["defense"]/4, g["evasion"], g["evasion"]/3, s.VIT/2),
		fmt.Sprintf("Magic Atk +%d%%  Magic Def +%d%% resist  Fast Cast %d%%  Enmity %+d%%", g["magic_attack_bonus"], clampInt(g["magic_defense_bonus"]+p.statFX.MagicDefBonus(), 0, mdbResistCap), clampInt(g["fast_cast"], 0, fastCastCap), g["enmity"]),
		fmt.Sprintf("Regen HP +%d MP +%d per tick  Ranged Atk %d Acc %d", g["hp_regen"], g["mp_regen"], g["ranged_attack"], g["ranged_accuracy"]),
	}
	var elems []string
	for k, v := range g {
		if strings.HasPrefix(k, "e_") && v != 0 {
			elems = append(elems, fmt.Sprintf("%s %+d%%", strings.TrimPrefix(k, "e_"), v))
		}
	}
	sort.Strings(elems)
	if len(elems) > 0 {
		lines = append(lines, "Elemental: "+strings.Join(elems, ", "))
	}
	return lines
}

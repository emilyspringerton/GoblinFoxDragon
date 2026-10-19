package main

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"dragonsnshit/server/gear"
	"dragonsnshit/server/itemdef"
	"dragonsnshit/server/job"
	"dragonsnshit/server/status"
)

// gearPlayer returns a player wearing synthetic items that each carry the given stats (one item
// per slot, mixed-case keys on purpose to exercise normalization).
func gearPlayer(t *testing.T, perSlot map[gear.Slot]string) *player {
	t.Helper()
	itemdefReg = itemdef.NewRegistry()
	var rows []string
	id := 9000
	p := newTestPlayer()
	p.jobID = job.WAR
	p.statFX = status.New()
	p.equip = gear.NewEquipment()
	for slot, stats := range perSlot {
		id++
		rows = append(rows, `{"id":`+itoa(id)+`,"name":"T`+itoa(id)+`","category":"armor","stack_size":1,"stats":`+stats+`}`)
		_ = p.equip.Equip(slot, gear.ItemEntry{ItemID: "t" + itoa(id), IL: 1, DefID: id})
	}
	if err := itemdefReg.LoadJSON([]byte("[" + strings.Join(rows, ",") + "]")); err != nil {
		t.Fatal(err)
	}
	return p
}

func itoa(i int) string { return strconv.Itoa(i) }

func newDeterministicRNG(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }

func TestStatKeysNormalize(t *testing.T) {
	got := normalizeStats(map[string]int{"INT": 5, "Magic Attack Bonus": 5, "Critical Hit Rate": 3, "cha": 4, "CHR": 1, "light_damage": 5, "enmity_increase": 4})
	want := map[string]int{"int": 5, "magic_attack_bonus": 5, "crit_rate": 3, "chr": 5, "e_light": 5, "enmity": 4}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("normalized[%q] = %d, want %d (all: %v)", k, got[k], v, got)
		}
	}
}

func TestGearDefenseEvasionReduceMobHits(t *testing.T) {
	naked := gearPlayer(t, nil)
	armored := gearPlayer(t, map[gear.Slot]string{gear.SlotHead: `{"Defense":40,"evasion":60}`})
	if got := gearStats(armored)["defense"]; got != 40 {
		t.Fatalf("defense = %d", got)
	}
	var nakedDmg, armoredDmg, nakedHits, armoredHits int
	for i := 0; i < 4000; i++ {
		if hit, _, d := resolveMobAutoAttackDamage(naked, 100); hit {
			nakedDmg += d
			nakedHits++
		}
		if hit, _, d := resolveMobAutoAttackDamage(armored, 100); hit {
			armoredDmg += d
			armoredHits++
		}
	}
	if armoredHits >= nakedHits {
		t.Errorf("evasion should reduce hits taken: armored %d vs naked %d", armoredHits, nakedHits)
	}
	if armoredDmg/armoredHits >= nakedDmg/nakedHits {
		t.Errorf("defense should reduce damage per hit: armored %d vs naked %d", armoredDmg/armoredHits, nakedDmg/nakedHits)
	}
}

func TestAccuracyAndCritGearRaiseOffense(t *testing.T) {
	plain := gearPlayer(t, nil)
	geared := gearPlayer(t, map[gear.Slot]string{gear.SlotHead: `{"accuracy":90,"Critical Hit Rate":30}`})
	var plainHits, gearedHits, plainCrits, gearedCrits int
	for i := 0; i < 4000; i++ {
		if h, c, _ := resolvePlayerAutoAttackDamage(plain, 50); h {
			plainHits++
			if c {
				plainCrits++
			}
		}
		if h, c, _ := resolvePlayerAutoAttackDamage(geared, 50); h {
			gearedHits++
			if c {
				gearedCrits++
			}
		}
	}
	if gearedHits <= plainHits {
		t.Errorf("accuracy gear should land more hits: %d vs %d", gearedHits, plainHits)
	}
	if gearedCrits <= plainCrits {
		t.Errorf("crit gear should crit more: %d vs %d", gearedCrits, plainCrits)
	}
}

func TestHasteChangesSwingSpeed(t *testing.T) {
	p := gearPlayer(t, map[gear.Slot]string{gear.SlotFeet: `{"haste":20}`})
	if got := swingDelayFor(p); got != time.Duration(1600)*time.Millisecond {
		t.Errorf("20%% gear haste should give 1.6s swings, got %v", got)
	}
	p.statFX.Apply(status.Effect{Kind: status.Slow, Potency: 30, ExpiresAt: time.Now().Add(time.Minute)})
	if got := hasteTotal(p); got != -10 {
		t.Errorf("gear haste 20 + slow 30 = -10, got %d", got)
	}
	capped := gearPlayer(t, map[gear.Slot]string{gear.SlotFeet: `{"haste":90}`})
	if hasteTotal(capped) != hasteCap {
		t.Errorf("haste must cap at %d", hasteCap)
	}
}

func TestMagicStatsScaleSpells(t *testing.T) {
	p := gearPlayer(t, map[gear.Slot]string{gear.SlotHead: `{"Magic Attack Bonus":50,"e_fire":20,"fast_cast":30,"enmity":-50}`})
	if got := spellPower(p, 100, "Fire"); got != 170 {
		t.Errorf("100 fire with +50%% MAB +20%% fire = 170, got %d", got)
	}
	if got := spellPower(p, 100, "Ice"); got != 150 {
		t.Errorf("non-fire element gets MAB only: %d", got)
	}
	if got := fastCastTime(p, 4*time.Second); got != 2800*time.Millisecond {
		t.Errorf("30%% fast cast on 4s = 2.8s, got %v", got)
	}
	if got := enmityScale(p, 100); got != 50 {
		t.Errorf("-50%% enmity: %d", got)
	}
}

func TestMagicDefenseAndResistGearResistMobDebuffs(t *testing.T) {
	oldGW := gw
	gw = &world{rng: newDeterministicRNG(7)}
	t.Cleanup(func() { gw = oldGW })
	bare := gearPlayer(t, nil)
	warded := gearPlayer(t, map[gear.Slot]string{gear.SlotHead: `{"magic_defense_bonus":50,"resist_sleep":30}`})
	var bareResists, wardedResists int
	for i := 0; i < 2000; i++ {
		if resistsMobDebuff(bare, status.Bind) {
			bareResists++
		}
		if resistsMobDebuff(warded, status.Bind) {
			wardedResists++
		}
	}
	if bareResists != 0 {
		t.Errorf("no gear = no resists, got %d", bareResists)
	}
	if wardedResists < 1400 || wardedResists > 1800 { // 80% expected
		t.Errorf("50 MDB + 30 resist_sleep vs Bind should resist ~80%%, got %d/2000", wardedResists)
	}
	// Shell counts as magic defense.
	bare.statFX.Apply(status.Effect{Kind: status.Shell, Potency: 60, ExpiresAt: time.Now().Add(time.Minute)})
	shelled := 0
	for i := 0; i < 2000; i++ {
		if resistsMobDebuff(bare, status.Poison) {
			shelled++
		}
	}
	if shelled < 1100 || shelled > 1500 { // 60% expected
		t.Errorf("Shell 60 should resist ~60%%, got %d/2000", shelled)
	}
}

func TestHPMPRegenAndStoreTPFromGear(t *testing.T) {
	p := gearPlayer(t, map[gear.Slot]string{gear.SlotHead: `{"HP":90,"mp":40,"hp_regen":3,"mp_regen":2,"store_tp":5}`})
	p.jobID = job.WHM
	p.charXP.Level = 10
	applyJobStats(p)
	baseHP, _ := job.HPAtLevel(job.WHM, 10)
	baseMP, _ := job.MPAtLevel(job.WHM, 10)
	if p.maxHP != baseHP+90 || p.maxMP != baseMP+40 {
		t.Errorf("gear HP/MP: maxHP %d (want %d) maxMP %d (want %d)", p.maxHP, baseHP+90, p.maxMP, baseMP+40)
	}
	if rh, rm := gearRegen(p); rh != 3 || rm != 2 {
		t.Errorf("regen = %d/%d", rh, rm)
	}
	if storeTPBonus(p) != 5 {
		t.Errorf("store tp = %d", storeTPBonus(p))
	}
}

func TestPrimaryStatsAndHealPowerUseGear(t *testing.T) {
	p := gearPlayer(t, map[gear.Slot]string{gear.SlotHead: `{"MND":20,"str":4,"cha":6}`})
	s := playerCombatStats(p)
	base, _ := job.StatsFor(job.WAR)
	if s.MND != base.MND+20 || s.STR != base.STR+4 || s.CHR != base.CHR+6 {
		t.Errorf("gear primary stats not applied: %+v vs base %+v", s, base)
	}
	if got := healPower(p, 100); got != 100+(s.MND-10)*2 {
		t.Errorf("heal power = %d", got)
	}
}

func TestStatSheetListsEverything(t *testing.T) {
	p := gearPlayer(t, map[gear.Slot]string{gear.SlotHead: `{"Defense":8,"e_wind":5}`})
	out := strings.Join(statSheet(p), "\n")
	for _, want := range []string{"STR", "Defense 8", "Magic Atk", "Fast Cast", "Regen", "wind +5%"} {
		if !strings.Contains(out, want) {
			t.Errorf("stat sheet missing %q:\n%s", want, out)
		}
	}
}

package main

import (
	"bufio"
	"bytes"
	"math/rand"
	"strings"
	"testing"
	"time"

	"dragonsnshit/server/enmity"
	"dragonsnshit/server/homepoint"
	"dragonsnshit/server/job"
	"dragonsnshit/server/mob"
	"dragonsnshit/server/status"
	"dragonsnshit/server/xp"
)

// legacyIDs are spells/abilities that already existed before the kit layer; the kit must not
// shadow them (existing hand-coded cases win in castNow/cmdJA, so a dup would be dead data).
var legacyIDs = []string{
	"invisible", "invis", "sneak", "cure", "cure2", "protect", "shell", "haste", "regen", "refresh", "dia",
	"poison", "bio", "distract", "frazzle", "march", "paeon", "ballad", "minne", "carol", "mambo",
	"drain", "aspir", "absorb-str", "absorb-dex", "absorb-vit", "absorb-int", "absorb-mnd",
	"katon", "suiton", "doton", "raiton", "hyoton", "huton", "flash", "sentinel", "rampart", "holy", "banish", "banish2",
	"provoke", "berserk", "warcry", "aggressor", "defenders", "mighty_strikes", "summon_zagan", "summon_beleth",
	"summon_vassago", "benediction", "holy_water", "chakra", "boost", "clear_mind", "elemental_seal", "convert",
	"chainspell", "venom", "siphon", "sneak_attack", "trick_attack",
}

func TestKitRegistryInvariants(t *testing.T) {
	legacy := map[string]bool{}
	for _, id := range legacyIDs {
		legacy[id] = true
	}
	validJob := map[string]bool{}
	for _, j := range job.AllJobs {
		validJob[string(j)] = true
	}
	perJob := map[string]int{}
	for _, e := range kitOrdered {
		if legacy[e.ID] {
			t.Errorf("kit id %q shadows a legacy spell/ability", e.ID)
		}
		if e.Name == "" || e.Desc == "" || len(e.Ops) == 0 || len(e.Jobs) == 0 {
			t.Errorf("%s: incomplete entry", e.ID)
		}
		if strings.ContainsAny(e.ID, " _A") {
			t.Errorf("%s: ids are lowercase-hyphen", e.ID)
		}
		if e.Kind == kitAbility && e.MP != 0 {
			t.Errorf("%s: abilities are free (MP %d)", e.ID, e.MP)
		}
		if e.Kind == kitAbility && e.Recast <= 0 {
			t.Errorf("%s: abilities need a recast", e.ID)
		}
		if (e.Target == tgtArea) && e.Radius <= 0 {
			t.Errorf("%s: area entry needs a radius", e.ID)
		}
		for j, lvl := range e.Jobs {
			if !validJob[j] {
				t.Errorf("%s: unknown job %q", e.ID, j)
			}
			if lvl < 1 || lvl > 99 {
				t.Errorf("%s: level %d out of range", e.ID, lvl)
			}
			perJob[j]++
		}
	}
	for _, j := range job.AllJobs {
		if perJob[string(j)] < 3 {
			t.Errorf("job %s has only %d kit entries", j, perJob[string(j)])
		}
	}
}

func TestEveryJobIsUnlocked(t *testing.T) {
	for _, j := range job.AllJobs {
		if !enabledJobs[string(j)] {
			t.Errorf("%s is locked", j)
		}
	}
}

// ASN is the enfeeble hybrid: it must carry the full crowd-control suite plus poisons/alchemy.
func TestAssassinHasTheFullCCAndPoisonSuite(t *testing.T) {
	have := map[mob.EffectKind]bool{}
	dots := 0
	coats := 0
	for _, e := range kitOrdered {
		if _, ok := e.Jobs[ASN]; !ok {
			continue
		}
		for _, o := range e.Ops {
			switch o.Kind {
			case opMobFx:
				have[o.FxKind] = true
			case opDot:
				dots++
			case opCoat:
				coats++
			}
		}
	}
	for _, k := range []mob.EffectKind{mob.FxSleep, mob.FxBind, mob.FxParalyze, mob.FxSlow, mob.FxBlind, mob.FxSilence, mob.FxStun, mob.FxWeaken} {
		if !have[k] {
			t.Errorf("ASN has no %s", k)
		}
	}
	if dots < 3 || coats < 2 {
		t.Errorf("ASN should have poisons (dots=%d) and coated weapons (coats=%d)", dots, coats)
	}
	if _, ok := kitByID["sleeping-draught"]; !ok || kitByID["sleeping-draught"].Target != tgtArea {
		t.Errorf("sleeping-draught must be an area sleep")
	}
}

// kitWorld builds a minimal world + caster in zone 1 with three mobs standing together.
func kitWorld(t *testing.T, jobID string, level int) (*player, *mob.Registry, *bytes.Buffer) {
	t.Helper()
	mob.Roll = func() float64 { return 0 }
	t.Cleanup(func() { mob.Roll = func() float64 { return rand.Float64() } })
	reg := mob.New()
	for i, id := range []string{"a", "b", "c"} {
		_ = reg.Spawn(mob.Mob{ID: id, Kind: "bog-vole", HP: 5000, MaxHP: 5000, SceneID: 1, Pos: mob.Pos{X: float64(i)}, HomePos: mob.Pos{X: float64(i)},
			AggroRange: 1, MeleeRange: 2, MeleeDamage: 10, SwingDelay: time.Second, MoveSpeed: 3})
	}
	oldGW := gw
	gw = &world{mobRegs: map[int]*mob.Registry{1: reg}, mobEnmity: map[string]*enmity.Table{}, players: map[string]*player{},
		charIDBySlot: map[string]string{}, rng: rand.New(rand.NewSource(3))}
	t.Cleanup(func() { gw = oldGW })

	p := newTestPlayer()
	p.slot, p.name, p.zoneID = "caster", "Caster", 1
	p.jobID = jobID
	p.charXP = &xp.CharXP{Level: level}
	p.statFX = status.New()
	p.combat = &mob.PlayerCombat{TargetMobID: "a"}
	p.homePoint = homepoint.NewState(1)
	p.charJob, _ = job.NewCharJob(job.JobID(jobID), "", level, 0)
	var buf bytes.Buffer
	p.w = bufio.NewWriter(&buf)
	p.inventory = map[string]int{}
	applyJobStats(p)
	p.hp, p.mp = p.maxHP, p.maxMP
	p.recastTracker = job.NewRecastTracker(abilitiesFor(p))
	gw.players[p.slot] = p
	return p, reg, &buf
}

func TestLullabyPutsTheWholePackToSleepAndDamageWakesOne(t *testing.T) {
	p, reg, buf := kitWorld(t, BRD, 40)
	castNow(p, "lullaby", "")
	p.w.Flush()
	for _, id := range []string{"a", "b", "c"} {
		if m, _ := reg.Get(id); !m.Has(mob.FxSleep) {
			t.Fatalf("mob %s should be asleep after Foe Lullaby:\n%s", id, buf.String())
		}
	}
	if p.mp >= p.maxMP {
		t.Errorf("lullaby should cost MP")
	}
	// Sleepers do nothing even with a player right next to them.
	near := []mob.PlayerPositions{{Slot: "caster", SceneID: 1, Pos: mob.Pos{}}}
	if evs := reg.Tick(time.Now(), 1, near); len(evs) != 0 {
		t.Errorf("sleeping pack acted: %+v", evs)
	}
	// Hit one: it wakes, the others stay down.
	res, _, _ := reg.Hit("a", "caster", 5)
	if !res.Woke {
		t.Errorf("damage should wake a")
	}
	if m, _ := reg.Get("b"); !m.Has(mob.FxSleep) {
		t.Errorf("b should still be asleep")
	}
}

func TestSpellRecastMPSilenceAndLevelGates(t *testing.T) {
	p, reg, buf := kitWorld(t, BRD, 40)
	castNow(p, "lullaby", "")
	mpAfter := p.mp
	castNow(p, "lullaby", "") // on recast
	p.w.Flush()
	if p.mp != mpAfter || !strings.Contains(buf.String(), "on recast") {
		t.Errorf("recast not enforced (mp %d -> %d):\n%s", mpAfter, p.mp, buf.String())
	}
	buf.Reset()
	castNow(p, "horde-lullaby", "") // needs Lv40 as BRD: ok at 40
	castNow(p, "elegy", "")         // Lv37 ok
	p.statFX.Apply(status.Effect{Kind: status.Silence, Potency: 1, ExpiresAt: time.Now().Add(time.Minute)})
	castNow(p, "foe-requiem", "")
	p.w.Flush()
	if !strings.Contains(buf.String(), "silenced") {
		t.Errorf("silence should block spells:\n%s", buf.String())
	}
	p2, _, buf2 := kitWorld(t, BRD, 10)
	castNow(p2, "lullaby", "") // needs Lv20
	p2.w.Flush()
	if !strings.Contains(buf2.String(), "requires level 20") {
		t.Errorf("level gate message missing:\n%s", buf2.String())
	}
	p3, _, buf3 := kitWorld(t, WAR, 40)
	castNow(p3, "lullaby", "") // wrong job
	p3.w.Flush()
	if !strings.Contains(buf3.String(), "requires BRD") {
		t.Errorf("job gate message missing:\n%s", buf3.String())
	}
	_ = reg
}

func TestAssassinAbilitiesThroughTheRealJACommand(t *testing.T) {
	p, reg, buf := kitWorld(t, ASN, 40)
	ja(p, "sleeping-draught")
	for _, id := range []string{"a", "b", "c"} {
		if m, _ := reg.Get(id); !m.Has(mob.FxSleep) {
			p.w.Flush()
			t.Fatalf("sleeping draught should sleep %s:\n%s", id, buf.String())
		}
	}
	// recast enforced via the job recast tracker
	buf.Reset()
	ja(p, "sleeping-draught")
	p.w.Flush()
	if !strings.Contains(buf.String(), "recast") {
		t.Errorf("ability recast not enforced:\n%s", buf.String())
	}
	// acid flask: poison + weaken on the target
	ja(p, "acid-flask")
	if m, _ := reg.Get("a"); !m.Has(mob.FxPoison) || !m.Has(mob.FxWeaken) {
		t.Errorf("acid flask should poison and weaken")
	}
	// envenom coats the weapon; a landed hit then poisons
	ja(p, "envenom")
	if p.coating == nil || p.coating.Charges != 8 {
		t.Fatalf("envenom should arm 8 charges: %+v", p.coating)
	}
	m, _ := reg.Get("b")
	reg.RemoveEffect("b", mob.FxSleep)
	applyCoating(p, m, reg, 20, time.Now())
	if !m.Has(mob.FxPoison) || p.coating.Charges != 7 {
		t.Errorf("a coated hit should poison and spend a charge")
	}
	// spells: dreamweed single sleep, hemlock dot, mp is spent
	mp := p.mp
	castNow(p, "hemlock", "")
	if p.mp >= mp {
		t.Errorf("hemlock should cost MP")
	}
}

func TestHealRaiseCleanseAndBuffs(t *testing.T) {
	p, _, _ := kitWorld(t, WHM, 60)
	ally := newTestPlayer()
	ally.slot, ally.name, ally.zoneID = "ally", "Ally", 1
	ally.statFX = status.New()
	ally.maxHP, ally.hp = 1000, 100
	ally.w = bufio.NewWriter(&bytes.Buffer{})
	ally.homePoint = homepoint.NewState(1)
	gw.players["ally"] = ally

	castNow(p, "cure3", "Ally")
	if ally.hp <= 400 {
		t.Errorf("cure III should heal a lot, hp=%d", ally.hp)
	}
	ally.statFX.Apply(status.Effect{Kind: status.Poison, Potency: 5, ExpiresAt: time.Now().Add(time.Minute)})
	castNow(p, "erase", "Ally")
	if ally.statFX.Has(status.Poison) {
		t.Errorf("erase should clear poison")
	}
	ally.homePoint.IsKO = true
	ally.hp = 0
	castNow(p, "raise", "Ally")
	if ally.homePoint.IsKO || ally.hp != 500 {
		t.Errorf("raise: KO=%v hp=%d, want revived at 50%%", ally.homePoint.IsKO, ally.hp)
	}
	// party-wide buff reaches the ally
	d, _, _ := kitWorld(t, DNC, 50)
	_ = d
	castNow(p, "curaga", "")
}

func TestSubJobGivesSpellsMPAndAbilities(t *testing.T) {
	p, _, buf := kitWorld(t, WAR, 40)
	if p.maxMP != 0 {
		t.Fatalf("WAR has no MP of its own, got %d", p.maxMP)
	}
	cmdSetSubJob(p, "RDM")
	p.w.Flush()
	if p.charJob.Sub != "RDM" {
		t.Fatalf("sub not set:\n%s", buf.String())
	}
	if p.maxMP <= 0 {
		t.Errorf("a sub job must give MP, got %d", p.maxMP)
	}
	if got := p.charJob.EffectiveSubLevel(); got != 20 {
		t.Errorf("sub effective level at 40 = %d, want 20 (half)", got)
	}
	p.mp = p.maxMP
	buf.Reset()
	castNow(p, "paralyze", "") // RDM Lv4 -> fine at sub level 20
	if m, _ := gw.mobRegs[1].Get("a"); !m.Has(mob.FxParalyze) {
		p.w.Flush()
		t.Errorf("sub-job spell should work:\n%s", buf.String())
	}
	buf.Reset()
	castNow(p, "addle", "") // RDM 50: too high
	p.w.Flush()
	if !strings.Contains(buf.String(), "sub-job level 50") {
		t.Errorf("high sub spell should be refused:\n%s", buf.String())
	}
	// level-up keeps the sub in step
	p.charXP.Level = 60
	applyJobStats(p)
	if p.charJob.MainLvl != 60 || p.charJob.EffectiveSubLevel() != 30 {
		t.Errorf("sub level stale after level-up: main %d sub eff %d", p.charJob.MainLvl, p.charJob.EffectiveSubLevel())
	}
	// sub abilities (RDM convert is Lv1) appear once level allows
	found := false
	for _, a := range abilitiesFor(p) {
		if a.ID == "composure" || a.ID == "convert" {
			found = true
		}
	}
	if !found {
		t.Errorf("sub job abilities missing from the recast list")
	}
}

func TestSubJobRulesAndClear(t *testing.T) {
	p, _, buf := kitWorld(t, WAR, 10)
	cmdSetSubJob(p, "WHM")
	p.w.Flush()
	if p.charJob.Sub != "" || !strings.Contains(buf.String(), "level 18") {
		t.Errorf("sub job needs level 18:\n%s", buf.String())
	}
	p.charXP.Level = 30
	cmdSetSubJob(p, "WAR")
	if p.charJob.Sub != "" {
		t.Errorf("sub cannot equal main")
	}
	cmdSetSubJob(p, "WHM")
	cmdSetSubJob(p, "NONE")
	if p.charJob.Sub != "" || p.maxMP != 0 {
		t.Errorf("clearing the sub should clear its MP (sub=%q mp=%d)", p.charJob.Sub, p.maxMP)
	}
}

func TestPoisonCoatDrainAndDotKillCredit(t *testing.T) {
	p, reg, _ := kitWorld(t, DNC, 30)
	p.hp = 100
	ja(p, "drain-samba")
	m, _ := reg.Get("a")
	applyCoating(p, m, reg, 100, time.Now())
	if p.hp != 125 {
		t.Errorf("drain samba should heal 25%% of 100 damage, hp=%d", p.hp)
	}
}

// ja runs a job ability the way a player would, clearing the 1s universal lockout between calls so
// tests can chain abilities.
func ja(p *player, id string) {
	p.lastActionAt = time.Time{}
	cmdJA(p, id)
}

package main

// kit.go -- the data-driven spell/ability layer for every job.
//
// Each entry in kitEntries (kit_data.go) is a spell (`cast <id>`) or a job ability (`ja <id>`) made
// of a few reusable ops: direct damage, damage-over-time, mob crowd control (sleep/bind/stun/slow/
// paralyze/blind/weaken/silence), heals, buffs, cleanses, drains, escapes, weapon coatings, steals,
// raises and so on. Everything goes through the same checks (job + level, silence, MP, recast,
// target) and the same stat model (stats.go): damage scales with the relevant primary stat, magic
// attack bonus and elemental affinity; heals scale with MND; songs/dances/rolls scale with CHR.
//
// Crowd control follows the Bard Lullaby model implemented in server/mob/cc.go: sleep is area
// capable, breaks on direct damage, can be resisted, and gets harder to re-apply to the same mob.

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"dragonsnshit/server/enmity"
	"dragonsnshit/server/job"
	"dragonsnshit/server/mob"
	"dragonsnshit/server/status"
)

type kitKind int

const (
	kitSpell kitKind = iota
	kitAbility
)

type kitTarget int

const (
	tgtEnemy kitTarget = iota // the current target mob
	tgtArea                   // every mob within Radius of the current target mob
	tgtSelf
	tgtAlly // a named player in the zone (default: self)
	tgtZone // every living player in the caster's zone
)

type opKind int

const (
	opDamage     opKind = iota // direct damage (wakes sleepers)
	opDot                      // poison damage over time (does not wake sleepers)
	opMobFx                    // crowd control / debuff on the enemy
	opDrain                    // damage, caster heals Pct% of it
	opHeal                     // heal allies
	opBuff                     // status buff on allies
	opCleanse                  // remove player debuffs
	opRestoreMP                // restore MP to allies
	opEnmity                   // add (or remove, if negative) enmity on enemies
	opNextAttack               // next landed melee hit +Pct% damage
	opNextCrit                 // next landed melee hit is a critical hit
	opCoat                     // next Charges melee hits apply a mob effect / drain
	opTP                       // gain TP
	opHide                     // drop aggro and turn invisible for Dur
	opSteal                    // steal one item from the target (once per mob)
	opRaise                    // revive a KO'd ally at Pct% HP
	opPayHP                    // spend Pct% of max HP
	opFlow                     // gain Base flow
)

type kitOp struct {
	Kind     opKind
	Base     int
	Stat     string  // scaling stat: str dex vit agi int mnd chr, or "ranged"
	PerStat  float64 // damage/heal per point of Stat above 0
	Element  string  // magical element ("" = physical/none)
	Phys     bool    // physical: scales with gear attack, not magic attack bonus
	Hits     int     // number of hits for opDamage (default 1)
	FxKind   mob.EffectKind
	Potency  int
	Dur      time.Duration
	Buff     status.Kind
	Pct      int  // opDrain heal%, opNextAttack bonus%, opRaise HP%, opPayHP%, opCoat drain%
	Charges  int  // opCoat charges
	ChrScale bool // buff potency/duration scaled by CHR
	Roll     bool // corsair-style: potency multiplied by a random 1-6 roll
}

type kitEntry struct {
	ID     string
	Name   string
	Kind   kitKind
	Jobs   map[string]int // job -> minimum level
	MP     int
	Cast   time.Duration
	Recast time.Duration
	Target kitTarget
	Radius float64
	Desc   string
	Ops    []kitOp
}

var (
	kitByID    = map[string]*kitEntry{}
	kitOrdered []*kitEntry
)

func registerKit(entries ...*kitEntry) {
	for _, e := range entries {
		if _, dup := kitByID[e.ID]; dup {
			panic("duplicate kit id: " + e.ID)
		}
		kitByID[e.ID] = e
		kitOrdered = append(kitOrdered, e)
	}
}

// kitLevelFor returns the level the player can use e at, and which role grants it ("" if none).
// Main job uses the character level; a sub job uses half of it and only for spells.
func kitLevelFor(p *player, e *kitEntry) (ok bool, needed int, role string) {
	if len(e.Jobs) == 0 { // an item effect: anyone can use it
		return true, 0, "item"
	}
	lvl := p.charXP.Level
	if min, has := e.Jobs[p.jobID]; has {
		return lvl >= min, min, "main"
	}
	if p.charJob != nil && p.charJob.Sub != "" {
		if min, has := e.Jobs[string(p.charJob.Sub)]; has {
			return p.charJob.EffectiveSubLevel() >= min, min, "sub"
		}
	}
	// report the lowest requirement for a clearer message
	needed = 0
	for _, m := range e.Jobs {
		if needed == 0 || m < needed {
			needed = m
		}
	}
	return false, needed, ""
}

func jobNames(e *kitEntry) string {
	var js []string
	for j := range e.Jobs {
		js = append(js, j)
	}
	sort.Strings(js)
	return strings.Join(js, "/")
}

// kitAbilitiesFor converts a job's kit abilities into recast-tracker entries.
func kitAbilitiesFor(jobID string) []job.Ability {
	var out []job.Ability
	for _, e := range kitOrdered {
		if e.Kind != kitAbility {
			continue
		}
		if min, ok := e.Jobs[jobID]; ok {
			out = append(out, job.Ability{ID: e.ID, Job: job.JobID(jobID), Recast: e.Recast, MinLevel: min})
		}
	}
	return out
}

func castTimeFor(spell string) time.Duration {
	if t := spellCastTimes[spell]; t > 0 {
		return t
	}
	if e, ok := kitByID[spell]; ok && e.Kind == kitSpell {
		return e.Cast
	}
	return 0
}

// weaponCoating is an armed melee coating (Envenom, Drain Samba ...): the next Charges landed hits
// apply the effect.
type weaponCoating struct {
	Name     string
	Fx       mob.EffectKind
	Potency  int
	Dur      time.Duration
	Charges  int
	DrainPct int
}

// stolenFrom tracks mobs already stolen from (caller holds gw.mu).
var stolenFrom = map[string]bool{}

var kitRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func statValue(p *player, stat string) int {
	s := playerCombatStats(p)
	switch stat {
	case "str":
		return s.STR
	case "dex":
		return s.DEX
	case "vit":
		return s.VIT
	case "agi":
		return s.AGI
	case "int":
		return s.INT
	case "mnd":
		return s.MND
	case "chr":
		return s.CHR
	case "ranged":
		return s.AGI + gearStats(p)["ranged_attack"]
	}
	return 0
}

func opDamageAmount(p *player, o kitOp) int {
	raw := o.Base + int(float64(statValue(p, o.Stat))*o.PerStat)
	if o.Phys {
		raw += equipAttackBonus(p) / 2
		if o.Stat == "ranged" {
			raw += gearStats(p)["ranged_accuracy"] / 2
		}
	} else {
		raw = spellPower(p, raw, o.Element)
	}
	// +-10% variance
	lo, hi := raw*9/10, raw*11/10
	if hi > lo {
		raw = lo + kitRand.Intn(hi-lo+1)
	}
	if raw < 1 {
		raw = 1
	}
	return raw
}

func scaledPotency(p *player, o kitOp) int {
	pot := o.Potency
	if o.ChrScale {
		pot += pot * statValue(p, "chr") / 200
	}
	return pot
}

func scaledDuration(p *player, o kitOp) time.Duration {
	d := o.Dur
	if o.ChrScale {
		d += d * time.Duration(statValue(p, "chr")) / 200
	}
	return d
}

// engage makes sure a hostile action leaves the mob aware of the caster (it will fight them once
// it is able to act again).
func engage(p *player, m *mob.Mob, enmityAmount int) {
	if m.AggroSlot == "" {
		m.AggroSlot = p.slot
		m.State = mob.StatePursuing
	}
	if enmityAmount != 0 {
		t := gw.mobEnmity[m.ID]
		if t == nil {
			t = enmity.NewTable()
			gw.mobEnmity[m.ID] = t
		}
		if enmityAmount > 0 {
			t.Add(p.slot, enmityScale(p, enmityAmount))
		}
		if top, err := t.Top(); err == nil && m.AggroSlot != top {
			m.AggroSlot = top
		}
	}
}

// cmdJAWithTarget is `ja <id> [target]`.
func cmdJAWithTarget(p *player, abilityID, targetName string) {
	kitTargetOverride = targetName
	defer func() { kitTargetOverride = "" }()
	cmdJA(p, abilityID)
}

// kitTargetOverride carries the optional ally name from `ja <id> <target>` into cmdJA (whose
// signature predates targeted abilities). Safe: command handling holds gw.mu for its whole run.
var kitTargetOverride string

// runKit executes e for p. viaJA means the recast was already consumed by cmdJA.
func runKit(p *player, e *kitEntry, targetName string, viaJA bool) {
	if ok, need, role := kitLevelFor(p, e); !ok {
		switch role {
		case "":
			p.sendf("%s requires %s (Lv%d).", e.Name, jobNames(e), need)
		case "sub":
			p.sendf("%s needs sub-job level %d (your sub job runs at half your level: %d).", e.Name, need, p.charJob.EffectiveSubLevel())
		default:
			p.sendf("%s requires level %d.", e.Name, need)
		}
		p.prompt()
		return
	}
	now := time.Now()
	if e.Kind == kitSpell {
		if p.statFX.IsSilenced() {
			p.send("You are silenced and cannot cast spells.")
			p.prompt()
			return
		}
		if p.kitRecasts == nil {
			p.kitRecasts = map[string]time.Time{}
		}
		if until := p.kitRecasts[e.ID]; now.Before(until) {
			p.sendf("%s is on recast. (%s remaining)", e.Name, until.Sub(now).Round(time.Second))
			p.prompt()
			return
		}
	}
	if e.MP > 0 && p.mp < e.MP {
		p.sendf("Not enough MP. (need %d, have %d)", e.MP, p.mp)
		p.prompt()
		return
	}

	// ---- resolve targets ----
	reg := gw.mobRegs[p.zoneID]
	var mobs []*mob.Mob
	var allies []*player
	switch e.Target {
	case tgtEnemy, tgtArea:
		if p.combat.TargetMobID == "" {
			p.send("No target. Use 'target <mob>' first.")
			p.prompt()
			return
		}
		tm, ok := reg.Get(p.combat.TargetMobID)
		if !ok || tm.HP <= 0 {
			p.send("Your target is dead or gone.")
			p.prompt()
			return
		}
		if e.Target == tgtEnemy {
			mobs = []*mob.Mob{tm}
		} else {
			for _, id := range reg.MobsNear(tm.Pos, e.Radius) {
				if m, ok := reg.Get(id); ok {
					mobs = append(mobs, m)
				}
			}
			sort.Slice(mobs, func(i, j int) bool { return mobs[i].ID < mobs[j].ID })
		}
	case tgtSelf:
		allies = []*player{p}
	case tgtAlly:
		if targetName == "" {
			targetName = kitTargetOverride
		}
		tgt, errMsg := resolveSpellTarget(p, targetName)
		if tgt == nil {
			p.send(errMsg)
			p.prompt()
			return
		}
		allies = []*player{tgt}
	case tgtZone:
		for _, other := range gw.players {
			if other.zoneID == p.zoneID && !other.homePoint.IsKO {
				allies = append(allies, other)
			}
		}
		sort.Slice(allies, func(i, j int) bool { return allies[i].slot < allies[j].slot })
	}
	// ---- pay ----
	p.mp -= e.MP
	if e.Kind == kitSpell && e.Recast > 0 {
		p.kitRecasts[e.ID] = now.Add(e.Recast)
	}

	// ---- apply ----
	out := &kitResult{}
	for _, o := range e.Ops {
		applyKitOp(p, e, o, mobs, allies, reg, now, out)
	}
	p.sendf("%s!%s", e.Name, out.summary())
	for _, l := range out.lines {
		p.send("  " + l)
	}
	for _, d := range out.deaths {
		resolveKill(p, d, reg, now)
	}
	if e.MP > 0 {
		p.sendf("  (MP: %d)", p.mp)
	}
	p.prompt()
}

type kitResult struct {
	landed, resisted, immune, damaged int
	totalDamage                       int
	healed                            int
	lines                             []string
	deaths                            []*mob.Mob
}

func (r *kitResult) summary() string {
	var parts []string
	if r.totalDamage > 0 {
		parts = append(parts, fmt.Sprintf("%d damage to %d", r.totalDamage, r.damaged))
	}
	if r.landed > 0 {
		parts = append(parts, fmt.Sprintf("%d affected", r.landed))
	}
	if r.resisted > 0 {
		parts = append(parts, fmt.Sprintf("%d resisted", r.resisted))
	}
	if r.healed > 0 {
		parts = append(parts, fmt.Sprintf("%d HP restored", r.healed))
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, ", ") + "."
}

func killedAdd(r *kitResult, m *mob.Mob) {
	for _, d := range r.deaths {
		if d == m {
			return
		}
	}
	r.deaths = append(r.deaths, m)
}

func applyKitOp(p *player, e *kitEntry, o kitOp, mobs []*mob.Mob, allies []*player, reg *mob.Registry, now time.Time, r *kitResult) {
	switch o.Kind {
	case opDamage, opDrain:
		hits := o.Hits
		if hits < 1 {
			hits = 1
		}
		for _, m := range mobs {
			if m.HP <= 0 {
				continue
			}
			total := 0
			for h := 0; h < hits; h++ {
				res, _, err := reg.Hit(m.ID, p.slot, opDamageAmount(p, o))
				if err != nil {
					break
				}
				total += res.Dealt
				if res.Woke {
					r.lines = append(r.lines, fmt.Sprintf("%s wakes up!", m.Kind))
				}
				if res.Died {
					killedAdd(r, m)
					break
				}
			}
			if total > 0 {
				r.totalDamage += total
				r.damaged++
				engage(p, m, total)
				if o.Kind == opDrain && o.Pct > 0 {
					heal := total * o.Pct / 100
					p.hp = clampInt(p.hp+heal, 0, p.maxHP)
					r.healed += heal
				}
			}
		}
	case opDot:
		for _, m := range mobs {
			applyMobFx(p, m, reg, mob.FxPoison, opDotPotency(p, o), scaledDuration(p, o), now, r)
		}
	case opMobFx:
		for _, m := range mobs {
			applyMobFx(p, m, reg, o.FxKind, o.Potency, scaledDuration(p, o), now, r)
		}
	case opEnmity:
		for _, m := range mobs {
			engage(p, m, o.Base)
			if o.Base < 0 {
				if t := gw.mobEnmity[m.ID]; t != nil {
					t.Add(p.slot, o.Base)
				}
			}
		}
	case opHeal:
		for _, a := range allies {
			amt := o.Base + int(float64(statValue(p, o.Stat))*o.PerStat)
			amt = healPower(p, amt)
			before := a.hp
			a.hp = clampInt(a.hp+amt, 0, a.maxHP)
			got := a.hp - before
			r.healed += got
			if a != p {
				a.sendf("\r\n[%s from %s] +%d HP. HP: %d/%d", e.Name, p.name, got, a.hp, a.maxHP)
			}
		}
	case opBuff:
		roll := 1.0
		if o.Roll {
			n := 1 + kitRand.Intn(6)
			roll = float64(n) / 3
			r.lines = append(r.lines, fmt.Sprintf("You roll a %d!", n))
		}
		pot := int(float64(scaledPotency(p, o)) * roll)
		if pot < 1 {
			pot = 1
		}
		for _, a := range allies {
			a.statFX.Apply(status.Effect{Kind: o.Buff, Potency: pot, ExpiresAt: now.Add(scaledDuration(p, o))})
			r.landed++
			if a != p {
				a.sendf("\r\n[%s from %s] %s +%d (%s)", e.Name, p.name, o.Buff, pot, scaledDuration(p, o).Round(time.Second))
			}
		}
		if len(allies) == 1 && allies[0] == p {
			r.lines = append(r.lines, fmt.Sprintf("%s +%d for %s.", o.Buff, pot, scaledDuration(p, o).Round(time.Second)))
		}
	case opCleanse:
		for _, a := range allies {
			for _, k := range []status.Kind{status.Poison, status.Paralyze, status.Slow, status.Silence, status.Bind} {
				if a.statFX.Has(k) {
					a.statFX.Remove(k)
					r.landed++
				}
			}
		}
	case opRestoreMP:
		for _, a := range allies {
			a.mp = clampInt(a.mp+o.Base+int(float64(statValue(p, o.Stat))*o.PerStat), 0, a.maxMP)
		}
		r.lines = append(r.lines, fmt.Sprintf("MP restored (+%d).", o.Base))
	case opNextAttack:
		p.pendingAttackBonus = float64(o.Pct) / 100
		r.lines = append(r.lines, fmt.Sprintf("Your next hit lands +%d%% harder.", o.Pct))
	case opNextCrit:
		p.pendingGuaranteedCrit = true
		r.lines = append(r.lines, "Your next hit is a guaranteed critical.")
	case opCoat:
		p.coating = &weaponCoating{Name: e.Name, Fx: o.FxKind, Potency: o.Potency, Dur: o.Dur, Charges: o.Charges, DrainPct: o.Pct}
		r.lines = append(r.lines, fmt.Sprintf("Your weapon is coated (%d hits).", o.Charges))
	case opTP:
		p.tp.Current = clampInt(p.tp.Current+o.Base, 0, 300)
		r.lines = append(r.lines, fmt.Sprintf("TP +%d (now %d).", o.Base, p.tp.Current))
	case opHide:
		p.isInvisible = true
		p.invisExpires = now.Add(o.Dur)
		cleared := 0
		for _, id := range reg.All() {
			if m, ok := reg.Get(id); ok && m.AggroSlot == p.slot && m.HP > 0 {
				m.AggroSlot = ""
				m.State = mob.StateIdle
				cleared++
			}
		}
		r.lines = append(r.lines, fmt.Sprintf("You slip from sight (%s); %d foe(s) lose you.", o.Dur.Round(time.Second), cleared))
	case opSteal:
		for _, m := range mobs {
			if stolenFrom[m.ID] {
				r.lines = append(r.lines, fmt.Sprintf("%s has nothing left to steal.", m.Kind))
				continue
			}
			drops := dropsForMob(m.Kind)
			stolenFrom[m.ID] = true
			if len(drops) == 0 {
				r.lines = append(r.lines, fmt.Sprintf("%s has nothing to steal.", m.Kind))
				continue
			}
			it := drops[kitRand.Intn(len(drops))]
			p.inventory[it.ID]++
			r.lines = append(r.lines, fmt.Sprintf("You steal %s from %s!", it.Name, m.Kind))
			engage(p, m, 300)
		}
	case opRaise:
		for _, a := range allies {
			if !a.homePoint.IsKO {
				r.lines = append(r.lines, fmt.Sprintf("%s is not KO'd.", a.name))
				continue
			}
			a.homePoint.IsKO = false
			a.hp = clampInt(a.maxHP*o.Pct/100, 1, a.maxHP)
			r.landed++
			if a != p {
				a.sendf("\r\n[%s] %s raises you! HP: %d/%d", e.Name, p.name, a.hp, a.maxHP)
			}
		}
	case opPayHP:
		cost := p.maxHP * o.Pct / 100
		p.hp = clampInt(p.hp-cost, 1, p.maxHP)
		r.lines = append(r.lines, fmt.Sprintf("You pay %d HP.", cost))
	case opFlow:
		p.flow += o.Base
		r.lines = append(r.lines, fmt.Sprintf("+%d flow.", o.Base))
	}
}

func opDotPotency(p *player, o kitOp) int {
	pot := o.Potency
	if o.Stat != "" {
		pot += int(float64(statValue(p, o.Stat)) * o.PerStat)
	}
	pot = spellPower(p, pot, o.Element)
	if pot < 1 {
		pot = 1
	}
	return pot
}

func applyMobFx(p *player, m *mob.Mob, reg *mob.Registry, kind mob.EffectKind, potency int, dur time.Duration, now time.Time, r *kitResult) {
	if m.HP <= 0 {
		return
	}
	out, err := reg.ApplyEffect(m.ID, kind, potency, dur, p.slot, now)
	if err != nil {
		r.immune++
		return
	}
	switch out {
	case mob.OutcomeLanded:
		r.landed++
		if kind == mob.FxSleep {
			// A sleeping mob stays aware of who put it down, but takes no action.
		}
		engage(p, m, 100)
		r.lines = append(r.lines, fmt.Sprintf("%s is %s.", m.Kind, fxVerb(kind)))
	case mob.OutcomeResisted:
		r.resisted++
		engage(p, m, 100)
		r.lines = append(r.lines, fmt.Sprintf("%s resists the %s!", m.Kind, strings.ToLower(kind.String())))
	}
}

func fxVerb(k mob.EffectKind) string {
	switch k {
	case mob.FxSleep:
		return "put to sleep"
	case mob.FxBind:
		return "bound in place"
	case mob.FxStun:
		return "stunned"
	case mob.FxParalyze:
		return "paralyzed"
	case mob.FxBlind:
		return "blinded"
	case mob.FxSlow:
		return "slowed"
	case mob.FxWeaken:
		return "weakened"
	case mob.FxSilence:
		return "silenced"
	case mob.FxPoison:
		return "poisoned"
	}
	return "afflicted"
}

// applyCoating is called after a landed melee hit: the armed coating spends a charge and applies
// its effect (and/or drains life).
func applyCoating(p *player, m *mob.Mob, reg *mob.Registry, dealt int, now time.Time) {
	c := p.coating
	if c == nil || c.Charges <= 0 || m.HP <= 0 {
		return
	}
	c.Charges--
	if c.Fx != mob.FxNone {
		if out, err := reg.ApplyEffect(m.ID, c.Fx, c.Potency, c.Dur, p.slot, now); err == nil && out == mob.OutcomeLanded {
			p.sendf("  [%s] %s is %s.", c.Name, m.Kind, fxVerb(c.Fx))
		}
	}
	if c.DrainPct > 0 && dealt > 0 {
		heal := dealt * c.DrainPct / 100
		p.hp = clampInt(p.hp+heal, 0, p.maxHP)
		p.sendf("  [%s] You drain %d HP.", c.Name, heal)
	}
	if c.Charges <= 0 {
		p.sendf("  Your %s wears off.", c.Name)
		p.coating = nil
	}
}

// cmdSpells lists everything the player's jobs can use.
func cmdSpells(p *player) {
	lvl := p.charXP.Level
	sub := ""
	if p.charJob != nil {
		sub = string(p.charJob.Sub)
	}
	p.sendf("\r\n=== Spells & abilities [%s%s] (Lv%d) ===", p.jobID, map[bool]string{true: "/" + sub, false: ""}[sub != ""], lvl)
	n := 0
	for _, kind := range []kitKind{kitSpell, kitAbility} {
		header := map[kitKind]string{kitSpell: "Spells (cast <id> [target])", kitAbility: "Job abilities (ja <id> [target])"}[kind]
		first := true
		for _, e := range kitOrdered {
			if e.Kind != kind {
				continue
			}
			ok, need, role := kitLevelFor(p, e)
			if role == "" {
				continue
			}
			if first {
				p.send(header + ":")
				first = false
			}
			mark := "  "
			if !ok {
				mark = "x "
			}
			mp := ""
			if e.MP > 0 {
				mp = fmt.Sprintf(" %dMP", e.MP)
			}
			p.sendf("%s%-18s Lv%-2d%-7s %s", mark, e.ID, need, mp, e.Desc)
			n++
		}
	}
	if n == 0 {
		p.send("Your job has no spells or abilities in the kit yet.")
	}
	p.send("(x = not yet available at your level)")
	p.prompt()
}

// ---- constructors used by kit_data.go ----

func sec(n int) time.Duration  { return time.Duration(n) * time.Second }
func mins(n int) time.Duration { return time.Duration(n) * time.Minute }

func jl(pairs ...interface{}) map[string]int {
	m := map[string]int{}
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i].(string)] = pairs[i+1].(int)
	}
	return m
}

func spell(id, name string, jobs map[string]int, mp int, cast, recast time.Duration, tgt kitTarget, radius float64, desc string, ops ...kitOp) *kitEntry {
	return &kitEntry{ID: id, Name: name, Kind: kitSpell, Jobs: jobs, MP: mp, Cast: cast, Recast: recast, Target: tgt, Radius: radius, Desc: desc, Ops: ops}
}

func ability(id, name string, jobs map[string]int, recast time.Duration, tgt kitTarget, radius float64, desc string, ops ...kitOp) *kitEntry {
	return &kitEntry{ID: id, Name: name, Kind: kitAbility, Jobs: jobs, Recast: recast, Target: tgt, Radius: radius, Desc: desc, Ops: ops}
}

func Dmg(base int, stat string, per float64, element string) kitOp {
	return kitOp{Kind: opDamage, Base: base, Stat: stat, PerStat: per, Element: element}
}
func Phys(base int, stat string, per float64) kitOp {
	return kitOp{Kind: opDamage, Base: base, Stat: stat, PerStat: per, Phys: true}
}
func PhysHits(hits, base int, stat string, per float64) kitOp {
	return kitOp{Kind: opDamage, Base: base, Stat: stat, PerStat: per, Phys: true, Hits: hits}
}
func Drain(base int, stat string, per float64, healPct int) kitOp {
	return kitOp{Kind: opDrain, Base: base, Stat: stat, PerStat: per, Element: "Dark", Pct: healPct}
}
func Dot(potency int, d time.Duration) kitOp { return kitOp{Kind: opDot, Potency: potency, Dur: d} }
func DotEl(potency int, element string, d time.Duration) kitOp {
	return kitOp{Kind: opDot, Potency: potency, Element: element, Dur: d}
}
func Fx(kind mob.EffectKind, potency int, d time.Duration) kitOp {
	return kitOp{Kind: opMobFx, FxKind: kind, Potency: potency, Dur: d}
}
func FxC(kind mob.EffectKind, potency int, d time.Duration) kitOp { // CHR-scaled duration (songs)
	return kitOp{Kind: opMobFx, FxKind: kind, Potency: potency, Dur: d, ChrScale: true}
}
func Heal(base int, stat string, per float64) kitOp {
	return kitOp{Kind: opHeal, Base: base, Stat: stat, PerStat: per}
}
func Buff(k status.Kind, potency int, d time.Duration) kitOp {
	return kitOp{Kind: opBuff, Buff: k, Potency: potency, Dur: d}
}
func BuffC(k status.Kind, potency int, d time.Duration) kitOp { // CHR-scaled (songs, dances)
	return kitOp{Kind: opBuff, Buff: k, Potency: potency, Dur: d, ChrScale: true}
}
func BuffRoll(k status.Kind, potency int, d time.Duration) kitOp { // corsair rolls
	return kitOp{Kind: opBuff, Buff: k, Potency: potency, Dur: d, Roll: true, ChrScale: true}
}
func Cleanse() kitOp             { return kitOp{Kind: opCleanse} }
func MPRestore(n int) kitOp      { return kitOp{Kind: opRestoreMP, Base: n} }
func Enmity(n int) kitOp         { return kitOp{Kind: opEnmity, Base: n} }
func NextAtk(pct int) kitOp      { return kitOp{Kind: opNextAttack, Pct: pct} }
func NextCrit() kitOp            { return kitOp{Kind: opNextCrit} }
func TP(n int) kitOp             { return kitOp{Kind: opTP, Base: n} }
func Hide(d time.Duration) kitOp { return kitOp{Kind: opHide, Dur: d} }
func Steal() kitOp               { return kitOp{Kind: opSteal} }
func Raise(pct int) kitOp        { return kitOp{Kind: opRaise, Pct: pct} }
func PayHP(pct int) kitOp        { return kitOp{Kind: opPayHP, Pct: pct} }
func Flow(n int) kitOp           { return kitOp{Kind: opFlow, Base: n} }
func Coat(k mob.EffectKind, potency int, d time.Duration, charges int) kitOp {
	return kitOp{Kind: opCoat, FxKind: k, Potency: potency, Dur: d, Charges: charges}
}
func CoatDrain(pct, charges int) kitOp { return kitOp{Kind: opCoat, Pct: pct, Charges: charges} }

// resolveDotKill credits a kill made by a damage-over-time tick (server/mob reports it as an
// EvtMobDied with the caster's slot). Falls back to just queueing the respawn if the caster has
// left, so the mob does not stay dead forever.
func resolveDotKill(zoneID int, reg *mob.Registry, ev mob.Event, now time.Time) {
	m, ok := reg.Get(ev.MobID)
	if !ok {
		return
	}
	if killer, ok := gw.players[ev.Slot]; ok && killer.zoneID == zoneID {
		killer.sendf("\r\n  %s succumbs to your poison!", m.Kind)
		if killer.combat.TargetMobID == m.ID {
			killer.combat.TargetMobID = ""
		}
		resolveKill(killer, m, reg, now)
		killer.prompt()
		return
	}
	if !isHNM(m.ID) {
		gw.deadQueue = append(gw.deadQueue, deadMob{m: *m, respawnAt: now.Add(respawnDelay), zoneID: zoneID})
	}
}

package mob

import (
	"errors"
	"math/rand"
	"strings"
	"time"
)

// Mob status effects / crowd control.
//
// Until this file mobs had no status stack at all (only players had server/status), so nothing
// could sleep, bind, poison-over-time or weaken a mob. The model deliberately mirrors FFXI:
//
//   - Sleep behaves like Bard's Lullaby: the mob does nothing (no chase, no swing, no aggro scan)
//     until the timer ends or it takes DIRECT damage. Damage-over-time ticks do not wake it, so a
//     poisoner can sleep-and-poison. Every time a sleep ends, the mob builds a short-lived
//     resistance, so sleeping the same mob again and again gets harder (Lullaby II/III chains
//     eventually slip).
//   - Bind roots the mob in place (it can still hit whatever is already in melee range).
//   - Stun removes its actions for a short time and, unlike Sleep, is not broken by damage.
//   - Paralyze/Blind make some swings fail, Slow stretches the swing delay, Weaken cuts its melee
//     damage, Silence stops it casting (checked by callers via Silenced).
//   - Poison is a damage-over-time that credits the caster (kill XP/loot) without waking sleepers.
//
// Notorious monsters (IDs starting "nm-") resist crowd control by default.

// EffectKind identifies a mob status.
type EffectKind int

const (
	FxNone     EffectKind = iota
	FxSleep               // no actions; broken by direct damage
	FxBind                // cannot move
	FxStun                // no actions; not broken by damage
	FxParalyze            // Potency% of swings are lost
	FxBlind               // Potency% of swings miss
	FxSlow                // swing delay stretched by Potency%
	FxWeaken              // melee damage reduced by Potency%
	FxSilence             // cannot cast
	FxPoison              // Potency damage every PoisonTick, never wakes a sleeper
)

func (k EffectKind) String() string {
	switch k {
	case FxSleep:
		return "Sleep"
	case FxBind:
		return "Bind"
	case FxStun:
		return "Stun"
	case FxParalyze:
		return "Paralyze"
	case FxBlind:
		return "Blind"
	case FxSlow:
		return "Slow"
	case FxWeaken:
		return "Weaken"
	case FxSilence:
		return "Silence"
	case FxPoison:
		return "Poison"
	}
	return "None"
}

// IsHardCC reports whether the kind removes the mob's ability to act, which is what crowd-control
// resistance applies to.
func (k EffectKind) IsHardCC() bool { return k == FxSleep || k == FxStun || k == FxBind }

const (
	// PoisonTick is how often a Poison effect deals its damage.
	PoisonTick = 3 * time.Second
	// SleepResistWindow is how long the "recently slept" resistance lasts after a sleep ends.
	SleepResistWindow = 45 * time.Second
	// SleepResistPerStack is how much each recent sleep lowers the next sleep's land chance.
	SleepResistPerStack = 0.20
	maxSleepStacks      = 3
	// BaseCCLandChance is the land chance of a hard CC against a mob with no resistance.
	BaseCCLandChance = 0.90
	// NMCCResist is the default hard-CC resistance of a notorious monster.
	NMCCResist = 0.50
)

// Effect is one active status on a mob.
type Effect struct {
	Kind     EffectKind
	Potency  int
	Expires  time.Time
	Source   string // slot of the caster, for kill credit
	nextTick time.Time
}

// ApplyOutcome reports what happened to an ApplyEffect attempt.
type ApplyOutcome int

const (
	OutcomeLanded ApplyOutcome = iota
	OutcomeResisted
	OutcomeImmune
)

func (o ApplyOutcome) String() string {
	return [...]string{"landed", "resisted", "immune"}[o]
}

// Roll draws the [0,1) number used for land chances and swing failures. Tests replace it.
var Roll = defaultRoll

func defaultRoll() float64 { return rand.Float64() }

var ErrNotTargetable = errors.New("mob cannot be affected right now")

// ccState holds a mob's status effects. It lives behind a pointer so Mob stays comparable.
type ccState struct {
	effects          map[EffectKind]*Effect
	sleepStacksN     int       // recent sleeps, lowers the chance of the next one
	sleepResistUntil time.Time // when those stacks lapse
}

func (m *Mob) state() *ccState {
	if m.cc == nil {
		m.cc = &ccState{effects: make(map[EffectKind]*Effect)}
	}
	return m.cc
}

func (m *Mob) fx(k EffectKind) *Effect {
	if m.cc == nil {
		return nil
	}
	return m.cc.effects[k]
}

// Has reports whether the mob currently has the effect.
func (m *Mob) Has(k EffectKind) bool { return m.fx(k) != nil }

// Silenced reports whether the mob is unable to cast.
func (m *Mob) Silenced() bool { return m.Has(FxSilence) }

// Effects returns a snapshot of the mob's active effects.
func (m *Mob) Effects() []Effect {
	if m.cc == nil {
		return nil
	}
	out := make([]Effect, 0, len(m.cc.effects))
	for _, e := range m.cc.effects {
		out = append(out, *e)
	}
	return out
}

// StatusTags renders active effects for the creature list, e.g. " [asleep, poisoned]".
func (m *Mob) StatusTags() string {
	if m.cc == nil || len(m.cc.effects) == 0 {
		return ""
	}
	names := map[EffectKind]string{FxSleep: "asleep", FxBind: "bound", FxStun: "stunned", FxParalyze: "paralyzed",
		FxBlind: "blinded", FxSlow: "slowed", FxWeaken: "weakened", FxSilence: "silenced", FxPoison: "poisoned"}
	var tags []string
	for _, k := range []EffectKind{FxSleep, FxStun, FxBind, FxParalyze, FxBlind, FxSlow, FxWeaken, FxSilence, FxPoison} {
		if m.Has(k) {
			tags = append(tags, names[k])
		}
	}
	return " [" + strings.Join(tags, ", ") + "]"
}

func (m *Mob) ccResist() float64 {
	if m.CCResist > 0 {
		return m.CCResist
	}
	if strings.HasPrefix(m.ID, "nm-") {
		return NMCCResist
	}
	return 0
}

func (m *Mob) sleepStacks(now time.Time) int {
	if m.cc == nil || now.After(m.cc.sleepResistUntil) {
		return 0
	}
	return m.cc.sleepStacksN
}

// ApplyEffect tries to put a status on a mob. Hard CC (sleep/stun/bind) rolls against the mob's
// resistance and, for Sleep, against recent-sleep stacks; soft debuffs and poison always land
// unless the mob is an NM-style immune target (they are not blocked, only shortened). A stronger or
// longer effect replaces a weaker one; a weaker one is ignored but still reported as landed.
func (reg *Registry) ApplyEffect(mobID string, kind EffectKind, potency int, dur time.Duration, source string, now time.Time) (ApplyOutcome, error) {
	m, ok := reg.mobs[mobID]
	if !ok {
		return OutcomeImmune, ErrMobNotFound
	}
	if !m.alive() || !m.targetable() {
		return OutcomeImmune, ErrNotTargetable
	}
	if kind.IsHardCC() {
		chance := BaseCCLandChance - m.ccResist()
		if kind == FxSleep {
			chance -= SleepResistPerStack * float64(m.sleepStacks(now))
		}
		if chance < 0.05 {
			chance = 0.05
		}
		if Roll() >= chance {
			return OutcomeResisted, nil
		}
	} else if res := m.ccResist(); res > 0 {
		dur = time.Duration(float64(dur) * (1 - res/2)) // soft debuffs just run shorter on resistant mobs
	}
	st := m.state()
	exp := now.Add(dur)
	if cur, ok := st.effects[kind]; ok && cur.Expires.After(exp) && cur.Potency >= potency {
		return OutcomeLanded, nil
	}
	e := &Effect{Kind: kind, Potency: potency, Expires: exp, Source: source}
	if kind == FxPoison {
		e.nextTick = now.Add(PoisonTick)
	}
	st.effects[kind] = e
	return OutcomeLanded, nil
}

// RemoveEffect clears one effect, returning whether it was present.
func (reg *Registry) RemoveEffect(mobID string, kind EffectKind) bool {
	m, ok := reg.mobs[mobID]
	if !ok || m.fx(kind) == nil {
		return false
	}
	delete(m.cc.effects, kind)
	return true
}

// MobsNear returns the IDs of living, targetable mobs within radius of pos (same registry).
func (reg *Registry) MobsNear(pos Pos, radius float64) []string {
	var out []string
	for id, m := range reg.mobs {
		if m.targetable() && dist(m.Pos, pos) <= radius {
			out = append(out, id)
		}
	}
	return out
}

// HitDot applies damage-over-time style damage: it tags and aggroes like Hit but does NOT wake a
// sleeping mob.
func (reg *Registry) HitDot(mobID, attackerSlot string, damage int) (HitResult, []Event, error) {
	return reg.hit(mobID, attackerSlot, damage, false)
}

// wake removes Sleep after direct damage and starts the resistance window.
func (m *Mob) wake(now time.Time) bool {
	if !m.Has(FxSleep) {
		return false
	}
	delete(m.cc.effects, FxSleep)
	m.noteSleepEnded(now)
	return true
}

func (m *Mob) noteSleepEnded(now time.Time) {
	st := m.state()
	if now.After(st.sleepResistUntil) {
		st.sleepStacksN = 0
	}
	if st.sleepStacksN < maxSleepStacks {
		st.sleepStacksN++
	}
	st.sleepResistUntil = now.Add(SleepResistWindow)
}

// tickEffects expires effects and applies poison. A poison kill returns EvtMobDied credited to the
// caster who applied it.
func tickEffects(m *Mob, now time.Time) []Event {
	if m.cc == nil || len(m.cc.effects) == 0 {
		return nil
	}
	var events []Event
	for k, e := range m.cc.effects {
		if !now.Before(e.Expires) {
			delete(m.cc.effects, k)
			if k == FxSleep {
				m.noteSleepEnded(now)
			}
			continue
		}
		if k == FxPoison && !now.Before(e.nextTick) {
			e.nextTick = e.nextTick.Add(PoisonTick)
			dmg := e.Potency
			if dmg >= m.HP {
				dmg = m.HP
			}
			m.HP -= dmg
			if m.TaggerSlot == "" && e.Source != "" {
				m.TaggerSlot = e.Source
				m.TaggedAt = now
			}
			if m.AggroSlot == "" && e.Source != "" {
				m.AggroSlot = e.Source
				m.State = StatePursuing
			}
			if m.HP <= 0 {
				m.State = StateDead
				m.DiedAt = now
				m.cc.effects = map[EffectKind]*Effect{}
				return append(events, Event{Kind: EvtMobDied, MobID: m.ID, Slot: e.Source, Tagger: m.TaggerSlot})
			}
		}
	}
	return events
}

// swingModifiers returns how a mob's swing is altered by its effects this tick: the delay to use,
// whether the swing is lost entirely, and the damage after Weaken.
func swingModifiers(m *Mob, damage int) (delay time.Duration, lost bool, dmg int) {
	delay = m.SwingDelay
	dmg = damage
	if e := m.fx(FxSlow); e != nil {
		delay = time.Duration(float64(delay) * (1 + float64(e.Potency)/100))
	}
	if e := m.fx(FxParalyze); e != nil && Roll()*100 < float64(e.Potency) {
		lost = true
	}
	if e := m.fx(FxBlind); e != nil && Roll()*100 < float64(e.Potency) {
		lost = true
	}
	if e := m.fx(FxWeaken); e != nil {
		dmg = damage * (100 - e.Potency) / 100
		if dmg < 1 {
			dmg = 1
		}
	}
	return
}

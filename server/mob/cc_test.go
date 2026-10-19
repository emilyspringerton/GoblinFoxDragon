package mob

import (
	"testing"
	"time"
)

func ccFixture(t *testing.T) (*Registry, time.Time) {
	t.Helper()
	Roll = func() float64 { return 0 } // always land / always fail swing chances
	t.Cleanup(func() { Roll = defaultRoll })
	reg := New()
	m := Mob{ID: "m1", Kind: "wolf", HP: 100, MaxHP: 100, Pos: Pos{X: 1}, HomePos: Pos{X: 1}, AggroRange: 20,
		MeleeRange: 2, MoveSpeed: 5, SwingDelay: time.Second, MeleeDamage: 10, State: StateIdle}
	if err := reg.Spawn(m); err != nil {
		t.Fatal(err)
	}
	return reg, time.Now()
}

var near = []PlayerPositions{{Slot: "p", Pos: Pos{X: 0}}}

func TestSleepFreezesMobAndDamageWakesIt(t *testing.T) {
	reg, now := ccFixture(t)
	if out, _ := reg.ApplyEffect("m1", FxSleep, 0, 30*time.Second, "p", now); out != OutcomeLanded {
		t.Fatalf("sleep should land, got %v", out)
	}
	for i := 0; i < 5; i++ {
		if evs := reg.Tick(now.Add(time.Duration(i)*time.Second), 1, near); len(evs) != 0 {
			t.Fatalf("sleeping mob acted: %+v", evs)
		}
	}
	m, _ := reg.Get("m1")
	if m.State != StateIdle {
		t.Errorf("sleeping mob should not aggro, state=%v", m.State)
	}
	res, _, _ := reg.Hit("m1", "p", 5)
	if !res.Woke || m.Has(FxSleep) {
		t.Fatalf("direct damage must wake a sleeper")
	}
	if evs := reg.Tick(now.Add(6*time.Second), 1, near); len(evs) == 0 {
		t.Errorf("woken mob should aggro and act")
	}
}

func TestPoisonTicksWithoutWakingAndCreditsCaster(t *testing.T) {
	reg, now := ccFixture(t)
	reg.ApplyEffect("m1", FxSleep, 0, time.Minute, "p", now)
	reg.ApplyEffect("m1", FxPoison, 40, time.Minute, "p", now)
	var died bool
	var slot string
	for i := 1; i <= 12 && !died; i++ {
		for _, e := range reg.Tick(now.Add(time.Duration(i)*PoisonTick), 1, near) {
			if e.Kind == EvtMobDied {
				died, slot = true, e.Slot
			}
		}
	}
	m, _ := reg.Get("m1")
	if !died || slot != "p" {
		t.Fatalf("poison should kill (100hp / 40 per tick) and credit the caster, died=%v slot=%q", died, slot)
	}
	_ = m
}

func TestPoisonDoesNotWakeSleeper(t *testing.T) {
	reg, now := ccFixture(t)
	reg.ApplyEffect("m1", FxSleep, 0, time.Minute, "p", now)
	reg.ApplyEffect("m1", FxPoison, 5, time.Minute, "p", now)
	reg.Tick(now.Add(PoisonTick), 1, near)
	m, _ := reg.Get("m1")
	if m.HP != 95 || !m.Has(FxSleep) {
		t.Fatalf("poison should tick and leave sleep intact, hp=%d asleep=%v", m.HP, m.Has(FxSleep))
	}
	if res, _, _ := reg.HitDot("m1", "p", 1); res.Woke {
		t.Errorf("HitDot must not wake")
	}
}

func TestSleepExpiresOnItsOwn(t *testing.T) {
	reg, now := ccFixture(t)
	reg.ApplyEffect("m1", FxSleep, 0, 10*time.Second, "p", now)
	reg.Tick(now.Add(11*time.Second), 1, near)
	m, _ := reg.Get("m1")
	if m.Has(FxSleep) {
		t.Fatal("sleep should have expired")
	}
}

func TestRepeatedSleepGetsHarderAndNMsResist(t *testing.T) {
	reg, now := ccFixture(t)
	Roll = func() float64 { return 0.65 } // lands at 0.90, fails once stacks pull chance under 0.65
	m, _ := reg.Get("m1")
	if out, _ := reg.ApplyEffect("m1", FxSleep, 0, time.Minute, "p", now); out != OutcomeLanded {
		t.Fatalf("first sleep should land")
	}
	reg.Hit("m1", "p", 1) // wakes -> 1 stack (chance 0.70)
	if out, _ := reg.ApplyEffect("m1", FxSleep, 0, time.Minute, "p", now); out != OutcomeLanded {
		t.Fatalf("second sleep at 0.70 should still land vs roll 0.65")
	}
	reg.Hit("m1", "p", 1) // 2 stacks (chance 0.50)
	if out, _ := reg.ApplyEffect("m1", FxSleep, 0, time.Minute, "p", now); out != OutcomeResisted {
		t.Fatalf("third sleep should be resisted after two recent sleeps")
	}
	if out, _ := reg.ApplyEffect("m1", FxSleep, 0, time.Minute, "p", now.Add(SleepResistWindow+time.Second)); out != OutcomeLanded {
		t.Fatalf("resistance should lapse after the window")
	}
	_ = m
	// NM: 0.90-0.50 = 0.40 land chance
	reg.Spawn(Mob{ID: "nm-boss", Kind: "Boss", HP: 500, MaxHP: 500})
	if out, _ := reg.ApplyEffect("nm-boss", FxSleep, 0, time.Minute, "p", now); out != OutcomeResisted {
		t.Errorf("NM should resist a 0.65 roll against 0.40 chance")
	}
}

func TestBindStopsMovementButNotMeleeInRange(t *testing.T) {
	reg, now := ccFixture(t)
	far := []PlayerPositions{{Slot: "p", Pos: Pos{X: -30}}}
	reg.ApplyEffect("m1", FxBind, 0, time.Minute, "p", now)
	m, _ := reg.Get("m1")
	m.State, m.AggroSlot = StatePursuing, "p"
	reg.Tick(now, 1, far)
	if m.Pos.X != 1 {
		t.Errorf("bound mob moved to %v", m.Pos)
	}
	m.Pos = Pos{X: 1}
	if evs := reg.Tick(now.Add(5*time.Second), 1, near); len(evs) != 1 || evs[0].Kind != EvtMobAttack {
		t.Errorf("bound mob in melee range should still swing: %+v", evs)
	}
}

func TestSlowParalyzeWeakenAndStun(t *testing.T) {
	reg, now := ccFixture(t)
	m, _ := reg.Get("m1")
	m.State, m.AggroSlot, m.Pos = StatePursuing, "p", Pos{X: 1}
	reg.ApplyEffect("m1", FxWeaken, 50, time.Minute, "p", now)
	evs := reg.Tick(now.Add(2*time.Second), 1, near)
	if len(evs) != 1 || evs[0].Damage != 5 {
		t.Fatalf("weaken should halve damage: %+v", evs)
	}
	reg.ApplyEffect("m1", FxSlow, 100, time.Minute, "p", now)
	if evs := reg.Tick(now.Add(3*time.Second), 1, near); len(evs) != 0 {
		t.Errorf("slowed mob (2s delay) swung after 1s: %+v", evs)
	}
	reg.RemoveEffect("m1", FxSlow)
	reg.ApplyEffect("m1", FxParalyze, 100, time.Minute, "p", now)
	if evs := reg.Tick(now.Add(10*time.Second), 1, near); len(evs) != 0 {
		t.Errorf("fully paralyzed mob swung: %+v", evs)
	}
	reg.RemoveEffect("m1", FxParalyze)
	reg.ApplyEffect("m1", FxStun, 0, 5*time.Second, "p", now.Add(20*time.Second))
	reg.Hit("m1", "p", 1)
	if !m.Has(FxStun) {
		t.Errorf("stun must survive damage")
	}
	if evs := reg.Tick(now.Add(22*time.Second), 1, near); len(evs) != 0 {
		t.Errorf("stunned mob acted: %+v", evs)
	}
}

func TestMobsNearAndReviveClearsEffects(t *testing.T) {
	reg, now := ccFixture(t)
	reg.Spawn(Mob{ID: "m2", Kind: "wolf", HP: 10, MaxHP: 10, Pos: Pos{X: 50}})
	if ids := reg.MobsNear(Pos{X: 0}, 5); len(ids) != 1 || ids[0] != "m1" {
		t.Errorf("MobsNear = %v", ids)
	}
	reg.ApplyEffect("m1", FxPoison, 3, time.Minute, "p", now)
	reg.Revive("m1", Pos{})
	if m, _ := reg.Get("m1"); m.Has(FxPoison) {
		t.Errorf("revive should clear effects")
	}
}

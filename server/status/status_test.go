package status

import (
	"testing"
	"time"
)

// helpers

func future(d time.Duration) time.Time { return time.Now().Add(d) }
func past(d time.Duration) time.Time   { return time.Now().Add(-d) }

func makeEffect(k Kind, potency int, dur time.Duration) Effect {
	e := Effect{Kind: k, Potency: potency, AppliedAt: time.Now()}
	if dur > 0 {
		e.ExpiresAt = future(dur)
	}
	return e
}

// --- Kind.String() ---

func TestKindString_AllNamed(t *testing.T) {
	cases := []struct {
		k    Kind
		want string
	}{
		{Poison, "Poison"},
		{Paralyze, "Paralyze"},
		{Slow, "Slow"},
		{Silence, "Silence"},
		{Bind, "Bind"},
		{Haste, "Haste"},
		{Regen, "Regen"},
		{Refresh, "Refresh"},
		{Protect, "Protect"},
		{Shell, "Shell"},
	}
	for _, c := range cases {
		if got := c.k.String(); got != c.want {
			t.Errorf("Kind(%d).String() = %q, want %q", int(c.k), got, c.want)
		}
	}
}

func TestKindString_Unknown(t *testing.T) {
	got := Kind(99).String()
	if got == "" || got == "Unknown" {
		// acceptable: as long as it doesn't panic
		return
	}
}

// --- Cat ---

func TestCat_DebuffsClassifiedCorrectly(t *testing.T) {
	debuffs := []Kind{Poison, Paralyze, Slow, Silence, Bind}
	for _, k := range debuffs {
		if Cat(k) != CategoryDebuff {
			t.Errorf("Cat(%v) should be CategoryDebuff", k)
		}
	}
}

func TestCat_BuffsClassifiedCorrectly(t *testing.T) {
	buffs := []Kind{Haste, Regen, Refresh, Protect, Shell}
	for _, k := range buffs {
		if Cat(k) != CategoryBuff {
			t.Errorf("Cat(%v) should be CategoryBuff", k)
		}
	}
}

// --- Apply: basic ---

func TestApply_NewEffect_Added(t *testing.T) {
	s := New()
	result := s.Apply(makeEffect(Poison, 5, time.Minute))
	if result != ApplyAdded {
		t.Errorf("Apply new effect: got %d, want ApplyAdded", result)
	}
	if !s.Has(Poison) {
		t.Error("Poison should be active after Apply")
	}
}

func TestApply_StrongerOverwrites(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Poison, 5, time.Minute))
	result := s.Apply(makeEffect(Poison, 10, time.Minute))
	if result != ApplyUpgraded {
		t.Errorf("stronger Apply: got %d, want ApplyUpgraded", result)
	}
	e, ok := s.Get(Poison)
	if !ok {
		t.Fatal("Poison not found after upgrade")
	}
	if e.Potency != 10 {
		t.Errorf("after upgrade: Potency=%d, want 10", e.Potency)
	}
}

func TestApply_WeakerRejected(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Poison, 10, time.Minute))
	result := s.Apply(makeEffect(Poison, 3, time.Minute))
	if result != ApplyRejected {
		t.Errorf("weaker Apply: got %d, want ApplyRejected", result)
	}
	e, _ := s.Get(Poison)
	if e.Potency != 10 {
		t.Errorf("weaker apply should not change Potency: got %d", e.Potency)
	}
}

func TestApply_SamePotencyRefreshesDuration(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Poison, 5, 30*time.Second))
	newExpiry := future(2 * time.Minute)
	e2 := Effect{Kind: Poison, Potency: 5, AppliedAt: time.Now(), ExpiresAt: newExpiry}
	result := s.Apply(e2)
	if result != ApplyRefreshed {
		t.Errorf("same potency Apply: got %d, want ApplyRefreshed", result)
	}
	got, _ := s.Get(Poison)
	if !got.ExpiresAt.Equal(newExpiry) {
		t.Errorf("duration not refreshed: got %v, want %v", got.ExpiresAt, newExpiry)
	}
	if got.Potency != 5 {
		t.Errorf("potency changed on refresh: got %d", got.Potency)
	}
}

func TestApply_PermanentEffect(t *testing.T) {
	s := New()
	e := Effect{Kind: Regen, Potency: 10} // no ExpiresAt → permanent
	s.Apply(e)
	if !s.Has(Regen) {
		t.Error("permanent Regen should be active")
	}
}

// --- Has / Get ---

func TestHas_FalseOnEmpty(t *testing.T) {
	s := New()
	if s.Has(Poison) {
		t.Error("empty stack: Has(Poison) should be false")
	}
}

func TestGet_ReturnsEffect(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Haste, 20, time.Minute))
	e, ok := s.Get(Haste)
	if !ok {
		t.Fatal("Get(Haste): not found")
	}
	if e.Kind != Haste || e.Potency != 20 {
		t.Errorf("Get returned wrong effect: %+v", e)
	}
}

func TestGet_FalseWhenAbsent(t *testing.T) {
	s := New()
	_, ok := s.Get(Slow)
	if ok {
		t.Error("Get(Slow) on empty stack should return false")
	}
}

// --- Remove / DispelOne ---

func TestRemove_RemovesEffect(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Bind, 1, time.Minute))
	removed := s.Remove(Bind)
	if !removed {
		t.Error("Remove(Bind) should return true")
	}
	if s.Has(Bind) {
		t.Error("Bind should be gone after Remove")
	}
}

func TestRemove_FalseWhenAbsent(t *testing.T) {
	s := New()
	if s.Remove(Silence) {
		t.Error("Remove on absent effect should return false")
	}
}

func TestDispelOne_RemovesDebuff(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Poison, 5, time.Minute))
	s.Apply(makeEffect(Haste, 15, time.Minute)) // buff — should not be removed
	e, ok := s.DispelOne(CategoryDebuff)
	if !ok {
		t.Fatal("DispelOne(debuff): nothing removed")
	}
	if Cat(e.Kind) != CategoryDebuff {
		t.Errorf("DispelOne removed wrong category: %v", e.Kind)
	}
	if s.Has(Haste) == false {
		t.Error("DispelOne(debuff) should not remove buffs")
	}
}

func TestDispelOne_FalseWhenNoneOfCategory(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Regen, 3, time.Minute))
	_, ok := s.DispelOne(CategoryDebuff)
	if ok {
		t.Error("DispelOne(debuff) on buff-only stack should return false")
	}
	if !s.Has(Regen) {
		t.Error("Regen should still be active")
	}
}

// --- All ---

func TestAll_ReturnsAllEffects(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Poison, 5, time.Minute))
	s.Apply(makeEffect(Regen, 3, time.Minute))
	all := s.All()
	if len(all) != 2 {
		t.Errorf("All() returned %d effects, want 2", len(all))
	}
}

func TestAll_EmptyOnNewStack(t *testing.T) {
	s := New()
	if len(s.All()) != 0 {
		t.Error("All() on new stack should be empty")
	}
}

// --- Tick: expiry ---

func TestTick_ExpiresEffect(t *testing.T) {
	s := New()
	e := Effect{Kind: Bind, Potency: 1, AppliedAt: time.Now(), ExpiresAt: past(time.Second)}
	s.Apply(e)
	res := s.Tick(time.Now())
	if len(res.Expired) != 1 {
		t.Errorf("Tick: expected 1 expired, got %d", len(res.Expired))
	}
	if res.Expired[0].Kind != Bind {
		t.Errorf("wrong expired kind: %v", res.Expired[0].Kind)
	}
	if s.Has(Bind) {
		t.Error("Bind should be gone after expiry")
	}
}

func TestTick_RetainsNonExpired(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Haste, 20, time.Minute)) // expires in the future
	res := s.Tick(time.Now())
	if len(res.Expired) != 0 {
		t.Errorf("no expiry expected, got %d expired", len(res.Expired))
	}
	if !s.Has(Haste) {
		t.Error("Haste should still be active")
	}
}

func TestTick_PermanentEffectNeverExpires(t *testing.T) {
	s := New()
	e := Effect{Kind: Protect, Potency: 40} // no ExpiresAt
	s.Apply(e)
	// tick far into the future
	res := s.Tick(time.Now().Add(365 * 24 * time.Hour))
	if len(res.Expired) != 0 {
		t.Error("permanent effect should not expire via Tick")
	}
	if !s.Has(Protect) {
		t.Error("permanent Protect should still be active")
	}
}

func TestTick_MultipleEffects_OnlyExpiresExpired(t *testing.T) {
	s := New()
	s.Apply(Effect{Kind: Bind, Potency: 1, AppliedAt: time.Now(), ExpiresAt: past(time.Second)})
	s.Apply(makeEffect(Regen, 5, time.Minute))
	res := s.Tick(time.Now())
	if len(res.Expired) != 1 || res.Expired[0].Kind != Bind {
		t.Errorf("only Bind should expire: got %v", res.Expired)
	}
	if !s.Has(Regen) {
		t.Error("Regen should survive")
	}
}

// --- Tick: DoT / HoT events ---

func TestTick_PoisonEmitsNegativeHPEvent(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Poison, 8, time.Minute))
	res := s.Tick(time.Now())
	found := false
	for _, ev := range res.Events {
		if ev.Kind == Poison {
			found = true
			if ev.Target != TargetHP {
				t.Errorf("Poison tick target: got %d, want TargetHP", ev.Target)
			}
			if ev.Value != -8 {
				t.Errorf("Poison tick value: got %d, want -8", ev.Value)
			}
		}
	}
	if !found {
		t.Error("Poison should emit a tick event")
	}
}

func TestTick_RegenEmitsPositiveHPEvent(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Regen, 12, time.Minute))
	res := s.Tick(time.Now())
	found := false
	for _, ev := range res.Events {
		if ev.Kind == Regen {
			found = true
			if ev.Target != TargetHP {
				t.Errorf("Regen tick target: got %d, want TargetHP", ev.Target)
			}
			if ev.Value != 12 {
				t.Errorf("Regen tick value: got %d, want 12", ev.Value)
			}
		}
	}
	if !found {
		t.Error("Regen should emit a tick event")
	}
}

func TestTick_RefreshEmitsPositiveMPEvent(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Refresh, 3, time.Minute))
	res := s.Tick(time.Now())
	found := false
	for _, ev := range res.Events {
		if ev.Kind == Refresh {
			found = true
			if ev.Target != TargetMP {
				t.Errorf("Refresh tick target: got %d, want TargetMP", ev.Target)
			}
			if ev.Value != 3 {
				t.Errorf("Refresh tick value: got %d, want 3", ev.Value)
			}
		}
	}
	if !found {
		t.Error("Refresh should emit a tick event")
	}
}

func TestTick_PassiveBuffsDoNotEmitTickEvents(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Haste, 20, time.Minute))
	s.Apply(makeEffect(Protect, 40, time.Minute))
	s.Apply(makeEffect(Shell, 30, time.Minute))
	res := s.Tick(time.Now())
	if len(res.Events) != 0 {
		t.Errorf("passive buffs should not emit tick events, got %d events", len(res.Events))
	}
}

func TestTick_ExpiredEffectDoesNotEmitTickEvent(t *testing.T) {
	s := New()
	// Poison already expired
	s.Apply(Effect{Kind: Poison, Potency: 5, AppliedAt: time.Now(), ExpiresAt: past(time.Second)})
	res := s.Tick(time.Now())
	if len(res.Expired) != 1 {
		t.Fatalf("Poison should be in Expired, got %d", len(res.Expired))
	}
	for _, ev := range res.Events {
		if ev.Kind == Poison {
			t.Error("expired Poison should not emit a tick event")
		}
	}
}

// --- Combat integration helpers ---

func TestNetHastePct_HasteOnly(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Haste, 25, time.Minute))
	if got := s.NetHastePct(); got != 25 {
		t.Errorf("NetHastePct (Haste=25): got %d, want 25", got)
	}
}

func TestNetHastePct_SlowOnly(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Slow, 10, time.Minute))
	if got := s.NetHastePct(); got != -10 {
		t.Errorf("NetHastePct (Slow=10): got %d, want -10", got)
	}
}

func TestNetHastePct_HasteAndSlow(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Haste, 25, time.Minute))
	s.Apply(makeEffect(Slow, 10, time.Minute))
	if got := s.NetHastePct(); got != 15 {
		t.Errorf("NetHastePct (Haste=25, Slow=10): got %d, want 15", got)
	}
}

func TestNetHastePct_NeitherPresent(t *testing.T) {
	s := New()
	if got := s.NetHastePct(); got != 0 {
		t.Errorf("NetHastePct (none): got %d, want 0", got)
	}
}

func TestPhysDefBonus_WithProtect(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Protect, 40, time.Minute))
	if got := s.PhysDefBonus(); got != 40 {
		t.Errorf("PhysDefBonus: got %d, want 40", got)
	}
}

func TestPhysDefBonus_WithoutProtect(t *testing.T) {
	s := New()
	if got := s.PhysDefBonus(); got != 0 {
		t.Errorf("PhysDefBonus (none): got %d, want 0", got)
	}
}

func TestMagicDefBonus_WithShell(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Shell, 30, time.Minute))
	if got := s.MagicDefBonus(); got != 30 {
		t.Errorf("MagicDefBonus: got %d, want 30", got)
	}
}

func TestMagicDefBonus_WithoutShell(t *testing.T) {
	s := New()
	if got := s.MagicDefBonus(); got != 0 {
		t.Errorf("MagicDefBonus (none): got %d, want 0", got)
	}
}

func TestIsParalyzed(t *testing.T) {
	s := New()
	if s.IsParalyzed() {
		t.Error("IsParalyzed should be false on empty stack")
	}
	s.Apply(makeEffect(Paralyze, 20, time.Minute))
	if !s.IsParalyzed() {
		t.Error("IsParalyzed should be true when Paralyze is active")
	}
	s.Remove(Paralyze)
	if s.IsParalyzed() {
		t.Error("IsParalyzed should be false after Remove")
	}
}

func TestIsSilenced(t *testing.T) {
	s := New()
	if s.IsSilenced() {
		t.Error("IsSilenced should be false on empty stack")
	}
	s.Apply(makeEffect(Silence, 1, time.Minute))
	if !s.IsSilenced() {
		t.Error("IsSilenced should be true when Silence is active")
	}
}

func TestIsBound(t *testing.T) {
	s := New()
	if s.IsBound() {
		t.Error("IsBound should be false on empty stack")
	}
	s.Apply(makeEffect(Bind, 1, time.Minute))
	if !s.IsBound() {
		t.Error("IsBound should be true when Bind is active")
	}
}

// --- Effect.Active ---

func TestEffectActive_FutureExpiry(t *testing.T) {
	e := makeEffect(Poison, 5, time.Minute)
	if !e.Active(time.Now()) {
		t.Error("effect expiring in the future should be Active")
	}
}

func TestEffectActive_PastExpiry(t *testing.T) {
	e := Effect{Kind: Poison, Potency: 5, ExpiresAt: past(time.Second)}
	if e.Active(time.Now()) {
		t.Error("effect with past ExpiresAt should not be Active")
	}
}

func TestEffectActive_Permanent(t *testing.T) {
	e := Effect{Kind: Regen, Potency: 3} // zero ExpiresAt
	if !e.Active(time.Now().Add(9999 * time.Hour)) {
		t.Error("permanent effect should always be Active")
	}
}

// --- Integration ---

func TestIntegration_FullDebuffLifecycle(t *testing.T) {
	s := New()
	// apply Poison
	s.Apply(makeEffect(Poison, 6, 9*time.Second))

	// tick 1: 3 s elapsed — Poison should still be active and emit event
	tick1 := time.Now().Add(3 * time.Second)
	res1 := s.Tick(tick1)
	if len(res1.Expired) != 0 {
		t.Error("Poison should not expire at tick 1")
	}
	if len(res1.Events) == 0 {
		t.Error("Poison should emit event at tick 1")
	}

	// weaker Poison rejected
	result := s.Apply(makeEffect(Poison, 3, time.Minute))
	if result != ApplyRejected {
		t.Errorf("weaker Poison should be rejected: got %d", result)
	}

	// tick 2: past expiry
	tick2 := time.Now().Add(15 * time.Second)
	res2 := s.Tick(tick2)
	if len(res2.Expired) != 1 || res2.Expired[0].Kind != Poison {
		t.Errorf("Poison should expire at tick 2: %v", res2.Expired)
	}
	if s.Has(Poison) {
		t.Error("Poison should be gone after expiry")
	}
}

func TestIntegration_MultipleBuffsAndDebuffs(t *testing.T) {
	s := New()
	s.Apply(makeEffect(Regen, 5, time.Minute))
	s.Apply(makeEffect(Refresh, 3, time.Minute))
	s.Apply(makeEffect(Haste, 25, time.Minute))
	s.Apply(makeEffect(Poison, 4, time.Minute))
	s.Apply(makeEffect(Slow, 10, time.Minute))

	if len(s.All()) != 5 {
		t.Errorf("expected 5 effects, got %d", len(s.All()))
	}
	if s.NetHastePct() != 15 {
		t.Errorf("NetHastePct: got %d, want 15", s.NetHastePct())
	}

	res := s.Tick(time.Now())
	evKinds := make(map[Kind]int)
	for _, ev := range res.Events {
		evKinds[ev.Kind] = ev.Value
	}
	if evKinds[Regen] != 5 {
		t.Errorf("Regen tick: want 5, got %d", evKinds[Regen])
	}
	if evKinds[Refresh] != 3 {
		t.Errorf("Refresh tick: want 3, got %d", evKinds[Refresh])
	}
	if evKinds[Poison] != -4 {
		t.Errorf("Poison tick: want -4, got %d", evKinds[Poison])
	}
	// Haste and Slow should not generate tick events
	if _, ok := evKinds[Haste]; ok {
		t.Error("Haste should not emit a tick event")
	}
	if _, ok := evKinds[Slow]; ok {
		t.Error("Slow should not emit a tick event")
	}
}

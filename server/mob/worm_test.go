package mob

import (
	"testing"
	"time"
)

// --- NewWorm ---

func TestNewWorm_Kind(t *testing.T) {
	w := NewWorm("w1", Pos{10, 2, 0})
	if w.Kind != KindWorm {
		t.Errorf("Kind=%q, want %q", w.Kind, KindWorm)
	}
}

func TestNewWorm_Stats(t *testing.T) {
	w := NewWorm("w1", Pos{10, 2, 0})
	if w.HP != WormHP || w.MaxHP != WormHP {
		t.Errorf("HP: got %d/%d, want %d", w.HP, w.MaxHP, WormHP)
	}
	if w.MeleeDamage != WormDamage {
		t.Errorf("Damage=%d, want %d", w.MeleeDamage, WormDamage)
	}
	if w.MoveSpeed != WormMoveSpeed {
		t.Errorf("MoveSpeed=%f, want %f", w.MoveSpeed, WormMoveSpeed)
	}
	if w.AggroRange != WormAggroRange {
		t.Errorf("AggroRange=%f, want %f", w.AggroRange, WormAggroRange)
	}
	if w.BurrowInterval != WormBurrowEvery {
		t.Errorf("BurrowInterval=%v, want %v", w.BurrowInterval, WormBurrowEvery)
	}
	if w.BurrowDuration != WormBurrowFor {
		t.Errorf("BurrowDuration=%v, want %v", w.BurrowDuration, WormBurrowFor)
	}
}

func TestNewWorm_SceneID(t *testing.T) {
	w := NewWorm("w1", Pos{})
	if w.SceneID != 0 {
		t.Errorf("SceneID=%d, want 0 (Meadow)", w.SceneID)
	}
}

func TestNewWorm_HomePosSameAsSpawn(t *testing.T) {
	p := Pos{25, 2, 25}
	w := NewWorm("w1", p)
	if w.HomePos != p {
		t.Errorf("HomePos=%v, want %v", w.HomePos, p)
	}
}

func TestNewWorm_SlowerThanDefault(t *testing.T) {
	if WormMoveSpeed >= DefaultMobMoveSpeed {
		t.Errorf("worm should be slower than default: worm=%f default=%f", WormMoveSpeed, DefaultMobMoveSpeed)
	}
}

func TestNewWorm_ShorterAggroThanDefault(t *testing.T) {
	if WormAggroRange >= DefaultAggroRange {
		t.Errorf("worm aggro range should be shorter than default: worm=%f default=%f", WormAggroRange, DefaultAggroRange)
	}
}

// --- MeadowWormSpawns ---

func TestMeadowWormSpawns_Count(t *testing.T) {
	spawns := MeadowWormSpawns()
	if len(spawns) != 8 {
		t.Errorf("want 8 worm spawns, got %d", len(spawns))
	}
}

func TestMeadowWormSpawns_UniqueIDs(t *testing.T) {
	spawns := MeadowWormSpawns()
	seen := map[string]bool{}
	for _, w := range spawns {
		if seen[w.ID] {
			t.Errorf("duplicate worm ID: %q", w.ID)
		}
		seen[w.ID] = true
	}
}

func TestMeadowWormSpawns_AllKindWorm(t *testing.T) {
	for _, w := range MeadowWormSpawns() {
		if w.Kind != KindWorm {
			t.Errorf("spawn %q has kind %q, want worm", w.ID, w.Kind)
		}
	}
}

func TestMeadowWormSpawns_OutsideTownRadius(t *testing.T) {
	const townRadius = 15.0 // anything within 15u is considered "town"
	for _, w := range MeadowWormSpawns() {
		d := dist(w.HomePos, Pos{0, w.HomePos.Y, 0})
		if d < townRadius {
			t.Errorf("worm %q too close to town: dist=%.1f < %.1f", w.ID, d, townRadius)
		}
	}
}

func TestMeadowWormSpawns_AllInMeadow(t *testing.T) {
	for _, w := range MeadowWormSpawns() {
		if w.SceneID != 0 {
			t.Errorf("worm %q: SceneID=%d, want 0 (Meadow)", w.ID, w.SceneID)
		}
	}
}

// --- Burrow state machine ---

func spawnWormReg(id string, pos Pos) *Registry {
	reg := New()
	reg.Spawn(NewWorm(id, pos))
	return reg
}

func TestBurrow_StateTransitionOnTick(t *testing.T) {
	reg := spawnWormReg("w1", Pos{0, 2, 0})
	m, _ := reg.Get("w1")

	// Manually set lastBurrow in the past so the burrow triggers immediately.
	m.lastBurrow = time.Now().Add(-(WormBurrowEvery + time.Second))

	now := time.Now()
	events := reg.Tick(now, 0.05, nil)

	if m.State != StateBurrowed {
		t.Fatalf("worm should be burrowed after interval; state=%v", m.State)
	}
	foundBurrow := false
	for _, e := range events {
		if e.Kind == EvtMobBurrow && e.MobID == "w1" {
			foundBurrow = true
		}
	}
	if !foundBurrow {
		t.Error("expected EvtMobBurrow event")
	}
}

func TestBurrow_SurfaceAfterDuration(t *testing.T) {
	reg := spawnWormReg("w1", Pos{0, 2, 0})
	m, _ := reg.Get("w1")

	// Put it into burrowed state with burrowEnd in the past.
	m.State = StateBurrowed
	m.burrowEnd = time.Now().Add(-time.Second)

	events := reg.Tick(time.Now(), 0.05, nil)

	if m.State != StateIdle {
		t.Fatalf("worm should surface after burrow ends; state=%v", m.State)
	}
	foundSurface := false
	for _, e := range events {
		if e.Kind == EvtMobSurface && e.MobID == "w1" {
			foundSurface = true
		}
	}
	if !foundSurface {
		t.Error("expected EvtMobSurface event")
	}
}

func TestBurrow_UnhittableWhileBurrowed(t *testing.T) {
	reg := spawnWormReg("w1", Pos{0, 2, 0})
	m, _ := reg.Get("w1")

	m.State = StateBurrowed
	m.burrowEnd = time.Now().Add(5 * time.Second)

	_, _, err := reg.Hit("w1", "player1", 20)
	if err == nil {
		t.Error("Hit on burrowed worm should return error")
	}
	if m.HP != WormHP {
		t.Errorf("burrowed worm HP should not change: got %d, want %d", m.HP, WormHP)
	}
}

func TestBurrow_TickPlayerUnhittableWhileBurrowed(t *testing.T) {
	reg := spawnWormReg("w1", Pos{0, 2, 0})
	m, _ := reg.Get("w1")

	m.State = StateBurrowed
	m.burrowEnd = time.Now().Add(5 * time.Second)

	combat := &PlayerCombat{
		TargetMobID: "w1",
		SwingDelay:  100 * time.Millisecond,
		BaseDamage:  20,
		MeleeRange:  5,
	}
	_, _, err := reg.TickPlayer("p1", combat, Pos{0, 2, 0}, 0, time.Now().Add(time.Second))
	if err == nil {
		t.Error("TickPlayer on burrowed worm should return error")
	}
	if m.HP != WormHP {
		t.Errorf("burrowed worm HP unchanged: got %d", m.HP)
	}
}

func TestBurrow_DoesNotBurrowWhilePursuing(t *testing.T) {
	reg := spawnWormReg("w1", Pos{0, 2, 0})
	m, _ := reg.Get("w1")

	m.State = StatePursuing
	m.AggroSlot = "player1"
	m.lastBurrow = time.Now().Add(-(WormBurrowEvery + time.Second))

	players := []PlayerPositions{{Slot: "player1", SceneID: 0, Pos: Pos{1, 2, 0}}}
	reg.Tick(time.Now(), 0.05, players)

	// Should still be pursuing, not burrowed.
	if m.State == StateBurrowed {
		t.Error("worm should not burrow while pursuing")
	}
}

func TestBurrow_DropsAggroOnBurrow(t *testing.T) {
	reg := spawnWormReg("w1", Pos{0, 2, 0})
	m, _ := reg.Get("w1")

	m.AggroSlot = "player1"
	m.State = StateIdle // back to idle but had aggro
	m.lastBurrow = time.Now().Add(-(WormBurrowEvery + time.Second))

	reg.Tick(time.Now(), 0.05, nil)

	if m.AggroSlot != "" {
		t.Errorf("AggroSlot should be cleared on burrow: got %q", m.AggroSlot)
	}
}

func TestBurrow_IdleWhileBurrowed(t *testing.T) {
	reg := spawnWormReg("w1", Pos{0, 2, 0})
	m, _ := reg.Get("w1")

	m.State = StateBurrowed
	m.burrowEnd = time.Now().Add(10 * time.Second)

	players := []PlayerPositions{{Slot: "player1", SceneID: 0, Pos: Pos{1, 2, 0}}}
	events := reg.Tick(time.Now(), 0.05, players)

	// Should not aggro or attack while burrowed.
	for _, e := range events {
		if e.Kind == EvtMobAggro || e.Kind == EvtMobAttack {
			t.Errorf("burrowed worm should not aggro or attack: got %v", e.Kind)
		}
	}
	if m.State != StateBurrowed {
		t.Errorf("state changed unexpectedly: got %v", m.State)
	}
}

// --- MobState.String addition ---

func TestStateBurrowed_String(t *testing.T) {
	if got := StateBurrowed.String(); got != "burrowed" {
		t.Errorf("StateBurrowed.String() = %q, want \"burrowed\"", got)
	}
}

// --- itoa helper ---

func TestItoa(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{7, "7"},
		{10, "10"},
		{99, "99"},
	}
	for _, c := range cases {
		if got := itoa(c.n); got != c.want {
			t.Errorf("itoa(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

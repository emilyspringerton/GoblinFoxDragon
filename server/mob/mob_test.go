package mob

import (
	"testing"
	"time"
)

// --- helpers ---

func skeleton(id string, pos Pos) Mob {
	return Mob{
		ID:      id,
		Kind:    "skeleton",
		SceneID: 1,
		Pos:     pos,
		HP:      100,
		MaxHP:   100,
	}
}

func playerAt(slot string, pos Pos) PlayerPositions {
	return PlayerPositions{Slot: slot, SceneID: 1, Pos: pos}
}

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// --- Spawn / Get ---

func TestSpawnAndGet(t *testing.T) {
	reg := New()
	if err := reg.Spawn(skeleton("m1", Pos{0, 0, 0})); err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	m, ok := reg.Get("m1")
	if !ok {
		t.Fatal("Get: mob not found after Spawn")
	}
	if m.State != StateIdle {
		t.Errorf("initial state: got %v, want idle", m.State)
	}
	if m.HP != 100 {
		t.Errorf("HP: got %d, want 100", m.HP)
	}
}

func TestSpawnDuplicateErrors(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{}))
	if err := reg.Spawn(skeleton("m1", Pos{})); err == nil {
		t.Error("duplicate Spawn should return error")
	}
}

func TestSpawnSetsDefaults(t *testing.T) {
	reg := New()
	reg.Spawn(Mob{ID: "m1", Kind: "wolf", SceneID: 1, HP: 50, MaxHP: 50})
	m, _ := reg.Get("m1")
	if m.AggroRange != DefaultAggroRange {
		t.Errorf("AggroRange default: got %v", m.AggroRange)
	}
	if m.MoveSpeed != DefaultMobMoveSpeed {
		t.Errorf("MoveSpeed default: got %v", m.MoveSpeed)
	}
	if m.SwingDelay != DefaultMobSwingDelay {
		t.Errorf("SwingDelay default: got %v", m.SwingDelay)
	}
	if m.MeleeDamage != DefaultMobDamage {
		t.Errorf("MeleeDamage default: got %v", m.MeleeDamage)
	}
}

// --- Hit / tagging ---

func TestHitTagsOnFirstDamage(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{}))
	res, _, err := reg.Hit("m1", "alice", 10)
	if err != nil {
		t.Fatalf("Hit: %v", err)
	}
	if !res.Tagged {
		t.Error("first hit should set tag")
	}
	m, _ := reg.Get("m1")
	if m.TaggerSlot != "alice" {
		t.Errorf("tagger: got %q, want alice", m.TaggerSlot)
	}
}

func TestHitDoesNotStealTag(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{}))
	reg.Hit("m1", "alice", 10)
	res, _, _ := reg.Hit("m1", "bob", 10)
	if res.Tagged {
		t.Error("second hitter should not steal the tag")
	}
	m, _ := reg.Get("m1")
	if m.TaggerSlot != "alice" {
		t.Errorf("tag should remain alice, got %q", m.TaggerSlot)
	}
}

func TestHitDealsDamage(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{}))
	res, _, _ := reg.Hit("m1", "alice", 30)
	if res.Dealt != 30 {
		t.Errorf("dealt: got %d, want 30", res.Dealt)
	}
	m, _ := reg.Get("m1")
	if m.HP != 70 {
		t.Errorf("HP after hit: got %d, want 70", m.HP)
	}
}

func TestHitCapsDamageAtRemainingHP(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{}))
	res, _, _ := reg.Hit("m1", "alice", 999)
	if res.Dealt != 100 {
		t.Errorf("overkill dealt: got %d, want 100", res.Dealt)
	}
}

func TestHitKillsMob(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{}))
	res, events, _ := reg.Hit("m1", "alice", 100)
	if !res.Died {
		t.Error("100 damage to 100 HP mob should kill it")
	}
	if len(events) == 0 || events[0].Kind != EvtMobDied {
		t.Error("EvtMobDied not emitted on kill")
	}
	if events[0].Tagger != "alice" {
		t.Errorf("EvtMobDied.Tagger: got %q, want alice", events[0].Tagger)
	}
}

func TestHitDeadMobErrors(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{}))
	reg.Hit("m1", "alice", 100)
	_, _, err := reg.Hit("m1", "alice", 10)
	if err != ErrMobDead {
		t.Errorf("hit dead mob: want ErrMobDead, got %v", err)
	}
}

func TestHitUnknownMobErrors(t *testing.T) {
	reg := New()
	_, _, err := reg.Hit("nope", "alice", 10)
	if err != ErrMobNotFound {
		t.Errorf("unknown mob: want ErrMobNotFound, got %v", err)
	}
}

func TestHitAggroesIdleMob(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{}))
	_, events, _ := reg.Hit("m1", "alice", 10)
	aggroed := false
	for _, e := range events {
		if e.Kind == EvtMobAggro && e.Slot == "alice" {
			aggroed = true
		}
	}
	if !aggroed {
		t.Error("hitting an idle mob should produce EvtMobAggro")
	}
	m, _ := reg.Get("m1")
	if m.State != StatePursuing {
		t.Errorf("mob state after hit: got %v, want pursuing", m.State)
	}
}

// --- Tick / mob AI ---

func TestTickIdleAggrosNearbyPlayer(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{0, 0, 0}))
	// Player within aggro range (default 20u).
	players := []PlayerPositions{playerAt("alice", Pos{5, 0, 0})}
	events := reg.Tick(t0, 0.1, players)
	m, _ := reg.Get("m1")
	if m.State != StatePursuing {
		t.Errorf("mob should aggro: got state %v", m.State)
	}
	hasAggro := false
	for _, e := range events {
		if e.Kind == EvtMobAggro {
			hasAggro = true
		}
	}
	if !hasAggro {
		t.Error("EvtMobAggro not emitted on proximity aggro")
	}
}

func TestTickIdleNoAggroOutOfRange(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{0, 0, 0}))
	players := []PlayerPositions{playerAt("alice", Pos{100, 0, 0})} // outside 20u range
	reg.Tick(t0, 0.1, players)
	m, _ := reg.Get("m1")
	if m.State != StateIdle {
		t.Errorf("far player should not aggro mob: got %v", m.State)
	}
}

func TestTickIdleNoAggroDifferentScene(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{0, 0, 0}))
	players := []PlayerPositions{{Slot: "alice", SceneID: 2, Pos: Pos{5, 0, 0}}}
	reg.Tick(t0, 0.1, players)
	m, _ := reg.Get("m1")
	if m.State != StateIdle {
		t.Error("player in different scene should not aggro mob")
	}
}

func TestTickPursuingMobMoves(t *testing.T) {
	reg := New()
	reg.Spawn(Mob{
		ID: "m1", Kind: "skeleton", SceneID: 1,
		Pos: Pos{0, 0, 0}, HomePos: Pos{0, 0, 0},
		HP: 100, MaxHP: 100,
		State: StatePursuing, AggroSlot: "alice",
		MoveSpeed: 5.0, LeashRange: 100,
		SwingDelay: 3 * time.Second, MeleeDamage: 10, MeleeRange: 2,
		AggroRange: 20,
	})
	players := []PlayerPositions{playerAt("alice", Pos{10, 0, 0})}
	reg.Tick(t0, 1.0, players) // 1 second: mob should move 5 units
	m, _ := reg.Get("m1")
	if m.Pos.X < 4.9 {
		t.Errorf("mob should have moved toward player: pos.X = %v", m.Pos.X)
	}
}

func TestTickPursuingMobAttacksInRange(t *testing.T) {
	reg := New()
	reg.Spawn(Mob{
		ID: "m1", Kind: "skeleton", SceneID: 1,
		Pos: Pos{1, 0, 0}, HomePos: Pos{0, 0, 0}, // 1u from player
		HP: 100, MaxHP: 100,
		State: StatePursuing, AggroSlot: "alice",
		MoveSpeed: 5.0, LeashRange: 100,
		SwingDelay: 0, MeleeDamage: 15, MeleeRange: 2,
		AggroRange: 20,
	})
	players := []PlayerPositions{playerAt("alice", Pos{0, 0, 0})}
	events := reg.Tick(t0, 0.1, players)
	hasAttack := false
	for _, e := range events {
		if e.Kind == EvtMobAttack && e.Slot == "alice" && e.Damage == 15 {
			hasAttack = true
		}
	}
	if !hasAttack {
		t.Error("mob in melee range should emit EvtMobAttack")
	}
}

func TestTickMobLeashResets(t *testing.T) {
	reg := New()
	reg.Spawn(Mob{
		ID: "m1", Kind: "skeleton", SceneID: 1,
		Pos: Pos{200, 0, 0}, HomePos: Pos{0, 0, 0}, // already past leash
		HP: 50, MaxHP: 100,
		State: StatePursuing, AggroSlot: "alice",
		MoveSpeed: 5.0, LeashRange: 50,
		SwingDelay: 3 * time.Second, MeleeDamage: 10, MeleeRange: 2,
		AggroRange: 20,
	})
	players := []PlayerPositions{playerAt("alice", Pos{200, 0, 0})}
	events := reg.Tick(t0, 0.1, players)
	m, _ := reg.Get("m1")
	if m.State != StateReturning {
		t.Errorf("leashed mob should be Returning, got %v", m.State)
	}
	if m.HP != 100 {
		t.Errorf("leashed mob should full-heal: got HP %d", m.HP)
	}
	hasReset := false
	for _, e := range events {
		if e.Kind == EvtMobReset {
			hasReset = true
		}
	}
	if !hasReset {
		t.Error("EvtMobReset not emitted on leash")
	}
}

func TestTickReturningMobWalksHome(t *testing.T) {
	reg := New()
	reg.Spawn(Mob{
		ID: "m1", Kind: "skeleton", SceneID: 1,
		Pos: Pos{10, 0, 0}, HomePos: Pos{0, 0, 0},
		HP: 100, MaxHP: 100,
		State: StateReturning,
		MoveSpeed: 5.0, LeashRange: 100, AggroRange: 20,
		SwingDelay: 3 * time.Second, MeleeDamage: 10, MeleeRange: 2,
	})
	reg.Tick(t0, 1.0, nil) // no players
	m, _ := reg.Get("m1")
	if m.Pos.X >= 10 {
		t.Errorf("returning mob should move toward home: pos.X = %v", m.Pos.X)
	}
}

func TestTickReturningMobReachesHome(t *testing.T) {
	reg := New()
	reg.Spawn(Mob{
		ID: "m1", Kind: "skeleton", SceneID: 1,
		Pos: Pos{1, 0, 0}, HomePos: Pos{0, 0, 0},
		HP: 100, MaxHP: 100,
		State: StateReturning,
		MoveSpeed: 10.0, LeashRange: 100, AggroRange: 20,
		SwingDelay: 3 * time.Second, MeleeDamage: 10, MeleeRange: 2,
	})
	reg.Tick(t0, 1.0, nil)
	m, _ := reg.Get("m1")
	if m.State != StateIdle {
		t.Errorf("returned mob should be Idle, got %v", m.State)
	}
}

// --- Player auto-attack ---

func TestTickPlayerHitsInRange(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{1, 0, 0}))
	pc := &PlayerCombat{
		TargetMobID: "m1",
		SwingDelay:  100 * time.Millisecond,
		BaseDamage:  25,
		MeleeRange:  3.0,
	}
	res, _, err := reg.TickPlayer("alice", pc, Pos{0, 0, 0}, 1, t0.Add(time.Second))
	if err != nil {
		t.Fatalf("TickPlayer: %v", err)
	}
	if res.Dealt != 25 {
		t.Errorf("dealt: got %d, want 25", res.Dealt)
	}
	if !res.Tagged {
		t.Error("first player hit should tag mob")
	}
}

func TestTickPlayerSwingTimerGates(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{1, 0, 0}))
	pc := &PlayerCombat{
		TargetMobID: "m1",
		SwingDelay:  2 * time.Second,
		BaseDamage:  25,
		MeleeRange:  3.0,
	}
	// First swing.
	reg.TickPlayer("alice", pc, Pos{0, 0, 0}, 1, t0)
	// Second swing 1s later — should not fire yet.
	res, _, err := reg.TickPlayer("alice", pc, Pos{0, 0, 0}, 1, t0.Add(time.Second))
	if err != nil {
		t.Fatalf("TickPlayer (too soon): %v", err)
	}
	if res.Dealt != 0 {
		t.Error("swing should be blocked by timer, no damage expected")
	}
}

func TestTickPlayerOutOfRangeErrors(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{100, 0, 0})) // 100u away
	pc := &PlayerCombat{
		TargetMobID: "m1",
		SwingDelay:  0,
		BaseDamage:  25,
		MeleeRange:  3.0,
	}
	_, _, err := reg.TickPlayer("alice", pc, Pos{0, 0, 0}, 1, t0)
	if err != ErrOutOfRange {
		t.Errorf("out of range: want ErrOutOfRange, got %v", err)
	}
}

func TestTickPlayerNoTargetErrors(t *testing.T) {
	reg := New()
	pc := &PlayerCombat{}
	_, _, err := reg.TickPlayer("alice", pc, Pos{}, 1, t0)
	if err != ErrNoTarget {
		t.Errorf("no target: want ErrNoTarget, got %v", err)
	}
}

func TestTickPlayerDeadTargetClearsTarget(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{1, 0, 0}))
	reg.Hit("m1", "alice", 100) // kill it
	pc := &PlayerCombat{TargetMobID: "m1", SwingDelay: 0, BaseDamage: 25, MeleeRange: 5}
	_, _, err := reg.TickPlayer("alice", pc, Pos{0, 0, 0}, 1, t0)
	if err != ErrMobDead {
		t.Errorf("dead target: want ErrMobDead, got %v", err)
	}
	if pc.TargetMobID != "" {
		t.Error("TargetMobID should be cleared after targeting dead mob")
	}
}

func TestTickPlayerWrongSceneErrors(t *testing.T) {
	reg := New()
	reg.Spawn(skeleton("m1", Pos{1, 0, 0})) // sceneID=1
	pc := &PlayerCombat{TargetMobID: "m1", SwingDelay: 0, BaseDamage: 25, MeleeRange: 5}
	_, _, err := reg.TickPlayer("alice", pc, Pos{0, 0, 0}, 2, t0) // player in scene 2
	if err != ErrOutOfRange {
		t.Errorf("wrong scene: want ErrOutOfRange, got %v", err)
	}
}

// --- Dist (exported wrapper) ---

func TestDist(t *testing.T) {
	a := Pos{X: 0, Y: 0, Z: 0}
	b := Pos{X: 3, Y: 4, Z: 0}
	if got := Dist(a, b); got != 5 {
		t.Errorf("Dist(%v, %v) = %v, want 5 (3-4-5 triangle)", a, b, got)
	}
	if got := Dist(a, a); got != 0 {
		t.Errorf("Dist(a, a) = %v, want 0", got)
	}
}

// TestMeadowWormSpawnsOutsideDefaultMeleeRange documents the real bug this
// was written to catch: MeadowWormSpawns places every worm 25-35 units from
// the zone's town-centre spawn point (0,2,0) -- see MeadowWormSpawns' own
// comment, "away from the town centre" -- which is far outside
// DefaultPlayerMeleeRange (3.0). Before apps2/mud's cmdAttack gained an
// auto-approach step, a brand-new player typing "attack worm" from spawn
// could never land a single hit: TickPlayer's range check silently returned
// ErrOutOfRange every tick with no player-visible feedback. This test pins
// the geometry fact that made the bug real, so a future spawn-layout change
// can't silently reintroduce it without a test failure flagging the gap.
func TestMeadowWormSpawnsOutsideDefaultMeleeRange(t *testing.T) {
	spawnPoint := Pos{X: 0, Y: 2, Z: 0}
	for _, w := range MeadowWormSpawns() {
		if d := Dist(spawnPoint, w.Pos); d <= DefaultPlayerMeleeRange {
			t.Errorf("worm %s at %v is within DefaultPlayerMeleeRange (%v) of spawn %v (dist=%v) -- "+
				"the real-world bug this suite guards against assumes every worm spawns OUTSIDE melee range",
				w.ID, w.Pos, DefaultPlayerMeleeRange, spawnPoint, d)
		}
	}
}

// --- MobState.String ---

func TestMobStateString(t *testing.T) {
	cases := []struct {
		s    MobState
		want string
	}{
		{StateIdle, "idle"},
		{StatePursuing, "pursuing"},
		{StateReturning, "returning"},
		{StateDead, "dead"},
		{MobState(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.s.String(); got != tc.want {
			t.Errorf("MobState(%d).String() = %q, want %q", tc.s, got, tc.want)
		}
	}
}

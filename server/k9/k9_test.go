package k9

import (
	"fmt"
	"math"
	"testing"
	"time"
)

var t0 = time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

func newDog(id string) *DogUnit { return NewDog(id, "crew-a") }

// ── Deploy + Mode ─────────────────────────────────────────────────────────────

func TestDeployAlive(t *testing.T) {
	d := newDog("dog-1")
	ev, err := d.Deploy("fo-1", t0)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if d.FOID != "fo-1" {
		t.Errorf("expected FOID fo-1, got %q", d.FOID)
	}
	if ev.Verb != "DEPLOYED" {
		t.Errorf("expected DEPLOYED event, got %q", ev.Verb)
	}
}

func TestDeployDeadBattery(t *testing.T) {
	d := newDog("dog-1")
	d.Battery = 0
	_, err := d.Deploy("fo-1", t0)
	if err != ErrBatteryDead {
		t.Errorf("expected ErrBatteryDead, got %v", err)
	}
}

func TestSetMode(t *testing.T) {
	d := newDog("dog-1")
	ev, err := d.SetMode(ModeAudit, t0)
	if err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	if d.Mode != ModeAudit {
		t.Errorf("expected ModeAudit, got %v", d.Mode)
	}
	if ev.Verb != "MODE_CHANGED" {
		t.Errorf("expected MODE_CHANGED, got %q", ev.Verb)
	}
}

func TestSetModeDeadBattery(t *testing.T) {
	d := newDog("dog-1")
	d.Battery = 0
	_, err := d.SetMode(ModeAudit, t0)
	if err != ErrBatteryDead {
		t.Errorf("expected ErrBatteryDead on dead dog, got %v", err)
	}
}

// ── Abilities ─────────────────────────────────────────────────────────────────

func TestMarkRaisesHeat(t *testing.T) {
	d := newDog("dog-1")
	d.FOID = "fo-1"
	before := d.HeatLevel
	_, err := d.Mark("crew-b", t0)
	if err != nil {
		t.Fatalf("Mark: %v", err)
	}
	if d.HeatLevel <= before {
		t.Errorf("expected HeatLevel to rise after Mark")
	}
}

func TestMarkRequiresFO(t *testing.T) {
	d := newDog("dog-1")
	_, err := d.Mark("crew-b", t0)
	if err != ErrNotOnFO {
		t.Errorf("expected ErrNotOnFO, got %v", err)
	}
}

func TestLatchStartAndEnd(t *testing.T) {
	d := newDog("dog-1")
	d.FOID = "fo-1"
	_, err := d.StartLatch(t0)
	if err != nil {
		t.Fatalf("StartLatch: %v", err)
	}
	if !d.Latching {
		t.Error("expected Latching=true after StartLatch")
	}
	_, err = d.EndLatch(t0)
	if err != nil {
		t.Fatalf("EndLatch: %v", err)
	}
	if d.Latching {
		t.Error("expected Latching=false after EndLatch")
	}
}

func TestLatchDoubleStart(t *testing.T) {
	d := newDog("dog-1")
	d.FOID = "fo-1"
	d.StartLatch(t0)
	_, err := d.StartLatch(t0)
	if err != ErrAlreadyLatching {
		t.Errorf("expected ErrAlreadyLatching, got %v", err)
	}
}

func TestEndLatchNotLatching(t *testing.T) {
	d := newDog("dog-1")
	_, err := d.EndLatch(t0)
	if err != ErrNotLatching {
		t.Errorf("expected ErrNotLatching, got %v", err)
	}
}

func TestHowlBeaconRaisesHeat(t *testing.T) {
	d := newDog("dog-1")
	d.FOID = "fo-1"
	before := d.HeatLevel
	_, err := d.HowlBeacon(t0)
	if err != nil {
		t.Fatalf("HowlBeacon: %v", err)
	}
	if d.HeatLevel <= before {
		t.Errorf("expected HeatLevel to rise after HowlBeacon")
	}
}

func TestHowlBeaconRequiresFO(t *testing.T) {
	d := newDog("dog-1")
	_, err := d.HowlBeacon(t0)
	if err != ErrNotOnFO {
		t.Errorf("expected ErrNotOnFO, got %v", err)
	}
}

func TestCustodyLockDuringContest(t *testing.T) {
	d := newDog("dog-1")
	d.FOID = "fo-1"
	ev, err := d.CustodyLock(true, t0)
	if err != nil {
		t.Fatalf("CustodyLock during contest: %v", err)
	}
	if ev.Verb != "CUSTODY_LOCK" {
		t.Errorf("expected CUSTODY_LOCK event, got %q", ev.Verb)
	}
}

func TestCustodyLockRequiresContest(t *testing.T) {
	d := newDog("dog-1")
	d.FOID = "fo-1"
	_, err := d.CustodyLock(false, t0)
	if err != ErrNotContested {
		t.Errorf("expected ErrNotContested, got %v", err)
	}
}

func TestCustodyLockRaisesHeat(t *testing.T) {
	d := newDog("dog-1")
	d.FOID = "fo-1"
	before := d.HeatLevel
	d.CustodyLock(true, t0)
	if d.HeatLevel <= before {
		t.Error("expected HeatLevel to rise after CustodyLock")
	}
}

func TestReceiptBurst(t *testing.T) {
	d := newDog("dog-1")
	d.FOID = "fo-1"
	ev, err := d.ReceiptBurst(t0)
	if err != nil {
		t.Fatalf("ReceiptBurst: %v", err)
	}
	if ev.Verb != "RECEIPT_BURST" {
		t.Errorf("expected RECEIPT_BURST, got %q", ev.Verb)
	}
}

func TestReceiptBurstRequiresFO(t *testing.T) {
	d := newDog("dog-1")
	_, err := d.ReceiptBurst(t0)
	if err != ErrNotOnFO {
		t.Errorf("expected ErrNotOnFO, got %v", err)
	}
}

// ── Battery / Tick ────────────────────────────────────────────────────────────

func TestTickDrainsSentry(t *testing.T) {
	d := newDog("dog-1")
	d.Tick(10*time.Second, t0)
	expected := BatteryMax - BatterySentryDrain*10
	if math.Abs(d.Battery-expected) > 0.001 {
		t.Errorf("expected battery %.2f after 10s sentry drain, got %.2f", expected, d.Battery)
	}
}

func TestTickDrainsAuditFaster(t *testing.T) {
	dS := newDog("sentry")
	dA := newDog("audit")
	dA.Mode = ModeAudit
	dS.Tick(10*time.Second, t0)
	dA.Tick(10*time.Second, t0)
	if dA.Battery >= dS.Battery {
		t.Error("audit mode should drain faster than sentry")
	}
}

func TestLatchExtraDrain(t *testing.T) {
	dA := newDog("no-latch")
	dB := newDog("latch")
	dB.FOID = "fo-1"
	dB.StartLatch(t0)
	dA.Tick(10*time.Second, t0)
	dB.Tick(10*time.Second, t0)
	if dB.Battery >= dA.Battery {
		t.Error("latching dog should drain battery faster")
	}
}

func TestLatchEndedOnBatteryDead(t *testing.T) {
	d := newDog("dog-1")
	d.FOID = "fo-1"
	d.StartLatch(t0)
	d.Battery = 0.1 // nearly dead
	d.Tick(time.Second, t0)
	if d.Latching {
		t.Error("latch should end when battery dies")
	}
}

func TestBatteryLowEvent(t *testing.T) {
	d := newDog("dog-1")
	d.Battery = 21.0 // just above 20%
	evs := d.Tick(time.Duration(3*time.Second), t0)
	// drain = 0.5/s * 3s = 1.5 → 19.5 → crosses 20%
	found := false
	for _, e := range evs {
		if e.Verb == "BATTERY_LOW" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected BATTERY_LOW event, got %v", evs)
	}
}

func TestBatteryDeadEvent(t *testing.T) {
	d := newDog("dog-1")
	d.Battery = 0.5
	evs := d.Tick(5*time.Second, t0) // 0.5/s * 5s = 2.5 drain → dead
	found := false
	for _, e := range evs {
		if e.Verb == "BATTERY_DEAD" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected BATTERY_DEAD event")
	}
}

func TestTickOnDeadDog(t *testing.T) {
	d := newDog("dog-1")
	d.Battery = 0
	evs := d.Tick(time.Second, t0)
	if len(evs) != 0 {
		t.Error("dead dog should produce no tick events")
	}
}

// ── Swarm math ────────────────────────────────────────────────────────────────

func TestSwarmCustodyScoreOnedog(t *testing.T) {
	score := SwarmCustodyScore(1, 10.0)
	if math.Abs(score-10.0) > 0.001 {
		t.Errorf("1 dog: expected 10.0, got %.4f", score)
	}
}

func TestSwarmCustodyScoreDiminishingReturns(t *testing.T) {
	s1 := SwarmCustodyScore(1, 10.0)
	s2 := SwarmCustodyScore(2, 10.0)
	s8 := SwarmCustodyScore(8, 10.0)
	// Each additional dog should add less than the previous
	if s2 >= s1*2 {
		t.Errorf("2 dogs should score less than 2× 1 dog (diminishing returns); got s1=%.2f s2=%.2f", s1, s2)
	}
	// 8 dogs should not exceed the geometric series limit: 10 / (1-0.85) = 66.67
	limit := 10.0 / (1 - SwarmDecay)
	if s8 > limit {
		t.Errorf("8-dog score %.2f exceeds geometric series limit %.2f", s8, limit)
	}
}

func TestSwarmCustodyScoreZero(t *testing.T) {
	if SwarmCustodyScore(0, 10.0) != 0 {
		t.Error("0 dogs should score 0")
	}
}

// ── Swarm struct ──────────────────────────────────────────────────────────────

func TestSwarmAddAndActiveCount(t *testing.T) {
	s := NewSwarm("fo-1")
	for i := 0; i < 3; i++ {
		d := NewDog(fmt.Sprintf("dog-%d", i), "crew-a")
		if err := s.Add(d); err != nil {
			t.Fatalf("Add dog-%d: %v", i, err)
		}
	}
	if s.ActiveCount() != 3 {
		t.Errorf("expected 3 active, got %d", s.ActiveCount())
	}
}

func TestSwarmCapEnforced(t *testing.T) {
	s := NewSwarm("fo-1")
	for i := 0; i < MaxActivePerOffice; i++ {
		s.Add(NewDog(fmt.Sprintf("dog-%d", i), "crew-a"))
	}
	extra := NewDog("dog-overflow", "crew-a")
	err := s.Add(extra)
	if err != ErrMaxDogsOnFO {
		t.Errorf("expected ErrMaxDogsOnFO at cap, got %v", err)
	}
}

func TestSwarmAddDeadDog(t *testing.T) {
	s := NewSwarm("fo-1")
	d := newDog("dog-dead")
	d.Battery = 0
	err := s.Add(d)
	if err != ErrBatteryDead {
		t.Errorf("expected ErrBatteryDead for dead dog, got %v", err)
	}
}

func TestSwarmCustodyScore(t *testing.T) {
	s := NewSwarm("fo-1")
	s.Add(NewDog("d1", "crew-a"))
	s.Add(NewDog("d2", "crew-a"))
	score := s.CustodyScore()
	expected := SwarmCustodyScore(2, 10.0)
	if math.Abs(score-expected) > 0.001 {
		t.Errorf("expected %.4f, got %.4f", expected, score)
	}
}

func TestSwarmPrune(t *testing.T) {
	s := NewSwarm("fo-1")
	alive := NewDog("alive", "crew-a")
	dead := NewDog("dead", "crew-a")
	dead.Battery = 0
	s.Dogs = append(s.Dogs, alive, dead)
	removed := s.Prune()
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}
	if len(s.Dogs) != 1 || s.Dogs[0].ID != "alive" {
		t.Error("expected only alive dog after prune")
	}
}

func TestSwarmIsLatched(t *testing.T) {
	s := NewSwarm("fo-1")
	d := NewDog("d1", "crew-a")
	d.FOID = "fo-1"
	s.Dogs = append(s.Dogs, d)
	if s.IsLatched() {
		t.Error("should not be latched before StartLatch")
	}
	d.StartLatch(t0)
	if !s.IsLatched() {
		t.Error("should be latched after StartLatch")
	}
}

func TestSwarmTickAll(t *testing.T) {
	s := NewSwarm("fo-1")
	d := NewDog("d1", "crew-a")
	d.Battery = 0.5 // will die on 5s tick
	s.Dogs = append(s.Dogs, d)
	evs := s.TickAll(5*time.Second, t0)
	found := false
	for _, e := range evs {
		if e.Verb == "BATTERY_DEAD" {
			found = true
		}
	}
	if !found {
		t.Error("expected BATTERY_DEAD in TickAll result")
	}
}


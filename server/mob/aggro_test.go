package mob

import "testing"

var (
	mobAt   = Pos2D{X: 0, Z: 0}
	nearBy  = Pos2D{X: 0, Z: 5}   // 5m away, straight ahead (heading=0)
	farAway = Pos2D{X: 0, Z: 100} // 100m away
	behind  = Pos2D{X: 0, Z: -5}  // directly behind mob facing +Z
	side    = Pos2D{X: 10, Z: 0}  // 10m to the right (outside 45° half-cone)
)

var noFlags = StatusFlags{}
var invisible = StatusFlags{Invisible: true}
var sneaked = StatusFlags{Sneak: true}
var bothFlags = StatusFlags{Invisible: true, Sneak: true}

// ── Sight ─────────────────────────────────────────────────────────────────────

func TestSight_DetectsInCone(t *testing.T) {
	at := AggroSight
	// Mob facing +Z (heading=0), target at 5m ahead, 90° FOV
	if !at.DetectsDefault(mobAt, nearBy, 0, noFlags) {
		t.Error("sight should detect target straight ahead in cone")
	}
}

func TestSight_BlockedByInvisible(t *testing.T) {
	at := AggroSight
	if at.DetectsDefault(mobAt, nearBy, 0, invisible) {
		t.Error("sight should not detect Invisible target")
	}
}

func TestSight_BlockedByDistance(t *testing.T) {
	at := AggroSight
	if at.DetectsDefault(mobAt, farAway, 0, noFlags) {
		t.Error("sight should not detect target beyond range")
	}
}

func TestSight_NotBehind(t *testing.T) {
	at := AggroSight
	// Mob facing +Z (heading=0), target at -Z (behind)
	if at.DetectsDefault(mobAt, behind, 0, noFlags) {
		t.Error("sight should not detect target behind the mob")
	}
}

func TestSight_SideOutsideCone(t *testing.T) {
	at := AggroSight
	// target at 10m to the side (90° off heading), outside 45° half-cone
	if at.DetectsDefault(mobAt, side, 0, noFlags) {
		t.Error("sight should not detect target outside FOV cone")
	}
}

// ── Sound ─────────────────────────────────────────────────────────────────────

func TestSound_Detects360(t *testing.T) {
	at := AggroSound
	// Sound is 360° — behind mob, within range
	if !at.DetectsDefault(mobAt, behind, 0, noFlags) {
		t.Error("sound should detect player behind mob")
	}
}

func TestSound_BlockedBySneak(t *testing.T) {
	at := AggroSound
	if at.DetectsDefault(mobAt, nearBy, 0, sneaked) {
		t.Error("sound should not detect Sneaked player")
	}
}

func TestSound_BlockedByDistance(t *testing.T) {
	at := AggroSound
	// Default sound range = 8m; target at 100m
	if at.DetectsDefault(mobAt, farAway, 0, noFlags) {
		t.Error("sound should not detect target beyond range")
	}
}

func TestSound_NotBlockedByInvisible(t *testing.T) {
	at := AggroSound
	// Invisible doesn't affect sound
	if !at.DetectsDefault(mobAt, nearBy, 0, invisible) {
		t.Error("sound should detect Invisible (but not Sneaked) player")
	}
}

// ── SightSound ────────────────────────────────────────────────────────────────

func TestSightSound_SightBlockedButSoundWorks(t *testing.T) {
	at := AggroSightSound
	// Invisible blocks sight, but sound still triggers
	if !at.DetectsDefault(mobAt, nearBy, 0, invisible) {
		t.Error("sight+sound: sound should still detect Invisible player")
	}
}

func TestSightSound_BothBlocked(t *testing.T) {
	at := AggroSightSound
	if at.DetectsDefault(mobAt, nearBy, 0, bothFlags) {
		t.Error("sight+sound: both blocked should not detect")
	}
}

// ── Job ───────────────────────────────────────────────────────────────────────

func TestJob_NotBlockedBySneakOrInvis(t *testing.T) {
	at := AggroJob
	if !at.DetectsDefault(mobAt, nearBy, 0, bothFlags) {
		t.Error("job detect should not be blockable by Sneak/Invisible")
	}
}

func TestJob_BlockedByDistance(t *testing.T) {
	at := AggroJob
	if at.DetectsDefault(mobAt, farAway, 0, noFlags) {
		t.Error("job detect should be range-limited")
	}
}

// ── Passive ───────────────────────────────────────────────────────────────────

func TestPassive_NeverAggros(t *testing.T) {
	at := AggroPassive
	for _, target := range []Pos2D{nearBy, behind, side} {
		if at.DetectsDefault(mobAt, target, 0, noFlags) {
			t.Errorf("passive mob should never aggro (target=%v)", target)
		}
	}
}

// ── distance2D ────────────────────────────────────────────────────────────────

func TestDistance2D(t *testing.T) {
	a := Pos2D{X: 0, Z: 0}
	b := Pos2D{X: 3, Z: 4}
	got := distance2D(a, b)
	if got < 4.99 || got > 5.01 {
		t.Errorf("distance2D 3-4-5 triangle: got %f, want 5", got)
	}
}

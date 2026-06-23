package mob

import "math"

// AggroType describes how a mob detects players.
// FFXI parity: sight, sound, and job-specific detection can be individually toggled.
type AggroType struct {
	Sight     bool // detects by sight; blocked by Invisible status
	Sound     bool // detects by proximity regardless of facing; blocked by Sneak status
	JobDetect bool // detects specific jobs by sensing their aura (not blockable)
}

// StatusFlags contains the detection-relevant status flags for a player.
type StatusFlags struct {
	Invisible bool // Invisible spell/item active — blocks Sight aggro
	Sneak     bool // Sneak spell/item active — blocks Sound aggro
}

// Pos2D is a flat XZ position used for aggro range checks (Y ignored).
type Pos2D struct{ X, Z float64 }

// distance2D returns the horizontal distance between two Pos2D positions.
func distance2D(a, b Pos2D) float64 {
	dx := a.X - b.X
	dz := a.Z - b.Z
	return math.Sqrt(dx*dx + dz*dz)
}

// inSightCone returns true if the target is within the given half-angle (degrees)
// and range of the mob facing the given headingDeg (compass degrees, 0=+Z).
func inSightCone(mobPos, targetPos Pos2D, mobHeadingDeg, halfAngleDeg, rangeMeter float64) bool {
	dist := distance2D(mobPos, targetPos)
	if dist > rangeMeter || dist == 0 {
		return false
	}
	dx := targetPos.X - mobPos.X
	dz := targetPos.Z - mobPos.Z
	// Angle from +Z axis (heading=0 means looking toward +Z)
	angleToTarget := math.Atan2(dx, dz) * 180 / math.Pi // degrees
	diff := angleToTarget - mobHeadingDeg
	// Normalise to [-180, 180]
	for diff > 180 {
		diff -= 360
	}
	for diff < -180 {
		diff += 360
	}
	return math.Abs(diff) <= halfAngleDeg
}

const (
	DefaultSightRange    = 20.0 // metres
	DefaultSightHalfCone = 45.0 // degrees (90° total FOV)
	DefaultSoundRange    = 8.0  // metres; sound is 360°
)

// Detects reports whether a mob with this AggroType detects a player.
//
//   - mobPos, targetPos: horizontal positions
//   - mobHeadingDeg: direction the mob faces (0 = +Z, 90 = +X, etc.)
//   - facingRange, halfAngle: sight range and half-FOV angle in degrees
//   - soundRange: radius for sound detection
//   - flags: player's active status effects
func (at AggroType) Detects(
	mobPos, targetPos Pos2D,
	mobHeadingDeg float64,
	sightRange, sightHalfAngle float64,
	soundRange float64,
	flags StatusFlags,
) bool {
	// Sight: blocked by Invisible; requires facing cone.
	if at.Sight && !flags.Invisible {
		if inSightCone(mobPos, targetPos, mobHeadingDeg, sightHalfAngle, sightRange) {
			return true
		}
	}
	// Sound: blocked by Sneak; 360° proximity.
	if at.Sound && !flags.Sneak {
		if distance2D(mobPos, targetPos) <= soundRange {
			return true
		}
	}
	// Job detect: not blockable; 360° at sound range.
	if at.JobDetect {
		if distance2D(mobPos, targetPos) <= soundRange {
			return true
		}
	}
	return false
}

// DetectsDefault is a convenience wrapper using default range/angle constants.
func (at AggroType) DetectsDefault(mobPos, targetPos Pos2D, mobHeadingDeg float64, flags StatusFlags) bool {
	return at.Detects(mobPos, targetPos, mobHeadingDeg,
		DefaultSightRange, DefaultSightHalfCone, DefaultSoundRange, flags)
}

// ── Preset AggroTypes ─────────────────────────────────────────────────────────

// AggroSight is sight-only aggro (standard aggressive mob).
var AggroSight = AggroType{Sight: true}

// AggroSound is sound-only aggro (undead, slime types).
var AggroSound = AggroType{Sound: true}

// AggroSightSound is dual aggro (NM, high-tier mobs).
var AggroSightSound = AggroType{Sight: true, Sound: true}

// AggroJob detects by job aura only (can't be sneaked or invis'd).
var AggroJob = AggroType{JobDetect: true}

// AggroPassive is a mob that never aggros (critters, fish).
var AggroPassive = AggroType{}

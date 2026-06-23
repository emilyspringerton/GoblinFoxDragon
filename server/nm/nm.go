// Package nm implements Notorious Monster spawn logic for DragonsNShit.
//
// # NM spawn model (FFXI-parity)
//
// A Notorious Monster (NM) either:
//   - Spawns in a time window after its placeholder is killed (forced spawn
//     with a chance roll), or
//   - Has a fixed spawn schedule (window-only, no placeholder requirement).
//
// TrySpawn is called by the server whenever the conditions might be met.
// It returns true (and clears the window) if the NM should appear.
package nm

import (
	"math/rand"
	"time"
)

// NMSpawn defines the spawn conditions for one Notorious Monster.
type NMSpawn struct {
	ID            string        // NM mob ID (used to Spawn into mob.Registry)
	PlaceholderID string        // mob whose death opens the window; "" = window-only NM
	SpawnChance   float64       // probability per TrySpawn call when in window (0.0–1.0)

	// Window is the time range during which the NM can spawn.
	// Zero values mean "always in window."
	WindowOpen  time.Duration // duration after placeholder kill before window opens
	WindowClose time.Duration // duration after placeholder kill when window closes

	windowOpenAt  time.Time // absolute time window opened; zero = not triggered
	windowCloseAt time.Time // absolute time window closes
	spawned       bool       // true if NM already spawned in current window
}

// OnPlaceholderKilled should be called when the placeholder mob dies.
// It opens the spawn window for this NM.
// No-op if PlaceholderID is "" (window-only NM must use OpenWindow directly).
func (s *NMSpawn) OnPlaceholderKilled(now time.Time) {
	if s.PlaceholderID == "" {
		return
	}
	s.windowOpenAt = now.Add(s.WindowOpen)
	s.windowCloseAt = now.Add(s.WindowClose)
	s.spawned = false
}

// OpenWindow directly opens the spawn window (for scheduled NMs or GM triggers).
// open and close are absolute times.
func (s *NMSpawn) OpenWindow(open, close time.Time) {
	s.windowOpenAt = open
	s.windowCloseAt = close
	s.spawned = false
}

// InWindow reports whether now falls within the open spawn window.
func (s *NMSpawn) InWindow(now time.Time) bool {
	if s.spawned {
		return false
	}
	if s.windowOpenAt.IsZero() {
		return false
	}
	return !now.Before(s.windowOpenAt) && now.Before(s.windowCloseAt)
}

// TrySpawn rolls for an NM spawn if the window is open.
// Returns true if the NM spawns; false if not in window or the roll fails.
// On success the window is marked closed (NM can't spawn again until re-triggered).
func (s *NMSpawn) TrySpawn(now time.Time, rng *rand.Rand) bool {
	if !s.InWindow(now) {
		return false
	}
	if rng.Float64() < s.SpawnChance {
		s.spawned = true
		return true
	}
	return false
}

// WindowExpired reports whether the spawn window has passed without the NM spawning.
func (s *NMSpawn) WindowExpired(now time.Time) bool {
	if s.windowCloseAt.IsZero() {
		return false
	}
	return !s.spawned && !now.Before(s.windowCloseAt)
}

// Reset clears the window state (e.g. after NM is killed and will re-pop).
func (s *NMSpawn) Reset() {
	s.windowOpenAt = time.Time{}
	s.windowCloseAt = time.Time{}
	s.spawned = false
}

// ── Preset NMs ────────────────────────────────────────────────────────────────

// MeadowNMs returns the NM spawn definitions for the Meadow zone.
// Placeholder worms have a 30-minute pop window after the last worm dies.
func MeadowNMs() []*NMSpawn {
	return []*NMSpawn{
		{
			ID:            "nm-king-worm",
			PlaceholderID: "worm-meadow-0",
			SpawnChance:   0.25, // 25% per check
			WindowOpen:    5 * time.Minute,
			WindowClose:   35 * time.Minute,
		},
	}
}

// SwampNMs returns NM spawn definitions for Swampville.
func SwampNMs() []*NMSpawn {
	return []*NMSpawn{
		{
			ID:            "nm-marsh-leech",
			PlaceholderID: "leech-swamp-0",
			SpawnChance:   0.30,
			WindowOpen:    10 * time.Minute,
			WindowClose:   40 * time.Minute,
		},
	}
}

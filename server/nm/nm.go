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
	"sync"
	"time"
)

// NMSpawn defines the spawn conditions for one Notorious Monster.
type NMSpawn struct {
	ID            string        // NM mob ID (used to Spawn into mob.Registry)
	PlaceholderID string        // mob whose death opens the window; "" = window-only NM
	SpawnChance   float64       // probability per TrySpawn call when in window (0.0–1.0)
	RespawnMinutes int          // minutes before NM respawns after being killed (0 = no respawn)

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

// HillsNMs returns NM spawn definitions for Hills (zone 1).
//
// nm-great-beetle: popped from any beetle-hills placeholder, 20% chance per
// check in a 30-minute window. Boss beetle with enrage at low HP.
//
// nm-ancient-wolf: window-only (no placeholder requirement); spawns once every
// 90 minutes at the Hills far north, independent of normal wolf kills.
func HillsNMs() []*NMSpawn {
	return []*NMSpawn{
		{
			ID:             "nm-great-beetle",
			PlaceholderID:  "beetle-hills-0",
			SpawnChance:    0.20,
			WindowOpen:     8 * time.Minute,
			WindowClose:    38 * time.Minute,
			RespawnMinutes: 90,
		},
		{
			ID:             "nm-ancient-wolf",
			PlaceholderID:  "", // window-only; scheduled via OpenWindow
			SpawnChance:    1.0,
			WindowOpen:     0,
			WindowClose:    15 * time.Minute,
			RespawnMinutes: 90,
		},
	}
}

// CavesNMs returns NM spawn definitions for Caves (zone 2).
//
// nm-bone-knight: popped from skeleton-caves-0 placeholder, 35% chance per
// check in a 30-minute window. Undead boss; wide aggro, heavy melee.
//
// nm-venom-queen: popped from spider-caves-0 placeholder, 25% chance per
// check. The Venom Queen drops rare accessories.
func CavesNMs() []*NMSpawn {
	return []*NMSpawn{
		{
			ID:             "nm-bone-knight",
			PlaceholderID:  "skeleton-caves-0",
			SpawnChance:    0.35,
			WindowOpen:     5 * time.Minute,
			WindowClose:    35 * time.Minute,
			RespawnMinutes: 75,
		},
		{
			ID:             "nm-venom-queen",
			PlaceholderID:  "spider-caves-0",
			SpawnChance:    0.25,
			WindowOpen:     12 * time.Minute,
			WindowClose:    42 * time.Minute,
			RespawnMinutes: 60,
		},
	}
}

// AllStartingZoneNMs returns all NM definitions for the four default starting zones.
// Callers can register these with a Registry at startup.
func AllStartingZoneNMs() []*NMSpawn {
	var all []*NMSpawn
	all = append(all, MeadowNMs()...)
	all = append(all, HillsNMs()...)
	all = append(all, CavesNMs()...)
	all = append(all, SwampNMs()...)
	return all
}

// ── Registry ──────────────────────────────────────────────────────────────────

// Registry holds all NMSpawn definitions for a single zone.
type Registry struct {
	mu    sync.RWMutex
	spawns map[string]*NMSpawn // NM ID → spawn definition
}

// NewRegistry creates an empty NM registry.
func NewRegistry() *Registry {
	return &Registry{spawns: make(map[string]*NMSpawn)}
}

// Register adds an NMSpawn to the registry, keyed by NM ID.
func (r *Registry) Register(s *NMSpawn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.spawns[s.ID] = s
}

// Get returns the NMSpawn for the given ID, or nil.
func (r *Registry) Get(id string) *NMSpawn {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.spawns[id]
}

// All returns a snapshot of all registered NMSpawn definitions.
func (r *Registry) All() []*NMSpawn {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*NMSpawn, 0, len(r.spawns))
	for _, s := range r.spawns {
		out = append(out, s)
	}
	return out
}

// Size returns the number of registered NMs.
func (r *Registry) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.spawns)
}

// ── NMRespawnScheduler ────────────────────────────────────────────────────────

// PopEvent is emitted by the scheduler when an NM pops.
type PopEvent struct {
	NMID   string
	ZoneID int
	At     time.Time
}

// NMRespawnScheduler tracks respawn timers for NMs that have been killed.
// The caller must call OnKill when an NM dies, then call Tick each game
// tick; Tick returns a PopEvent for each NM that is now due to respawn.
type NMRespawnScheduler struct {
	registry *Registry
	mu       sync.Mutex
	timers   map[string]time.Time // NM ID → time to respawn
	zones    map[string]int       // NM ID → zone ID for the pop event
}

// NewNMRespawnScheduler creates a scheduler backed by the given registry.
func NewNMRespawnScheduler(reg *Registry) *NMRespawnScheduler {
	return &NMRespawnScheduler{
		registry: reg,
		timers:   make(map[string]time.Time),
		zones:    make(map[string]int),
	}
}

// OnKill records that nmID was killed in zoneID at time killedAt.
// If the NM has RespawnMinutes > 0, a respawn timer is scheduled.
func (s *NMRespawnScheduler) OnKill(nmID string, zoneID int, killedAt time.Time) {
	spawn := s.registry.Get(nmID)
	if spawn == nil || spawn.RespawnMinutes <= 0 {
		return
	}
	respawnAt := killedAt.Add(time.Duration(spawn.RespawnMinutes) * time.Minute)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.timers[nmID] = respawnAt
	s.zones[nmID] = zoneID
}

// Tick checks all pending respawn timers and returns PopEvents for NMs due to respawn.
// The fired timers are removed from the scheduler.
func (s *NMRespawnScheduler) Tick(now time.Time) []PopEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	var pops []PopEvent
	for nmID, respawnAt := range s.timers {
		if !now.Before(respawnAt) {
			pops = append(pops, PopEvent{
				NMID:   nmID,
				ZoneID: s.zones[nmID],
				At:     now,
			})
			delete(s.timers, nmID)
			delete(s.zones, nmID)
		}
	}
	return pops
}

// PendingCount returns the number of NMs currently awaiting respawn.
func (s *NMRespawnScheduler) PendingCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.timers)
}

// Cancel removes the respawn timer for nmID (e.g. if a GM force-popped it).
func (s *NMRespawnScheduler) Cancel(nmID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.timers, nmID)
	delete(s.zones, nmID)
}

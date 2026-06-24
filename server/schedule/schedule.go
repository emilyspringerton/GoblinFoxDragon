// Package schedule implements the TRAPX NPC daily schedule system.
//
// Each NPC has a 24-entry array mapping each hour (0-23 UTC) to the zone ID
// they should occupy at that hour. The Registry ticks every hour and emits
// NPC moves for any NPC whose current zone doesn't match the scheduled zone.
package schedule

import (
	"fmt"
	"sync"
	"time"
)

// NPCSchedule maps hours 0-23 to zone IDs for one NPC.
// Hour index 0 = midnight UTC, 12 = noon UTC.
type NPCSchedule struct {
	NPCID       string
	ZoneAtHour  [24]int // index = hour (0-23), value = zone ID
	Description string  // flavor text for examine
}

// Move is emitted when an NPC should change zones.
type Move struct {
	NPCID   string
	FromZone int
	ToZone  int
	Hour    int
}

// Registry manages all NPC schedules and tracks their current zones.
type Registry struct {
	mu       sync.RWMutex
	schedules map[string]*NPCSchedule
	current   map[string]int // NPCID → current zone
	clock     func() time.Time
}

// NewRegistry creates an empty schedule registry.
func NewRegistry() *Registry {
	return &Registry{
		schedules: make(map[string]*NPCSchedule),
		current:   make(map[string]int),
		clock:     time.Now,
	}
}

// Register adds an NPC schedule. The NPC's initial zone is set from hour 0.
func (r *Registry) Register(s *NPCSchedule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.schedules[s.NPCID] = s
	r.current[s.NPCID] = s.ZoneAtHour[0]
}

// Tick processes the schedule for the given hour (0-23).
// Returns all NPCs that need to move this tick.
func (r *Registry) Tick(hour int) []Move {
	if hour < 0 || hour > 23 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var moves []Move
	for id, s := range r.schedules {
		wantZone := s.ZoneAtHour[hour]
		haveZone := r.current[id]
		if wantZone != haveZone {
			moves = append(moves, Move{NPCID: id, FromZone: haveZone, ToZone: wantZone, Hour: hour})
			r.current[id] = wantZone
		}
	}
	return moves
}

// TickNow advances to the current UTC hour and returns any needed moves.
func (r *Registry) TickNow() []Move {
	return r.Tick(r.clock().UTC().Hour())
}

// CurrentZone returns the NPC's current tracked zone.
func (r *Registry) CurrentZone(npcID string) (int, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	z, ok := r.current[npcID]
	return z, ok
}

// Get returns the NPCSchedule for the given NPC ID, or nil if not registered.
func (r *Registry) Get(npcID string) *NPCSchedule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.schedules[npcID]
}

// All returns all registered NPC IDs.
func (r *Registry) All() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.schedules))
	for id := range r.schedules {
		ids = append(ids, id)
	}
	return ids
}

// Count returns the number of registered NPC schedules.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.schedules)
}

// ExamineFlavor returns time-of-day flavor text for an NPC at the given hour.
// It describes what the NPC is currently doing based on their schedule context.
func ExamineFlavor(s *NPCSchedule, hour int) string {
	if s == nil {
		return ""
	}
	if s.Description != "" {
		return fmt.Sprintf("[%02d:00] %s — %s", hour, s.NPCID, s.Description)
	}
	return fmt.Sprintf("[%02d:00] %s is in zone %d.", hour, s.NPCID, s.ZoneAtHour[hour])
}

// --- Seed schedules for canonical TRAPX NPCs ---

// allZones returns a [24]int where every hour maps to zoneID.
func allZones(zoneID int) [24]int {
	var arr [24]int
	for i := range arr {
		arr[i] = zoneID
	}
	return arr
}

// makeSchedule fills hours [start,end) with activeZone and the rest with idleZone.
func makeSchedule(activeZone, idleZone, start, end int) [24]int {
	var arr [24]int
	for i := range arr {
		if i >= start && i < end {
			arr[i] = activeZone
		} else {
			arr[i] = idleZone
		}
	}
	return arr
}

// JiangshiWardenSchedule is the Jiangshi warden: patrols zone 50 by night,
// returns to barracks (zone 51) during the day.
var JiangshiWardenSchedule = &NPCSchedule{
	NPCID:       "jiangshi-warden",
	ZoneAtHour:  makeSchedule(50, 51, 20, 6), // 20:00-05:59 UTC in zone 50, else zone 51
	Description: "patrols the eastern checkpoint by night; rests in barracks by day",
}

// EastwindArchivistSchedule is the archivist: in the archive (zone 60) 08-18, café (zone 61) rest.
var EastwindArchivistSchedule = &NPCSchedule{
	NPCID:       "eastwind-archivist",
	ZoneAtHour:  makeSchedule(60, 61, 8, 18),
	Description: "studies manuscripts in the archive 08:00-18:00; frequents the café otherwise",
}

// HeikeganiDockBossSchedule is the dock boss: at the docks (zone 70) 06-22, home (zone 71) nights.
var HeikeganiDockBossSchedule = &NPCSchedule{
	NPCID:       "heikegani-dock-boss",
	ZoneAtHour:  makeSchedule(70, 71, 6, 22),
	Description: "runs the dock operation 06:00-22:00; sleeps at home overnight",
}

// DefaultSchedules returns the canonical TRAPX NPC schedules for seeding a Registry.
func DefaultSchedules() []*NPCSchedule {
	return []*NPCSchedule{
		JiangshiWardenSchedule,
		EastwindArchivistSchedule,
		HeikeganiDockBossSchedule,
	}
}

// Note: makeSchedule wraps hours: if start > end it covers [start,24)∪[0,end).
func init() {
	// Patch jiangshi warden for wrap-around: 20:00 → 05:59 spans midnight.
	var arr [24]int
	for i := range arr {
		if i >= 20 || i < 6 {
			arr[i] = 50
		} else {
			arr[i] = 51
		}
	}
	JiangshiWardenSchedule.ZoneAtHour = arr
}

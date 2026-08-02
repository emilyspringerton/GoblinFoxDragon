// Package zone manages the DragonsNShit zone (scene) backend.
//
// # Concepts
//
// A Zone is a named game region with an integer ID, a display name, and a
// default spawn position.  All player visibility, yell chat, mob ticks, and
// snapshot broadcasts are scoped per-zone — a player in zone A does not see
// events from zone B.
//
// The Manager is the single source of truth for which zone every connected
// player is in.  The server main loop calls:
//
//	mgr.Enter(slot, name, zoneID)   — on PacketConnect
//	mgr.Leave(slot)                  — on disconnect / timeout
//	mgr.Transfer(slot, toZoneID)     — on Telecrystal / portal use
//
// and reads zone membership to scope snapshots:
//
//	slots := mgr.PlayersIn(zoneID)   — broadcast only to co-zone players
//	id, ok := mgr.ZoneOf(slot)       — look up a specific player's zone
//
// # PacketSceneChange
//
// When the server performs a Transfer it must notify the client to reload the
// scene.  Use EncodeSceneChange to build the packet, then send it to the
// player's UDP address before moving the server-side slot.
//
//	Wire layout (14 bytes):
//	  [0]   PacketSceneChange (0x07)
//	  [1]   zoneID uint8
//	  [2–5] spawnX float32 little-endian
//	  [6–9] spawnY float32 little-endian
//	  [10–13] spawnZ float32 little-endian
//
// # Built-in zones
//
// DefaultZones returns the three zones wired to the worldapi chunk scenes
// (scene 0 = Meadow, 1 = Hills, 2 = Caves).  More zones can be added at
// runtime with Manager.AddZone.
package zone

import (
	"encoding/binary"
	"fmt"
	"math"

	"dragonsnshit/packages2/common"
)

// Zone describes a single game region.
type Zone struct {
	ID             int
	Name           string
	SpawnX, SpawnY, SpawnZ float64 // default entry position
}

// DefaultZones returns the starter zones matching worldapi scene IDs.
//
// Zone 4 ("Town Square", 2026-08-02, founder: "you may need to add the next zone" while
// building GoblinFoxDragon's Battlegrounds-GUI Town scene) is deliberately its own zone, not a
// reuse of zone 0 (Meadow) -- Meadow already has a real, live telnet MUD presence (worm combat,
// "X has entered the world" broadcasts, etc.); conflating it with the GUI's own local-only,
// no-combat Town scene would be exactly the "gui and the mud play nice... might get weird"
// tension the founder already named. Town Square gets a real zone identity of its own instead,
// same shared 0-3 scene-ID convention worldapi/scenes.go's ProceduralWorldStore already follows
// for its own voxel terrain -- not wired into that voxel generator yet (a real, separate,
// larger undertaking: the GUI client would need a whole new UDP protocol client speaking
// apps2/server-go's own wire format, packages2/common/protocol.h, to ever render it), flagged
// here rather than guessed at.
func DefaultZones() []*Zone {
	return []*Zone{
		{ID: 0, Name: "Meadow", SpawnX: 0, SpawnY: 2, SpawnZ: 0},
		{ID: 1, Name: "Hills", SpawnX: 0, SpawnY: 8, SpawnZ: 0},
		{ID: 2, Name: "Caves", SpawnX: 0, SpawnY: 4, SpawnZ: 0},
		{ID: 3, Name: "Swampville", SpawnX: 0, SpawnY: 1, SpawnZ: 0},
		{ID: 4, Name: "Town Square", SpawnX: 0, SpawnY: 2, SpawnZ: 0},
	}
}

// Manager is the authoritative registry of player zone membership.
// Not goroutine-safe — callers synchronise via the server tick loop.
type Manager struct {
	zones   map[int]*Zone
	players map[string]int    // slot → current zoneID
	names   map[string]string // slot → display name
}

// New creates a Manager pre-populated with the given zones.
// Panics if any two zones share the same ID.
func New(zones []*Zone) *Manager {
	m := &Manager{
		zones:   make(map[int]*Zone, len(zones)),
		players: make(map[string]int),
		names:   make(map[string]string),
	}
	for _, z := range zones {
		if _, dup := m.zones[z.ID]; dup {
			panic(fmt.Sprintf("zone: duplicate zone ID %d", z.ID))
		}
		m.zones[z.ID] = z
	}
	return m
}

// AddZone registers a new zone at runtime.
// Returns an error if a zone with the same ID already exists.
func (m *Manager) AddZone(z *Zone) error {
	if _, exists := m.zones[z.ID]; exists {
		return fmt.Errorf("zone %d already registered", z.ID)
	}
	m.zones[z.ID] = z
	return nil
}

// KnownZone reports whether zoneID is registered.
func (m *Manager) KnownZone(id int) bool {
	_, ok := m.zones[id]
	return ok
}

// Get returns the Zone metadata for id (nil, false if unknown).
func (m *Manager) Get(id int) (*Zone, bool) {
	z, ok := m.zones[id]
	return z, ok
}

// ZoneIDs returns all registered zone IDs (unordered).
func (m *Manager) ZoneIDs() []int {
	out := make([]int, 0, len(m.zones))
	for id := range m.zones {
		out = append(out, id)
	}
	return out
}

// Enter registers a player in the given zone.
// If the player is already registered their zone is updated to zoneID.
// Returns an error if zoneID is not known.
func (m *Manager) Enter(slot, name string, zoneID int) error {
	if !m.KnownZone(zoneID) {
		return fmt.Errorf("zone %d not registered", zoneID)
	}
	m.players[slot] = zoneID
	m.names[slot] = name
	return nil
}

// Leave removes a player from the manager entirely (disconnect, timeout).
// No-op if the slot was not registered.
func (m *Manager) Leave(slot string) {
	delete(m.players, slot)
	delete(m.names, slot)
}

// Transfer moves a player to a new zone.
// Returns an error if toZoneID is not known or the player is not registered.
// On error the player's current zone is unchanged.
func (m *Manager) Transfer(slot string, toZoneID int) error {
	if _, ok := m.players[slot]; !ok {
		return fmt.Errorf("slot %q not registered", slot)
	}
	if !m.KnownZone(toZoneID) {
		return fmt.Errorf("zone %d not registered", toZoneID)
	}
	m.players[slot] = toZoneID
	return nil
}

// ZoneOf returns the current zone ID for slot.
func (m *Manager) ZoneOf(slot string) (int, bool) {
	id, ok := m.players[slot]
	return id, ok
}

// NameOf returns the display name registered for slot.
func (m *Manager) NameOf(slot string) (string, bool) {
	n, ok := m.names[slot]
	return n, ok
}

// PlayersIn returns all slot strings currently in zoneID.
// Returns an empty slice (never nil) for unknown or empty zones.
func (m *Manager) PlayersIn(zoneID int) []string {
	var out []string
	for slot, id := range m.players {
		if id == zoneID {
			out = append(out, slot)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

// SameZone reports whether both slots are in the same zone and both registered.
func (m *Manager) SameZone(slotA, slotB string) bool {
	a, okA := m.players[slotA]
	b, okB := m.players[slotB]
	return okA && okB && a == b
}

// PlayerCount returns the total number of registered players across all zones.
func (m *Manager) PlayerCount() int { return len(m.players) }

// PlayersInCount returns the number of players currently in zoneID.
func (m *Manager) PlayersInCount(zoneID int) int {
	n := 0
	for _, id := range m.players {
		if id == zoneID {
			n++
		}
	}
	return n
}

// SpawnFor returns the spawn position of a zone.
// Returns zero position if zoneID is unknown.
func (m *Manager) SpawnFor(zoneID int) (x, y, z float64) {
	if zn, ok := m.zones[zoneID]; ok {
		return zn.SpawnX, zn.SpawnY, zn.SpawnZ
	}
	return 0, 0, 0
}

// EncodeSceneChange encodes a PacketSceneChange (type=7) to send to the client
// before transferring it server-side.  The client should freeze input, play the
// transition, set its local scene_id = zoneID, and spawn at (x, y, z).
func EncodeSceneChange(zoneID int, x, y, z float64) []byte {
	buf := make([]byte, 14)
	buf[0] = common.PacketSceneChange
	buf[1] = uint8(zoneID)
	binary.LittleEndian.PutUint32(buf[2:], math.Float32bits(float32(x)))
	binary.LittleEndian.PutUint32(buf[6:], math.Float32bits(float32(y)))
	binary.LittleEndian.PutUint32(buf[10:], math.Float32bits(float32(z)))
	return buf
}

// DecodeSceneChange parses a PacketSceneChange received from the server.
// Returns (zoneID, x, y, z, ok).  ok is false if buf is too short or not type 7.
func DecodeSceneChange(buf []byte) (zoneID int, x, y, z float64, ok bool) {
	if len(buf) < 14 || buf[0] != common.PacketSceneChange {
		return 0, 0, 0, 0, false
	}
	zoneID = int(buf[1])
	x = float64(math.Float32frombits(binary.LittleEndian.Uint32(buf[2:])))
	y = float64(math.Float32frombits(binary.LittleEndian.Uint32(buf[6:])))
	z = float64(math.Float32frombits(binary.LittleEndian.Uint32(buf[10:])))
	return zoneID, x, y, z, true
}

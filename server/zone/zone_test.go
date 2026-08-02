package zone

import (
	"testing"

	"dragonsnshit/packages2/common"
)

// helpers

func defaultMgr() *Manager {
	return New(DefaultZones())
}

// --- DefaultZones ---

func TestDefaultZones_ThreeZones(t *testing.T) {
	zones := DefaultZones()
	if len(zones) != 5 {
		t.Fatalf("DefaultZones: want 5, got %d", len(zones))
	}
}

func TestDefaultZones_IDs(t *testing.T) {
	ids := map[int]bool{}
	for _, z := range DefaultZones() {
		ids[z.ID] = true
	}
	for _, want := range []int{0, 1, 2} {
		if !ids[want] {
			t.Errorf("missing zone ID %d in DefaultZones", want)
		}
	}
}

func TestDefaultZones_NonEmptyNames(t *testing.T) {
	for _, z := range DefaultZones() {
		if z.Name == "" {
			t.Errorf("zone %d has empty Name", z.ID)
		}
	}
}

// --- New / AddZone ---

func TestNew_KnownZonesRegistered(t *testing.T) {
	m := defaultMgr()
	for _, id := range []int{0, 1, 2} {
		if !m.KnownZone(id) {
			t.Errorf("zone %d should be known", id)
		}
	}
}

func TestNew_UnknownZoneNotKnown(t *testing.T) {
	m := defaultMgr()
	if m.KnownZone(99) {
		t.Error("zone 99 should not be known")
	}
}

func TestAddZone_RegistersNewZone(t *testing.T) {
	m := defaultMgr()
	err := m.AddZone(&Zone{ID: 9, Name: "Dungeon"})
	if err != nil {
		t.Fatalf("AddZone: %v", err)
	}
	if !m.KnownZone(9) {
		t.Error("zone 9 should be known after AddZone")
	}
}

func TestAddZone_DuplicateReturnsError(t *testing.T) {
	m := defaultMgr()
	err := m.AddZone(&Zone{ID: 0, Name: "Dup"})
	if err == nil {
		t.Error("AddZone duplicate should return error")
	}
}

func TestGet_ReturnsZone(t *testing.T) {
	m := defaultMgr()
	z, ok := m.Get(0)
	if !ok {
		t.Fatal("Get(0) not found")
	}
	if z.Name != "Meadow" {
		t.Errorf("Get(0).Name = %q, want Meadow", z.Name)
	}
}

func TestGet_UnknownReturnsNil(t *testing.T) {
	m := defaultMgr()
	_, ok := m.Get(99)
	if ok {
		t.Error("Get(99) should return false")
	}
}

// --- Enter ---

func TestEnter_RegistersPlayer(t *testing.T) {
	m := defaultMgr()
	err := m.Enter("slot-1", "Alice", 0)
	if err != nil {
		t.Fatalf("Enter: %v", err)
	}
	id, ok := m.ZoneOf("slot-1")
	if !ok || id != 0 {
		t.Errorf("ZoneOf after Enter: got (%d, %v), want (0, true)", id, ok)
	}
}

func TestEnter_UnknownZoneReturnsError(t *testing.T) {
	m := defaultMgr()
	err := m.Enter("slot-1", "Alice", 99)
	if err == nil {
		t.Error("Enter unknown zone should return error")
	}
	if _, ok := m.ZoneOf("slot-1"); ok {
		t.Error("player should not be registered after failed Enter")
	}
}

func TestEnter_ReenterUpdatesZone(t *testing.T) {
	m := defaultMgr()
	m.Enter("slot-1", "Alice", 0)
	err := m.Enter("slot-1", "Alice", 1) // move to zone 1
	if err != nil {
		t.Fatalf("re-enter: %v", err)
	}
	id, _ := m.ZoneOf("slot-1")
	if id != 1 {
		t.Errorf("re-enter: got zone %d, want 1", id)
	}
}

func TestEnter_RegistersName(t *testing.T) {
	m := defaultMgr()
	m.Enter("slot-1", "Bob", 0)
	name, ok := m.NameOf("slot-1")
	if !ok || name != "Bob" {
		t.Errorf("NameOf after Enter: got (%q, %v)", name, ok)
	}
}

// --- Leave ---

func TestLeave_RemovesPlayer(t *testing.T) {
	m := defaultMgr()
	m.Enter("slot-1", "Alice", 0)
	m.Leave("slot-1")
	if _, ok := m.ZoneOf("slot-1"); ok {
		t.Error("ZoneOf should return false after Leave")
	}
	if _, ok := m.NameOf("slot-1"); ok {
		t.Error("NameOf should return false after Leave")
	}
}

func TestLeave_NoopOnUnknownSlot(t *testing.T) {
	m := defaultMgr()
	m.Leave("nonexistent") // should not panic
}

// --- Transfer ---

func TestTransfer_MovesPlayerToNewZone(t *testing.T) {
	m := defaultMgr()
	m.Enter("slot-1", "Alice", 0)
	err := m.Transfer("slot-1", 2)
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	id, _ := m.ZoneOf("slot-1")
	if id != 2 {
		t.Errorf("after Transfer: zone=%d, want 2", id)
	}
}

func TestTransfer_UnknownZoneReturnsError(t *testing.T) {
	m := defaultMgr()
	m.Enter("slot-1", "Alice", 0)
	err := m.Transfer("slot-1", 99)
	if err == nil {
		t.Error("Transfer to unknown zone should error")
	}
	id, _ := m.ZoneOf("slot-1")
	if id != 0 {
		t.Errorf("zone should not change after failed Transfer: got %d", id)
	}
}

func TestTransfer_UnregisteredSlotReturnsError(t *testing.T) {
	m := defaultMgr()
	err := m.Transfer("nobody", 1)
	if err == nil {
		t.Error("Transfer unregistered slot should error")
	}
}

// --- PlayersIn ---

func TestPlayersIn_ReturnsPlayersInZone(t *testing.T) {
	m := defaultMgr()
	m.Enter("a", "Alice", 1)
	m.Enter("b", "Bob", 1)
	m.Enter("c", "Carol", 0) // different zone
	in := m.PlayersIn(1)
	if len(in) != 2 {
		t.Errorf("PlayersIn(1): got %d, want 2", len(in))
	}
	found := map[string]bool{}
	for _, s := range in {
		found[s] = true
	}
	if !found["a"] || !found["b"] {
		t.Errorf("PlayersIn(1) missing players: %v", in)
	}
	if found["c"] {
		t.Error("Carol (zone 0) should not appear in zone 1")
	}
}

func TestPlayersIn_EmptyZoneReturnsEmptySlice(t *testing.T) {
	m := defaultMgr()
	in := m.PlayersIn(0)
	if in == nil {
		t.Error("PlayersIn should return empty slice, not nil")
	}
	if len(in) != 0 {
		t.Errorf("empty zone: got %d players", len(in))
	}
}

func TestPlayersIn_UnknownZoneReturnsEmpty(t *testing.T) {
	m := defaultMgr()
	in := m.PlayersIn(99)
	if len(in) != 0 {
		t.Errorf("unknown zone: got %d players, want 0", len(in))
	}
}

func TestPlayersIn_AfterLeave(t *testing.T) {
	m := defaultMgr()
	m.Enter("a", "Alice", 0)
	m.Leave("a")
	if len(m.PlayersIn(0)) != 0 {
		t.Error("zone should be empty after player leaves")
	}
}

func TestPlayersIn_AfterTransfer(t *testing.T) {
	m := defaultMgr()
	m.Enter("a", "Alice", 0)
	m.Transfer("a", 1)
	if len(m.PlayersIn(0)) != 0 {
		t.Error("zone 0 should be empty after player transfers out")
	}
	if len(m.PlayersIn(1)) != 1 {
		t.Error("zone 1 should have 1 player after transfer")
	}
}

// --- SameZone ---

func TestSameZone_TrueForCoZonePlayers(t *testing.T) {
	m := defaultMgr()
	m.Enter("a", "Alice", 1)
	m.Enter("b", "Bob", 1)
	if !m.SameZone("a", "b") {
		t.Error("Alice and Bob in zone 1 should be same zone")
	}
}

func TestSameZone_FalseForDifferentZones(t *testing.T) {
	m := defaultMgr()
	m.Enter("a", "Alice", 0)
	m.Enter("b", "Bob", 1)
	if m.SameZone("a", "b") {
		t.Error("players in different zones should not be same zone")
	}
}

func TestSameZone_FalseIfEitherUnregistered(t *testing.T) {
	m := defaultMgr()
	m.Enter("a", "Alice", 0)
	if m.SameZone("a", "ghost") {
		t.Error("SameZone with unregistered slot should be false")
	}
}

// --- PlayerCount / PlayersInCount ---

func TestPlayerCount(t *testing.T) {
	m := defaultMgr()
	m.Enter("a", "A", 0)
	m.Enter("b", "B", 1)
	m.Enter("c", "C", 0)
	if m.PlayerCount() != 3 {
		t.Errorf("PlayerCount: got %d, want 3", m.PlayerCount())
	}
	m.Leave("a")
	if m.PlayerCount() != 2 {
		t.Errorf("PlayerCount after leave: got %d, want 2", m.PlayerCount())
	}
}

func TestPlayersInCount(t *testing.T) {
	m := defaultMgr()
	m.Enter("a", "A", 0)
	m.Enter("b", "B", 0)
	m.Enter("c", "C", 1)
	if m.PlayersInCount(0) != 2 {
		t.Errorf("PlayersInCount(0): got %d, want 2", m.PlayersInCount(0))
	}
	if m.PlayersInCount(1) != 1 {
		t.Errorf("PlayersInCount(1): got %d, want 1", m.PlayersInCount(1))
	}
	if m.PlayersInCount(2) != 0 {
		t.Errorf("PlayersInCount(2): got %d, want 0", m.PlayersInCount(2))
	}
}

// --- SpawnFor ---

func TestSpawnFor_ReturnsZoneSpawn(t *testing.T) {
	m := defaultMgr()
	z, _ := m.Get(0)
	x, y, z2 := m.SpawnFor(0)
	if x != z.SpawnX || y != z.SpawnY || z2 != z.SpawnZ {
		t.Errorf("SpawnFor mismatch: got %v %v %v", x, y, z2)
	}
}

func TestSpawnFor_UnknownZoneReturnsZero(t *testing.T) {
	m := defaultMgr()
	x, y, z := m.SpawnFor(99)
	if x != 0 || y != 0 || z != 0 {
		t.Errorf("SpawnFor unknown: want 0,0,0 got %v %v %v", x, y, z)
	}
}

// --- EncodeSceneChange / DecodeSceneChange ---

func TestEncodeSceneChange_Length(t *testing.T) {
	buf := EncodeSceneChange(2, 10.5, 4.0, -3.0)
	if len(buf) != 14 {
		t.Errorf("EncodeSceneChange: len=%d, want 14", len(buf))
	}
}

func TestEncodeSceneChange_PacketType(t *testing.T) {
	buf := EncodeSceneChange(1, 0, 0, 0)
	if buf[0] != common.PacketSceneChange {
		t.Errorf("first byte: got %d, want PacketSceneChange (%d)", buf[0], common.PacketSceneChange)
	}
}

func TestDecodeSceneChange_RoundTrip(t *testing.T) {
	buf := EncodeSceneChange(2, 10.5, 4.0, -3.0)
	id, x, y, z, ok := DecodeSceneChange(buf)
	if !ok {
		t.Fatal("DecodeSceneChange: not ok")
	}
	if id != 2 {
		t.Errorf("zoneID: got %d, want 2", id)
	}
	// float32 round-trip; tolerance 1e-4
	check := func(name string, got, want float64) {
		if diff := got - want; diff > 0.001 || diff < -0.001 {
			t.Errorf("%s: got %f, want %f", name, got, want)
		}
	}
	check("x", x, 10.5)
	check("y", y, 4.0)
	check("z", z, -3.0)
}

func TestDecodeSceneChange_TooShort(t *testing.T) {
	_, _, _, _, ok := DecodeSceneChange([]byte{0x07, 0x01})
	if ok {
		t.Error("short buffer should return ok=false")
	}
}

func TestDecodeSceneChange_WrongType(t *testing.T) {
	buf := make([]byte, 14)
	buf[0] = 0x00 // PacketConnect, not PacketSceneChange
	_, _, _, _, ok := DecodeSceneChange(buf)
	if ok {
		t.Error("wrong packet type should return ok=false")
	}
}

func TestDecodeSceneChange_EmptyBuffer(t *testing.T) {
	_, _, _, _, ok := DecodeSceneChange(nil)
	if ok {
		t.Error("nil buffer should return ok=false")
	}
}

// --- ZoneIDs ---

func TestZoneIDs_ReturnsAllIDs(t *testing.T) {
	m := defaultMgr()
	ids := m.ZoneIDs()
	found := map[int]bool{}
	for _, id := range ids {
		found[id] = true
	}
	for _, want := range []int{0, 1, 2} {
		if !found[want] {
			t.Errorf("missing zone ID %d in ZoneIDs()", want)
		}
	}
}

// --- Integration ---

func TestIntegration_ZoneTransferLifecycle(t *testing.T) {
	m := New(DefaultZones())

	// Three players connect into Meadow (zone 0).
	for _, s := range []string{"alice", "bob", "carol"} {
		if err := m.Enter(s, s, 0); err != nil {
			t.Fatalf("Enter %s: %v", s, err)
		}
	}
	if m.PlayerCount() != 3 {
		t.Fatalf("PlayerCount: got %d, want 3", m.PlayerCount())
	}

	// Bob uses a Telecrystal → moves to Hills (zone 1).
	if err := m.Transfer("bob", 1); err != nil {
		t.Fatalf("Transfer bob: %v", err)
	}
	if !m.SameZone("alice", "carol") {
		t.Error("alice and carol should still be co-zone in 0")
	}
	if m.SameZone("alice", "bob") {
		t.Error("alice and bob should NOT be co-zone after transfer")
	}
	if m.PlayersInCount(0) != 2 {
		t.Errorf("zone 0 count: got %d, want 2", m.PlayersInCount(0))
	}
	if m.PlayersInCount(1) != 1 {
		t.Errorf("zone 1 count: got %d, want 1", m.PlayersInCount(1))
	}

	// alice disconnects.
	m.Leave("alice")
	if m.PlayersInCount(0) != 1 {
		t.Errorf("zone 0 after alice leaves: got %d, want 1", m.PlayersInCount(0))
	}

	// Encode a scene change packet for bob.
	sx, sy, sz := m.SpawnFor(1)
	pkt := EncodeSceneChange(1, sx, sy, sz)
	if len(pkt) != 14 {
		t.Errorf("scene change packet length: %d", len(pkt))
	}
	zid, _, _, _, ok := DecodeSceneChange(pkt)
	if !ok || zid != 1 {
		t.Errorf("scene change decode: ok=%v zid=%d", ok, zid)
	}
}

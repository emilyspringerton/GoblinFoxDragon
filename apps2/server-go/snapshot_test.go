package main

import (
	"encoding/binary"
	"math"
	"testing"

	"dragonsnshit/packages2/common"
	"dragonsnshit/server/system"
)

func TestIntegrateMovement_ForwardMovesTowardFacing(t *testing.T) {
	// apps/lobby's own input mapping makes cmd.Fwd NEGATIVE when the player presses forward
	// (main.c: "if(k[SDL_SCANCODE_W]) fwd-=1") -- integrateMovement's own doc comment names this
	// explicitly; this test is the real check that the sign convention lands correctly, not just
	// documented. yaw=0 faces +Z (this function's own defined convention).
	pos := system.Vec3{}
	cmd := common.UserCmd{Fwd: -1, Str: 0, Yaw: 0, Msec: 1000} // "pressing W" for a full second
	got := integrateMovement(pos, cmd, 1.0)
	if got.Z <= 0 {
		t.Fatalf("expected forward movement to increase Z (yaw=0 faces +Z), got Z=%.3f", got.Z)
	}
	if math.Abs(got.X) > 1e-9 {
		t.Fatalf("expected no lateral drift for pure forward movement, got X=%.6f", got.X)
	}
	wantZ := walkSpeed * 1.0
	if math.Abs(got.Z-wantZ) > 1e-9 {
		t.Fatalf("expected Z=%.3f (walkSpeed * 1s), got %.3f", wantZ, got.Z)
	}
}

func TestIntegrateMovement_StrafeIsPerpendicularToForward(t *testing.T) {
	pos := system.Vec3{}
	cmd := common.UserCmd{Fwd: 0, Str: 1, Yaw: 0, Msec: 1000}
	got := integrateMovement(pos, cmd, 1.0)
	if math.Abs(got.Z) > 1e-9 {
		t.Fatalf("expected no forward/back drift for pure strafe, got Z=%.6f", got.Z)
	}
	if got.X == 0 {
		t.Fatal("expected real lateral movement from strafe input, got X=0")
	}
}

func TestIntegrateMovement_ZeroInputNoMovement(t *testing.T) {
	pos := system.Vec3{X: 5, Y: 2, Z: -3}
	cmd := common.UserCmd{Fwd: 0, Str: 0, Yaw: 47, Msec: 500}
	got := integrateMovement(pos, cmd, 0.5)
	if got != pos {
		t.Fatalf("expected zero input to leave position unchanged, got %+v (was %+v)", got, pos)
	}
}

func TestIntegrateMovement_ZeroDtNoMovement(t *testing.T) {
	pos := system.Vec3{X: 1, Y: 1, Z: 1}
	cmd := common.UserCmd{Fwd: -1, Str: 1, Yaw: 90, Msec: 0}
	got := integrateMovement(pos, cmd, 0.0)
	if got != pos {
		t.Fatalf("expected zero dt to leave position unchanged, got %+v (was %+v)", got, pos)
	}
}

// TestBuildSnapshotPacket_MatchesRealCStructLayout checks the exact byte offsets/sizes against
// apps2/lobby's own real, compiler-verified NetHeader (12 bytes)/NetPlayer (44 bytes) layout
// (packages/common/protocol.h) -- confirmed via a standalone gcc sizeof/offsetof probe,
// 2026-08-03, not assumed from the field list. A byte-layout mismatch here would silently
// corrupt what the one real client speaking this protocol parses -- this test exists
// specifically to catch that class of bug before it ships.
func TestBuildSnapshotPacket_MatchesRealCStructLayout(t *testing.T) {
	peers := []snapshotPeer{
		{id: 3, pos: system.Vec3{X: 1.5, Y: 2.5, Z: -3.5}, yaw: 90, health: 80, maxHP: 100, isKO: false},
		{id: 7, pos: system.Vec3{X: -10, Y: 0, Z: 20}, yaw: 180, health: 0, maxHP: 50, isKO: true},
	}
	pkt := buildSnapshotPacket(1, peers)

	wantLen := snapshotHeaderSize + 1 + len(peers)*snapshotEntitySize
	if len(pkt) != wantLen {
		t.Fatalf("expected packet length %d (12 header + 1 count + %d*44 entities), got %d",
			wantLen, len(peers), len(pkt))
	}
	if pkt[0] != common.PacketSnapshot {
		t.Fatalf("expected type byte %d, got %d", common.PacketSnapshot, pkt[0])
	}
	if pkt[1] != 1 {
		t.Fatalf("expected recipient client_id=1, got %d", pkt[1])
	}
	if pkt[8] != uint8(len(peers)) {
		t.Fatalf("expected header entity_count=%d, got %d", len(peers), pkt[8])
	}
	if pkt[snapshotHeaderSize] != uint8(len(peers)) {
		t.Fatalf("expected the real count byte (offset %d, what net_process_snapshot actually "+
			"reads) = %d, got %d", snapshotHeaderSize, len(peers), pkt[snapshotHeaderSize])
	}

	off := snapshotHeaderSize + 1
	p0 := peers[0]
	if pkt[off] != p0.id {
		t.Fatalf("entity 0: expected id=%d at offset %d, got %d", p0.id, off, pkt[off])
	}
	gotX := math.Float32frombits(binary.LittleEndian.Uint32(pkt[off+8:]))
	gotY := math.Float32frombits(binary.LittleEndian.Uint32(pkt[off+12:]))
	gotZ := math.Float32frombits(binary.LittleEndian.Uint32(pkt[off+16:]))
	gotYaw := math.Float32frombits(binary.LittleEndian.Uint32(pkt[off+20:]))
	if float64(gotX) != p0.pos.X || float64(gotY) != p0.pos.Y || float64(gotZ) != p0.pos.Z {
		t.Fatalf("entity 0: expected pos (%.2f,%.2f,%.2f) at offsets x=8/y=12/z=16, got (%.2f,%.2f,%.2f)",
			p0.pos.X, p0.pos.Y, p0.pos.Z, gotX, gotY, gotZ)
	}
	if gotYaw != p0.yaw {
		t.Fatalf("entity 0: expected yaw=%.2f at offset 20, got %.2f", p0.yaw, gotYaw)
	}
	if pkt[off+30] != uint8(p0.health) {
		t.Fatalf("entity 0: expected health=%d at offset 30, got %d", p0.health, pkt[off+30])
	}
	if pkt[off+29] != common.StateAlive {
		t.Fatalf("entity 0: expected state=StateAlive at offset 29, got %d", pkt[off+29])
	}

	off1 := off + snapshotEntitySize
	p1 := peers[1]
	if pkt[off1] != p1.id {
		t.Fatalf("entity 1: expected id=%d at offset %d, got %d", p1.id, off1, pkt[off1])
	}
	if pkt[off1+29] != common.StateDead {
		t.Fatalf("entity 1: expected state=StateDead (isKO=true) at offset 29, got %d", pkt[off1+29])
	}
}

func TestBuildSnapshotPacket_EmptyPeers(t *testing.T) {
	pkt := buildSnapshotPacket(5, nil)
	if len(pkt) != snapshotHeaderSize+1 {
		t.Fatalf("expected header+count only (%d bytes) for zero peers, got %d",
			snapshotHeaderSize+1, len(pkt))
	}
	if pkt[8] != 0 || pkt[snapshotHeaderSize] != 0 {
		t.Fatalf("expected both entity-count fields to be 0, got header=%d count-byte=%d",
			pkt[8], pkt[snapshotHeaderSize])
	}
}

// TestGroundClampY_MeadowSetsRealHeight -- Meadow is a real gentle roll now (2026-08-03,
// founder: "meadows are not completely flat my bro"), range [3,5], not a hardcoded flat 4.
// Checks the real range rather than an exact value the terrain generator no longer guarantees
// at any specific column.
func TestGroundClampY_MeadowSetsRealHeight(t *testing.T) {
	pos := system.Vec3{X: 3, Y: 999, Z: -7} // Y=999 simulates drift with no prior clamping
	got := groundClampY(pos)
	if got.Y < 3 || got.Y > 5 {
		t.Fatalf("expected Meadow ground height in [3,5] (gentle roll), got Y=%.2f", got.Y)
	}
	if got.X != pos.X || got.Z != pos.Z {
		t.Fatalf("expected X/Z untouched by ground clamping, got (%.2f,%.2f), want (%.2f,%.2f)",
			got.X, got.Z, pos.X, pos.Z)
	}
}

func TestGroundClampY_NegativeCoordinates(t *testing.T) {
	// Real regression coverage for floorDiv16-class bugs -- negative world coordinates are
	// routine here (origin-centered coordinate system), not an edge case to skip.
	pos := system.Vec3{X: -20.7, Y: 0, Z: -33.2}
	got := groundClampY(pos)
	if got.Y != 4 {
		t.Fatalf("expected Meadow ground height 4 at negative coordinates, got Y=%.2f", got.Y)
	}
}

func TestBuildSnapshotPacket_CapsAt255Peers(t *testing.T) {
	peers := make([]snapshotPeer, 300)
	for i := range peers {
		peers[i] = snapshotPeer{id: uint8(i % 256)}
	}
	pkt := buildSnapshotPacket(0, peers)
	wantLen := snapshotHeaderSize + 1 + 255*snapshotEntitySize
	if len(pkt) != wantLen {
		t.Fatalf("expected cap at 255 peers (%d bytes), got %d bytes", wantLen, len(pkt))
	}
	if pkt[8] != 255 {
		t.Fatalf("expected entity_count capped at 255, got %d", pkt[8])
	}
}

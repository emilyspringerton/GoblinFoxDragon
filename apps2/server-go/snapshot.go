package main

import (
	"encoding/binary"
	"math"
	"time"

	"dragonsnshit/packages2/common"
	"dragonsnshit/server/system"
)

// walkSpeed is Minecraft's own real sprint speed (5.6 blocks/sec) -- not an arbitrary number,
// a deliberate reference point matching this system's own Bedrock-protocol lineage (SHANKPIT's
// own CLAUDE.md: "DragonsNShit... runs on Dragonfly (Minecraft Bedrock Protocol fork)").
const walkSpeed = 5.6

// integrateMovement (backend-unification, 2026-08-03, founder: "server-authoritative position" --
// picked specifically over trusting client-reported position, a real cheat vector). This is the
// first place in this server that continuously simulates ON-FOOT player movement from raw
// UserCmd input -- racing.go's own applyRacingTick (SHANKPIT's sibling repo) is the only other
// continuous-movement precedent anywhere in this codebase family, and it's vehicle physics, the
// wrong shape for walking. No existing yaw/forward convention to match either: apps2/lobby (the
// one real client already speaking this protocol) sends raw Fwd/Str/Yaw and waits for the
// server's own snapshot to tell it where it ended up -- it does no local movement integration of
// its own. This function is the one place that convention is defined:
//
//	yaw 0 degrees faces +Z; increasing yaw rotates toward +X (standard atan2-style, arbitrary but
//	now fixed by this being the first and only definition).
//	apps/lobby's own input mapping (main.c: "if(k[SDL_SCANCODE_W]) fwd-=1") makes cmd.Fwd NEGATIVE
//	when the player presses forward -- so moving in the facing direction uses -cmd.Fwd, not +.
//
// dt is seconds (cmd.Msec / 1000.0, matching applyRacingTick's own dt convention).
func integrateMovement(pos system.Vec3, cmd common.UserCmd, dt float64) system.Vec3 {
	yawRad := float64(cmd.Yaw) * math.Pi / 180.0
	fwdX, fwdZ := math.Sin(yawRad), math.Cos(yawRad)
	rightX, rightZ := math.Cos(yawRad), -math.Sin(yawRad)

	dx := (-float64(cmd.Fwd)*fwdX + float64(cmd.Str)*rightX) * walkSpeed * dt
	dz := (-float64(cmd.Fwd)*fwdZ + float64(cmd.Str)*rightZ) * walkSpeed * dt

	pos.X += dx
	pos.Z += dz
	return pos
}

// snapshotEntitySize/snapshotHeaderSize match the real, compiler-determined C struct layout in
// apps2/lobby/packages/common/protocol.h's own NetHeader/NetPlayer -- verified directly (a
// standalone gcc probe printing sizeof/offsetof for both structs, 2026-08-03), not assumed from
// field lists alone, since C struct alignment padding is a real, silent wire-format risk
// (unsigned char id; unsigned char scene_id; unsigned int last_seq -- the compiler inserts 2
// padding bytes before last_seq to 4-byte-align it, invisible from the field list). Both structs
// come out to compiler-padded sizes (12 and 44 bytes) that happen to need no manual padding
// bytes in the Go-side buffer beyond what's written at each field's own real, verified offset.
const (
	snapshotHeaderSize = 12 // real sizeof(NetHeader): type,client_id,sequence,timestamp,entity_count,scene_id + padding
	snapshotEntitySize = 44 // real sizeof(NetPlayer): id,scene_id,last_seq,x,y,z,yaw,pitch,... + padding
)

// snapshotPeer is the subset of clientInfo buildSnapshotPacket actually needs -- kept separate
// from clientInfo itself so this stays a pure, easily-unit-tested function with no dependency on
// the real connection map.
type snapshotPeer struct {
	id     uint8
	pos    system.Vec3
	yaw    float32
	health int
	maxHP  int
	isKO   bool
}

// buildSnapshotPacket matches apps2/lobby's own net_process_snapshot byte-for-byte: a 12-byte
// NetHeader, one count byte immediately after it (net_process_snapshot reads this SEPARATELY
// from the header's own entity_count field, at cursor=sizeof(NetHeader) -- both are written here
// for consistency, but the client only actually consumes the standalone byte), then `count`
// tightly-packed 44-byte NetPlayer entries. Fields this backend has no real tracking for yet
// (current_weapon, is_shooting, in_vehicle, crouching, shield, ammo, hit_feedback,
// storm_charges, reward_feedback) are zero-filled -- honest, not a guess dressed up as data; the
// client simply renders a neutral/default state for those until this backend tracks them for
// real (weapon/inventory system, no PvE mobs yet to reward-feedback against, etc).
func buildSnapshotPacket(recipientID uint8, peers []snapshotPeer) []byte {
	if len(peers) > 255 {
		peers = peers[:255] // entity_count is a single byte; a real, named cap, not a silent truncation bug
	}
	out := make([]byte, snapshotHeaderSize+1+len(peers)*snapshotEntitySize)

	// NetHeader
	out[0] = common.PacketSnapshot
	out[1] = recipientID
	binary.LittleEndian.PutUint16(out[2:], 0) // sequence -- no real per-packet sequencing yet, named gap
	binary.LittleEndian.PutUint32(out[4:], uint32(time.Now().UnixMilli()))
	out[8] = uint8(len(peers))
	out[9] = 0 // scene_id -- this backend has no multi-scene concept yet (unlike SHANKPIT's own sibling)

	out[snapshotHeaderSize] = uint8(len(peers)) // the real count byte net_process_snapshot actually reads

	off := snapshotHeaderSize + 1
	for _, p := range peers {
		out[off+0] = p.id
		out[off+1] = 0 // scene_id
		binary.LittleEndian.PutUint32(out[off+4:], 0) // last_seq -- no client-side reconciliation yet, named gap
		binary.LittleEndian.PutUint32(out[off+8:], math.Float32bits(float32(p.pos.X)))
		binary.LittleEndian.PutUint32(out[off+12:], math.Float32bits(float32(p.pos.Y)))
		binary.LittleEndian.PutUint32(out[off+16:], math.Float32bits(float32(p.pos.Z)))
		binary.LittleEndian.PutUint32(out[off+20:], math.Float32bits(p.yaw))
		binary.LittleEndian.PutUint32(out[off+24:], 0) // pitch -- not tracked server-side, named gap
		out[off+28] = 0                                // current_weapon
		state := uint8(common.StateAlive)
		if p.isKO {
			state = common.StateDead
		}
		out[off+29] = state
		health := p.health
		if health < 0 {
			health = 0
		}
		if health > 255 {
			health = 255
		}
		out[off+30] = uint8(health)
		out[off+31] = 0 // shield
		out[off+32] = 0 // is_shooting
		out[off+33] = 0 // crouching
		binary.LittleEndian.PutUint32(out[off+36:], 0) // reward_feedback
		out[off+40] = 0                                // ammo
		out[off+41] = 0                                // in_vehicle
		out[off+42] = 0                                // hit_feedback
		out[off+43] = 0                                // storm_charges
		off += snapshotEntitySize
	}
	return out
}

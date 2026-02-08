package main

import (
	"encoding/binary"
	"math"
	"testing"

	"dragonsnshit/packages/common"
)

func TestParseUserCmd(t *testing.T) {
	buf := make([]byte, 64)
	const netHeaderSize = 12
	buf[0] = common.PacketUserCmd
	buf[netHeaderSize] = 1
	off := netHeaderSize + 1

	binary.LittleEndian.PutUint32(buf[off:], 42)
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], 99)
	off += 4
	binary.LittleEndian.PutUint16(buf[off:], 16)
	off += 4

	binary.LittleEndian.PutUint32(buf[off:], math.Float32bits(1.25))
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], math.Float32bits(-0.5))
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], math.Float32bits(90))
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], math.Float32bits(-10))
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], common.BtnAttack)
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], 1)

	cmd := parseUserCmd(buf, netHeaderSize+1)
	if cmd.Sequence != 42 {
		t.Fatalf("expected sequence 42, got %d", cmd.Sequence)
	}
	if cmd.Timestamp != 99 {
		t.Fatalf("expected timestamp 99, got %d", cmd.Timestamp)
	}
	if cmd.Msec != 16 {
		t.Fatalf("expected msec 16, got %d", cmd.Msec)
	}
	if cmd.Fwd != 1.25 {
		t.Fatalf("expected fwd 1.25, got %f", cmd.Fwd)
	}
	if cmd.Str != -0.5 {
		t.Fatalf("expected str -0.5, got %f", cmd.Str)
	}
	if cmd.Yaw != 90 {
		t.Fatalf("expected yaw 90, got %f", cmd.Yaw)
	}
	if cmd.Pitch != -10 {
		t.Fatalf("expected pitch -10, got %f", cmd.Pitch)
	}
	if cmd.Buttons != common.BtnAttack {
		t.Fatalf("expected buttons %d, got %d", common.BtnAttack, cmd.Buttons)
	}
	if cmd.WeaponIdx != 1 {
		t.Fatalf("expected weapon 1, got %d", cmd.WeaponIdx)
	}
}

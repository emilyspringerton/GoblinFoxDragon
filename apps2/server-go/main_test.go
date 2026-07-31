package main

import (
	"encoding/binary"
	"math"
	"os"
	"testing"
	"time"

	"dragonsnshit/packages2/common"
	"dragonsnshit/server/idunaclient"
	jobpkg "dragonsnshit/server/job"
	"dragonsnshit/server/skillchain"
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

// Backend-unification Sprint 3 (EMILY/BACKLOG.md, 2026-07-31): resolveWSCast is the pure
// decision core of PacketWSCast handling, extracted for the same reason parseUserCmd was
// (TestParseUserCmd above) -- unit-testable without a live UDP loop.

func TestResolveWSCastUnknownSkill(t *testing.T) {
	_, _, ok := resolveWSCast("Not A Real Weapon Skill", map[string]wsChainState{}, "target-1", 1, 2, time.Now())
	if ok {
		t.Fatal("expected ok=false for an unknown weapon skill")
	}
}

func TestResolveWSCastNoChain(t *testing.T) {
	result, newState, ok := resolveWSCast("Fast Blade", map[string]wsChainState{}, "target-1", 1, 2, time.Now())
	if !ok {
		t.Fatal("expected a real weapon skill to resolve successfully")
	}
	if result.Chained {
		t.Fatal("expected no chain with no prior weapon skill on this target")
	}
	if result.Damage != placeholderPlayerDamage*3 {
		t.Fatalf("expected unchained damage %d, got %d", placeholderPlayerDamage*3, result.Damage)
	}
	if len(newState.Attrs) == 0 {
		t.Fatal("expected newState to carry Fast Blade's real resonance attributes")
	}
}

func TestResolveWSCastFormsSkillchain(t *testing.T) {
	// Shining Blade (Transfixion) -> Burning Blade (Liquefaction) is a real Tier-2 Fusion
	// closure per server/skillchain's own combinationTable.
	now := time.Now()
	wsChains := map[string]wsChainState{
		"target-1": {Attrs: skillchain.CanonicalWeaponSkills["Shining Blade"].Attrs, At: now},
	}
	result, _, ok := resolveWSCast("Burning Blade", wsChains, "target-1", 1, 2, now.Add(2*time.Second))
	if !ok {
		t.Fatal("expected Burning Blade to resolve successfully")
	}
	if !result.Chained {
		t.Fatal("expected Shining Blade -> Burning Blade to form a real skillchain")
	}
	if result.Resonance != "Fusion" {
		t.Fatalf("expected Fusion resonance, got %q", result.Resonance)
	}
	if result.Tier != 2 {
		t.Fatalf("expected Tier 2, got %d", result.Tier)
	}
	baseDamage := placeholderPlayerDamage * 3
	wantDamage := baseDamage + int(float64(baseDamage)*0.35)
	if result.Damage != wantDamage {
		t.Fatalf("expected chained damage %d, got %d", wantDamage, result.Damage)
	}
}

// Backend-unification follow-up (2026-07-31): fetchCharacterCombatStats falls back to WAR/level
// 1 when IDUNA is unreachable or has no character row -- a real, deterministic path to test
// without a live IDUNA server (point the client at a port nothing is listening on).
func TestFetchCharacterCombatStatsFallsBackOnUnreachableIDUNA(t *testing.T) {
	oldURL := os.Getenv("IDUNA_BASE_URL")
	defer os.Setenv("IDUNA_BASE_URL", oldURL)
	os.Setenv("IDUNA_BASE_URL", "http://127.0.0.1:1") // nothing listens here

	client := idunaclient.New()
	jobMain, level, maxHP, currentXP := fetchCharacterCombatStats(client, "some-character-id")

	if jobMain != jobpkg.WAR {
		t.Fatalf("expected fallback job %q, got %q", jobpkg.WAR, jobMain)
	}
	if level != 1 {
		t.Fatalf("expected fallback level 1, got %d", level)
	}
	if currentXP != 0 {
		t.Fatalf("expected fallback currentXP 0, got %d", currentXP)
	}
	wantHP, err := jobpkg.HPAtLevel(jobpkg.WAR, 1)
	if err != nil {
		t.Fatalf("jobpkg.HPAtLevel(WAR, 1) unexpectedly errored: %v", err)
	}
	if maxHP != wantHP {
		t.Fatalf("expected fallback HP %d (WAR level 1), got %d", wantHP, maxHP)
	}
}

func TestResolveWSCastChainWindowExpired(t *testing.T) {
	now := time.Now()
	wsChains := map[string]wsChainState{
		"target-1": {Attrs: skillchain.CanonicalWeaponSkills["Shining Blade"].Attrs, At: now},
	}
	// Well outside skillchain.DefaultChainWindow (8s).
	result, _, ok := resolveWSCast("Burning Blade", wsChains, "target-1", 1, 2, now.Add(30*time.Second))
	if !ok {
		t.Fatal("expected Burning Blade to still resolve, just unchained")
	}
	if result.Chained {
		t.Fatal("expected no chain once the window has expired")
	}
}

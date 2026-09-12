package main

import (
	"testing"

	"dragonsnshit/server/gear"
	"dragonsnshit/server/itemdef"
	"dragonsnshit/server/xp"
)

func testRegistryWithTypedSword(t *testing.T) *itemdef.Registry {
	t.Helper()
	reg := itemdef.NewRegistry()
	if err := reg.LoadJSON([]byte(`[{"id":3,"name":"Sword","category":"weapon","weapon_type":"sword",
		"equip_slots":["main","off"],"stack_size":1,"stats":{"attack":10,"str":1}}]`)); err != nil {
		t.Fatalf("load test sword: %v", err)
	}
	return reg
}

func TestCurrentWeaponType_UnarmedIsH2H(t *testing.T) {
	p := newTestPlayer()
	p.equip = gear.NewEquipment()
	if got := currentWeaponType(p); got != weaponTypeH2H {
		t.Errorf("currentWeaponType(unarmed) = %q, want %q", got, weaponTypeH2H)
	}
}

func TestCurrentWeaponType_NilEquipIsH2H(t *testing.T) {
	p := newTestPlayer()
	p.equip = nil
	if got := currentWeaponType(p); got != weaponTypeH2H {
		t.Errorf("currentWeaponType(nil equip) = %q, want %q", got, weaponTypeH2H)
	}
}

func TestCurrentWeaponType_EquippedSwordReturnsSword(t *testing.T) {
	oldReg := itemdefReg
	itemdefReg = testRegistryWithTypedSword(t)
	t.Cleanup(func() { itemdefReg = oldReg })

	p := newTestPlayer()
	p.equip = gear.NewEquipment()
	if err := p.equip.Equip(gear.SlotMainHand, gear.ItemEntry{ItemID: "sword", DefID: 3}); err != nil {
		t.Fatalf("equip: %v", err)
	}
	if got := currentWeaponType(p); got != "sword" {
		t.Errorf("currentWeaponType(Sword equipped) = %q, want %q", got, "sword")
	}
}

// TestCurrentWeaponType_UntaggedWeaponFallsBackToH2H guards a real, honest degrade: an item with
// Category "weapon" but no authored weapon_type (not yet backfilled) must not crash or silently
// resolve to a made-up type -- it degrades to h2h, the same real default an empty main hand gets.
func TestCurrentWeaponType_UntaggedWeaponFallsBackToH2H(t *testing.T) {
	oldReg := itemdefReg
	reg := itemdef.NewRegistry()
	if err := reg.LoadJSON([]byte(`[{"id":99,"name":"Mystery Weapon","category":"weapon",
		"equip_slots":["main"],"stack_size":1}]`)); err != nil {
		t.Fatalf("load test item: %v", err)
	}
	itemdefReg = reg
	t.Cleanup(func() { itemdefReg = oldReg })

	p := newTestPlayer()
	p.equip = gear.NewEquipment()
	if err := p.equip.Equip(gear.SlotMainHand, gear.ItemEntry{ItemID: "mystery", DefID: 99}); err != nil {
		t.Fatalf("equip: %v", err)
	}
	if got := currentWeaponType(p); got != weaponTypeH2H {
		t.Errorf("currentWeaponType(untagged weapon) = %q, want fallback %q", got, weaponTypeH2H)
	}
}

func TestWeaponSkillCap_ScalesWithCharacterLevel(t *testing.T) {
	cases := []struct {
		level, want int
	}{
		{1, 10},
		{10, 100},
		{25, 250},
	}
	for _, c := range cases {
		if got := weaponSkillCap(c.level); got != c.want {
			t.Errorf("weaponSkillCap(%d) = %d, want %d", c.level, got, c.want)
		}
	}
}

func TestWeaponSkillCap_ZeroLevelStillGrantsARealFloor(t *testing.T) {
	if got := weaponSkillCap(0); got != weaponSkillCapPerLevel {
		t.Errorf("weaponSkillCap(0) = %d, want the real floor %d", got, weaponSkillCapPerLevel)
	}
}

func TestWeaponSkillLevel_NilMapReturnsZero(t *testing.T) {
	p := newTestPlayer()
	if got := weaponSkillLevel(p, "sword"); got != 0 {
		t.Errorf("weaponSkillLevel(nil map) = %d, want 0", got)
	}
}

func TestGainWeaponSkill_LazilyInitializesAndIncrements(t *testing.T) {
	p := newTestPlayer()
	p.equip = gear.NewEquipment() // unarmed -> h2h
	if p.weaponSkills != nil {
		t.Fatalf("expected nil weaponSkills before first gain")
	}
	gainWeaponSkill(p, 5)
	if got := weaponSkillLevel(p, weaponTypeH2H); got != 1 {
		t.Errorf("weaponSkillLevel(h2h) after 1 gain = %d, want 1", got)
	}
}

func TestGainWeaponSkill_StopsAtLevelCap(t *testing.T) {
	p := newTestPlayer()
	p.equip = gear.NewEquipment() // unarmed -> h2h
	const level = 2               // cap = 20
	cap := weaponSkillCap(level)
	for i := 0; i < cap+50; i++ {
		gainWeaponSkill(p, level)
	}
	if got := weaponSkillLevel(p, weaponTypeH2H); got != cap {
		t.Errorf("weaponSkillLevel(h2h) after over-training = %d, want capped at %d", got, cap)
	}
}

func TestGainWeaponSkill_TracksSeparateWeaponTypesIndependently(t *testing.T) {
	oldReg := itemdefReg
	itemdefReg = testRegistryWithTypedSword(t)
	t.Cleanup(func() { itemdefReg = oldReg })

	p := newTestPlayer()
	p.equip = gear.NewEquipment()
	gainWeaponSkill(p, 10) // unarmed -> h2h
	gainWeaponSkill(p, 10)

	if err := p.equip.Equip(gear.SlotMainHand, gear.ItemEntry{ItemID: "sword", DefID: 3}); err != nil {
		t.Fatalf("equip: %v", err)
	}
	gainWeaponSkill(p, 10) // now sword

	if got := weaponSkillLevel(p, weaponTypeH2H); got != 2 {
		t.Errorf("h2h skill = %d, want 2 (unaffected by later sword training)", got)
	}
	if got := weaponSkillLevel(p, "sword"); got != 1 {
		t.Errorf("sword skill = %d, want 1", got)
	}
}

func TestCharLevelOf_NilCharXPDefaultsToOne(t *testing.T) {
	p := &player{}
	if got := charLevelOf(p); got != 1 {
		t.Errorf("charLevelOf(nil charXP) = %d, want 1", got)
	}
}

func TestCharLevelOf_ReadsRealLevel(t *testing.T) {
	p := newTestPlayer()
	p.charXP = xp.NewCharXP()
	p.charXP.Level = 7
	if got := charLevelOf(p); got != 7 {
		t.Errorf("charLevelOf = %d, want 7", got)
	}
}

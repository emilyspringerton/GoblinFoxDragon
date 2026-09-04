package main

import (
	"testing"

	combatTp "dragonsnshit/server/combat"
	"dragonsnshit/server/gear"
	"dragonsnshit/server/itemdef"
)

// TestWeaponDelayFor is this package's first real unit test -- weaponDelayFor is small and
// self-contained enough (only p.equip + the package-level itemdefReg, not the full gw global
// state every other command handler needs) to test in isolation. Closes the real, found-live
// bug this function fixes: TP gain used to always call combatTp.AddTP with the hardcoded
// combatTp.Delay1HSword constant regardless of what weapon (or nothing) a player had equipped.
func TestWeaponDelayFor_NoWeaponEquipped_UsesHandToHand(t *testing.T) {
	p := &player{equip: gear.NewEquipment()}
	got := weaponDelayFor(p)
	if got != combatTp.DelayHtH {
		t.Errorf("expected DelayHtH (%d) for bare hands, got %d", combatTp.DelayHtH, got)
	}
}

func TestWeaponDelayFor_EquippedWeaponWithRealDelay_UsesItsOwnValue(t *testing.T) {
	itemdefReg = itemdef.NewRegistry()
	if err := itemdefReg.LoadJSON([]byte(`[{"id":9001,"name":"Test Rapier","category":"weapon","equip_slots":["main"],"stack_size":1,"delay":168}]`)); err != nil {
		t.Fatalf("load test item: %v", err)
	}

	p := &player{equip: gear.NewEquipment()}
	if err := p.equip.Equip(gear.SlotMainHand, gear.ItemEntry{ItemID: "test-rapier", DefID: 9001}); err != nil {
		t.Fatalf("equip: %v", err)
	}

	got := weaponDelayFor(p)
	if got != 168 {
		t.Errorf("expected the equipped weapon's own real delay (168), got %d", got)
	}
}

func TestWeaponDelayFor_EquippedWeaponWithoutRealDelayYet_FallsBackToDelay1HSword(t *testing.T) {
	itemdefReg = itemdef.NewRegistry()
	// Real, honest legacy case: an item that exists in the registry but was never backfilled
	// with a real Delay value (the zero value) -- must fall back to the same Delay1HSword
	// every player got before this fix, not silently treat 0 as "instant."
	if err := itemdefReg.LoadJSON([]byte(`[{"id":9002,"name":"Legacy Weapon","category":"weapon","equip_slots":["main"],"stack_size":1}]`)); err != nil {
		t.Fatalf("load test item: %v", err)
	}

	p := &player{equip: gear.NewEquipment()}
	if err := p.equip.Equip(gear.SlotMainHand, gear.ItemEntry{ItemID: "legacy-weapon", DefID: 9002}); err != nil {
		t.Fatalf("equip: %v", err)
	}

	got := weaponDelayFor(p)
	if got != combatTp.Delay1HSword {
		t.Errorf("expected the pre-fix fallback Delay1HSword (%d) for a not-yet-backfilled weapon, got %d", combatTp.Delay1HSword, got)
	}
}

func TestWeaponDelayFor_EquippedWeaponNotInRegistry_FallsBackToDelay1HSword(t *testing.T) {
	itemdefReg = itemdef.NewRegistry() // empty registry -- DefID 0 (the zero value) never resolves

	p := &player{equip: gear.NewEquipment()}
	if err := p.equip.Equip(gear.SlotMainHand, gear.ItemEntry{ItemID: "unregistered-item"}); err != nil {
		t.Fatalf("equip: %v", err)
	}

	got := weaponDelayFor(p)
	if got != combatTp.Delay1HSword {
		t.Errorf("expected the fallback Delay1HSword (%d) for a legacy item with no real itemdef entry, got %d", combatTp.Delay1HSword, got)
	}
}

package main

import (
	"bufio"
	"bytes"
	"testing"

	"dragonsnshit/server/gear"
	"dragonsnshit/server/itemdef"
)

// TestGrantStartingGear_EquipsRealSword guards the real, founder-requested feature (2026-09-12,
// S412-03): "we need a sword given to the player when they make their account." Uses the exact
// real item def (id 3, "Sword", {"attack":10,"str":1}) the founder named directly.
func TestGrantStartingGear_EquipsRealSword(t *testing.T) {
	oldReg := itemdefReg
	itemdefReg = itemdef.NewRegistry()
	if err := itemdefReg.LoadJSON([]byte(`[{"id":3,"name":"Sword","category":"weapon",
		"equip_slots":["main","off"],"stack_size":1,"stats":{"attack":10,"str":1}}]`)); err != nil {
		t.Fatalf("load test sword: %v", err)
	}
	t.Cleanup(func() { itemdefReg = oldReg })

	p := newTestPlayer()
	p.equip = gear.NewEquipment()
	var buf bytes.Buffer
	p.w = bufio.NewWriter(&buf)

	grantStartingGear(p)

	entry, err := p.equip.ItemAt(gear.SlotMainHand)
	if err != nil {
		t.Fatalf("expected a real item in the main-hand slot after grantStartingGear, got error: %v", err)
	}
	if entry.DefID != 3 {
		t.Errorf("equipped DefID = %d, want 3 (the real Sword)", entry.DefID)
	}
	if equipAttackBonus(p) != 10 {
		t.Errorf("equipAttackBonus after granting the starting sword = %d, want 10", equipAttackBonus(p))
	}
}

func TestGrantStartingGear_MissingItemDefDoesNotPanic(t *testing.T) {
	oldReg := itemdefReg
	itemdefReg = itemdef.NewRegistry() // deliberately empty -- no "sword" registered
	t.Cleanup(func() { itemdefReg = oldReg })

	p := newTestPlayer()
	p.equip = gear.NewEquipment()
	grantStartingGear(p) // must not panic

	if _, err := p.equip.ItemAt(gear.SlotMainHand); err == nil {
		t.Error("expected the main-hand slot to remain empty when the item def is missing")
	}
}

func TestGrantStartingGear_NilEquipDoesNotPanic(t *testing.T) {
	p := newTestPlayer()
	p.equip = nil
	grantStartingGear(p) // must not panic
}

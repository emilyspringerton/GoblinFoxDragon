package main

import (
	"os"
	"testing"

	"dragonsnshit/server/gear"
	"dragonsnshit/server/idunaclient"
	"dragonsnshit/server/itemdef"
)

// equip_persist_test.go guards the real production data-loss fix (2026-09-12, founder real-time:
// "gear needs to persist what the fuck why was that deferred"). gw.iduna is a concrete
// *idunaclient.Client (not an interface), so (matching this file's own established testing
// boundary, see char_progress_sync_test.go) persistEquipSlot/loadEquipmentFromIDUNA's real
// gw.iduna calls are untested directly; resolveEquipEntry -- the one real, pure piece of new
// logic -- is fully testable in isolation.

func TestResolveEquipEntry_KnownItemResolvesRealDefIDAndIL(t *testing.T) {
	oldReg := itemdefReg
	itemdefReg = testRegistryWithTypedSword(t)
	t.Cleanup(func() { itemdefReg = oldReg })

	entry, ok := resolveEquipEntry("sword")
	if !ok {
		t.Fatal("expected sword to resolve")
	}
	if entry.ItemID != "sword" || entry.DefID != 3 {
		t.Errorf("unexpected entry: %+v", entry)
	}
	if entry.IL != 1 {
		t.Errorf("expected default IL 1 (no item_level stat authored), got %d", entry.IL)
	}
}

func TestResolveEquipEntry_UnknownItemDegradesGracefully(t *testing.T) {
	oldReg := itemdefReg
	itemdefReg = itemdef.NewRegistry() // empty -- "sword" was removed/renamed since persisting
	t.Cleanup(func() { itemdefReg = oldReg })

	_, ok := resolveEquipEntry("sword")
	if ok {
		t.Error("expected a removed/unknown item_id to fail to resolve, not silently succeed")
	}
}

func TestResolveEquipEntry_UsesRealAuthoredItemLevel(t *testing.T) {
	oldReg := itemdefReg
	reg := itemdef.NewRegistry()
	if err := reg.LoadJSON([]byte(`[{"id":50,"name":"Fancy Sword","category":"weapon","weapon_type":"sword",
		"equip_slots":["main"],"stack_size":1,"stats":{"item_level":117,"attack":20}}]`)); err != nil {
		t.Fatalf("load test item: %v", err)
	}
	itemdefReg = reg
	t.Cleanup(func() { itemdefReg = oldReg })

	entry, ok := resolveEquipEntry("fancy-sword")
	if !ok {
		t.Fatal("expected fancy-sword to resolve")
	}
	if entry.IL != 117 {
		t.Errorf("expected IL 117 from the item's own authored item_level stat, got %d", entry.IL)
	}
}

// TestPersistEquipSlot_NoBoundCharacterIsANoOp guards a real, defensive case: a player whose
// slot isn't (yet) bound to a real character ID (e.g. mid-guest-name-entry) must not panic.
func TestPersistEquipSlot_NoBoundCharacterIsANoOp(t *testing.T) {
	oldGw := gw
	gw = &world{charIDBySlot: map[string]string{}}
	t.Cleanup(func() { gw = oldGw })

	p := &player{slot: "unbound-slot"}
	persistEquipSlot(p, "main", "sword") // must not panic
}

// TestLoadEquipmentFromIDUNA_NilEquipIsANoOp guards the same real, defensive case
// grantStartingGear's own nil-equip check already established.
func TestLoadEquipmentFromIDUNA_NilEquipIsANoOp(t *testing.T) {
	oldGw := gw
	gw = &world{charIDBySlot: map[string]string{}}
	t.Cleanup(func() { gw = oldGw })

	p := &player{equip: nil}
	loadEquipmentFromIDUNA(p, "char-1") // must not panic
}

// TestLoadEquipmentFromIDUNA_UnreachableIDUNADegradesGracefully guards a real, defensive case:
// an IDUNA outage (GetEquipment returns an error) must leave p.equip untouched, not panic --
// matching every other idunaclient call site's own best-effort convention in this file.
func TestLoadEquipmentFromIDUNA_UnreachableIDUNADegradesGracefully(t *testing.T) {
	oldURL := os.Getenv("IDUNA_BASE_URL")
	os.Setenv("IDUNA_BASE_URL", "http://127.0.0.1:1") // real, always-refused port
	t.Cleanup(func() { os.Setenv("IDUNA_BASE_URL", oldURL) })

	oldGw := gw
	gw = &world{charIDBySlot: map[string]string{}, iduna: idunaclient.New()}
	t.Cleanup(func() { gw = oldGw })

	p := &player{equip: gear.NewEquipment()}
	loadEquipmentFromIDUNA(p, "char-1") // must not panic

	entry, err := p.equip.ItemAt(gear.SlotMainHand)
	if err == nil && entry != nil {
		t.Errorf("expected main hand to stay empty after a failed load, got %+v", entry)
	}
}

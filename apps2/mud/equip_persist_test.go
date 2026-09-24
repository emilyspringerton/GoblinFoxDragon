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

// TestResolveEquipEntry_LegacyItemFallsBackToItemIL is the real regression guard for the
// 2026-09-24 gear-persistence bug (see resolveEquipEntry's own doc comment): every armor piece
// itemIL names as "legacy" (never registered in itemdefReg -- leather-legs among them) must
// still resolve on reload, not silently vanish, mirroring cmdEquip's own real fallback.
func TestResolveEquipEntry_LegacyItemFallsBackToItemIL(t *testing.T) {
	oldReg := itemdefReg
	itemdefReg = itemdef.NewRegistry() // empty -- leather-legs was never a real registry entry
	t.Cleanup(func() { itemdefReg = oldReg })

	for itemID, wantIL := range itemIL {
		entry, ok := resolveEquipEntry(itemID)
		if !ok {
			t.Errorf("expected legacy item %q to resolve via the itemIL fallback, got ok=false", itemID)
			continue
		}
		if entry.ItemID != itemID || entry.IL != wantIL || entry.DefID != 0 {
			t.Errorf("resolveEquipEntry(%q) = %+v, want ItemID=%q IL=%d DefID=0", itemID, entry, itemID, wantIL)
		}
	}
}

// TestResolveEquipEntry_RegistryTakesPriorityOverItemIL guards the real precedence order
// cmdEquip's own resolution already establishes: a real itemdefReg entry wins even if the same
// item_id also happens to appear in the itemIL legacy map.
func TestResolveEquipEntry_RegistryTakesPriorityOverItemIL(t *testing.T) {
	oldReg := itemdefReg
	reg := itemdef.NewRegistry()
	if err := reg.LoadJSON([]byte(`[{"id":9001,"name":"Leather Legs","category":"armor",
		"equip_slots":["legs"],"stack_size":1,"stats":{"item_level":42}}]`)); err != nil {
		t.Fatalf("load test item: %v", err)
	}
	itemdefReg = reg
	t.Cleanup(func() { itemdefReg = oldReg })

	entry, ok := resolveEquipEntry("leather-legs")
	if !ok {
		t.Fatal("expected leather-legs to resolve")
	}
	if entry.DefID != 9001 || entry.IL != 42 {
		t.Errorf("expected the real registry entry (DefID 9001, IL 42) to win over itemIL's DefID-less fallback, got %+v", entry)
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

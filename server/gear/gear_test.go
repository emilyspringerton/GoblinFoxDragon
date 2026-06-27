package gear

import (
	"testing"

	"dragonsnshit/server/itemdef"
)

func equip(e *Equipment, slot Slot, il int) {
	e.Equip(slot, ItemEntry{ItemID: "item-" + slot, IL: il})
}

// ── Equip + ItemAt ────────────────────────────────────────────────────────────

func TestEquip_SetsItem(t *testing.T) {
	eq := NewEquipment()
	if err := equip2(eq); err != nil {
		t.Fatal(err)
	}
	item, err := eq.ItemAt(SlotHead)
	if err != nil || item.IL != 100 {
		t.Errorf("ItemAt head: item=%v err=%v", item, err)
	}
}

func equip2(eq *Equipment) error {
	return eq.Equip(SlotHead, ItemEntry{ItemID: "helm-01", IL: 100})
}

func TestEquip_UnknownSlot(t *testing.T) {
	eq := NewEquipment()
	if err := eq.Equip("noggin", ItemEntry{ItemID: "x", IL: 50}); err != ErrUnknownSlot {
		t.Errorf("unknown slot: got %v, want ErrUnknownSlot", err)
	}
}

func TestItemAt_EmptySlot(t *testing.T) {
	eq := NewEquipment()
	_, err := eq.ItemAt(SlotBody)
	if err != ErrSlotEmpty {
		t.Errorf("empty slot: got %v, want ErrSlotEmpty", err)
	}
}

func TestItemAt_UnknownSlot(t *testing.T) {
	eq := NewEquipment()
	if _, err := eq.ItemAt("noggin"); err != ErrUnknownSlot {
		t.Errorf("unknown slot: got %v, want ErrUnknownSlot", err)
	}
}

// ── Unequip ───────────────────────────────────────────────────────────────────

func TestUnequip_RemovesItem(t *testing.T) {
	eq := NewEquipment()
	eq.Equip(SlotHead, ItemEntry{ItemID: "helm", IL: 100})
	item, err := eq.Unequip(SlotHead)
	if err != nil || item.ItemID != "helm" {
		t.Errorf("Unequip: item=%v err=%v", item, err)
	}
	if _, err := eq.ItemAt(SlotHead); err != ErrSlotEmpty {
		t.Error("slot should be empty after Unequip")
	}
}

func TestUnequip_EmptySlot(t *testing.T) {
	eq := NewEquipment()
	if _, err := eq.Unequip(SlotHead); err != ErrSlotEmpty {
		t.Errorf("unequip empty: got %v, want ErrSlotEmpty", err)
	}
}

// ── EffectiveIL ───────────────────────────────────────────────────────────────

func TestEffectiveIL_Average(t *testing.T) {
	eq := NewEquipment()
	eq.Equip(SlotHead, ItemEntry{ItemID: "a", IL: 100})
	eq.Equip(SlotBody, ItemEntry{ItemID: "b", IL: 200})
	// average = 150
	il, err := eq.EffectiveIL()
	if err != nil || il != 150 {
		t.Errorf("EffectiveIL: got %d err=%v, want 150", il, err)
	}
}

func TestEffectiveIL_EmptySlotsExcluded(t *testing.T) {
	eq := NewEquipment()
	eq.Equip(SlotHead, ItemEntry{ItemID: "a", IL: 120})
	// only 1 slot occupied
	il, _ := eq.EffectiveIL()
	if il != 120 {
		t.Errorf("single item IL: got %d, want 120", il)
	}
}

func TestEffectiveIL_NoEquipment(t *testing.T) {
	eq := NewEquipment()
	_, err := eq.EffectiveIL()
	if err != ErrNoEquipment {
		t.Errorf("empty: got %v, want ErrNoEquipment", err)
	}
}

func TestEffectiveIL_AllSlots(t *testing.T) {
	eq := NewEquipment()
	for _, slot := range AllSlots {
		eq.Equip(slot, ItemEntry{ItemID: "x-" + slot, IL: 100})
	}
	il, err := eq.EffectiveIL()
	if err != nil || il != 100 {
		t.Errorf("all IL=100: got %d err=%v", il, err)
	}
}

// ── OccupiedCount ─────────────────────────────────────────────────────────────

func TestOccupiedCount(t *testing.T) {
	eq := NewEquipment()
	eq.Equip(SlotHead, ItemEntry{ItemID: "a", IL: 100})
	eq.Equip(SlotBody, ItemEntry{ItemID: "b", IL: 100})
	if eq.OccupiedCount() != 2 {
		t.Errorf("OccupiedCount: got %d, want 2", eq.OccupiedCount())
	}
}

func TestOccupiedCount_AfterUnequip(t *testing.T) {
	eq := NewEquipment()
	eq.Equip(SlotHead, ItemEntry{ItemID: "a", IL: 100})
	eq.Unequip(SlotHead)
	if eq.OccupiedCount() != 0 {
		t.Errorf("after unequip: got %d, want 0", eq.OccupiedCount())
	}
}

// ── ComputeStats ──────────────────────────────────────────────────────────────

func makeTestRegistry(t *testing.T) *itemdef.Registry {
	t.Helper()
	reg := itemdef.NewRegistry()
	if err := reg.LoadJSON([]byte(`[
		{"id":1,"name":"Iron Sword","category":"weapon","equip_slots":["main"],"jobs":["WAR","DRK"],"level":1,"stats":{"attack":12,"str":2},"stack_size":1},
		{"id":2,"name":"Leather Helm","category":"armor","equip_slots":["head"],"level":1,"stats":{"defense":8,"vit":1},"stack_size":1},
		{"id":3,"name":"Bronze Ring","category":"accessory","equip_slots":["ring-l","ring-r"],"level":1,"stats":{"mnd":3},"stack_size":1}
	]`)); err != nil {
		t.Fatalf("load registry: %v", err)
	}
	return reg
}

func TestComputeStats_Empty(t *testing.T) {
	eq := NewEquipment()
	reg := makeTestRegistry(t)
	stats := eq.ComputeStats(reg)
	if len(stats) != 0 {
		t.Errorf("expected empty stats, got %v", stats)
	}
}

func TestComputeStats_SingleItem(t *testing.T) {
	eq := NewEquipment()
	eq.Equip(SlotMainHand, ItemEntry{ItemID: "sword-1", IL: 1, DefID: 1})
	reg := makeTestRegistry(t)
	stats := eq.ComputeStats(reg)
	if stats["attack"] != 12 || stats["str"] != 2 {
		t.Errorf("single item stats: got %v", stats)
	}
}

func TestComputeStats_MultipleItems(t *testing.T) {
	eq := NewEquipment()
	eq.Equip(SlotMainHand, ItemEntry{ItemID: "sword-1", IL: 1, DefID: 1}) // attack+12, str+2
	eq.Equip(SlotHead, ItemEntry{ItemID: "helm-1", IL: 1, DefID: 2})      // defense+8, vit+1
	eq.Equip(SlotRingL, ItemEntry{ItemID: "ring-1", IL: 1, DefID: 3})     // mnd+3
	reg := makeTestRegistry(t)
	stats := eq.ComputeStats(reg)
	if stats["attack"] != 12 || stats["defense"] != 8 || stats["mnd"] != 3 {
		t.Errorf("multi-item stats: got %v", stats)
	}
}

func TestComputeStats_UnknownDefIDSkipped(t *testing.T) {
	eq := NewEquipment()
	eq.Equip(SlotHead, ItemEntry{ItemID: "mystery", IL: 50, DefID: 999})
	reg := makeTestRegistry(t)
	stats := eq.ComputeStats(reg)
	if len(stats) != 0 {
		t.Errorf("unknown def_id should contribute no stats, got %v", stats)
	}
}

// ── CanEquip ─────────────────────────────────────────────────────────────────

func TestCanEquip_ValidWAR(t *testing.T) {
	eq := NewEquipment()
	reg := makeTestRegistry(t)
	def, _ := reg.ByID(1) // Iron Sword: WAR/DRK, main, level 1
	if err := eq.CanEquip(SlotMainHand, def, "WAR", 1); err != nil {
		t.Errorf("WAR should be able to equip Iron Sword: %v", err)
	}
}

func TestCanEquip_WrongJob(t *testing.T) {
	eq := NewEquipment()
	reg := makeTestRegistry(t)
	def, _ := reg.ByID(1) // Iron Sword: WAR/DRK only
	if err := eq.CanEquip(SlotMainHand, def, "WHM", 1); err == nil {
		t.Error("WHM should not be able to equip Iron Sword")
	}
}

func TestCanEquip_LevelTooLow(t *testing.T) {
	eq := NewEquipment()
	reg := itemdef.NewRegistry()
	reg.LoadJSON([]byte(`[{"id":10,"name":"HQ Sword","category":"weapon","equip_slots":["main"],"level":30,"stats":{},"stack_size":1}]`))
	def, _ := reg.ByID(10)
	if err := eq.CanEquip(SlotMainHand, def, "WAR", 5); err == nil {
		t.Error("level 5 should not equip a level 30 item")
	}
}

func TestCanEquip_WrongSlot(t *testing.T) {
	eq := NewEquipment()
	reg := makeTestRegistry(t)
	def, _ := reg.ByID(1) // Iron Sword: main slot only
	if err := eq.CanEquip(SlotHead, def, "WAR", 1); err == nil {
		t.Error("sword should not equip in head slot")
	}
}

func TestCanEquip_UnknownSlot(t *testing.T) {
	eq := NewEquipment()
	reg := makeTestRegistry(t)
	def, _ := reg.ByID(2)
	if err := eq.CanEquip("noggin", def, "WAR", 1); err != ErrUnknownSlot {
		t.Errorf("unknown slot: got %v, want ErrUnknownSlot", err)
	}
}

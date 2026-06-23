package gear

import "testing"

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

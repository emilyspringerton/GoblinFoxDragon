package inventory

import "testing"

func mkStack(itemID string, defID, qty, stackSize int) Stack {
	return Stack{ItemID: itemID, DefID: defID, Quantity: qty, StackSize: stackSize}
}

// ── Bag ───────────────────────────────────────────────────────────────────────

func TestBagAdd(t *testing.T) {
	b := NewBag(10)
	idx, err := b.Add(mkStack("uuid-1", 1, 1, 1))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if idx != 0 {
		t.Errorf("idx=%d want 0", idx)
	}
	if b.OccupiedCount() != 1 {
		t.Errorf("count=%d want 1", b.OccupiedCount())
	}
}

func TestBagFull(t *testing.T) {
	b := NewBag(2)
	b.Add(mkStack("a", 1, 1, 1))
	b.Add(mkStack("b", 2, 1, 1))
	_, err := b.Add(mkStack("c", 3, 1, 1))
	if err != ErrBagFull {
		t.Errorf("expected ErrBagFull, got %v", err)
	}
}

func TestBagStackMerge(t *testing.T) {
	b := NewBag(10)
	b.Add(mkStack("crystal-1", 4, 6, 12))
	idx, err := b.Add(mkStack("crystal-2", 4, 3, 12))
	if err != nil {
		t.Fatalf("merge Add: %v", err)
	}
	if b.OccupiedCount() != 1 {
		t.Errorf("expected 1 slot after merge, got %d", b.OccupiedCount())
	}
	s, _ := b.At(idx)
	if s.Quantity != 9 {
		t.Errorf("merged qty=%d want 9", s.Quantity)
	}
}

func TestBagStackOverflow(t *testing.T) {
	b := NewBag(10)
	b.Add(mkStack("c1", 4, 10, 12))
	_, err := b.Add(mkStack("c2", 4, 5, 12))
	if err != ErrStackOverflow {
		t.Errorf("expected ErrStackOverflow, got %v", err)
	}
}

func TestBagRemove(t *testing.T) {
	b := NewBag(10)
	b.Add(mkStack("sword", 1, 1, 1))
	if err := b.Remove(0, 1); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if b.OccupiedCount() != 0 {
		t.Errorf("expected empty bag after remove")
	}
}

func TestBagRemovePartial(t *testing.T) {
	b := NewBag(10)
	b.Add(mkStack("c", 4, 10, 12))
	b.Remove(0, 4)
	s, _ := b.At(0)
	if s.Quantity != 6 {
		t.Errorf("qty=%d want 6", s.Quantity)
	}
}

func TestBagRemoveEmptySlot(t *testing.T) {
	b := NewBag(10)
	if err := b.Remove(0, 1); err != ErrSlotEmpty {
		t.Errorf("expected ErrSlotEmpty, got %v", err)
	}
}

func TestBagMove(t *testing.T) {
	b := NewBag(10)
	b.Add(mkStack("sword", 1, 1, 1))
	if err := b.Move(0, 5); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if _, err := b.At(0); err != ErrSlotEmpty {
		t.Error("slot 0 should be empty after move")
	}
	s, err := b.At(5)
	if err != nil || s.ItemID != "sword" {
		t.Errorf("slot 5 should contain sword, got %v %v", s, err)
	}
}

func TestBagFind(t *testing.T) {
	b := NewBag(10)
	b.Add(mkStack("c1", 4, 6, 12))
	b.Add(mkStack("sword", 1, 1, 1))
	b.Add(mkStack("c2", 4, 3, 12)) // new slot (c1 is full at 6, StackSize 12 not full)
	indices := b.Find(4)
	if len(indices) != 1 {
		// They merged into one slot.
		t.Errorf("Find returned %d slots want 1", len(indices))
	}
}

func TestBagCount(t *testing.T) {
	b := NewBag(10)
	b.Add(mkStack("c1", 4, 6, 12))
	b.Add(mkStack("c2", 4, 3, 12))
	if b.Count(4) != 9 {
		t.Errorf("Count=%d want 9", b.Count(4))
	}
}

func TestBagBadIndex(t *testing.T) {
	b := NewBag(5)
	_, err := b.At(10)
	if err != ErrBadIndex {
		t.Errorf("expected ErrBadIndex, got %v", err)
	}
}

// ── Mog ───────────────────────────────────────────────────────────────────────

func TestMogDefaults(t *testing.T) {
	m := NewMog()
	if m.Inventory.Capacity != DefaultInventorySize {
		t.Errorf("inventory capacity=%d want %d", m.Inventory.Capacity, DefaultInventorySize)
	}
	if m.Storage.Capacity != DefaultStorageSize {
		t.Errorf("storage capacity=%d want %d", m.Storage.Capacity, DefaultStorageSize)
	}
}

func TestMogKeyItems(t *testing.T) {
	m := NewMog()
	m.AddKeyItem(100)
	if !m.HasKeyItem(100) {
		t.Error("should have key item 100")
	}
	m.RemoveKeyItem(100)
	if m.HasKeyItem(100) {
		t.Error("should not have key item 100 after remove")
	}
}

func TestMogRareConflict(t *testing.T) {
	m := NewMog()
	if err := m.CheckRare(5); err != nil {
		t.Error("first rare check should pass")
	}
	m.AddRare(5)
	if err := m.CheckRare(5); err != ErrRareConflict {
		t.Errorf("expected ErrRareConflict, got %v", err)
	}
	m.RemoveRare(5)
	if err := m.CheckRare(5); err != nil {
		t.Error("after remove, rare check should pass again")
	}
}

func TestMogExpandInventory(t *testing.T) {
	m := NewMog()
	newCap, err := m.ExpandInventory()
	if err != nil {
		t.Fatalf("ExpandInventory: %v", err)
	}
	if newCap != DefaultInventorySize+GobbiebagIncrement {
		t.Errorf("newCap=%d want %d", newCap, DefaultInventorySize+GobbiebagIncrement)
	}
	// Items from before expansion should still be there.
	m2 := NewMog()
	m2.Inventory.Add(mkStack("sword", 1, 1, 1))
	m2.ExpandInventory()
	if m2.Inventory.OccupiedCount() != 1 {
		t.Error("items lost after expansion")
	}
}

func TestMogExpandInventoryAtMax(t *testing.T) {
	m := NewMog()
	for m.Inventory.Capacity < MaxInventorySize {
		m.ExpandInventory()
	}
	_, err := m.ExpandInventory()
	if err == nil {
		t.Error("expected error at max capacity")
	}
}

func TestMogClearTemporary(t *testing.T) {
	m := NewMog()
	m.Temporary.Add(mkStack("tmp", 99, 1, 1))
	m.ClearTemporary()
	if m.Temporary.OccupiedCount() != 0 {
		t.Error("temporary bag should be empty after clear")
	}
}

package moghouse_test

import (
	"testing"

	"dragonsnshit/server/moghouse"
)

func TestNewMogHouseIsEmpty(t *testing.T) {
	m := moghouse.New("alice")
	if m.TotalItems() != 0 {
		t.Errorf("want 0 items, got %d", m.TotalItems())
	}
}

func TestStoreAndCount(t *testing.T) {
	m := moghouse.New("alice")
	if err := m.Store("sword-001", 2); err != nil {
		t.Fatal(err)
	}
	if m.Count("sword-001") != 2 {
		t.Errorf("want 2, got %d", m.Count("sword-001"))
	}
}

func TestStoreAccumulates(t *testing.T) {
	m := moghouse.New("alice")
	_ = m.Store("potion", 3)
	_ = m.Store("potion", 2)
	if m.Count("potion") != 5 {
		t.Errorf("want 5, got %d", m.Count("potion"))
	}
}

func TestStoreZeroQtyError(t *testing.T) {
	m := moghouse.New("alice")
	if err := m.Store("potion", 0); err == nil {
		t.Error("expected error for 0 qty")
	}
}

func TestStoreNegativeQtyError(t *testing.T) {
	m := moghouse.New("alice")
	if err := m.Store("potion", -1); err == nil {
		t.Error("expected error for negative qty")
	}
}

func TestStoreFull(t *testing.T) {
	m := moghouse.New("alice")
	if err := m.Store("potion", moghouse.Capacity); err != nil {
		t.Fatalf("store to capacity: %v", err)
	}
	if err := m.Store("potion", 1); err == nil {
		t.Error("expected ErrFull when over capacity")
	}
}

func TestStoreExactCapacity(t *testing.T) {
	m := moghouse.New("alice")
	if err := m.Store("item-a", moghouse.Capacity); err != nil {
		t.Fatalf("store exactly capacity: %v", err)
	}
	if m.TotalItems() != moghouse.Capacity {
		t.Errorf("want %d, got %d", moghouse.Capacity, m.TotalItems())
	}
}

func TestRetrieveSuccess(t *testing.T) {
	m := moghouse.New("alice")
	_ = m.Store("sword-001", 3)
	if err := m.Retrieve("sword-001", 2); err != nil {
		t.Fatal(err)
	}
	if m.Count("sword-001") != 1 {
		t.Errorf("want 1 remaining, got %d", m.Count("sword-001"))
	}
}

func TestRetrieveAllClearsEntry(t *testing.T) {
	m := moghouse.New("alice")
	_ = m.Store("ring", 1)
	if err := m.Retrieve("ring", 1); err != nil {
		t.Fatal(err)
	}
	if m.Count("ring") != 0 {
		t.Error("expected 0 after retrieving all")
	}
	if m.TotalItems() != 0 {
		t.Errorf("want 0 total, got %d", m.TotalItems())
	}
}

func TestRetrieveNotStored(t *testing.T) {
	m := moghouse.New("alice")
	if err := m.Retrieve("not-an-item", 1); err == nil {
		t.Error("expected ErrItemNotStored")
	}
}

func TestRetrieveInsufficientQty(t *testing.T) {
	m := moghouse.New("alice")
	_ = m.Store("arrow", 2)
	if err := m.Retrieve("arrow", 3); err == nil {
		t.Error("expected ErrInsufficientQty")
	}
}

func TestRetrieveZeroQtyError(t *testing.T) {
	m := moghouse.New("alice")
	_ = m.Store("item", 1)
	if err := m.Retrieve("item", 0); err == nil {
		t.Error("expected error for 0 qty retrieve")
	}
}

func TestAvailable(t *testing.T) {
	m := moghouse.New("alice")
	_ = m.Store("item-a", 10)
	want := moghouse.Capacity - 10
	if m.Available() != want {
		t.Errorf("want available=%d, got %d", want, m.Available())
	}
}

func TestSnapshot(t *testing.T) {
	m := moghouse.New("alice")
	_ = m.Store("axe", 2)
	_ = m.Store("helm", 1)
	snap := m.Snapshot()
	if snap["axe"] != 2 || snap["helm"] != 1 {
		t.Errorf("snapshot mismatch: %v", snap)
	}
	// Verify snapshot is a copy — mutating it shouldn't affect the house.
	snap["axe"] = 999
	if m.Count("axe") != 2 {
		t.Error("snapshot mutation affected MogHouse state")
	}
}

func TestCapacityConstant(t *testing.T) {
	if moghouse.Capacity != 50 {
		t.Errorf("want Capacity=50, got %d", moghouse.Capacity)
	}
}

func TestRegistryGetCreates(t *testing.T) {
	reg := moghouse.NewRegistry()
	h := reg.Get("bob")
	if h == nil {
		t.Fatal("expected non-nil MogHouse")
	}
	if h.OwnerID != "bob" {
		t.Errorf("OwnerID: want bob, got %s", h.OwnerID)
	}
}

func TestRegistryGetSameInstance(t *testing.T) {
	reg := moghouse.NewRegistry()
	h1 := reg.Get("carol")
	h2 := reg.Get("carol")
	if h1 != h2 {
		t.Error("expected same MogHouse instance for same ownerID")
	}
}

func TestRegistryPlayerCount(t *testing.T) {
	reg := moghouse.NewRegistry()
	reg.Get("x")
	reg.Get("y")
	reg.Get("z")
	if reg.PlayerCount() != 3 {
		t.Errorf("want 3 players, got %d", reg.PlayerCount())
	}
}

func TestRegistryIsolated(t *testing.T) {
	reg := moghouse.NewRegistry()
	h1 := reg.Get("p1")
	h2 := reg.Get("p2")
	_ = h1.Store("item", 5)
	if h2.Count("item") != 0 {
		t.Error("expected isolated mog houses for different players")
	}
}

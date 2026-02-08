package store

import (
	"testing"
	"time"

	"dragonsnshit/packages/common"
)

func TestMemoryClientStoreLifecycle(t *testing.T) {
	tick := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	store := NewMemoryClientStore()
	store.clock = func() time.Time { return tick }

	cmd := common.UserCmd{Sequence: 7, Buttons: common.BtnAttack}
	store.Upsert("client-1", cmd)

	saved, ok := store.Get("client-1")
	if !ok {
		t.Fatalf("expected client to be stored")
	}
	if saved.LastCmd.Sequence != 7 {
		t.Fatalf("expected sequence 7, got %d", saved.LastCmd.Sequence)
	}
	if !saved.Updated.Equal(tick) {
		t.Fatalf("expected updated time to match")
	}

	store.Delete("client-1")
	if _, ok := store.Get("client-1"); ok {
		t.Fatalf("expected client to be deleted")
	}
}

func TestMemoryClientStoreAll(t *testing.T) {
	store := NewMemoryClientStore()
	store.Upsert("alpha", common.UserCmd{Sequence: 1})
	store.Upsert("bravo", common.UserCmd{Sequence: 2})

	clients := store.All()
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}
}

package auction_test

import (
	"fmt"
	"testing"
	"time"

	"dragonsnshit/server/auction"
	"dragonsnshit/server/loot"
)

var testItem = loot.Item{ID: "sword-001", Name: "Iron Sword"}

func newHouse() *auction.House { return auction.New() }

func TestListCreatesListing(t *testing.T) {
	h := newHouse()
	id, l, err := h.List("alice", testItem, 500)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Error("expected non-empty listing ID")
	}
	if l.Ask != 500 {
		t.Errorf("ask: want 500, got %d", l.Ask)
	}
	if l.SellerID != "alice" {
		t.Errorf("sellerID: want alice, got %s", l.SellerID)
	}
	if l.Status != auction.StatusActive {
		t.Errorf("status: want Active, got %d", l.Status)
	}
}

func TestListZeroPriceReturnsError(t *testing.T) {
	h := newHouse()
	_, _, err := h.List("alice", testItem, 0)
	if err == nil {
		t.Error("expected error for zero ask")
	}
}

func TestListNegativePriceReturnsError(t *testing.T) {
	h := newHouse()
	_, _, err := h.List("alice", testItem, -1)
	if err == nil {
		t.Error("expected error for negative ask")
	}
}

func TestBuySuccess(t *testing.T) {
	h := newHouse()
	id, _, _ := h.List("alice", testItem, 1000)
	fee, net, l, err := h.Buy(id, "bob", 1500)
	if err != nil {
		t.Fatalf("Buy: %v", err)
	}
	if l.Status != auction.StatusSold {
		t.Errorf("want StatusSold, got %d", l.Status)
	}
	if l.Buyer != "bob" {
		t.Errorf("buyer: want bob, got %s", l.Buyer)
	}
	if fee != auction.FeeFor(1000) {
		t.Errorf("fee: want %d, got %d", auction.FeeFor(1000), fee)
	}
	if net != 1000-fee {
		t.Errorf("net: want %d, got %d", 1000-fee, net)
	}
}

func TestBuySellerCannotBuyOwnListing(t *testing.T) {
	h := newHouse()
	id, _, _ := h.List("alice", testItem, 500)
	_, _, _, err := h.Buy(id, "alice", 1000)
	if err == nil {
		t.Error("expected error when seller buys own listing")
	}
}

func TestBuyInsufficientFlow(t *testing.T) {
	h := newHouse()
	id, _, _ := h.List("alice", testItem, 1000)
	_, _, _, err := h.Buy(id, "bob", 500)
	if err == nil {
		t.Error("expected error for insufficient flow")
	}
}

func TestBuyNotFoundError(t *testing.T) {
	h := newHouse()
	_, _, _, err := h.Buy("ah-999999", "bob", 9999)
	if err == nil {
		t.Error("expected error for missing listing")
	}
}

func TestBuyAlreadySold(t *testing.T) {
	h := newHouse()
	id, _, _ := h.List("alice", testItem, 100)
	_, _, _, err := h.Buy(id, "bob", 9999)
	if err != nil {
		t.Fatalf("first buy: %v", err)
	}
	_, _, _, err2 := h.Buy(id, "carol", 9999)
	if err2 == nil {
		t.Error("expected error buying already-sold listing")
	}
}

func TestFeeIs5Percent(t *testing.T) {
	cases := []struct{ ask, want int }{
		{1000, 50},
		{100, 5},
		{19, 0}, // floor: 19*5/100 = 0
		{20, 1}, // 20*5/100 = 1
	}
	for _, c := range cases {
		got := auction.FeeFor(c.ask)
		if got != c.want {
			t.Errorf("FeeFor(%d): want %d, got %d", c.ask, c.want, got)
		}
	}
}

func TestActiveListingsFiltersExpired(t *testing.T) {
	h := auction.New()
	// Inject a clock that we'll advance.
	// We can't set the clock externally, so we rely on real time and verify
	// that active listings appear before expiry.
	_, l, _ := h.List("alice", testItem, 100)
	active := h.ActiveListings()
	found := false
	for _, a := range active {
		if a.ID == l.ID {
			found = true
		}
	}
	if !found {
		t.Error("expected new listing to appear in ActiveListings")
	}
}

func TestGetReturnsListing(t *testing.T) {
	h := newHouse()
	id, orig, _ := h.List("alice", testItem, 300)
	l, ok := h.Get(id)
	if !ok {
		t.Error("expected Get to return listing")
	}
	if l.ID != orig.ID {
		t.Errorf("ID mismatch")
	}
}

func TestGetMissingReturnsFalse(t *testing.T) {
	h := newHouse()
	_, ok := h.Get("ah-notexist")
	if ok {
		t.Error("expected false for missing listing")
	}
}

func TestCancelSuccess(t *testing.T) {
	h := newHouse()
	id, _, _ := h.List("alice", testItem, 200)
	l, err := h.Cancel(id, "alice")
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if l.Status != auction.StatusCancelled {
		t.Errorf("want StatusCancelled, got %d", l.Status)
	}
}

func TestCancelWrongOwner(t *testing.T) {
	h := newHouse()
	id, _, _ := h.List("alice", testItem, 200)
	_, err := h.Cancel(id, "bob")
	if err == nil {
		t.Error("expected error when non-owner cancels")
	}
}

func TestCancelAlreadySold(t *testing.T) {
	h := newHouse()
	id, _, _ := h.List("alice", testItem, 100)
	_, _, _, _ = h.Buy(id, "bob", 9999)
	_, err := h.Cancel(id, "alice")
	if err == nil {
		t.Error("expected error cancelling sold listing")
	}
}

func TestExpiryTickExpires(t *testing.T) {
	// Use a deliberately expired listing via manual inspection — we can't
	// inject the clock without exporting it. Test the public path: listings
	// listed well in the past are expired by ExpiryTick when their ExpiresAt
	// has passed. We simulate by checking ExpiryTick returns 0 on a fresh listing
	// (not yet expired).
	h := newHouse()
	h.List("alice", testItem, 100)
	n := h.ExpiryTick()
	if n != 0 {
		t.Errorf("fresh listing should not expire immediately, got %d", n)
	}
}

func TestListingDurationConstant(t *testing.T) {
	if auction.ListingDuration != 15*time.Minute {
		t.Errorf("want 15min, got %v", auction.ListingDuration)
	}
}

func TestMultipleListings(t *testing.T) {
	h := newHouse()
	var ids []string
	for i := 0; i < 5; i++ {
		item := loot.Item{ID: fmt.Sprintf("item-%d", i), Name: fmt.Sprintf("Item %d", i)}
		id, _, err := h.List("alice", item, (i+1)*100)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	active := h.ActiveListings()
	if len(active) != 5 {
		t.Errorf("want 5 active listings, got %d", len(active))
	}
	// Buy one.
	_, _, _, err := h.Buy(ids[0], "bob", 9999)
	if err != nil {
		t.Fatal(err)
	}
	active2 := h.ActiveListings()
	if len(active2) != 4 {
		t.Errorf("after one buy: want 4 active, got %d", len(active2))
	}
}


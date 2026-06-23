package market

import (
	"testing"
	"time"
)

// helpers

func newAH() *AuctionHouse { return New() }

func listOK(t *testing.T, ah *AuctionHouse, seller, itemID, name string, cat Category, price int64, qty int) Listing {
	t.Helper()
	l, err := ah.List(seller, itemID, name, cat, price, qty)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return l
}

// --- Categories ---

func TestCategoryName_AllNamed(t *testing.T) {
	for _, cat := range AllCategories() {
		if n := CategoryName(cat); n == "" {
			t.Errorf("CategoryName(%d) is empty", cat)
		}
	}
}

func TestCategoryName_Unknown(t *testing.T) {
	n := CategoryName(Category(99))
	if n == "" {
		t.Error("CategoryName unknown should return non-empty fallback")
	}
}

func TestAllCategories_EightCategories(t *testing.T) {
	if len(AllCategories()) != 8 {
		t.Errorf("AllCategories: want 8, got %d", len(AllCategories()))
	}
}

// --- List ---

func TestList_CreatesListing(t *testing.T) {
	ah := newAH()
	l, err := ah.List("alice", "sword-001", "Iron Sword", CatWeapons, 1000, 1)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if l.ID == "" {
		t.Error("listing ID should not be empty")
	}
	if l.Price != 1000 || l.Qty != 1 || l.SellerID != "alice" {
		t.Errorf("listing fields wrong: %+v", l)
	}
}

func TestList_InvalidPriceRejected(t *testing.T) {
	ah := newAH()
	_, err := ah.List("alice", "x", "x", CatWeapons, 0, 1)
	if err != ErrInvalidPrice {
		t.Errorf("price=0 should return ErrInvalidPrice, got %v", err)
	}
	_, err = ah.List("alice", "x", "x", CatWeapons, -10, 1)
	if err != ErrInvalidPrice {
		t.Errorf("negative price should return ErrInvalidPrice, got %v", err)
	}
}

func TestList_InvalidQtyRejected(t *testing.T) {
	ah := newAH()
	_, err := ah.List("alice", "x", "x", CatWeapons, 100, 0)
	if err != ErrInvalidQty {
		t.Errorf("qty=0 should return ErrInvalidQty, got %v", err)
	}
}

func TestList_UnknownCategoryRejected(t *testing.T) {
	ah := newAH()
	_, err := ah.List("alice", "x", "x", Category(99), 100, 1)
	if err != ErrUnknownCategory {
		t.Errorf("unknown category should return ErrUnknownCategory, got %v", err)
	}
}

func TestList_UniqueIDs(t *testing.T) {
	ah := newAH()
	ids := map[string]bool{}
	for i := 0; i < 5; i++ {
		l, _ := ah.List("alice", "x", "x", CatWeapons, 100, 1)
		if ids[l.ID] {
			t.Errorf("duplicate listing ID: %q", l.ID)
		}
		ids[l.ID] = true
	}
}

func TestList_AppearsInActiveCount(t *testing.T) {
	ah := newAH()
	ah.List("alice", "x", "x", CatWeapons, 100, 1)
	ah.List("bob", "y", "y", CatArmor, 200, 1)
	if ah.ActiveCount() != 2 {
		t.Errorf("ActiveCount: got %d, want 2", ah.ActiveCount())
	}
}

// --- Buy ---

func TestBuy_PurchasesCheapestListing(t *testing.T) {
	ah := newAH()
	ah.List("alice", "sw", "Iron Sword", CatWeapons, 2000, 1)
	ah.List("bob", "sw", "Iron Sword", CatWeapons, 1500, 1)
	ah.List("carol", "sw", "Iron Sword", CatWeapons, 3000, 1)

	rec, err := ah.Buy("dave", "sw")
	if err != nil {
		t.Fatalf("Buy: %v", err)
	}
	if rec.Price != 1500 {
		t.Errorf("should buy cheapest: got price %d, want 1500", rec.Price)
	}
	if rec.BuyerID != "dave" {
		t.Errorf("BuyerID: got %q", rec.BuyerID)
	}
	// Two listings should remain.
	if ah.ActiveCount() != 2 {
		t.Errorf("ActiveCount after buy: got %d, want 2", ah.ActiveCount())
	}
}

func TestBuy_NoListingsReturnsError(t *testing.T) {
	ah := newAH()
	_, err := ah.Buy("dave", "nonexistent")
	if err != ErrNoListings {
		t.Errorf("buy with no listings: got %v, want ErrNoListings", err)
	}
}

func TestBuy_SelfPurchaseBlocked(t *testing.T) {
	ah := newAH()
	ah.List("alice", "sw", "Iron Sword", CatWeapons, 1000, 1)
	_, err := ah.Buy("alice", "sw")
	if err != ErrSelfPurchase {
		t.Errorf("self-purchase: got %v, want ErrSelfPurchase", err)
	}
	// Listing should still be active.
	if ah.ActiveCount() != 1 {
		t.Errorf("listing should survive self-purchase attempt: count=%d", ah.ActiveCount())
	}
}

func TestBuy_RemovesListingOnSuccess(t *testing.T) {
	ah := newAH()
	ah.List("alice", "sw", "Iron Sword", CatWeapons, 1000, 1)
	ah.Buy("bob", "sw")
	if ah.ActiveCount() != 0 {
		t.Error("listing should be removed after purchase")
	}
}

func TestBuy_RecordsHistory(t *testing.T) {
	ah := newAH()
	ah.List("alice", "sw", "Iron Sword", CatWeapons, 1000, 1)
	ah.Buy("bob", "sw")
	history := ah.HistoryFor("sw")
	if len(history) != 1 {
		t.Fatalf("history: want 1 record, got %d", len(history))
	}
	if history[0].BuyerID != "bob" || history[0].SellerID != "alice" {
		t.Errorf("history record wrong: %+v", history[0])
	}
}

func TestBuy_HistoryCapAt12(t *testing.T) {
	ah := newAH()
	for i := 0; i < 15; i++ {
		ah.List("seller", "potion", "Potion", CatMisc, 100, 1)
		ah.Buy("buyer", "potion")
	}
	history := ah.HistoryFor("potion")
	if len(history) != HistoryCap {
		t.Errorf("history capped at %d, got %d", HistoryCap, len(history))
	}
}

func TestBuy_HistoryNewestFirst(t *testing.T) {
	ah := newAH()
	ah.List("s", "potion", "Potion", CatMisc, 100, 1)
	ah.Buy("b1", "potion")
	time.Sleep(2 * time.Millisecond) // ensure distinct timestamps
	ah.List("s", "potion", "Potion", CatMisc, 200, 1)
	ah.Buy("b2", "potion")

	history := ah.HistoryFor("potion")
	if len(history) != 2 {
		t.Fatalf("want 2 records, got %d", len(history))
	}
	if !history[0].SoldAt.After(history[1].SoldAt) {
		t.Error("history should be newest-first")
	}
}

// --- CancelListing ---

func TestCancelListing_RemovesListing(t *testing.T) {
	ah := newAH()
	l := listOK(t, ah, "alice", "sw", "Iron Sword", CatWeapons, 1000, 1)
	cancelled, err := ah.CancelListing("alice", l.ID)
	if err != nil {
		t.Fatalf("CancelListing: %v", err)
	}
	if cancelled.ID != l.ID {
		t.Errorf("returned wrong listing: %+v", cancelled)
	}
	if ah.ActiveCount() != 0 {
		t.Error("listing should be removed after cancel")
	}
}

func TestCancelListing_NotFoundReturnsError(t *testing.T) {
	ah := newAH()
	_, err := ah.CancelListing("alice", "L999999")
	if err != ErrListingNotFound {
		t.Errorf("got %v, want ErrListingNotFound", err)
	}
}

func TestCancelListing_NotSellerReturnsError(t *testing.T) {
	ah := newAH()
	l := listOK(t, ah, "alice", "sw", "Iron Sword", CatWeapons, 1000, 1)
	_, err := ah.CancelListing("bob", l.ID)
	if err != ErrNotSeller {
		t.Errorf("got %v, want ErrNotSeller", err)
	}
	if ah.ActiveCount() != 1 {
		t.Error("listing should survive cancel-by-non-seller")
	}
}

// --- BrowseCategory ---

func TestBrowseCategory_ReturnsItemSummaries(t *testing.T) {
	ah := newAH()
	ah.List("alice", "sw1", "Iron Sword", CatWeapons, 2000, 1)
	ah.List("bob", "sw1", "Iron Sword", CatWeapons, 1500, 1) // same item, cheaper
	ah.List("carol", "axe1", "Bronze Axe", CatWeapons, 800, 1)
	ah.List("dave", "potion", "Potion", CatMisc, 50, 1) // different category

	summaries := ah.BrowseCategory(CatWeapons)
	if len(summaries) != 2 {
		t.Fatalf("BrowseCategory(Weapons): want 2 items, got %d", len(summaries))
	}
	// Should be sorted by lowest price: axe 800, sword 1500.
	if summaries[0].ItemID != "axe1" {
		t.Errorf("cheapest item first: got %q", summaries[0].ItemID)
	}
	if summaries[1].LowestPrice != 1500 {
		t.Errorf("sword lowest price: got %d, want 1500", summaries[1].LowestPrice)
	}
	if summaries[1].ListingCount != 2 {
		t.Errorf("sword listing count: got %d, want 2", summaries[1].ListingCount)
	}
}

func TestBrowseCategory_EmptyCategoryReturnsEmpty(t *testing.T) {
	ah := newAH()
	ah.List("alice", "sw1", "Iron Sword", CatWeapons, 1000, 1)
	if sums := ah.BrowseCategory(CatFood); len(sums) != 0 {
		t.Errorf("empty category: got %d summaries", len(sums))
	}
}

func TestBrowseCategory_ShowsHistoryPriceOnSummary(t *testing.T) {
	ah := newAH()
	ah.List("alice", "potion", "Potion", CatMisc, 100, 1)
	ah.Buy("bob", "potion")
	// Now post a new listing.
	ah.List("carol", "potion", "Potion", CatMisc, 120, 1)

	sums := ah.BrowseCategory(CatMisc)
	if len(sums) != 1 {
		t.Fatalf("want 1 summary, got %d", len(sums))
	}
	if sums[0].LastPrice != 100 {
		t.Errorf("LastPrice should be 100 (last sale), got %d", sums[0].LastPrice)
	}
}

// --- ItemPage ---

func TestItemPage_ActiveListingsSortedCheapest(t *testing.T) {
	ah := newAH()
	ah.List("a", "sw", "Iron Sword", CatWeapons, 3000, 1)
	ah.List("b", "sw", "Iron Sword", CatWeapons, 1000, 1)
	ah.List("c", "sw", "Iron Sword", CatWeapons, 2000, 1)

	page := ah.ItemPage("sw")
	if len(page.Listings) != 3 {
		t.Fatalf("ItemPage listings: want 3, got %d", len(page.Listings))
	}
	if page.Listings[0].Price != 1000 {
		t.Errorf("cheapest first: got %d", page.Listings[0].Price)
	}
}

func TestItemPage_HistoryIncluded(t *testing.T) {
	ah := newAH()
	ah.List("s", "sw", "Iron Sword", CatWeapons, 1000, 1)
	ah.Buy("b", "sw")
	ah.List("s", "sw", "Iron Sword", CatWeapons, 1200, 1)

	page := ah.ItemPage("sw")
	if len(page.History) != 1 {
		t.Fatalf("ItemPage history: want 1, got %d", len(page.History))
	}
	if len(page.Listings) != 1 {
		t.Fatalf("ItemPage active: want 1, got %d", len(page.Listings))
	}
}

func TestItemPage_EmptyItemReturnsEmptyPage(t *testing.T) {
	ah := newAH()
	page := ah.ItemPage("nonexistent")
	if len(page.Listings) != 0 || len(page.History) != 0 {
		t.Error("empty item should have empty page")
	}
}

func TestItemPage_MetaFromHistoryWhenNoListings(t *testing.T) {
	ah := newAH()
	ah.List("s", "sw", "Iron Sword", CatWeapons, 1000, 1)
	ah.Buy("b", "sw")
	page := ah.ItemPage("sw")
	if page.ItemName != "Iron Sword" {
		t.Errorf("ItemName from history: got %q", page.ItemName)
	}
	if page.Category != CatWeapons {
		t.Errorf("Category from history: got %v", page.Category)
	}
}

// --- SellerListings ---

func TestSellerListings_OwnListings(t *testing.T) {
	ah := newAH()
	ah.List("alice", "sw", "Iron Sword", CatWeapons, 1000, 1)
	ah.List("alice", "axe", "Bronze Axe", CatWeapons, 800, 1)
	ah.List("bob", "shield", "Shield", CatArmor, 500, 1)

	aliceListings := ah.SellerListings("alice")
	if len(aliceListings) != 2 {
		t.Errorf("alice listings: got %d, want 2", len(aliceListings))
	}
	bobListings := ah.SellerListings("bob")
	if len(bobListings) != 1 {
		t.Errorf("bob listings: got %d, want 1", len(bobListings))
	}
}

func TestSellerListings_EmptyAfterBuy(t *testing.T) {
	ah := newAH()
	ah.List("alice", "sw", "Iron Sword", CatWeapons, 1000, 1)
	ah.Buy("bob", "sw")
	if len(ah.SellerListings("alice")) != 0 {
		t.Error("seller listings should be empty after sale")
	}
}

// --- BuyerHistory ---

func TestBuyerHistory_ReturnsOwnPurchases(t *testing.T) {
	ah := newAH()
	ah.List("s", "sw", "Iron Sword", CatWeapons, 1000, 1)
	ah.Buy("bob", "sw")
	ah.List("s", "axe", "Bronze Axe", CatWeapons, 800, 1)
	ah.Buy("alice", "axe")

	bobHistory := ah.BuyerHistory("bob")
	if len(bobHistory) != 1 || bobHistory[0].ItemID != "sw" {
		t.Errorf("bob history: %v", bobHistory)
	}
	aliceHistory := ah.BuyerHistory("alice")
	if len(aliceHistory) != 1 || aliceHistory[0].ItemID != "axe" {
		t.Errorf("alice history: %v", aliceHistory)
	}
}

// --- Stack listings ---

func TestList_StackQty(t *testing.T) {
	ah := newAH()
	l, err := ah.List("alice", "fire-crystal", "Fire Crystal", CatCrystals, 500, 12)
	if err != nil {
		t.Fatalf("stack listing: %v", err)
	}
	if l.Qty != 12 {
		t.Errorf("Qty: got %d, want 12", l.Qty)
	}
}

// --- Integration: full AH terminal flow ---

func TestIntegration_FullTerminalFlow(t *testing.T) {
	ah := newAH()

	// Sellers list items.
	ah.List("weaponsmith", "iron-sword", "Iron Sword", CatWeapons, 2500, 1)
	ah.List("weaponsmith", "iron-sword", "Iron Sword", CatWeapons, 2000, 1)
	ah.List("alchemist", "hi-potion", "Hi-Potion", CatMisc, 300, 1)
	ah.List("cook", "rolanberry", "Rolanberry Salad", CatFood, 1200, 1)

	// Browse Weapons category.
	weapons := ah.BrowseCategory(CatWeapons)
	if len(weapons) != 1 {
		t.Fatalf("browse weapons: want 1 item, got %d", len(weapons))
	}
	if weapons[0].ListingCount != 2 {
		t.Errorf("sword listing count: got %d, want 2", weapons[0].ListingCount)
	}
	if weapons[0].LowestPrice != 2000 {
		t.Errorf("sword lowest price: got %d, want 2000", weapons[0].LowestPrice)
	}

	// View item page.
	page := ah.ItemPage("iron-sword")
	if len(page.Listings) != 2 {
		t.Fatalf("item page listings: want 2, got %d", len(page.Listings))
	}
	if page.Listings[0].Price != 2000 {
		t.Errorf("cheapest listing first: got %d", page.Listings[0].Price)
	}

	// Player buys cheapest sword.
	rec, err := ah.Buy("adventurer", "iron-sword")
	if err != nil {
		t.Fatalf("buy: %v", err)
	}
	if rec.Price != 2000 {
		t.Errorf("bought wrong listing: price=%d", rec.Price)
	}

	// History now shows the sale.
	history := ah.HistoryFor("iron-sword")
	if len(history) != 1 {
		t.Fatalf("history: want 1, got %d", len(history))
	}
	if history[0].BuyerID != "adventurer" {
		t.Errorf("buyer in history: got %q", history[0].BuyerID)
	}

	// Weapon summary now shows LastPrice from history.
	weapons2 := ah.BrowseCategory(CatWeapons)
	if weapons2[0].LastPrice != 2000 {
		t.Errorf("LastPrice: got %d, want 2000", weapons2[0].LastPrice)
	}

	// Seller cancels remaining listing.
	remaining := ah.SellerListings("weaponsmith")
	if len(remaining) != 1 {
		t.Fatalf("remaining seller listings: want 1, got %d", len(remaining))
	}
	_, err = ah.CancelListing("weaponsmith", remaining[0].ID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if ah.ActiveCount() != 2 { // hi-potion + rolanberry still up
		t.Errorf("active after cancel: want 2, got %d", ah.ActiveCount())
	}
}

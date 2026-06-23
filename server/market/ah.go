// Package market implements the DragonsNShit Auction House — an FFXI-parity
// server-authoritative market for player-to-player item trading.
//
// # FFXI Auction House model
//
// Sellers list items at a fixed price (single-unit or stack). Buyers
// purchase at the lowest available price for a given item; the exact
// seller is not visible before purchase ("blind bid" — seller and price
// only revealed in the history after the sale closes).
//
// Key rules inherited from FFXI:
//   - One AH per server; all zones share the same listings.
//   - Price history: the 12 most recent sale prices per item are visible
//     (this is exactly what the FFXI PS2 terminal shows on the item page).
//   - Sellers can cancel un-sold listings and reclaim the item.
//   - A listing fee of AHFeePct % of the sale price is deducted on sale
//     (caller handles the currency transfer; this package just records it).
//   - Qty 1 = single unit; Qty > 1 = stack; buyers purchase the whole listing.
//
// # PS2 terminal navigation model
//
// The FFXI PS2 AH terminal has these screens:
//
//	[Main] → Browse → [Category list] → [Item list] → [Item page]
//	                                                  → Buy (cheapest listing)
//	       → Sell   → [price/qty prompt]
//	       → History → [own sale/purchase history]
//	       → Status  → [own active listings]
//
// The following methods map to each screen:
//
//	BrowseCategory(cat)     — category item list (ItemSummary per distinct item)
//	ItemPage(itemID)        — item detail: active listings + price history
//	Buy(buyerID, itemID)    — execute cheapest-listing purchase
//	List(...)               — post a new listing (Sell screen confirm)
//	CancelListing(...)      — withdraw a listing (Status screen action)
//	SellerListings(id)      — Status screen: seller's own active listings
//	BuyerHistory(id)        — own purchases in SaleHistory
package market

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// Category classifies items for AH browse navigation (mirrors FFXI categories).
type Category int

const (
	CatWeapons   Category = 1
	CatArmor     Category = 2
	CatAmmo      Category = 3
	CatFood      Category = 4
	CatCrystals  Category = 5
	CatMaterials Category = 6 // raw crafting materials
	CatCraftItems Category = 7 // craft-produced goods
	CatMisc      Category = 8
)

var categoryNames = map[Category]string{
	CatWeapons:   "Weapons",
	CatArmor:     "Armor",
	CatAmmo:      "Ammo / Ranged",
	CatFood:      "Food & Drinks",
	CatCrystals:  "Crystals",
	CatMaterials: "Materials",
	CatCraftItems: "Craft Items",
	CatMisc:      "Miscellaneous",
}

// CategoryName returns the display name for a category.
func CategoryName(c Category) string {
	if n, ok := categoryNames[c]; ok {
		return n
	}
	return fmt.Sprintf("Category(%d)", int(c))
}

// AllCategories returns all registered categories in display order.
func AllCategories() []Category {
	return []Category{
		CatWeapons, CatArmor, CatAmmo, CatFood,
		CatCrystals, CatMaterials, CatCraftItems, CatMisc,
	}
}

// AHFeePct is the percentage of the sale price charged as the AH listing fee.
// Callers apply this when transferring gil between buyer and seller.
const AHFeePct = 4

// HistoryCap is the number of recent sale records retained per item (FFXI shows 12).
const HistoryCap = 12

// Listing is a single item currently posted for sale on the AH.
type Listing struct {
	ID       string
	ItemID   string
	ItemName string
	Category Category
	Price    int64     // gil (must be > 0)
	Qty      int       // units in this listing (1 = single; > 1 = stack)
	SellerID string
	ListedAt time.Time
}

// SaleRecord captures a completed transaction (written when Buy succeeds).
type SaleRecord struct {
	ListingID string
	ItemID    string
	ItemName  string
	Category  Category
	Price     int64
	Qty       int
	SellerID  string
	BuyerID   string
	SoldAt    time.Time
}

// ItemSummary is the per-item row shown on the AH browse / category screen.
type ItemSummary struct {
	ItemID       string
	ItemName     string
	Category     Category
	LowestPrice  int64 // lowest active listing price
	ListingCount int   // number of active listings
	LastSoldAt   time.Time // zero if never sold
	LastPrice    int64     // most recent sale price; 0 if never sold
}

// ItemPage is returned by ItemPage() — the full AH item detail screen.
type ItemPage struct {
	ItemID   string
	ItemName string
	Category Category
	// Active listings sorted cheapest-first.
	Listings []Listing
	// Up to HistoryCap recent sales, newest-first.
	History []SaleRecord
}

// Errors
var (
	ErrNoListings       = errors.New("no listings for item")
	ErrListingNotFound  = errors.New("listing not found")
	ErrNotSeller        = errors.New("not the listing's seller")
	ErrInvalidPrice     = errors.New("price must be > 0")
	ErrInvalidQty       = errors.New("qty must be > 0")
	ErrUnknownCategory  = errors.New("unknown category")
	ErrSelfPurchase     = errors.New("cannot buy your own listing")
)

// AuctionHouse is the server-authoritative AH state.
// Not goroutine-safe; callers synchronise via the server tick loop.
type AuctionHouse struct {
	listings map[string]*Listing        // listingID → listing
	history  map[string][]SaleRecord    // itemID → recent sales (newest first, capped at HistoryCap)
	nextID   uint64
}

// New returns an empty AuctionHouse.
func New() *AuctionHouse {
	return &AuctionHouse{
		listings: make(map[string]*Listing),
		history:  make(map[string][]SaleRecord),
	}
}

func (ah *AuctionHouse) newID() string {
	ah.nextID++
	return fmt.Sprintf("L%06d", ah.nextID)
}

// List posts a new listing to the AH.
// Returns the created Listing or an error if price/qty/category are invalid.
func (ah *AuctionHouse) List(sellerID, itemID, itemName string, cat Category, price int64, qty int) (Listing, error) {
	if price <= 0 {
		return Listing{}, ErrInvalidPrice
	}
	if qty <= 0 {
		return Listing{}, ErrInvalidQty
	}
	if _, ok := categoryNames[cat]; !ok {
		return Listing{}, ErrUnknownCategory
	}
	l := Listing{
		ID:       ah.newID(),
		ItemID:   itemID,
		ItemName: itemName,
		Category: cat,
		Price:    price,
		Qty:      qty,
		SellerID: sellerID,
		ListedAt: time.Now(),
	}
	ah.listings[l.ID] = &l
	return l, nil
}

// Buy purchases the cheapest active listing for itemID on behalf of buyerID.
// Returns the completed SaleRecord and removes the listing from the AH.
// Returns ErrNoListings if no listings exist.
// Returns ErrSelfPurchase if the cheapest listing belongs to buyerID.
func (ah *AuctionHouse) Buy(buyerID, itemID string) (SaleRecord, error) {
	candidates := ah.listingsForItem(itemID)
	if len(candidates) == 0 {
		return SaleRecord{}, ErrNoListings
	}
	// Sort by price ascending (buy cheapest).
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Price < candidates[j].Price
	})
	cheapest := candidates[0]
	if cheapest.SellerID == buyerID {
		return SaleRecord{}, ErrSelfPurchase
	}

	delete(ah.listings, cheapest.ID)

	rec := SaleRecord{
		ListingID: cheapest.ID,
		ItemID:    cheapest.ItemID,
		ItemName:  cheapest.ItemName,
		Category:  cheapest.Category,
		Price:     cheapest.Price,
		Qty:       cheapest.Qty,
		SellerID:  cheapest.SellerID,
		BuyerID:   buyerID,
		SoldAt:    time.Now(),
	}
	ah.appendHistory(rec)
	return rec, nil
}

// CancelListing removes an active listing posted by sellerID and returns it.
// Returns ErrListingNotFound or ErrNotSeller on failure.
func (ah *AuctionHouse) CancelListing(sellerID, listingID string) (Listing, error) {
	l, ok := ah.listings[listingID]
	if !ok {
		return Listing{}, ErrListingNotFound
	}
	if l.SellerID != sellerID {
		return Listing{}, ErrNotSeller
	}
	delete(ah.listings, listingID)
	return *l, nil
}

// BrowseCategory returns one ItemSummary per distinct item in the given category,
// sorted by lowest active price ascending.  Returns an empty slice if no listings.
func (ah *AuctionHouse) BrowseCategory(cat Category) []ItemSummary {
	byItem := map[string]*ItemSummary{}
	for _, l := range ah.listings {
		if l.Category != cat {
			continue
		}
		s, ok := byItem[l.ItemID]
		if !ok {
			s = &ItemSummary{
				ItemID:   l.ItemID,
				ItemName: l.ItemName,
				Category: cat,
				LowestPrice: l.Price,
			}
			byItem[l.ItemID] = s
		}
		s.ListingCount++
		if l.Price < s.LowestPrice {
			s.LowestPrice = l.Price
		}
	}
	// Annotate with history data.
	for _, s := range byItem {
		if recs := ah.history[s.ItemID]; len(recs) > 0 {
			s.LastSoldAt = recs[0].SoldAt
			s.LastPrice = recs[0].Price
		}
	}
	out := make([]ItemSummary, 0, len(byItem))
	for _, s := range byItem {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LowestPrice != out[j].LowestPrice {
			return out[i].LowestPrice < out[j].LowestPrice
		}
		return out[i].ItemName < out[j].ItemName
	})
	return out
}

// ItemPage returns the full detail page for an item: active listings (cheapest-first)
// and the last HistoryCap sale records (newest-first).
func (ah *AuctionHouse) ItemPage(itemID string) ItemPage {
	listings := ah.listingsForItem(itemID)
	sort.Slice(listings, func(i, j int) bool { return listings[i].Price < listings[j].Price })

	history := append([]SaleRecord{}, ah.history[itemID]...) // snapshot

	page := ItemPage{ItemID: itemID, Listings: listings, History: history}
	if len(listings) > 0 {
		page.ItemName = listings[0].ItemName
		page.Category = listings[0].Category
	} else if len(history) > 0 {
		page.ItemName = history[0].ItemName
		page.Category = history[0].Category
	}
	return page
}

// SellerListings returns all active listings posted by sellerID, newest-first.
func (ah *AuctionHouse) SellerListings(sellerID string) []Listing {
	var out []Listing
	for _, l := range ah.listings {
		if l.SellerID == sellerID {
			out = append(out, *l)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ListedAt.After(out[j].ListedAt) })
	return out
}

// BuyerHistory returns sale records in which buyerID was the purchaser, newest-first.
func (ah *AuctionHouse) BuyerHistory(buyerID string) []SaleRecord {
	var out []SaleRecord
	for _, recs := range ah.history {
		for _, r := range recs {
			if r.BuyerID == buyerID {
				out = append(out, r)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SoldAt.After(out[j].SoldAt) })
	return out
}

// ActiveCount returns the total number of active listings on the AH.
func (ah *AuctionHouse) ActiveCount() int { return len(ah.listings) }

// HistoryFor returns the sale history for itemID (newest first, ≤HistoryCap entries).
func (ah *AuctionHouse) HistoryFor(itemID string) []SaleRecord {
	return append([]SaleRecord{}, ah.history[itemID]...)
}

// --- internal helpers ---

func (ah *AuctionHouse) listingsForItem(itemID string) []Listing {
	var out []Listing
	for _, l := range ah.listings {
		if l.ItemID == itemID {
			out = append(out, *l)
		}
	}
	return out
}

func (ah *AuctionHouse) appendHistory(rec SaleRecord) {
	recs := ah.history[rec.ItemID]
	recs = append([]SaleRecord{rec}, recs...) // prepend (newest first)
	if len(recs) > HistoryCap {
		recs = recs[:HistoryCap]
	}
	ah.history[rec.ItemID] = recs
}

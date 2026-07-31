// Package auction implements the GoblinFoxDragon Auction House (AH).
//
// Players list items for sale at an ask price. Other players purchase at ask.
// AH fee is 5% of the ask price (floored), deducted from the seller's return.
// Listings expire after 15 minutes if unsold.
package auction

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"dragonsnshit/server/loot"
)

const (
	// FeePercent is the AH commission on each sale.
	FeePercent = 5
	// ListingDuration is the maximum lifetime of an unsold listing.
	ListingDuration = 15 * time.Minute
)

var (
	ErrListingNotFound  = errors.New("listing not found")
	ErrSellerCannotBuy  = errors.New("you cannot buy your own listing")
	ErrListingExpired   = errors.New("listing has expired")
	ErrListingNotActive = errors.New("listing is not active")
	ErrInvalidPrice     = errors.New("ask price must be > 0")
	ErrInsufficientFlow  = errors.New("insufficient flow")
)

// Status tracks a listing's lifecycle.
type Status int

const (
	StatusActive Status = iota
	StatusSold
	StatusExpired
	StatusCancelled
)

// Listing is a single item listed for sale in the Auction House.
type Listing struct {
	ID       string
	Item     loot.Item
	SellerID string
	Ask      int
	Status   Status
	ListedAt time.Time
	ExpiresAt time.Time
	Buyer    string // set when Status == StatusSold
	SoldAt   time.Time
}

// House manages all AH listings.
type House struct {
	mu       sync.Mutex
	listings map[string]*Listing
	seqID    int
	clock    func() time.Time
}

// New creates an empty Auction House.
func New() *House {
	return &House{
		listings: make(map[string]*Listing),
		clock:    time.Now,
	}
}

// List creates a new listing for item at ask price.
// Returns the listing ID and the created Listing.
func (h *House) List(sellerID string, item loot.Item, ask int) (string, *Listing, error) {
	if ask <= 0 {
		return "", nil, ErrInvalidPrice
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.seqID++
	id := fmt.Sprintf("ah-%06d", h.seqID)
	now := h.clock()
	l := &Listing{
		ID:        id,
		Item:      item,
		SellerID:  sellerID,
		Ask:       ask,
		Status:    StatusActive,
		ListedAt:  now,
		ExpiresAt: now.Add(ListingDuration),
	}
	h.listings[id] = l
	return id, l, nil
}

// Buy purchases the listing for buyer, returning (fee, net_to_seller, error).
// The caller is responsible for adjusting player flow balances.
func (h *House) Buy(listingID, buyerID string, buyerFlow int) (fee int, netToSeller int, l *Listing, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	l, ok := h.listings[listingID]
	if !ok {
		return 0, 0, nil, ErrListingNotFound
	}
	if l.SellerID == buyerID {
		return 0, 0, l, ErrSellerCannotBuy
	}
	if l.Status != StatusActive {
		return 0, 0, l, ErrListingNotActive
	}
	now := h.clock()
	if now.After(l.ExpiresAt) {
		l.Status = StatusExpired
		return 0, 0, l, ErrListingExpired
	}
	if buyerFlow < l.Ask {
		return 0, 0, l, ErrInsufficientFlow
	}
	fee = l.Ask * FeePercent / 100
	netToSeller = l.Ask - fee
	l.Status = StatusSold
	l.Buyer = buyerID
	l.SoldAt = now
	return fee, netToSeller, l, nil
}

// Cancel removes an unsold listing by the owner.
func (h *House) Cancel(listingID, sellerID string) (*Listing, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	l, ok := h.listings[listingID]
	if !ok {
		return nil, ErrListingNotFound
	}
	if l.SellerID != sellerID {
		return nil, errors.New("not your listing")
	}
	if l.Status != StatusActive {
		return l, ErrListingNotActive
	}
	l.Status = StatusCancelled
	return l, nil
}

// ActiveListings returns all currently active listings, expiring stale ones.
func (h *House) ActiveListings() []*Listing {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.clock()
	out := make([]*Listing, 0)
	for _, l := range h.listings {
		if l.Status == StatusActive {
			if now.After(l.ExpiresAt) {
				l.Status = StatusExpired
				continue
			}
			out = append(out, l)
		}
	}
	return out
}

// Get returns a single listing by ID.
func (h *House) Get(listingID string) (*Listing, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	l, ok := h.listings[listingID]
	return l, ok
}

// ExpiryTick scans all active listings and expires those past ExpiresAt.
// Returns the count of listings that were expired.
func (h *House) ExpiryTick() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.clock()
	n := 0
	for _, l := range h.listings {
		if l.Status == StatusActive && now.After(l.ExpiresAt) {
			l.Status = StatusExpired
			n++
		}
	}
	return n
}

// FeeFor returns the AH fee that would be charged for an ask price.
func FeeFor(ask int) int { return ask * FeePercent / 100 }

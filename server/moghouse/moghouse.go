// Package moghouse implements personal item storage for GFD MUD players.
//
// Each player has a Mog House with capacity for 50 items. Items are stored by
// ID with a count (like the main inventory map[string]int). Players can store,
// retrieve, and list their Mog House inventory.
package moghouse

import (
	"errors"
	"sync"
)

const Capacity = 50

var (
	ErrFull            = errors.New("mog house is full (50-item limit)")
	ErrItemNotStored   = errors.New("item not in mog house")
	ErrInsufficientQty = errors.New("insufficient quantity in mog house")
	ErrInvalidQty      = errors.New("quantity must be > 0")
)

// MogHouse is one player's personal item storage.
type MogHouse struct {
	OwnerID string
	mu      sync.Mutex
	items   map[string]int // itemID → count
}

// New creates an empty Mog House for the given owner.
func New(ownerID string) *MogHouse {
	return &MogHouse{OwnerID: ownerID, items: make(map[string]int)}
}

// Store deposits qty units of itemID. Returns ErrFull if capacity would be exceeded.
func (m *MogHouse) Store(itemID string, qty int) error {
	if qty <= 0 {
		return ErrInvalidQty
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.totalItems()+qty > Capacity {
		return ErrFull
	}
	m.items[itemID] += qty
	return nil
}

// Retrieve removes qty units of itemID. Returns ErrItemNotStored or ErrInsufficientQty on failure.
func (m *MogHouse) Retrieve(itemID string, qty int) error {
	if qty <= 0 {
		return ErrInvalidQty
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	have, ok := m.items[itemID]
	if !ok || have == 0 {
		return ErrItemNotStored
	}
	if have < qty {
		return ErrInsufficientQty
	}
	have -= qty
	if have == 0 {
		delete(m.items, itemID)
	} else {
		m.items[itemID] = have
	}
	return nil
}

// Count returns how many of itemID are stored.
func (m *MogHouse) Count(itemID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.items[itemID]
}

// TotalItems returns the total number of items currently stored (sum of all counts).
func (m *MogHouse) TotalItems() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.totalItems()
}

func (m *MogHouse) totalItems() int {
	total := 0
	for _, n := range m.items {
		total += n
	}
	return total
}

// Available returns the number of remaining storage slots.
func (m *MogHouse) Available() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Capacity - m.totalItems()
}

// Snapshot returns a copy of the current inventory map (itemID → count).
func (m *MogHouse) Snapshot() map[string]int {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]int, len(m.items))
	for k, v := range m.items {
		out[k] = v
	}
	return out
}

// Registry manages Mog Houses for all players.
type Registry struct {
	mu    sync.Mutex
	houses map[string]*MogHouse // ownerID → *MogHouse
}

// NewRegistry creates an empty Mog House registry.
func NewRegistry() *Registry {
	return &Registry{houses: make(map[string]*MogHouse)}
}

// Get returns the Mog House for ownerID, creating one if it doesn't exist.
func (r *Registry) Get(ownerID string) *MogHouse {
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, ok := r.houses[ownerID]; ok {
		return h
	}
	h := New(ownerID)
	r.houses[ownerID] = h
	return h
}

// PlayerCount returns the number of players with Mog Houses in the registry.
func (r *Registry) PlayerCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.houses)
}

// Package inventory implements DragonsNShit character inventory containers.
//
// An inventory is organized into named bags (Bag). Each bag has a fixed slot
// capacity. Slots hold Stacks — an item UUID + def ID + quantity. The bag does
// not own item metadata; callers resolve metadata from itemdef.Registry.
//
// # Character storage model (FFXI-parity)
//
//	Inventory  — main carry bag, 30 slots (expandable to 80 via Gobbiebag)
//	Storage    — Mog House safe deposit, 80 slots
//	Temporary  — BCNM/instanced content, 10 slots, auto-cleared on exit
//	KeyItems   — non-spatial tab; a set of def IDs
//	Equipped   — virtual, 16 slots; mirrors gear.Equipment
//
// # Stack merging
//
// When adding a stackable item (StackSize > 1), the bag attempts to merge into
// an existing partial stack of the same defID before occupying a new slot.
package inventory

import (
	"errors"
	"fmt"
)

var (
	ErrBagFull        = errors.New("inventory: bag is full")
	ErrSlotEmpty      = errors.New("inventory: slot is empty")
	ErrSlotOccupied   = errors.New("inventory: slot is already occupied")
	ErrBadIndex       = errors.New("inventory: slot index out of range")
	ErrQuantity       = errors.New("inventory: invalid quantity")
	ErrStackOverflow  = errors.New("inventory: would exceed stack size")
	ErrRareConflict   = errors.New("inventory: character already carries a Rare copy of this item")
)

// Stack is a single bag slot: one item instance + stack count.
type Stack struct {
	ItemID    string // UUID from items table
	DefID     int    // itemdef.ItemDef.ID
	Quantity  int
	StackSize int    // max stack size for this def (1 = not stackable)
}

// Full reports whether the stack is at max capacity.
func (s *Stack) Full() bool { return s.Quantity >= s.StackSize }

// Bag is a fixed-capacity inventory container.
// Not goroutine-safe; callers synchronize via server tick.
type Bag struct {
	Capacity int
	slots    []*Stack // nil = empty slot
}

// NewBag creates an empty bag with the given slot capacity.
func NewBag(capacity int) *Bag {
	return &Bag{
		Capacity: capacity,
		slots:    make([]*Stack, capacity),
	}
}

// Add places a stack into the bag.
// If the item is stackable (StackSize > 1), attempts to merge into an
// existing partial stack of the same DefID before using a new slot.
// Returns the slot index where the item landed.
func (b *Bag) Add(s Stack) (int, error) {
	if s.Quantity <= 0 {
		return -1, ErrQuantity
	}
	// Attempt stack merge for stackable items.
	if s.StackSize > 1 {
		for i, existing := range b.slots {
			if existing == nil || existing.DefID != s.DefID {
				continue
			}
			room := existing.StackSize - existing.Quantity
			if room <= 0 {
				continue
			}
			if s.Quantity > room {
				return -1, ErrStackOverflow
			}
			existing.Quantity += s.Quantity
			return i, nil
		}
	}
	// Find first empty slot.
	for i, slot := range b.slots {
		if slot == nil {
			copy := s
			b.slots[i] = &copy
			return i, nil
		}
	}
	return -1, ErrBagFull
}

// Remove decrements quantity in a slot by qty. If the slot reaches 0, it is cleared.
func (b *Bag) Remove(slotIndex, qty int) error {
	if slotIndex < 0 || slotIndex >= b.Capacity {
		return ErrBadIndex
	}
	if b.slots[slotIndex] == nil {
		return ErrSlotEmpty
	}
	if qty <= 0 {
		return ErrQuantity
	}
	s := b.slots[slotIndex]
	if qty > s.Quantity {
		return fmt.Errorf("%w: have %d, removing %d", ErrQuantity, s.Quantity, qty)
	}
	s.Quantity -= qty
	if s.Quantity == 0 {
		b.slots[slotIndex] = nil
	}
	return nil
}

// At returns the stack at slotIndex, or (nil, ErrSlotEmpty).
func (b *Bag) At(slotIndex int) (*Stack, error) {
	if slotIndex < 0 || slotIndex >= b.Capacity {
		return nil, ErrBadIndex
	}
	if b.slots[slotIndex] == nil {
		return nil, ErrSlotEmpty
	}
	return b.slots[slotIndex], nil
}

// Move swaps the contents of two slots. Either or both may be empty.
func (b *Bag) Move(from, to int) error {
	if from < 0 || from >= b.Capacity || to < 0 || to >= b.Capacity {
		return ErrBadIndex
	}
	b.slots[from], b.slots[to] = b.slots[to], b.slots[from]
	return nil
}

// Find returns all slot indices containing an item with the given DefID.
func (b *Bag) Find(defID int) []int {
	var out []int
	for i, s := range b.slots {
		if s != nil && s.DefID == defID {
			out = append(out, i)
		}
	}
	return out
}

// Count returns the total quantity of all items with the given DefID across all slots.
func (b *Bag) Count(defID int) int {
	total := 0
	for _, s := range b.slots {
		if s != nil && s.DefID == defID {
			total += s.Quantity
		}
	}
	return total
}

// OccupiedCount returns the number of non-empty slots.
func (b *Bag) OccupiedCount() int {
	n := 0
	for _, s := range b.slots {
		if s != nil {
			n++
		}
	}
	return n
}

// Full reports whether all slots are occupied.
func (b *Bag) Full() bool { return b.OccupiedCount() >= b.Capacity }

// Slots returns a copy of all slots (nil entries are empty slots).
func (b *Bag) Slots() []*Stack {
	out := make([]*Stack, b.Capacity)
	copy(out, b.slots)
	return out
}

// ── Mog ──────────────────────────────────────────────────────────────────────

const (
	DefaultInventorySize = 30
	DefaultStorageSize   = 80
	DefaultTempSize      = 10
	GobbiebagIncrement   = 10
	MaxInventorySize     = 80
)

// Mog is the full character storage set. It owns all bags and the key-item tab.
// Not goroutine-safe; callers synchronize via server tick.
type Mog struct {
	Inventory *Bag
	Storage   *Bag
	Temporary *Bag
	KeyItems  map[int]bool // defID → present; non-spatial

	// RareOwned tracks defIDs of Rare items anywhere in this Mog (including equipped).
	// Callers must register/unregister when items enter/leave via AddRare/RemoveRare.
	RareOwned map[int]bool
}

// NewMog creates a Mog with default bag sizes.
func NewMog() *Mog {
	return &Mog{
		Inventory: NewBag(DefaultInventorySize),
		Storage:   NewBag(DefaultStorageSize),
		Temporary: NewBag(DefaultTempSize),
		KeyItems:  make(map[int]bool),
		RareOwned: make(map[int]bool),
	}
}

// ExpandInventory increases the Inventory capacity by GobbiebagIncrement,
// up to MaxInventorySize. Returns the new capacity.
func (m *Mog) ExpandInventory() (int, error) {
	current := m.Inventory.Capacity
	if current >= MaxInventorySize {
		return current, fmt.Errorf("inventory: already at max capacity %d", MaxInventorySize)
	}
	newCap := current + GobbiebagIncrement
	if newCap > MaxInventorySize {
		newCap = MaxInventorySize
	}
	newBag := NewBag(newCap)
	// Copy existing slots.
	copy(newBag.slots, m.Inventory.slots)
	m.Inventory = newBag
	return newCap, nil
}

// AddKeyItem adds a key item by defID.
func (m *Mog) AddKeyItem(defID int) { m.KeyItems[defID] = true }

// HasKeyItem reports whether the character carries a given key item.
func (m *Mog) HasKeyItem(defID int) bool { return m.KeyItems[defID] }

// RemoveKeyItem removes a key item.
func (m *Mog) RemoveKeyItem(defID int) { delete(m.KeyItems, defID) }

// CheckRare returns ErrRareConflict if defID is a Rare item already owned.
func (m *Mog) CheckRare(defID int) error {
	if m.RareOwned[defID] {
		return ErrRareConflict
	}
	return nil
}

// AddRare registers ownership of a Rare item.
func (m *Mog) AddRare(defID int) { m.RareOwned[defID] = true }

// RemoveRare unregisters a Rare item (called when dropped/traded/destroyed).
func (m *Mog) RemoveRare(defID int) { delete(m.RareOwned, defID) }

// ClearTemporary removes all items from the Temporary bag.
func (m *Mog) ClearTemporary() {
	for i := range m.Temporary.slots {
		m.Temporary.slots[i] = nil
	}
}

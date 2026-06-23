// Package gear implements the DragonsNShit item level / equipment slot system.
//
// # Item Level model (FFXI-parity)
//
// Every piece of equipment has an item level (IL). A character's effective IL
// is the average IL across all occupied equipment slots (empty slots excluded).
// Stat bonuses from gear scale with IL relative to a base IL of 117 (FFXI norm).
//
// Equipment slots match FFXI standard slots.
package gear

import "errors"

// Slot is a named equipment position.
type Slot = string

const (
	SlotHead    Slot = "head"
	SlotNeck    Slot = "neck"
	SlotEarL    Slot = "ear-l"
	SlotEarR    Slot = "ear-r"
	SlotBody    Slot = "body"
	SlotHandL   Slot = "hand-l"
	SlotHandR   Slot = "hand-r"
	SlotRingL   Slot = "ring-l"
	SlotRingR   Slot = "ring-r"
	SlotBack    Slot = "back"
	SlotWaist   Slot = "waist"
	SlotLegs    Slot = "legs"
	SlotFeet    Slot = "feet"
	SlotMainHand Slot = "main"
	SlotOffHand  Slot = "off"
	SlotAmmo    Slot = "ammo"
)

// AllSlots is the canonical equipment slot order.
var AllSlots = []Slot{
	SlotMainHand, SlotOffHand, SlotAmmo,
	SlotHead, SlotNeck, SlotEarL, SlotEarR,
	SlotBody, SlotHandL, SlotHandR,
	SlotRingL, SlotRingR,
	SlotBack, SlotWaist, SlotLegs, SlotFeet,
}

var (
	ErrUnknownSlot  = errors.New("unknown equipment slot")
	ErrSlotEmpty    = errors.New("slot is empty")
	ErrNoEquipment  = errors.New("no equipment equipped")
)

// ItemEntry is a piece of equipment in the inventory.
type ItemEntry struct {
	ItemID string
	IL     int // item level
}

// Equipment holds a character's equipped gear.
// Not goroutine-safe; callers synchronise via server tick.
type Equipment struct {
	slots map[Slot]*ItemEntry
}

// NewEquipment creates an empty equipment set.
func NewEquipment() *Equipment {
	s := make(map[Slot]*ItemEntry, len(AllSlots))
	for _, sl := range AllSlots {
		s[sl] = nil
	}
	return &Equipment{slots: s}
}

// Equip places an item in the given slot.
// Returns ErrUnknownSlot if the slot is not valid.
func (e *Equipment) Equip(slot Slot, item ItemEntry) error {
	if _, ok := e.slots[slot]; !ok {
		return ErrUnknownSlot
	}
	e.slots[slot] = &item
	return nil
}

// Unequip removes an item from a slot.
// Returns ErrUnknownSlot or ErrSlotEmpty if nothing there.
func (e *Equipment) Unequip(slot Slot) (*ItemEntry, error) {
	entry, ok := e.slots[slot]
	if !ok {
		return nil, ErrUnknownSlot
	}
	if entry == nil {
		return nil, ErrSlotEmpty
	}
	e.slots[slot] = nil
	return entry, nil
}

// ItemAt returns the item in the given slot, or (nil, ErrSlotEmpty) if empty.
func (e *Equipment) ItemAt(slot Slot) (*ItemEntry, error) {
	entry, ok := e.slots[slot]
	if !ok {
		return nil, ErrUnknownSlot
	}
	if entry == nil {
		return nil, ErrSlotEmpty
	}
	return entry, nil
}

// EffectiveIL returns the average IL across all occupied slots.
// Empty slots are excluded from the average.
// Returns 0 and ErrNoEquipment if nothing is equipped.
func (e *Equipment) EffectiveIL() (int, error) {
	sum := 0
	count := 0
	for _, entry := range e.slots {
		if entry != nil {
			sum += entry.IL
			count++
		}
	}
	if count == 0 {
		return 0, ErrNoEquipment
	}
	return sum / count, nil
}

// OccupiedCount returns the number of slots with items equipped.
func (e *Equipment) OccupiedCount() int {
	n := 0
	for _, entry := range e.slots {
		if entry != nil {
			n++
		}
	}
	return n
}

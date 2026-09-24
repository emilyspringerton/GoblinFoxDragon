package main

import "dragonsnshit/server/gear"

// equip_persist.go fixes a real, live production data-loss gap (2026-09-12, founder real-time,
// after finding S412's own starting-weapon feature had left equipment permanently
// non-persistent: "gear needs to persist what the fuck why was that deferred"). Checked directly
// before this fix: IDUNA's GET /api/v1/characters/:id/equipment was the ONLY real equipment
// endpoint that ever existed -- no PATCH/PUT existed anywhere, so nothing a player equipped
// survived a reconnect or a server restart, ever.
//
// Real correction to an earlier, overcautious assumption (starting_gear.go's own prior doc
// comment): character_equipment's real schema (`character_id, slot, item_id`) has NO foreign key
// on item_id (checked directly against the live schema) -- a real item-instance UUID in the
// `items` table was never actually required. item_id can safely hold apps2/mud's own plain
// itemdef.Registry lookup key (e.g. "sword"), the exact same string gear.ItemEntry.ItemID already
// holds for every real equip call in this file.

// persistEquipSlot writes one equipped (or cleared, if itemID == "") slot back to IDUNA,
// best-effort -- same silent-discard convention every other idunaclient call in this file already
// uses (a transient IDUNA outage shouldn't block a player from equipping/unequipping something
// in their own live session).
func persistEquipSlot(p *player, slot, itemID string) {
	if gw == nil {
		return // real, found-live case: grantStartingGear's own unit tests build a bare player
	}
	charID := gw.charIDBySlot[p.slot]
	if charID == "" {
		return
	}
	_ = gw.iduna.UpdateEquipmentSlot(charID, slot, itemID)
}

// resolveEquipEntry looks itemID up in itemdefReg to reconstruct a real gear.ItemEntry (IL/DefID)
// -- pure and testable in isolation, matching cmdEquip's own identical IL-resolution logic just
// above in main.go. Returns ok=false for an item_id that no longer resolves anywhere (neither
// itemdefReg nor the itemIL legacy fallback) -- a real, honest degrade, not equipping a broken,
// defID-less entry.
//
// Real, found-live bug fix (2026-09-24, founder real-time: "the gear doesn not get persisted
// (legs if i buy leather legs they do not persist) the sword does"). Root cause, checked
// directly: cmdEquip (main.go) already falls back to the itemIL map for any item that isn't in
// itemdefReg -- its own doc comment names these "legacy items" explicitly -- but this function,
// the read half used by loadEquipmentFromIDUNA at reconnect, never had that same fallback. Every
// armor piece the "scout" vendor sells (leather-legs, leather-body, leather-helm, leather-feet,
// leather-hands, leather-belt, bone-earring, iron-earring, cotton-cape, bronze-ring,
// bronze-sword, iron-sword, bronze-shield, iron-shield) is itemIL-only, never registered in
// itemdefReg (data/items.json's own item names don't nameKey-match these vendor IDs at all) --
// so every one of them equipped fine in the live session but silently failed to resolve here on
// the very next reconnect, while "sword" (a real, separate itemdefReg entry, id 3) survived.
// Mirroring cmdEquip's own fallback here fixes every legacy item's persistence at once, not just
// legs.
func resolveEquipEntry(itemID string) (gear.ItemEntry, bool) {
	if def, ok := itemdefReg.ByName(itemID); ok {
		il := 1
		if def.Stats != nil {
			if v, ok2 := def.Stats["item_level"]; ok2 {
				il = v
			}
		}
		return gear.ItemEntry{ItemID: itemID, IL: il, DefID: def.ID}, true
	}
	if il, ok := itemIL[itemID]; ok {
		return gear.ItemEntry{ItemID: itemID, IL: il, DefID: 0}, true
	}
	return gear.ItemEntry{}, false
}

// loadEquipmentFromIDUNA restores p.equip from whatever IDUNA already has persisted for this
// character -- the real read half of persistEquipSlot, called once at connect time (mirroring
// loadedInventory/loadedSkills' own established pattern).
func loadEquipmentFromIDUNA(p *player, characterID string) {
	if p.equip == nil || gw == nil {
		return
	}
	persisted, err := gw.iduna.GetEquipment(characterID)
	if err != nil {
		return
	}
	for slot, itemID := range persisted {
		if entry, ok := resolveEquipEntry(itemID); ok {
			_ = p.equip.Equip(slot, entry)
		}
	}
}

package main

// starting_gear.go — S412-03, founder real-time: "we need a sword given to the player when they
// make their account and not the currency that we currently give." Wired to matter for the first
// time this same session (S412-01/02, combat_formula.go) -- gear stats were previously computed
// only for a cosmetic "Stat changes:" display, never applied to real combat math, so a starting
// weapon would have been purely cosmetic before today.
//
// Real, honest gap found while scoping this, FIXED later the same day (see equip_persist.go):
// equipment did not persist across a reconnect at all -- no write path existed anywhere from
// p.equip back to IDUNA's real `character_equipment` table. That fix's own doc comment corrects
// this file's original, overcautious assumption that a real item-instance UUID was required --
// the schema has no foreign key on item_id, a bare itemdef.Registry key was always safe to store.

import (
	"log"

	"dragonsnshit/server/gear"
)

// startingWeaponName is the real item this grants -- itemdefReg's own "Sword" (id 3,
// {"attack":10,"str":1}), the exact item the founder named directly by its own real stats.
const startingWeaponName = "sword"

// grantStartingGear equips p with the real starting weapon, best-effort (matching every other
// idunaclient-adjacent "don't block a session over a missing catalog entry" convention in this
// file) -- a missing item def or a slot-mismatch degrades to a logged skip, never a panic or a
// blocked login.
func grantStartingGear(p *player) {
	def, ok := itemdefReg.ByName(startingWeaponName)
	if !ok {
		log.Printf("[startup] grantStartingGear: %q not found in itemdefReg, skipping starting weapon for %s", startingWeaponName, p.name)
		return
	}
	if p.equip == nil {
		log.Printf("[startup] grantStartingGear: %s has no Equipment set, skipping", p.name)
		return
	}
	if err := p.equip.Equip(gear.SlotMainHand, gear.ItemEntry{ItemID: startingWeaponName, DefID: def.ID}); err != nil {
		log.Printf("[startup] grantStartingGear: failed to equip %q for %s: %v", startingWeaponName, p.name, err)
		return
	}
	persistEquipSlot(p, gear.SlotMainHand, startingWeaponName) // see equip_persist.go
	p.send("A blacksmith presses a Sword into your hands. \"Every adventurer needs a real weapon,\" she says.")
}

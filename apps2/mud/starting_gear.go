package main

// starting_gear.go — S412-03, founder real-time: "we need a sword given to the player when they
// make their account and not the currency that we currently give." Wired to matter for the first
// time this same session (S412-01/02, combat_formula.go) -- gear stats were previously computed
// only for a cosmetic "Stat changes:" display, never applied to real combat math, so a starting
// weapon would have been purely cosmetic before today.
//
// Real, honest, NOT solved here (found while scoping this): equipment does not persist across a
// reconnect AT ALL today -- checked directly, there is no write path anywhere (client or server)
// from p.equip back to IDUNA's own real `character_equipment` table (a real GET exists,
// `/api/v1/characters/:id/equipment`; no PATCH/PUT ever existed to populate it). This is a real,
// pre-existing, universal limitation affecting every piece of equipment for every character, not
// something this feature introduces or makes worse -- a starting sword granted here behaves
// exactly as consistently (present for the session, gone on reconnect) as any other equipped
// item already does. Real, separate, larger follow-up, not rushed here: `character_equipment`'s
// own real schema has a foreign key to a real item INSTANCE row in `items` (not the bare
// itemdef.Registry catalog ID), so a correct fix needs a real item-instance-creation path wired
// through a new IDUNA write endpoint, not a quick patch to this file alone.

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
	p.send("A blacksmith presses a Sword into your hands. \"Every adventurer needs a real weapon,\" she says.")
}

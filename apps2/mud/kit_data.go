package main

// kit_data.go -- every job's spells and abilities, inspired by FFXI's job rosters and expressed in
// the op vocabulary of kit.go. IDs are unique (init panics on a duplicate) and never reuse an ID the
// older hand-coded spells/abilities already own (cure, fire, march, drain, katon, flash, provoke,
// venom, ...); those keep working exactly as before and are simply joined by this layer.
//
// Notes on fidelity: FFXI mechanics that need systems this MUD does not have (pets, maneuver
// stacking, geomancy bubbles, rune charges, ammo) are approximated with the closest working op and
// named for their FFXI original. Crowd control follows the Bard Lullaby model (kit.go / server/mob/cc.go).

import (
	"time"

	"dragonsnshit/server/mob"
	"dragonsnshit/server/status"
)

const (
	WAR = "WAR"
	MNK = "MNK"
	WHM = "WHM"
	BLM = "BLM"
	RDM = "RDM"
	THF = "THF"
	PLD = "PLD"
	DRK = "DRK"
	BST = "BST"
	BRD = "BRD"
	RNG = "RNG"
	SAM = "SAM"
	NIN = "NIN"
	DRG = "DRG"
	SMN = "SMN"
	BLU = "BLU"
	COR = "COR"
	PUP = "PUP"
	DNC = "DNC"
	SCH = "SCH"
	GEO = "GEO"
	RUN = "RUN"
	ASN = "ASN"
)

func init() {
	registerKit(warriorKit()...)
	registerKit(monkKit()...)
	registerKit(whiteMageKit()...)
	registerKit(blackMageKit()...)
	registerKit(redMageKit()...)
	registerKit(thiefKit()...)
	registerKit(paladinKit()...)
	registerKit(darkKnightKit()...)
	registerKit(beastmasterKit()...)
	registerKit(bardKit()...)
	registerKit(rangerKit()...)
	registerKit(samuraiKit()...)
	registerKit(ninjaKit()...)
	registerKit(dragoonKit()...)
	registerKit(summonerKit()...)
	registerKit(blueMageKit()...)
	registerKit(corsairKit()...)
	registerKit(puppetmasterKit()...)
	registerKit(dancerKit()...)
	registerKit(scholarKit()...)
	registerKit(geomancerKit()...)
	registerKit(runeFencerKit()...)
	registerKit(assassinKit()...)
	registerKit(sharedKit()...)
}

// ---- ASN: the enfeeble hybrid ------------------------------------------------------------------
// THF-ish melee frame with poisons and alchemy: coated weapons, thrown flasks, tinctures that
// weaken, and the best crowd control in the game short of a Bard (single and area sleep, bind,
// paralyze, slow, blind, silence, stun). Spells cost MP; abilities are free but have recasts.

func assassinKit() []*kitEntry {
	return []*kitEntry{
		// coatings and melee riders
		ability("envenom", "Envenom", jl(ASN, 1), sec(60), tgtSelf, 0, "coat your weapon: next 8 hits poison", Coat(mob.FxPoison, 6, sec(60), 8)),
		ability("lacing", "Numbing Lacing", jl(ASN, 10), sec(75), tgtSelf, 0, "coat your weapon: next 6 hits slow", Coat(mob.FxSlow, 25, sec(30), 6)),
		ability("assassinate", "Assassinate", jl(ASN, 40), mins(3), tgtSelf, 0, "next hit is a crit for +120%", NextCrit(), NextAtk(120)),
		// alchemy: thrown flasks (abilities)
		ability("vial-toss", "Vial Toss", jl(ASN, 5), sec(20), tgtEnemy, 0, "hurl a vial of acid", Dmg(55, "dex", 2.5, "Water")),
		ability("alchemists-fire", "Alchemist's Fire", jl(ASN, 12), sec(30), tgtArea, 5, "fire flask, small area", Dmg(80, "dex", 3, "Fire")),
		ability("acid-flask", "Acid Flask", jl(ASN, 15), sec(45), tgtEnemy, 0, "corrodes: poison + weaken", Dot(10, sec(45)), Fx(mob.FxWeaken, 15, sec(45))),
		ability("stun-grenade", "Flash Grenade", jl(ASN, 20), sec(60), tgtEnemy, 0, "stun for 5s", Fx(mob.FxStun, 0, sec(5)), Dmg(30, "dex", 1.5, "Light")),
		ability("smoke-bomb", "Smoke Bomb", jl(ASN, 8), sec(90), tgtArea, 10, "blind the area and slip away", Fx(mob.FxBlind, 40, sec(30)), Hide(sec(12))),
		ability("sleeping-draught", "Sleeping Draught", jl(ASN, 18), sec(90), tgtArea, 10, "area sleep, like a lullaby (breaks on damage)", Fx(mob.FxSleep, 0, sec(35))),
		ability("rot-tincture", "Rot Tincture", jl(ASN, 25), sec(60), tgtEnemy, 0, "weaken and slow a foe", Fx(mob.FxWeaken, 30, sec(60)), Fx(mob.FxSlow, 20, sec(60))),
		ability("antidote-brew", "Antidote Brew", jl(ASN, 14), sec(120), tgtSelf, 0, "cleanse and mend yourself", Cleanse(), Heal(120, "mnd", 3)),
		ability("elixir-of-vigor", "Elixir of Vigor", jl(ASN, 30), mins(5), tgtSelf, 0, "attack and crit up, MP surge", Buff(status.AttackUp, 20, mins(3)), Buff(status.CritUp, 10, mins(3)), MPRestore(60)),
		ability("black-lotus", "Black Lotus", jl(ASN, 60), mins(10), tgtArea, 14, "poison cloud: poison, weaken, slow and silence", Dot(15, sec(90)), Fx(mob.FxWeaken, 25, sec(90)), Fx(mob.FxSlow, 25, sec(90)), Fx(mob.FxSilence, 0, sec(30))),
		// poisons and tinctures (spells)
		spell("nightshade", "Nightshade", jl(ASN, 10, THF, 45), 30, sec(2), sec(8), tgtEnemy, 0, "poison DoT", Dot(8, sec(60))),
		spell("hemlock", "Hemlock", jl(ASN, 30), 55, ms(2500), sec(10), tgtEnemy, 0, "strong poison DoT", Dot(18, sec(90))),
		spell("blinding-dust", "Blinding Dust", jl(ASN, 8, NIN, 20), 20, sec(2), sec(8), tgtEnemy, 0, "blind a foe", Fx(mob.FxBlind, 40, sec(60))),
		spell("numbing-salve", "Numbing Salve", jl(ASN, 16, RDM, 4), 30, sec(2), sec(8), tgtEnemy, 0, "paralyze a foe", Fx(mob.FxParalyze, 30, sec(75))),
		spell("sluggard-draught", "Sluggard Draught", jl(ASN, 20), 35, sec(2), sec(8), tgtEnemy, 0, "slow a foe", Fx(mob.FxSlow, 30, sec(75))),
		spell("dreamweed", "Dreamweed", jl(ASN, 22), 45, ms(2500), sec(12), tgtEnemy, 0, "single sleep", Fx(mob.FxSleep, 0, sec(50))),
		spell("bind-tar", "Bind Tar", jl(ASN, 24), 35, sec(2), sec(10), tgtEnemy, 0, "root a foe in place", Fx(mob.FxBind, 0, sec(30))),
		spell("mute-powder", "Mute Powder", jl(ASN, 28), 40, sec(2), sec(10), tgtEnemy, 0, "silence a foe", Fx(mob.FxSilence, 0, sec(45))),
		spell("withering-tonic", "Withering Tonic", jl(ASN, 35), 55, ms(2500), sec(12), tgtEnemy, 0, "weaken a foe's strikes", Fx(mob.FxWeaken, 30, sec(75)), Dot(6, sec(45))),
		spell("poppy-fog", "Poppy Fog", jl(ASN, 45), 85, sec(3), sec(20), tgtArea, 10, "area sleep (lullaby-style)", Fx(mob.FxSleep, 0, sec(60))),
		spell("tincture-of-ruin", "Tincture of Ruin", jl(ASN, 50), 90, sec(3), sec(15), tgtEnemy, 0, "ruinous tincture: damage, weaken, poison", Dmg(150, "dex", 3, "Dark"), Fx(mob.FxWeaken, 20, sec(60)), Dot(12, sec(60))),
	}
}

// ---- shared helpers ------------------------------------------------------------------------------

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

// sharedKit holds spells several jobs learn at different levels (crowd control mostly).
func sharedKit() []*kitEntry {
	return []*kitEntry{
		spell("sleep", "Sleep", jl(BLM, 20, RDM, 30, SCH, 40), 40, ms(2500), sec(10), tgtEnemy, 0, "put a foe to sleep (breaks on damage)", Fx(mob.FxSleep, 0, sec(60))),
		spell("sleepga", "Sleepga", jl(BLM, 40, SCH, 60), 75, sec(3), sec(20), tgtArea, 10, "area sleep (breaks on damage)", Fx(mob.FxSleep, 0, sec(60))),
		spell("bind", "Bind", jl(BLM, 18, RDM, 25, SCH, 30, GEO, 30), 30, sec(2), sec(8), tgtEnemy, 0, "root a foe", Fx(mob.FxBind, 0, sec(40))),
		spell("gravity", "Gravity", jl(BLM, 25, RDM, 35, SCH, 40, GEO, 40), 40, ms(2500), sec(10), tgtEnemy, 0, "slow a foe", Fx(mob.FxSlow, 30, sec(75))),
		spell("stun", "Stun", jl(BLM, 45, DRK, 40, RDM, 55), 80, ms(1500), sec(25), tgtEnemy, 0, "stun for 6s", Fx(mob.FxStun, 0, sec(6))),
		spell("paralyze", "Paralyze", jl(RDM, 4, WHM, 12, SCH, 20), 30, sec(2), sec(8), tgtEnemy, 0, "paralyze a foe", Fx(mob.FxParalyze, 30, sec(90))),
		spell("slow", "Slow", jl(RDM, 13, WHM, 20), 35, sec(2), sec(8), tgtEnemy, 0, "slow a foe", Fx(mob.FxSlow, 30, sec(90))),
		spell("blind", "Blind", jl(RDM, 3, WHM, 8, BLM, 15), 25, sec(2), sec(8), tgtEnemy, 0, "blind a foe", Fx(mob.FxBlind, 40, sec(90))),
		spell("silence", "Silence", jl(RDM, 25, WHM, 30), 40, sec(2), sec(10), tgtEnemy, 0, "silence a foe", Fx(mob.FxSilence, 0, sec(60))),
		spell("repose", "Repose", jl(WHM, 28), 45, ms(2500), sec(12), tgtEnemy, 0, "single sleep", Fx(mob.FxSleep, 0, sec(60))),
		spell("phalanx", "Phalanx", jl(RDM, 33, PLD, 59, RUN, 68), 50, sec(3), 0, tgtAlly, 0, "physical defense up", Buff(status.Protect, 25, mins(3))),
	}
}

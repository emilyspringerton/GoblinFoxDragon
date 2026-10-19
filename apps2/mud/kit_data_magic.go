package main

import (
	"dragonsnshit/server/mob"
	"dragonsnshit/server/status"
)

func whiteMageKit() []*kitEntry {
	return []*kitEntry{
		spell("cure3", "Cure III", jl(WHM, 40, RDM, 48, SCH, 55), 120, sec(3), 0, tgtAlly, 0, "large heal", Heal(450, "mnd", 3)),
		spell("curaga", "Curaga", jl(WHM, 35, SCH, 60), 140, ms(3500), sec(8), tgtZone, 0, "heal the whole party", Heal(180, "mnd", 2)),
		spell("raise", "Raise", jl(WHM, 25, SCH, 50, RDM, 35), 150, sec(5), mins(1), tgtAlly, 0, "revive a fallen ally at half HP", Raise(50)),
		spell("poisona", "Poisona", jl(WHM, 6, RDM, 10, SCH, 10), 20, sec(1), 0, tgtAlly, 0, "cure poison and other ailments", Cleanse()),
		spell("erase", "Erase", jl(WHM, 32, SCH, 45), 40, sec(2), sec(10), tgtAlly, 0, "strip ailments", Cleanse()),
		ability("afflatus-solace", "Afflatus Solace", jl(WHM, 40), mins(3), tgtZone, 0, "party regen", Buff(status.Regen, 4, mins(3))),
		ability("divine-seal", "Divine Seal", jl(WHM, 15), mins(5), tgtSelf, 0, "restore your MP", MPRestore(120)),
	}
}

func blackMageKit() []*kitEntry {
	return []*kitEntry{
		spell("burn", "Burn", jl(BLM, 10), 35, sec(2), sec(6), tgtEnemy, 0, "fire DoT", DotEl(11, "Fire", sec(60))),
		spell("frost", "Frost", jl(BLM, 12), 35, sec(2), sec(6), tgtEnemy, 0, "ice DoT, slows", DotEl(10, "Ice", sec(60)), Fx(mob.FxSlow, 15, sec(30))),
		spell("choke", "Choke", jl(BLM, 14), 35, sec(2), sec(6), tgtEnemy, 0, "wind DoT, silences briefly", DotEl(10, "Wind", sec(60)), Fx(mob.FxSilence, 0, sec(15))),
		spell("rasp", "Rasp", jl(BLM, 16), 35, sec(2), sec(6), tgtEnemy, 0, "earth DoT, blinds", DotEl(10, "Earth", sec(60)), Fx(mob.FxBlind, 25, sec(30))),
		spell("shock", "Shock", jl(BLM, 18), 35, sec(2), sec(6), tgtEnemy, 0, "lightning DoT, paralyzes", DotEl(10, "Lightning", sec(60)), Fx(mob.FxParalyze, 20, sec(30))),
		spell("drown", "Drown", jl(BLM, 20), 35, sec(2), sec(6), tgtEnemy, 0, "water DoT, weakens", DotEl(10, "Water", sec(60)), Fx(mob.FxWeaken, 15, sec(30))),
		ability("manafont", "Manafont", jl(BLM, 30), mins(30), tgtSelf, 0, "flood yourself with mana", MPRestore(600)),
		ability("mana-wall", "Mana Wall", jl(BLM, 40), mins(5), tgtSelf, 0, "a barrier of mana", Buff(status.Protect, 40, sec(60)), Buff(status.Shell, 40, sec(60))),
		ability("enmity-douse", "Enmity Douse", jl(BLM, 35), mins(2), tgtEnemy, 0, "slip a foe's attention elsewhere", Enmity(-1500)),
	}
}

func redMageKit() []*kitEntry {
	return []*kitEntry{
		spell("dia2", "Dia II", jl(RDM, 40, WHM, 50), 60, sec(2), sec(8), tgtEnemy, 0, "stronger light DoT", DotEl(16, "Light", sec(120))),
		spell("diaga", "Diaga", jl(RDM, 60, WHM, 65), 90, sec(3), sec(15), tgtArea, 8, "area light DoT", DotEl(12, "Light", sec(90))),
		spell("addle", "Addle", jl(RDM, 50), 75, ms(2500), sec(15), tgtEnemy, 0, "slow and weaken", Fx(mob.FxSlow, 20, sec(90)), Fx(mob.FxWeaken, 20, sec(90))),
		spell("stoneskin", "Stoneskin", jl(RDM, 28, WHM, 28, SCH, 28), 55, sec(3), 0, tgtAlly, 0, "hard skin: defense up", Buff(status.Protect, 30, mins(2))),
		ability("composure", "Composure", jl(RDM, 50), mins(5), tgtSelf, 0, "poise: attack, accuracy and evasion up", Buff(status.AttackUp, 15, mins(5)), Buff(status.AccuracyUp, 15, mins(5)), Buff(status.EvasionUp, 15, mins(5))),
		ability("saboteur", "Saboteur", jl(RDM, 60), mins(5), tgtEnemy, 0, "cripple a foe: slow, weaken, blind", Fx(mob.FxSlow, 40, sec(60)), Fx(mob.FxWeaken, 35, sec(60)), Fx(mob.FxBlind, 40, sec(60))),
	}
}

func summonerKit() []*kitEntry {
	// Blood pacts: avatar powers expressed as abilities (the avatar duel summons live in main.go).
	return []*kitEntry{
		ability("searing-light", "Searing Light", jl(SMN, 5), sec(60), tgtArea, 6, "radiant pact: area light damage", Dmg(95, "int", 3, "Light")),
		ability("poison-nails", "Poison Nails", jl(SMN, 15), sec(60), tgtEnemy, 0, "garuda's talons: poison", Dot(10, sec(90))),
		ability("healing-ruby", "Healing Ruby", jl(SMN, 10), mins(2), tgtZone, 0, "carbuncle heals the party", Heal(160, "mnd", 2)),
		ability("hastega", "Hastega", jl(SMN, 20), mins(5), tgtZone, 0, "garuda speeds the party", Buff(status.Haste, 15, mins(3))),
		ability("shock-squall", "Shock Squall", jl(SMN, 25), mins(2), tgtEnemy, 0, "ramuh's squall: stun 6s", Dmg(40, "int", 1.5, "Lightning"), Fx(mob.FxStun, 0, sec(6))),
		ability("nightmare", "Nightmare", jl(SMN, 30), mins(2), tgtArea, 10, "diabolos's nightmare: area sleep", Fx(mob.FxSleep, 0, sec(60))),
		ability("lunar-cry", "Lunar Cry", jl(SMN, 35), mins(3), tgtArea, 10, "fenrir's cry: weaken", Fx(mob.FxWeaken, 25, sec(75))),
		ability("aerial-armor", "Aerial Armor", jl(SMN, 40), mins(5), tgtZone, 0, "ward the party", Buff(status.Protect, 25, mins(3)), Buff(status.Shell, 25, mins(3))),
		ability("perfect-defense", "Perfect Defense", jl(SMN, 55), mins(10), tgtSelf, 0, "ultimate ward", Buff(status.Protect, 60, sec(30)), Buff(status.Shell, 60, sec(30))),
		ability("ruinous-omen", "Ruinous Omen", jl(SMN, 70), mins(5), tgtEnemy, 0, "diabolos's omen: crushing damage", Dmg(450, "int", 6, "Dark")),
	}
}

func blueMageKit() []*kitEntry {
	return []*kitEntry{
		spell("cocoon", "Cocoon", jl(BLU, 8), 20, sec(2), sec(20), tgtSelf, 0, "harden: defense up", Buff(status.Protect, 25, mins(2))),
		spell("head-butt", "Head Butt", jl(BLU, 12), 15, ms(1500), sec(15), tgtEnemy, 0, "damage and stun", Phys(70, "str", 3), Fx(mob.FxStun, 0, sec(4))),
		spell("sheep-song", "Sheep Song", jl(BLU, 16), 25, sec(2), sec(15), tgtArea, 8, "lull nearby foes to sleep", Fx(mob.FxSleep, 0, sec(45))),
		spell("wild-carrot", "Wild Carrot", jl(BLU, 18), 30, sec(2), sec(5), tgtAlly, 0, "restore HP", Heal(130, "mnd", 3)),
		spell("sound-blast", "Sound Blast", jl(BLU, 20), 25, sec(2), sec(15), tgtArea, 8, "weaken nearby foes", Fx(mob.FxWeaken, 25, sec(60))),
		spell("poison-breath", "Poison Breath", jl(BLU, 22), 30, sec(2), sec(15), tgtArea, 8, "area poison", Dot(9, sec(60))),
		spell("bomb-toss", "Bomb Toss", jl(BLU, 24), 35, ms(2500), sec(12), tgtArea, 6, "area fire damage", Dmg(90, "int", 3, "Fire")),
		spell("healing-breeze", "Healing Breeze", jl(BLU, 26), 55, sec(3), sec(12), tgtZone, 0, "restore the party", Heal(160, "mnd", 2)),
		spell("frightful-roar", "Frightful Roar", jl(BLU, 30), 40, sec(2), sec(20), tgtArea, 10, "slow foes", Fx(mob.FxSlow, 30, sec(60))),
		spell("filamented-hold", "Filamented Hold", jl(BLU, 35), 40, sec(2), sec(15), tgtEnemy, 0, "slow a foe heavily", Fx(mob.FxSlow, 40, sec(75))),
		spell("mind-blast", "Mind Blast", jl(BLU, 36), 50, ms(2500), sec(15), tgtEnemy, 0, "damage and paralyze", Dmg(150, "int", 3, "Lightning"), Fx(mob.FxParalyze, 25, sec(45))),
		spell("magic-fruit", "Magic Fruit", jl(BLU, 40), 70, sec(2), sec(8), tgtAlly, 0, "big heal", Heal(300, "mnd", 3)),
		spell("jettatura", "Jettatura", jl(BLU, 45), 50, sec(2), sec(30), tgtEnemy, 0, "terrifying gaze: stun 8s", Fx(mob.FxStun, 0, sec(8))),
		spell("battery-charge", "Battery Charge", jl(BLU, 50), 60, sec(3), 0, tgtZone, 0, "party refresh", Buff(status.Refresh, 3, mins(3))),
	}
}

func scholarKit() []*kitEntry {
	return []*kitEntry{
		spell("pyrohelix", "Pyrohelix", jl(SCH, 20), 40, ms(2500), sec(8), tgtEnemy, 0, "fire helix DoT", DotEl(14, "Fire", sec(90))),
		spell("cryohelix", "Cryohelix", jl(SCH, 22), 40, ms(2500), sec(8), tgtEnemy, 0, "ice helix DoT, slows", DotEl(13, "Ice", sec(90)), Fx(mob.FxSlow, 15, sec(45))),
		spell("anemohelix", "Anemohelix", jl(SCH, 24), 40, ms(2500), sec(8), tgtEnemy, 0, "wind helix DoT", DotEl(13, "Wind", sec(90))),
		spell("geohelix", "Geohelix", jl(SCH, 26), 40, ms(2500), sec(8), tgtEnemy, 0, "earth helix DoT", DotEl(13, "Earth", sec(90))),
		spell("ionohelix", "Ionohelix", jl(SCH, 28), 40, ms(2500), sec(8), tgtEnemy, 0, "lightning helix DoT", DotEl(13, "Lightning", sec(90))),
		spell("hydrohelix", "Hydrohelix", jl(SCH, 30), 40, ms(2500), sec(8), tgtEnemy, 0, "water helix DoT", DotEl(13, "Water", sec(90))),
		spell("luminohelix", "Luminohelix", jl(SCH, 40), 55, sec(3), sec(10), tgtEnemy, 0, "light helix DoT", DotEl(18, "Light", sec(90))),
		spell("noctohelix", "Noctohelix", jl(SCH, 40), 55, sec(3), sec(10), tgtEnemy, 0, "dark helix DoT", DotEl(18, "Dark", sec(90))),
		ability("light-arts", "Light Arts", jl(SCH, 10), mins(3), tgtZone, 0, "healing study: party regen", Buff(status.Regen, 3, mins(3))),
		ability("dark-arts", "Dark Arts", jl(SCH, 10), mins(3), tgtSelf, 0, "dark study: spell focus", Buff(status.AttackUp, 15, mins(3)), MPRestore(40)),
		ability("sublimation", "Sublimation", jl(SCH, 35), mins(2), tgtSelf, 0, "store and release mana", MPRestore(120)),
	}
}

func geomancerKit() []*kitEntry {
	return []*kitEntry{
		spell("indi-fury", "Indi-Fury", jl(GEO, 10), 40, sec(2), 0, tgtZone, 0, "bubble: party attack up", Buff(status.AttackUp, 15, mins(5))),
		spell("indi-precision", "Indi-Precision", jl(GEO, 12), 40, sec(2), 0, tgtZone, 0, "bubble: party accuracy up", Buff(status.AccuracyUp, 15, mins(5))),
		spell("indi-haste", "Indi-Haste", jl(GEO, 30), 60, sec(2), 0, tgtZone, 0, "bubble: party haste", Buff(status.Haste, 15, mins(5))),
		spell("indi-regen", "Indi-Regen", jl(GEO, 14), 35, sec(2), 0, tgtZone, 0, "bubble: party regen", Buff(status.Regen, 3, mins(5))),
		spell("indi-refresh", "Indi-Refresh", jl(GEO, 16), 35, sec(2), 0, tgtZone, 0, "bubble: party refresh", Buff(status.Refresh, 2, mins(5))),
		spell("indi-barrier", "Indi-Barrier", jl(GEO, 18), 40, sec(2), 0, tgtZone, 0, "bubble: party defense up", Buff(status.Protect, 25, mins(5))),
		spell("indi-fend", "Indi-Fend", jl(GEO, 18), 40, sec(2), 0, tgtZone, 0, "bubble: party magic defense up", Buff(status.Shell, 25, mins(5))),
		spell("indi-fade", "Indi-Fade", jl(GEO, 22), 40, sec(2), 0, tgtZone, 0, "bubble: party evasion up", Buff(status.EvasionUp, 15, mins(5))),
		spell("geo-frailty", "Geo-Frailty", jl(GEO, 20), 50, sec(3), sec(15), tgtArea, 12, "hex bubble: weaken foes", Fx(mob.FxWeaken, 25, sec(90))),
		spell("geo-torpor", "Geo-Torpor", jl(GEO, 25), 50, sec(3), sec(15), tgtArea, 12, "hex bubble: slow foes", Fx(mob.FxSlow, 30, sec(90))),
		spell("geo-vex", "Geo-Vex", jl(GEO, 28), 50, sec(3), sec(15), tgtArea, 12, "hex bubble: blind foes", Fx(mob.FxBlind, 35, sec(90))),
		spell("geo-malaise", "Geo-Malaise", jl(GEO, 35), 60, sec(3), sec(15), tgtArea, 12, "hex bubble: poison foes", Dot(9, sec(90))),
		ability("radial-arcana", "Radial Arcana", jl(GEO, 40), mins(5), tgtZone, 0, "feed the party mana", MPRestore(60)),
	}
}

func bardKit() []*kitEntry {
	return []*kitEntry{
		// Lullabies: the reference crowd control. Area sleep, duration scales with CHR, breaks on
		// damage, harder to re-apply to the same mob.
		spell("lullaby", "Foe Lullaby", jl(BRD, 20), 40, ms(2500), sec(15), tgtArea, 12, "lull nearby foes to sleep", FxC(mob.FxSleep, 0, sec(45))),
		spell("horde-lullaby", "Horde Lullaby", jl(BRD, 40), 70, sec(3), sec(20), tgtArea, 15, "stronger, wider lullaby", FxC(mob.FxSleep, 0, sec(60))),
		spell("foe-requiem", "Foe Requiem", jl(BRD, 1), 35, sec(2), sec(10), tgtArea, 10, "requiem: area poison", Dot(5, sec(90))),
		spell("foe-requiem-ii", "Foe Requiem II", jl(BRD, 30), 55, sec(2), sec(10), tgtArea, 10, "requiem II: stronger area poison", Dot(9, sec(90))),
		spell("elegy", "Carnage Elegy", jl(BRD, 37), 50, sec(2), sec(20), tgtArea, 12, "elegy: slow foes", FxC(mob.FxSlow, 25, sec(90))),
		spell("threnody", "Threnody", jl(BRD, 45), 55, sec(2), sec(20), tgtArea, 12, "threnody: weaken foes", FxC(mob.FxWeaken, 25, sec(90))),
		spell("minuet", "Valor Minuet", jl(BRD, 3), 30, sec(2), 0, tgtZone, 0, "minuet: party attack up", BuffC(status.AttackUp, 8, mins(3))),
		spell("minuet-ii", "Valor Minuet II", jl(BRD, 30), 45, sec(2), 0, tgtZone, 0, "minuet II: stronger attack up", BuffC(status.AttackUp, 14, mins(3))),
		spell("madrigal", "Sword Madrigal", jl(BRD, 11), 35, sec(2), 0, tgtZone, 0, "madrigal: party accuracy up", BuffC(status.AccuracyUp, 12, mins(3))),
		spell("sheepfoe-mambo", "Sheepfoe Mambo", jl(BRD, 25), 35, sec(2), 0, tgtZone, 0, "mambo: party evasion up", BuffC(status.EvasionUp, 12, mins(3))),
		spell("scherzo", "Sentinel's Scherzo", jl(BRD, 50), 60, sec(2), 0, tgtZone, 0, "scherzo: party defense up", BuffC(status.Protect, 20, mins(3))),
		spell("finale", "Magic Finale", jl(BRD, 33), 40, sec(2), sec(15), tgtZone, 0, "finale: purge party ailments", Cleanse()),
	}
}

func corsairKit() []*kitEntry {
	return []*kitEntry{
		ability("fighters-roll", "Fighter's Roll", jl(COR, 5), mins(5), tgtZone, 0, "roll the dice: party attack up", BuffRoll(status.AttackUp, 10, mins(5))),
		ability("chaos-roll", "Chaos Roll", jl(COR, 14), mins(5), tgtZone, 0, "roll the dice: party big attack up", BuffRoll(status.AttackUp, 16, mins(5))),
		ability("hunters-roll", "Hunter's Roll", jl(COR, 11), mins(5), tgtZone, 0, "roll the dice: party accuracy up", BuffRoll(status.AccuracyUp, 12, mins(5))),
		ability("rogues-roll", "Rogue's Roll", jl(COR, 17), mins(5), tgtZone, 0, "roll the dice: party crit up", BuffRoll(status.CritUp, 8, mins(5))),
		ability("ninja-roll", "Ninja Roll", jl(COR, 20), mins(5), tgtZone, 0, "roll the dice: party evasion up", BuffRoll(status.EvasionUp, 14, mins(5))),
		ability("gallants-roll", "Gallant's Roll", jl(COR, 25), mins(5), tgtZone, 0, "roll the dice: party defense up", BuffRoll(status.Protect, 15, mins(5))),
		ability("healers-roll", "Healer's Roll", jl(COR, 30), mins(5), tgtZone, 0, "roll the dice: party regen", BuffRoll(status.Regen, 2, mins(5))),
		ability("evokers-roll", "Evoker's Roll", jl(COR, 35), mins(5), tgtZone, 0, "roll the dice: party refresh", BuffRoll(status.Refresh, 1, mins(5))),
		ability("fire-shot", "Fire Shot", jl(COR, 1), sec(20), tgtEnemy, 0, "quick draw: fire", Dmg(70, "ranged", 3, "Fire")),
		ability("ice-shot", "Ice Shot", jl(COR, 3), sec(20), tgtEnemy, 0, "quick draw: ice, slows", Dmg(65, "ranged", 3, "Ice"), Fx(mob.FxSlow, 15, sec(20))),
		ability("wind-shot", "Wind Shot", jl(COR, 5), sec(20), tgtEnemy, 0, "quick draw: wind", Dmg(65, "ranged", 3, "Wind")),
		ability("earth-shot", "Earth Shot", jl(COR, 7), sec(20), tgtEnemy, 0, "quick draw: earth", Dmg(65, "ranged", 3, "Earth")),
		ability("thunder-shot", "Thunder Shot", jl(COR, 9), sec(20), tgtEnemy, 0, "quick draw: lightning, paralyzes", Dmg(70, "ranged", 3, "Lightning"), Fx(mob.FxParalyze, 15, sec(20))),
		ability("water-shot", "Water Shot", jl(COR, 11), sec(20), tgtEnemy, 0, "quick draw: water", Dmg(65, "ranged", 3, "Water")),
		ability("light-shot", "Light Shot", jl(COR, 40), sec(45), tgtEnemy, 0, "quick draw: light, blinds", Dmg(120, "ranged", 4, "Light"), Fx(mob.FxBlind, 30, sec(30))),
		ability("dark-shot", "Dark Shot", jl(COR, 40), sec(45), tgtEnemy, 0, "quick draw: dark, weakens", Dmg(120, "ranged", 4, "Dark"), Fx(mob.FxWeaken, 25, sec(30))),
	}
}

func puppetmasterKit() []*kitEntry {
	return []*kitEntry{
		ability("fire-maneuver", "Fire Maneuver", jl(PUP, 1), sec(10), tgtSelf, 0, "attack up", Buff(status.AttackUp, 12, mins(3))),
		ability("ice-maneuver", "Ice Maneuver", jl(PUP, 1), sec(10), tgtSelf, 0, "magic defense up", Buff(status.Shell, 20, mins(3))),
		ability("wind-maneuver", "Wind Maneuver", jl(PUP, 1), sec(10), tgtSelf, 0, "evasion up", Buff(status.EvasionUp, 12, mins(3))),
		ability("earth-maneuver", "Earth Maneuver", jl(PUP, 1), sec(10), tgtSelf, 0, "defense up", Buff(status.Protect, 20, mins(3))),
		ability("thunder-maneuver", "Thunder Maneuver", jl(PUP, 1), sec(10), tgtSelf, 0, "accuracy up", Buff(status.AccuracyUp, 12, mins(3))),
		ability("water-maneuver", "Water Maneuver", jl(PUP, 1), sec(10), tgtSelf, 0, "refresh", Buff(status.Refresh, 2, mins(3))),
		ability("light-maneuver", "Light Maneuver", jl(PUP, 1), sec(10), tgtSelf, 0, "regen", Buff(status.Regen, 3, mins(3))),
		ability("dark-maneuver", "Dark Maneuver", jl(PUP, 1), sec(10), tgtSelf, 0, "crit up", Buff(status.CritUp, 8, mins(3))),
		ability("repair", "Repair", jl(PUP, 15), mins(1), tgtSelf, 0, "field repairs: heal", Heal(220, "vit", 3)),
		ability("maintenance", "Maintenance", jl(PUP, 20), mins(2), tgtSelf, 0, "clear ailments", Cleanse()),
		ability("overdrive", "Overdrive", jl(PUP, 50), mins(10), tgtSelf, 0, "overclock: attack and haste", Buff(status.AttackUp, 40, sec(60)), Buff(status.Haste, 20, sec(60))),
	}
}

func dancerKit() []*kitEntry {
	return []*kitEntry{
		ability("drain-samba", "Drain Samba", jl(DNC, 5), mins(1), tgtSelf, 0, "next 10 hits drain HP", CoatDrain(25, 10)),
		ability("curing-waltz", "Curing Waltz", jl(DNC, 15), sec(15), tgtAlly, 0, "waltz: heal an ally", Heal(150, "chr", 3)),
		ability("spectral-jig", "Spectral Jig", jl(DNC, 25), mins(1), tgtSelf, 0, "jig: vanish from sight", Hide(sec(45))),
		ability("quickstep", "Quickstep", jl(DNC, 20), sec(30), tgtEnemy, 0, "step: weaken a foe", Fx(mob.FxWeaken, 20, sec(45))),
		ability("box-step", "Box Step", jl(DNC, 30), sec(30), tgtEnemy, 0, "step: weaken harder, slowing", Fx(mob.FxWeaken, 25, sec(45)), Fx(mob.FxSlow, 15, sec(45))),
		ability("stutter-step", "Stutter Step", jl(DNC, 40), sec(30), tgtEnemy, 0, "step: slow a foe", Fx(mob.FxSlow, 25, sec(45))),
		ability("desperate-flourish", "Desperate Flourish", jl(DNC, 30), mins(1), tgtEnemy, 0, "flourish: bind a foe", Fx(mob.FxBind, 0, sec(25))),
		ability("violent-flourish", "Violent Flourish", jl(DNC, 45), mins(1), tgtEnemy, 0, "flourish: stun and strike", Phys(60, "dex", 2), Fx(mob.FxStun, 0, sec(5))),
		ability("animated-flourish", "Animated Flourish", jl(DNC, 25), sec(45), tgtEnemy, 0, "flourish: draw attention", Enmity(1200)),
		ability("curing-waltz-ii", "Curing Waltz II", jl(DNC, 30), sec(20), tgtAlly, 0, "waltz II: heal an ally", Heal(300, "chr", 4)),
		ability("healing-waltz", "Healing Waltz", jl(DNC, 35), sec(20), tgtAlly, 0, "waltz: purge ailments", Cleanse()),
		ability("divine-waltz", "Divine Waltz", jl(DNC, 40), mins(1), tgtZone, 0, "waltz: heal the party", Heal(180, "chr", 3)),
		ability("haste-samba", "Haste Samba", jl(DNC, 45), mins(1), tgtZone, 0, "samba: party haste", BuffC(status.Haste, 12, mins(2))),
		ability("fan-dance", "Fan Dance", jl(DNC, 35), mins(2), tgtSelf, 0, "defensive dance", Buff(status.Protect, 20, mins(2)), Buff(status.EvasionUp, 15, mins(2))),
		ability("saber-dance", "Saber Dance", jl(DNC, 50), mins(3), tgtSelf, 0, "offensive dance", Buff(status.AttackUp, 20, mins(2)), Buff(status.CritUp, 15, mins(2))),
	}
}

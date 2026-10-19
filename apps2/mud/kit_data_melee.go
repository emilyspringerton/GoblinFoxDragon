package main

import (
	"dragonsnshit/server/mob"
	"dragonsnshit/server/status"
)

func warriorKit() []*kitEntry {
	return []*kitEntry{
		ability("tomahawk", "Tomahawk", jl(WAR, 30), sec(45), tgtEnemy, 0, "hurl an axe: damage and weaken", Phys(70, "str", 3), Fx(mob.FxWeaken, 15, sec(20))),
		ability("retaliation", "Retaliation", jl(WAR, 35), mins(2), tgtSelf, 0, "evasion up", Buff(status.EvasionUp, 20, sec(90))),
		ability("blood-rage", "Blood Rage", jl(WAR, 45), mins(5), tgtZone, 0, "party attack up", Buff(status.AttackUp, 20, sec(60))),
		ability("warriors-charge", "Warrior's Charge", jl(WAR, 60), mins(5), tgtSelf, 0, "next hit crits for +100%", NextCrit(), NextAtk(100)),
	}
}

func monkKit() []*kitEntry {
	return []*kitEntry{
		ability("focus", "Focus", jl(MNK, 15), mins(2), tgtSelf, 0, "accuracy up", Buff(status.AccuracyUp, 25, mins(2))),
		ability("dodge", "Dodge", jl(MNK, 15), mins(2), tgtSelf, 0, "evasion up", Buff(status.EvasionUp, 25, mins(2))),
		ability("mantra", "Mantra", jl(MNK, 35), mins(5), tgtZone, 0, "party regen", Buff(status.Regen, 3, mins(2))),
		ability("impetus", "Impetus", jl(MNK, 40), mins(3), tgtSelf, 0, "attack and crit up", Buff(status.AttackUp, 25, sec(60)), Buff(status.CritUp, 15, sec(60))),
		ability("perfect-counter", "Perfect Counter", jl(MNK, 50), mins(3), tgtSelf, 0, "near-untouchable for 15s", Buff(status.EvasionUp, 60, sec(15))),
		ability("hundred-fists", "Hundred Fists", jl(MNK, 75), mins(30), tgtSelf, 0, "attack, haste and crit surge", Buff(status.AttackUp, 50, sec(30)), Buff(status.Haste, 25, sec(30)), Buff(status.CritUp, 20, sec(30))),
	}
}

func thiefKit() []*kitEntry {
	return []*kitEntry{
		ability("steal", "Steal", jl(THF, 5), mins(1), tgtEnemy, 0, "steal an item from a foe (once per foe)", Steal()),
		ability("hide", "Hide", jl(THF, 15), mins(5), tgtSelf, 0, "vanish: foes lose you", Hide(sec(30))),
		ability("flee", "Flee", jl(THF, 25), mins(5), tgtSelf, 0, "speed boost", Buff(status.Haste, 30, sec(30))),
		ability("bully", "Bully", jl(THF, 30), sec(90), tgtEnemy, 0, "damage and weaken", Phys(60, "str", 2), Fx(mob.FxWeaken, 20, sec(30))),
		ability("mug", "Mug", jl(THF, 40), mins(2), tgtEnemy, 0, "damage and lift some flow", Phys(80, "dex", 2.5), Flow(60)),
		ability("perfect-dodge", "Perfect Dodge", jl(THF, 45), mins(5), tgtSelf, 0, "evasion way up for 15s", Buff(status.EvasionUp, 60, sec(15))),
	}
}

func beastmasterKit() []*kitEntry {
	return []*kitEntry{
		ability("reward", "Reward", jl(BST, 12), mins(1), tgtSelf, 0, "feed on rewards: heal", Heal(200, "chr", 3)),
		ability("charm", "Charm", jl(BST, 23), mins(2), tgtEnemy, 0, "pacify a foe (breaks on damage)", FxC(mob.FxSleep, 0, sec(45))),
		ability("feral-howl", "Feral Howl", jl(BST, 50), mins(5), tgtArea, 10, "terrify foes: stun 8s", Fx(mob.FxStun, 0, sec(8))),
		ability("killer-instinct", "Killer Instinct", jl(BST, 52), mins(5), tgtZone, 0, "party attack up", Buff(status.AttackUp, 15, mins(3))),
		ability("snarl", "Snarl", jl(BST, 20), sec(45), tgtEnemy, 0, "bite and pull enmity", Phys(55, "str", 2), Enmity(600)),
	}
}

func rangerKit() []*kitEntry {
	return []*kitEntry{
		ability("sharpshot", "Sharpshot", jl(RNG, 1), mins(3), tgtSelf, 0, "accuracy up", Buff(status.AccuracyUp, 40, mins(1))),
		ability("scavenge", "Scavenge", jl(RNG, 10), mins(3), tgtSelf, 0, "recover your strength", Heal(100, "agi", 2), MPRestore(20)),
		ability("camouflage", "Camouflage", jl(RNG, 20), mins(5), tgtSelf, 0, "melt into the surroundings", Hide(sec(60))),
		ability("barrage", "Barrage", jl(RNG, 30), mins(3), tgtEnemy, 0, "three quick shots", PhysHits(3, 35, "ranged", 1.5)),
		ability("double-shot", "Double Shot", jl(RNG, 40), mins(2), tgtSelf, 0, "next hit crits harder", NextCrit(), NextAtk(50)),
		ability("decoy-shot", "Decoy Shot", jl(RNG, 45), mins(2), tgtEnemy, 0, "draw a foe's attention", Phys(40, "ranged", 1), Enmity(1500)),
		ability("flashy-shot", "Flashy Shot", jl(RNG, 55), mins(3), tgtEnemy, 0, "dazzling shot: stun 4s", Phys(45, "ranged", 1.5), Fx(mob.FxStun, 0, sec(4))),
		ability("eagle-eye-shot", "Eagle Eye Shot", jl(RNG, 75), mins(30), tgtEnemy, 0, "a single devastating shot", PayHP(5), PhysHits(1, 450, "ranged", 6)),
	}
}

func samuraiKit() []*kitEntry {
	return []*kitEntry{
		ability("meditate", "Meditate", jl(SAM, 1), mins(3), tgtSelf, 0, "gather TP", TP(100)),
		ability("third-eye", "Third Eye", jl(SAM, 15), mins(1), tgtSelf, 0, "foresee attacks: evasion up", Buff(status.EvasionUp, 50, sec(30))),
		ability("hasso", "Hasso", jl(SAM, 25), mins(1), tgtSelf, 0, "two-hand stance: attack and haste", Buff(status.AttackUp, 15, mins(3)), Buff(status.Haste, 10, mins(3))),
		ability("seigan", "Seigan", jl(SAM, 35), mins(1), tgtSelf, 0, "defensive stance", Buff(status.EvasionUp, 20, mins(3)), Buff(status.Protect, 15, mins(3))),
		ability("sekkanoki", "Sekkanoki", jl(SAM, 40), mins(3), tgtSelf, 0, "next hit lands twice as hard", NextAtk(100)),
		ability("blade-bash", "Blade Bash", jl(SAM, 30), mins(3), tgtEnemy, 0, "stun with the flat of the blade", Phys(40, "str", 2), Fx(mob.FxStun, 0, sec(4))),
		ability("meikyo-shisui", "Meikyo Shisui", jl(SAM, 50), mins(30), tgtSelf, 0, "fill your TP gauge", TP(300)),
	}
}

func dragoonKit() []*kitEntry {
	return []*kitEntry{
		ability("jump", "Jump", jl(DRG, 1), sec(60), tgtEnemy, 0, "leap and strike", Phys(110, "str", 4)),
		ability("ancient-circle", "Ancient Circle", jl(DRG, 5), mins(5), tgtZone, 0, "party defense up", Buff(status.Protect, 12, mins(3))),
		ability("spirit-link", "Spirit Link", jl(DRG, 25), mins(1), tgtSelf, 0, "share your wyvern's vitality: heal", Heal(250, "vit", 4)),
		ability("high-jump", "High Jump", jl(DRG, 35), mins(2), tgtEnemy, 0, "strike from above and drop out of the fight", Phys(150, "str", 5), Enmity(-1200)),
		ability("angon", "Angon", jl(DRG, 40), mins(3), tgtEnemy, 0, "pin a foe: damage and weaken", Phys(70, "str", 3), Fx(mob.FxWeaken, 25, sec(45))),
		ability("healing-breath", "Healing Breath", jl(DRG, 45), mins(3), tgtZone, 0, "wyvern's breath heals the party", Heal(150, "vit", 2)),
		ability("spirit-surge", "Spirit Surge", jl(DRG, 50), mins(10), tgtSelf, 0, "dragon's might: attack and haste", Buff(status.AttackUp, 40, sec(60)), Buff(status.Haste, 15, sec(60))),
		ability("dragon-breaker", "Dragon Breaker", jl(DRG, 60), mins(3), tgtEnemy, 0, "damage and slow", Phys(100, "str", 4), Fx(mob.FxSlow, 25, sec(45))),
	}
}

func ninjaKit() []*kitEntry {
	return []*kitEntry{
		// tool-fueled ninjutsu: cheap MP, short recasts (NIN's pool is small)
		spell("utsusemi", "Utsusemi", jl(NIN, 12), 25, sec(2), sec(20), tgtSelf, 0, "shadows: evasion up", Buff(status.EvasionUp, 40, sec(25))),
		spell("kurayami", "Kurayami", jl(NIN, 15), 20, sec(2), sec(10), tgtEnemy, 0, "blind a foe", Fx(mob.FxBlind, 40, sec(60))),
		spell("hojo", "Hojo", jl(NIN, 20), 25, sec(2), sec(10), tgtEnemy, 0, "slow a foe", Fx(mob.FxSlow, 30, sec(75))),
		spell("dokumori", "Dokumori", jl(NIN, 25), 30, sec(2), sec(10), tgtEnemy, 0, "poison a foe", Dot(9, sec(75))),
		spell("jubaku", "Jubaku", jl(NIN, 30), 30, sec(2), sec(10), tgtEnemy, 0, "paralyze a foe", Fx(mob.FxParalyze, 30, sec(75))),
		spell("aisha", "Aisha", jl(NIN, 35), 35, sec(2), sec(10), tgtEnemy, 0, "weaken a foe", Fx(mob.FxWeaken, 25, sec(75))),
		spell("tonko", "Tonko", jl(NIN, 18), 20, sec(2), sec(60), tgtSelf, 0, "vanish from sight", Hide(sec(45))),
		ability("yonin", "Yonin", jl(NIN, 40), mins(3), tgtSelf, 0, "defensive stance", Buff(status.EvasionUp, 30, mins(3)), Buff(status.Protect, 10, mins(3))),
		ability("innin", "Innin", jl(NIN, 40), mins(3), tgtSelf, 0, "offensive stance", Buff(status.CritUp, 20, mins(3)), Buff(status.AccuracyUp, 15, mins(3))),
		ability("futae", "Futae", jl(NIN, 50), mins(3), tgtSelf, 0, "next strike lands harder", NextAtk(60)),
		ability("sange", "Sange", jl(NIN, 60), mins(5), tgtEnemy, 0, "storm of blades", PhysHits(5, 40, "dex", 1.5)),
	}
}

func paladinKit() []*kitEntry {
	return []*kitEntry{
		ability("taunt", "Taunt", jl(PLD, 1), sec(30), tgtEnemy, 0, "draw a foe's attention", Enmity(1800)),
		ability("shield-bash", "Shield Bash", jl(PLD, 15), mins(3), tgtEnemy, 0, "stun and damage", Phys(40, "vit", 2), Fx(mob.FxStun, 0, sec(4))),
		ability("holy-circle", "Holy Circle", jl(PLD, 25), mins(5), tgtZone, 0, "party defense up", Buff(status.Protect, 15, mins(3))),
		ability("invincible", "Invincible", jl(PLD, 40), mins(10), tgtSelf, 0, "near-invulnerable for 15s", Buff(status.Protect, 80, sec(15)), Buff(status.EvasionUp, 30, sec(15))),
		ability("chivalry", "Chivalry", jl(PLD, 55), mins(5), tgtSelf, 0, "convert valor into MP", MPRestore(150)),
		ability("divine-emblem", "Divine Emblem", jl(PLD, 70), mins(5), tgtSelf, 0, "next strike holy-blessed", NextAtk(80)),
	}
}

func darkKnightKit() []*kitEntry {
	return []*kitEntry{
		ability("arcane-circle", "Arcane Circle", jl(DRK, 5), mins(5), tgtZone, 0, "party defense up", Buff(status.Protect, 10, mins(3))),
		ability("last-resort", "Last Resort", jl(DRK, 15), mins(1), tgtSelf, 0, "reckless power: attack up", Buff(status.AttackUp, 30, mins(3))),
		ability("weapon-bash", "Weapon Bash", jl(DRK, 20), mins(3), tgtEnemy, 0, "stun and damage", Phys(50, "str", 2), Fx(mob.FxStun, 0, sec(4))),
		ability("souleater", "Souleater", jl(DRK, 30), mins(1), tgtSelf, 0, "pay HP for a brutal next strike", PayHP(10), NextAtk(80)),
		ability("diabolic-eye", "Diabolic Eye", jl(DRK, 55), mins(5), tgtSelf, 0, "accuracy and crit up", Buff(status.AccuracyUp, 20, mins(3)), Buff(status.CritUp, 10, mins(3))),
		spell("dread-spikes", "Dread Spikes", jl(DRK, 45), 60, sec(2), 0, tgtSelf, 0, "dark armor", Buff(status.Protect, 30, mins(2))),
	}
}

func runeFencerKit() []*kitEntry {
	return []*kitEntry{
		ability("swordplay", "Swordplay", jl(RUN, 1), mins(2), tgtSelf, 0, "accuracy and evasion up", Buff(status.AccuracyUp, 20, mins(3)), Buff(status.EvasionUp, 20, mins(3))),
		ability("pflug", "Pflug", jl(RUN, 20), mins(2), tgtSelf, 0, "evasion up", Buff(status.EvasionUp, 25, mins(3))),
		ability("gambit", "Gambit", jl(RUN, 25), mins(3), tgtArea, 10, "unravel foes: weaken", Fx(mob.FxWeaken, 25, sec(90))),
		ability("rayke", "Rayke", jl(RUN, 40), mins(3), tgtArea, 10, "bind foes in slowing runes", Fx(mob.FxSlow, 30, sec(90))),
		ability("lunge", "Lunge", jl(RUN, 30), sec(45), tgtEnemy, 0, "rune-charged thrust", Phys(130, "str", 4)),
		ability("swipe", "Swipe", jl(RUN, 10), sec(30), tgtEnemy, 0, "quick rune-cut", Phys(70, "str", 3)),
		ability("vallation", "Vallation", jl(RUN, 50), mins(5), tgtSelf, 0, "ward against blows and spells", Buff(status.Protect, 25, mins(5)), Buff(status.Shell, 25, mins(5))),
		ability("valiance", "Valiance", jl(RUN, 65), mins(5), tgtZone, 0, "ward the whole party", Buff(status.Protect, 25, mins(5)), Buff(status.Shell, 25, mins(5))),
		ability("one-for-all", "One for All", jl(RUN, 55), mins(5), tgtZone, 0, "bolster the party's health", Heal(250, "vit", 3)),
		ability("vivacious-pulse", "Vivacious Pulse", jl(RUN, 45), mins(2), tgtSelf, 0, "restore and purify", Heal(200, "mnd", 3), Cleanse()),
		ability("battuta", "Battuta", jl(RUN, 60), mins(3), tgtSelf, 0, "counter stance", Buff(status.EvasionUp, 30, mins(3)), Buff(status.AttackUp, 15, mins(3))),
		spell("foil", "Foil", jl(RUN, 25), 40, sec(1), sec(20), tgtSelf, 0, "evasion surge", Buff(status.EvasionUp, 35, sec(20))),
	}
}

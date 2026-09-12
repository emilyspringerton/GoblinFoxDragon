package main

// weapon_skill_level.go implements S412-06 (founder real-time, a rapid weapon-skill-leveling
// design burst): a real, fishing/mining-shaped skill-gain-on-use mechanic per weapon TYPE (sword,
// dagger, axe, greatsword, staff, club, polearm, h2h, katana, bow, instrument), wired into real
// accuracy/damage, plus a real skill-level gate on which weapon skills (skillchain.
// CanonicalWeaponSkills) a player can actually fire via `ws`.
//
// This is distinct from p.wsSkill (the currently SELECTED weapon skill ability name, e.g. "Fast
// Blade") -- p.weaponSkills tracks numeric per-weapon-type mastery, the same real shape
// p.miningSkill/p.fishingSkill already established for gathering (a flat increment per
// successful use, capped). Per-job weapon-type CAPS (RDM S-tier sword, THF S-tier dagger, etc.,
// S412-07) are a real, separate, follow-up layer on top of this file's own flat, job-independent
// cap -- everyone's cap here is the same formula regardless of job until that lands.

import (
	"dragonsnshit/server/gear"
)

// weaponTypeH2H is both the real "no weapon equipped" fallback AND MNK's own real affinity type
// (S412-07) -- unarmed combat IS hand-to-hand combat, the same real weapon type a MNK trains,
// matching the founder's own framing: "when we first start out we dont have a weapon so hand to
// hand should level up."
const weaponTypeH2H = "h2h"

// weaponSkillCapPerLevel is the flat, job-independent skill cap this pass grants (S412-07 will
// multiply this per job/weapon-type affinity, not replace it) -- 10 skill points per character
// level, so cap == 100 at character level 10, matching the founder's own worked example number
// ("RDM is S tier at sword so at lvl 10 they cap at a higher level").
const weaponSkillCapPerLevel = 10

// currentWeaponType resolves the weapon TYPE (not the item name) of whatever a player currently
// has equipped in their main hand -- itemdef.ItemDef.WeaponType, a real, authored field (S412-06)
// distinct from Category ("weapon" alone doesn't say sword vs. dagger vs. axe). Falls back to
// weaponTypeH2H (unarmed = hand-to-hand) whenever there's no equipment system, no item equipped,
// the equipped item isn't in the registry, or it has no weapon_type authored yet -- a real,
// honest default, not a panic.
func currentWeaponType(p *player) string {
	if p.equip == nil {
		return weaponTypeH2H
	}
	entry, err := p.equip.ItemAt(gear.SlotMainHand)
	if err != nil || entry == nil {
		return weaponTypeH2H
	}
	def, ok := itemdefReg.ByID(entry.DefID)
	if !ok || def.WeaponType == "" {
		return weaponTypeH2H
	}
	return def.WeaponType
}

// weaponSkillCap returns the real, current skill cap at charLevel -- S412-07 will extend this
// with a per-job/weapon-type affinity multiplier; today every weapon type and every job shares
// the same flat formula.
func weaponSkillCap(charLevel int) int {
	cap := charLevel * weaponSkillCapPerLevel
	if cap < weaponSkillCapPerLevel {
		cap = weaponSkillCapPerLevel // level 0/uninitialized defensively still grants a real, non-zero cap
	}
	return cap
}

// gainWeaponSkill grants real skill-gain-on-use (fishing/mining-shaped: a flat +1 per successful
// use, capped) to whatever weapon type the player currently has equipped. Called once per LANDED
// player auto-attack (see resolvePlayerAutoAttackDamage) -- a miss teaches you nothing, matching
// this file's own established gather-skill convention (only a successful mine/catch grants points
// there too).
func gainWeaponSkill(p *player, charLevel int) {
	if p.weaponSkills == nil {
		p.weaponSkills = make(map[string]int)
	}
	wt := currentWeaponType(p)
	cap := weaponSkillCap(charLevel)
	if p.weaponSkills[wt] >= cap {
		return
	}
	p.weaponSkills[wt]++
	if p.weaponSkills[wt] > cap {
		p.weaponSkills[wt] = cap
	}
}

// weaponSkillLevel returns the player's real, current skill in the given weapon type (0 if never
// trained).
func weaponSkillLevel(p *player, weaponType string) int {
	if p.weaponSkills == nil {
		return 0
	}
	return p.weaponSkills[weaponType]
}

// charLevelOf returns p's real, current character level -- p.charXP should always be non-nil in
// practice (both real character-creation paths set it), but this degrades to level 1 rather than
// panicking if it's ever missing (e.g. a test-constructed player).
func charLevelOf(p *player) int {
	if p.charXP == nil {
		return 1
	}
	return p.charXP.Level
}

// Package job implements FFXI-parity job definitions for DragonsNShit.
//
// # Job system
//
// 22 base jobs are defined as string IDs (FFXI-canonical abbreviations).
// Each job has a base stat table (HP/MP + six primary stats).
// HPAtLevel and MPAtLevel use simplified FFXI-style scaling:
//   HP = BaseHP + (HPPerLevel * level)
//   MP = BaseMP + (MPPerLevel * level)
//
// Stats are purely informational here; combat application is done by the combat package.
package job

import "errors"

// JobID is a string abbreviation for a job (e.g. "WAR", "WHM").
type JobID = string

// All 22 base job IDs — FFXI canonical.
const (
	WAR = "WAR" // Warrior
	MNK = "MNK" // Monk
	WHM = "WHM" // White Mage
	BLM = "BLM" // Black Mage
	RDM = "RDM" // Red Mage
	THF = "THF" // Thief
	PLD = "PLD" // Paladin
	DRK = "DRK" // Dark Knight
	BST = "BST" // Beastmaster
	BRD = "BRD" // Bard
	RNG = "RNG" // Ranger
	SAM = "SAM" // Samurai
	NIN = "NIN" // Ninja
	DRG = "DRG" // Dragoon
	SMN = "SMN" // Summoner
	BLU = "BLU" // Blue Mage
	COR = "COR" // Corsair
	PUP = "PUP" // Puppetmaster
	DNC = "DNC" // Dancer
	SCH = "SCH" // Scholar
	GEO = "GEO" // Geomancer
	RUN = "RUN" // Rune Fencer

	// ASN (Assassin) is a real, new, non-FFXI-canonical 23rd job (GFD-JOB-244122, founder:
	// "NEW CLASS ENFEEBLE DD SUPPORT SUSTAIN (sorta like DNC)... can cast aoe buffs like a
	// bard for different poisons that have different impacts on the different mob types...
	// mp regen affordances"). A real, deliberate hybrid: DEX/AGI-focused damage dealer (the
	// "DD" half) with two support/utility abilities (see subjob.go's AssassinAbilities) --
	// an AoE poison JA scaled per real mob-Kind resistance (undead-tagged kinds resist it,
	// matching real FFXI's own undead-resist-poison convention) and an MP-regen support JA
	// (the "sustain" half, same real mechanism Bard's own Ballad song already uses).
	ASN = "ASN" // Assassin
)

// AllJobs lists all 23 jobs in FFXI definition order, plus this repo's own real ASN addition.
var AllJobs = []JobID{WAR, MNK, WHM, BLM, RDM, THF, PLD, DRK, BST, BRD, RNG, SAM, NIN, DRG, SMN, BLU, COR, PUP, DNC, SCH, GEO, RUN, ASN}

var ErrUnknownJob = errors.New("unknown job ID")

// Stats holds the base stat block for a job at level 1.
// HP/MP grow via HPPerLevel/MPPerLevel each level.
type Stats struct {
	BaseHP     int
	HPPerLevel int
	BaseMP     int
	MPPerLevel int
	STR        int
	DEX        int
	VIT        int
	AGI        int
	INT        int
	MND        int
	CHR        int
}

// jobStats is the base stat table for all 22 jobs.
// Values are approximate FFXI parity (simplified for DragonsNShit).
var jobStats = map[JobID]Stats{
	WAR: {BaseHP: 90,  HPPerLevel: 15, BaseMP: 0,  MPPerLevel: 0,  STR: 8, DEX: 7, VIT: 8, AGI: 6, INT: 5, MND: 5, CHR: 5},
	MNK: {BaseHP: 100, HPPerLevel: 16, BaseMP: 0,  MPPerLevel: 0,  STR: 8, DEX: 8, VIT: 8, AGI: 7, INT: 4, MND: 4, CHR: 5},
	WHM: {BaseHP: 65,  HPPerLevel: 10, BaseMP: 85,  MPPerLevel: 14, STR: 5, DEX: 5, VIT: 5, AGI: 5, INT: 6, MND: 9, CHR: 8},
	BLM: {BaseHP: 55,  HPPerLevel: 8,  BaseMP: 100, MPPerLevel: 16, STR: 4, DEX: 5, VIT: 4, AGI: 5, INT: 9, MND: 7, CHR: 5},
	RDM: {BaseHP: 70,  HPPerLevel: 11, BaseMP: 75,  MPPerLevel: 12, STR: 6, DEX: 6, VIT: 5, AGI: 6, INT: 7, MND: 7, CHR: 7},
	THF: {BaseHP: 75,  HPPerLevel: 12, BaseMP: 0,   MPPerLevel: 0,  STR: 6, DEX: 9, VIT: 6, AGI: 9, INT: 6, MND: 5, CHR: 6},
	PLD: {BaseHP: 95,  HPPerLevel: 15, BaseMP: 60,  MPPerLevel: 9,  STR: 8, DEX: 6, VIT: 9, AGI: 5, INT: 5, MND: 7, CHR: 7},
	DRK: {BaseHP: 90,  HPPerLevel: 14, BaseMP: 45,  MPPerLevel: 8,  STR: 9, DEX: 6, VIT: 7, AGI: 5, INT: 6, MND: 5, CHR: 4},
	BST: {BaseHP: 80,  HPPerLevel: 13, BaseMP: 0,   MPPerLevel: 0,  STR: 7, DEX: 7, VIT: 7, AGI: 7, INT: 5, MND: 5, CHR: 7},
	BRD: {BaseHP: 65,  HPPerLevel: 10, BaseMP: 50,  MPPerLevel: 9,  STR: 5, DEX: 7, VIT: 5, AGI: 7, INT: 6, MND: 6, CHR: 9},
	RNG: {BaseHP: 75,  HPPerLevel: 12, BaseMP: 0,   MPPerLevel: 0,  STR: 7, DEX: 8, VIT: 6, AGI: 8, INT: 6, MND: 5, CHR: 5},
	SAM: {BaseHP: 85,  HPPerLevel: 14, BaseMP: 0,   MPPerLevel: 0,  STR: 8, DEX: 7, VIT: 7, AGI: 7, INT: 6, MND: 6, CHR: 5},
	NIN: {BaseHP: 75,  HPPerLevel: 12, BaseMP: 0,   MPPerLevel: 0,  STR: 6, DEX: 9, VIT: 6, AGI: 9, INT: 6, MND: 5, CHR: 5},
	DRG: {BaseHP: 85,  HPPerLevel: 14, BaseMP: 0,   MPPerLevel: 0,  STR: 8, DEX: 6, VIT: 8, AGI: 6, INT: 5, MND: 6, CHR: 6},
	SMN: {BaseHP: 60,  HPPerLevel: 9,  BaseMP: 90,  MPPerLevel: 15, STR: 4, DEX: 4, VIT: 4, AGI: 5, INT: 7, MND: 8, CHR: 7},
	BLU: {BaseHP: 80,  HPPerLevel: 13, BaseMP: 55,  MPPerLevel: 9,  STR: 7, DEX: 7, VIT: 6, AGI: 6, INT: 7, MND: 6, CHR: 5},
	COR: {BaseHP: 70,  HPPerLevel: 11, BaseMP: 45,  MPPerLevel: 8,  STR: 6, DEX: 8, VIT: 5, AGI: 8, INT: 6, MND: 6, CHR: 6},
	PUP: {BaseHP: 75,  HPPerLevel: 12, BaseMP: 30,  MPPerLevel: 6,  STR: 7, DEX: 8, VIT: 6, AGI: 7, INT: 6, MND: 5, CHR: 5},
	DNC: {BaseHP: 80,  HPPerLevel: 13, BaseMP: 0,   MPPerLevel: 0,  STR: 7, DEX: 8, VIT: 7, AGI: 8, INT: 5, MND: 5, CHR: 7},
	SCH: {BaseHP: 55,  HPPerLevel: 8,  BaseMP: 95,  MPPerLevel: 15, STR: 4, DEX: 5, VIT: 4, AGI: 5, INT: 8, MND: 8, CHR: 6},
	GEO: {BaseHP: 60,  HPPerLevel: 9,  BaseMP: 85,  MPPerLevel: 14, STR: 5, DEX: 5, VIT: 5, AGI: 5, INT: 8, MND: 8, CHR: 7},
	RUN: {BaseHP: 80,  HPPerLevel: 13, BaseMP: 40,  MPPerLevel: 7,  STR: 7, DEX: 7, VIT: 7, AGI: 7, INT: 6, MND: 6, CHR: 6},
	// ASN: DEX/AGI-focused DD (real STR/DEX/AGI band close to THF/DNC), with enough MP for its
	// own two real support JAs (venom/siphon, subjob.go) -- not a caster, so BaseMP sits well
	// below any real magic-user job, closer to PLD/DRK's "some MP for a few real abilities."
	ASN: {BaseHP: 78,  HPPerLevel: 12, BaseMP: 45,  MPPerLevel: 8,  STR: 7, DEX: 9, VIT: 6, AGI: 8, INT: 5, MND: 6, CHR: 6},
}

// StatsFor returns the base stats for the given job.
func StatsFor(job JobID) (Stats, error) {
	s, ok := jobStats[job]
	if !ok {
		return Stats{}, ErrUnknownJob
	}
	return s, nil
}

// HPAtLevel returns the total HP for a job at the given level (1-based).
// Level 0 returns BaseHP.
func HPAtLevel(job JobID, level int) (int, error) {
	s, err := StatsFor(job)
	if err != nil {
		return 0, err
	}
	if level < 1 {
		level = 1
	}
	return s.BaseHP + s.HPPerLevel*(level-1), nil
}

// MPAtLevel returns the total MP for a job at the given level.
// Jobs with BaseMP=0 always return 0 (melee jobs have no MP in FFXI).
func MPAtLevel(job JobID, level int) (int, error) {
	s, err := StatsFor(job)
	if err != nil {
		return 0, err
	}
	if s.BaseMP == 0 {
		return 0, nil
	}
	if level < 1 {
		level = 1
	}
	return s.BaseMP + s.MPPerLevel*(level-1), nil
}

// AllJobsCount returns the number of defined jobs.
func AllJobsCount() int { return len(AllJobs) }

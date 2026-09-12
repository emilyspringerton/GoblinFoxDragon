// Package skillchain implements FFXI-style weapon skill chaining and magic burst.
//
// # Skillchain model
//
// Each weapon skill carries a list of Resonance attributes (its "SC properties").
// When two weapon skills land in sequence within the chain window, the server
// checks every (ws1_attr, ws2_attr) pair against the combination table and returns
// the highest-tier match. No match within the window = no skillchain.
//
// Tiers and multipliers (applied to WS2 damage):
//
//	Tier 1  — 20%  (same-element closure: Liquefaction+Liquefaction, etc.)
//	Tier 2  — 35%  (cross-element: Transfixion+Liquefaction=Fusion, etc.)
//	Tier 3  — 50%  (compound: Fusion+Fragmentation=Light, etc.)
//
// # Magic Burst model
//
// After a skillchain lands the burst window opens (default 15 s).  Any spell
// whose Element matches one of the active resonance's burst elements deals a
// flat +35% bonus.  Higher-tier skillchains accept more elements (Light accepts
// Fire/Wind/Lightning/Light; Darkness accepts Earth/Ice/Water/Dark).
package skillchain

import "time"

// Resonance is a skillchain elemental property carried by weapon skills.
type Resonance int

const (
	None          Resonance = 0
	Liquefaction  Resonance = 1  // Fire    — tier 1
	Impaction     Resonance = 2  // Lightning — tier 1
	Detonation    Resonance = 3  // Wind    — tier 1
	Scission      Resonance = 4  // Earth   — tier 1
	Reverberation Resonance = 5  // Water   — tier 1
	Induration    Resonance = 6  // Ice     — tier 1
	Compression   Resonance = 7  // Dark    — tier 1
	Transfixion   Resonance = 8  // Light   — tier 1
	Fusion        Resonance = 9  // Fire+Light    — tier 2
	Fragmentation Resonance = 10 // Wind+Lightning — tier 2
	Gravitation   Resonance = 11 // Earth+Dark    — tier 2
	Distortion    Resonance = 12 // Ice+Water     — tier 2
	Light         Resonance = 13 // all-light     — tier 3
	Darkness      Resonance = 14 // all-dark      — tier 3
)

func (r Resonance) String() string {
	switch r {
	case None:
		return "None"
	case Liquefaction:
		return "Liquefaction"
	case Impaction:
		return "Impaction"
	case Detonation:
		return "Detonation"
	case Scission:
		return "Scission"
	case Reverberation:
		return "Reverberation"
	case Induration:
		return "Induration"
	case Compression:
		return "Compression"
	case Transfixion:
		return "Transfixion"
	case Fusion:
		return "Fusion"
	case Fragmentation:
		return "Fragmentation"
	case Gravitation:
		return "Gravitation"
	case Distortion:
		return "Distortion"
	case Light:
		return "Light"
	case Darkness:
		return "Darkness"
	default:
		return "Unknown"
	}
}

// Tier is the skillchain level (1 = basic, 2 = compound, 3 = ultimate).
type Tier int

const (
	Tier1 Tier = 1
	Tier2 Tier = 2
	Tier3 Tier = 3
)

// Result is the outcome of a successful skillchain closure.
type Result struct {
	Resonance  Resonance
	Tier       Tier
	Multiplier float64 // bonus damage as a fraction of WS2 damage (0.20 / 0.35 / 0.50)
}

// DefaultChainWindow is how long ws1's chain stays open for ws2 to close.
const DefaultChainWindow = 8 * time.Second

// combinationTable maps (ws1_resonance, ws2_resonance) → skillchain result.
// Source: FFXI skillchain chart.  Bidirectional pairs are listed explicitly.
var combinationTable = map[[2]Resonance]Result{
	// --- Tier 1: same-element closure ---
	{Liquefaction, Liquefaction}:   {Liquefaction, Tier1, 0.20},
	{Impaction, Impaction}:         {Impaction, Tier1, 0.20},
	{Detonation, Detonation}:       {Detonation, Tier1, 0.20},
	{Scission, Scission}:           {Scission, Tier1, 0.20},
	{Reverberation, Reverberation}: {Reverberation, Tier1, 0.20},
	{Induration, Induration}:       {Induration, Tier1, 0.20},
	{Compression, Compression}:     {Compression, Tier1, 0.20},
	{Transfixion, Transfixion}:     {Transfixion, Tier1, 0.20},

	// --- Tier 2: cross-element closure (bidirectional) ---
	{Transfixion, Liquefaction}:    {Fusion, Tier2, 0.35},
	{Liquefaction, Transfixion}:    {Fusion, Tier2, 0.35},
	{Liquefaction, Impaction}:      {Fusion, Tier2, 0.35}, // some WS carry both Liq+Imp → Fusion
	{Impaction, Detonation}:        {Fragmentation, Tier2, 0.35},
	{Detonation, Impaction}:        {Fragmentation, Tier2, 0.35},
	{Detonation, Reverberation}:    {Fragmentation, Tier2, 0.35},
	{Reverberation, Detonation}:    {Fragmentation, Tier2, 0.35},
	{Scission, Compression}:        {Gravitation, Tier2, 0.35},
	{Compression, Scission}:        {Gravitation, Tier2, 0.35},
	{Scission, Reverberation}:      {Distortion, Tier2, 0.35},
	{Reverberation, Induration}:    {Distortion, Tier2, 0.35},
	{Induration, Reverberation}:    {Distortion, Tier2, 0.35},
	{Induration, Scission}:         {Distortion, Tier2, 0.35},

	// --- Tier 3: compound closure (bidirectional) ---
	{Fusion, Fragmentation}:   {Light, Tier3, 0.50},
	{Fragmentation, Fusion}:   {Light, Tier3, 0.50},
	{Gravitation, Distortion}: {Darkness, Tier3, 0.50},
	{Distortion, Gravitation}: {Darkness, Tier3, 0.50},
}

// Chain attempts to close a skillchain between two weapon skills.
//
// ws1Attrs and ws2Attrs are the resonance attributes each weapon skill carries.
// elapsed is the time since ws1 landed; window is the open duration (use
// DefaultChainWindow unless modified by gear or abilities).
//
// When multiple attribute pairs match, the highest tier wins; within the same
// tier, the first match found is returned (order is non-deterministic — callers
// should ensure WS attribute lists are canonical).
//
// Returns (Result{}, false) if no chain forms or the window has expired.
func Chain(ws1Attrs, ws2Attrs []Resonance, elapsed, window time.Duration) (Result, bool) {
	if elapsed < 0 || elapsed > window {
		return Result{}, false
	}
	best := Result{}
	found := false
	for _, a1 := range ws1Attrs {
		for _, a2 := range ws2Attrs {
			r, ok := combinationTable[[2]Resonance{a1, a2}]
			if !ok {
				continue
			}
			if !found || r.Tier > best.Tier {
				best = r
				found = true
			}
		}
	}
	return best, found
}

// --- Magic Burst ---

// Element is a spell's elemental affinity, used for magic burst matching.
type Element int

const (
	ElemFire      Element = 1
	ElemIce       Element = 2
	ElemWind      Element = 3
	ElemEarth     Element = 4
	ElemLightning Element = 5
	ElemWater     Element = 6
	ElemLight     Element = 7
	ElemDark      Element = 8
)

// DefaultBurstWindow is how long a skillchain remains burstable after it fires.
const DefaultBurstWindow = 15 * time.Second

// BurstMultiplier is the flat bonus multiplier applied to a burst spell's damage.
const BurstMultiplier = 0.35

// resonanceBurstElements lists which spell elements can burst each resonance.
// Higher-tier resonances accept more elements (all elements of their constituents).
var resonanceBurstElements = map[Resonance][]Element{
	Liquefaction:  {ElemFire},
	Impaction:     {ElemLightning},
	Detonation:    {ElemWind},
	Scission:      {ElemEarth},
	Reverberation: {ElemWater},
	Induration:    {ElemIce},
	Compression:   {ElemDark},
	Transfixion:   {ElemLight},
	// Tier 2: union of constituent elements
	Fusion:        {ElemFire, ElemLight},
	Fragmentation: {ElemWind, ElemLightning},
	Gravitation:   {ElemEarth, ElemDark},
	Distortion:    {ElemIce, ElemWater},
	// Tier 3: all elements of all constituents
	Light:    {ElemFire, ElemWind, ElemLightning, ElemLight},
	Darkness: {ElemEarth, ElemIce, ElemWater, ElemDark},
}

// BurstElements returns the spell elements that can magic-burst a given resonance.
// Returns nil for None or unknown resonances.
func BurstElements(r Resonance) []Element {
	return resonanceBurstElements[r]
}

// CanBurst reports whether spellElem bursts the given resonance within the window.
func CanBurst(resonance Resonance, spellElem Element, elapsed, window time.Duration) bool {
	if elapsed < 0 || elapsed > window {
		return false
	}
	elems := resonanceBurstElements[resonance]
	for _, e := range elems {
		if e == spellElem {
			return true
		}
	}
	return false
}

// Burst returns the burst damage multiplier (0.0 if no burst).
// Callers multiply spell base damage by (1 + Burst(...)) for the total.
func Burst(resonance Resonance, spellElem Element, elapsed, window time.Duration) float64 {
	if CanBurst(resonance, spellElem, elapsed, window) {
		return BurstMultiplier
	}
	return 0
}

// --- Weapon skill definitions ---

// WeaponSkill defines the resonance properties of a weapon skill.
//
// WeaponType and MinSkillLevel (S412-06, founder real-time weapon-skill-leveling design burst:
// "you start at lvl 1 sword and like that also impacts accuracy and damage then when you hit you
// can level up that skill... fast blade should be the sward lvl 10 weapon skill combo is the mnk
// lvl 10 weapon skill") are real and new -- before this pass `setws` accepted any canonical name
// with zero gate, and no field said which weapon TYPE a given skill even belonged to.
// WeaponType is one of apps2/mud's own real weaponSkillLevel-package constants ("sword",
// "greatsword", "axe", "dagger", "club", "staff", "polearm", "h2h", "katana"). MinSkillLevel is
// the real per-weapon-type skill points (apps2/mud's own p.weaponSkills[WeaponType], gained on
// every landed auto-attack, fishing/mining-shaped) a player needs before they can actually FIRE
// this skill via `ws` -- `setws` itself stays ungated (pre-selecting a name you can't use yet,
// e.g. while leveling toward it, is harmless), the real gate lives at execution time.
type WeaponSkill struct {
	Name          string
	Attrs         []Resonance // SC attributes this WS carries (can carry multiple)
	WeaponType    string
	MinSkillLevel int
}

// CanonicalWeaponSkills is a representative set of DragonsNShit weapon skills
// modelled on FFXI archetypes.  The server registers additional skills at runtime.
var CanonicalWeaponSkills = map[string]WeaponSkill{
	// 1-handed sword
	"Fast Blade":      {Name: "Fast Blade", Attrs: []Resonance{Scission}, WeaponType: "sword", MinSkillLevel: 10},
	"Flat Blade":      {Name: "Flat Blade", Attrs: []Resonance{Impaction}, WeaponType: "sword", MinSkillLevel: 20},
	"Burning Blade":   {Name: "Burning Blade", Attrs: []Resonance{Liquefaction}, WeaponType: "sword", MinSkillLevel: 30},
	"Shining Blade":   {Name: "Shining Blade", Attrs: []Resonance{Transfixion}, WeaponType: "sword", MinSkillLevel: 40},
	"Circle Blade":    {Name: "Circle Blade", Attrs: []Resonance{Reverberation}, WeaponType: "sword", MinSkillLevel: 55},
	"Seraph Blade":    {Name: "Seraph Blade", Attrs: []Resonance{Transfixion, Reverberation}, WeaponType: "sword", MinSkillLevel: 70},
	"Red Lotus Blade": {Name: "Red Lotus Blade", Attrs: []Resonance{Liquefaction, Transfixion}, WeaponType: "sword", MinSkillLevel: 85},
	// Great sword
	"Hard Slash":  {Name: "Hard Slash", Attrs: []Resonance{Scission}, WeaponType: "greatsword", MinSkillLevel: 10},
	"Power Slash": {Name: "Power Slash", Attrs: []Resonance{Transfixion}, WeaponType: "greatsword", MinSkillLevel: 25},
	"Frostbite":   {Name: "Frostbite", Attrs: []Resonance{Induration, Reverberation}, WeaponType: "greatsword", MinSkillLevel: 45},
	"Freezebite":  {Name: "Freezebite", Attrs: []Resonance{Distortion}, WeaponType: "greatsword", MinSkillLevel: 65},
	// Staff (real data/items.json items: Willow Wand, Brass/Healing/Shepherd's Staff -- WHM/BLM's
	// real casting-weapon affinity; founder's own wording said "whm club," mapped onto this
	// repo's real, existing "Staff"/"Wand" item type since no separate Club-category item exists)
	"Shell Crusher": {Name: "Shell Crusher", Attrs: []Resonance{Scission}, WeaponType: "staff", MinSkillLevel: 10},
	"Rock Crusher":  {Name: "Rock Crusher", Attrs: []Resonance{Reverberation}, WeaponType: "staff", MinSkillLevel: 25},
	"Earth Crusher": {Name: "Earth Crusher", Attrs: []Resonance{Reverberation, Compression}, WeaponType: "staff", MinSkillLevel: 45},
	"Sunburst":      {Name: "Sunburst", Attrs: []Resonance{Fragmentation}, WeaponType: "staff", MinSkillLevel: 65},
	"Starburst":     {Name: "Starburst", Attrs: []Resonance{Fusion}, WeaponType: "staff", MinSkillLevel: 85},
	// Club -- a real, distinct weapon type from Staff above; no data/items.json item grants this
	// type yet (an honest, named gap, same shape as polearm's own below), kept registered so the
	// skills exist the moment a real Club item is authored.
	"Shining Strike": {Name: "Shining Strike", Attrs: []Resonance{Transfixion}, WeaponType: "club", MinSkillLevel: 10},
	"Seraph Strike":  {Name: "Seraph Strike", Attrs: []Resonance{Transfixion, Reverberation}, WeaponType: "club", MinSkillLevel: 30},
	"Black Halo":     {Name: "Black Halo", Attrs: []Resonance{Gravitation}, WeaponType: "club", MinSkillLevel: 55},
	"Judgment":       {Name: "Judgment", Attrs: []Resonance{Fusion}, WeaponType: "club", MinSkillLevel: 80},
	// Dagger (THF's real affinity weapon; founder: "thf gets wasp sting for daggers etc")
	"Wasp Sting":    {Name: "Wasp Sting", Attrs: []Resonance{Detonation}, WeaponType: "dagger", MinSkillLevel: 10},
	"Gust Slash":    {Name: "Gust Slash", Attrs: []Resonance{Detonation}, WeaponType: "dagger", MinSkillLevel: 25},
	"Cyclone":       {Name: "Cyclone", Attrs: []Resonance{Detonation, Scission}, WeaponType: "dagger", MinSkillLevel: 50},
	"Mercy Stroke":  {Name: "Mercy Stroke", Attrs: []Resonance{Compression}, WeaponType: "dagger", MinSkillLevel: 75},
	// Axe (WAR's real affinity weapon)
	"Raging Axe":    {Name: "Raging Axe", Attrs: []Resonance{Liquefaction}, WeaponType: "axe", MinSkillLevel: 10},
	"Smash Axe":     {Name: "Smash Axe", Attrs: []Resonance{Impaction}, WeaponType: "axe", MinSkillLevel: 25},
	"Gale Axe":      {Name: "Gale Axe", Attrs: []Resonance{Detonation}, WeaponType: "axe", MinSkillLevel: 50},
	"Primal Rend":   {Name: "Primal Rend", Attrs: []Resonance{Gravitation}, WeaponType: "axe", MinSkillLevel: 75},
	// Polearm (DRG's real affinity weapon) -- honest gap: no real data/items.json item has
	// weapon_type "polearm" yet (this repo has no polearm item at all today), so this type is
	// currently unreachable in live gameplay; registered here so DRG's own affinity and these
	// skills are real and ready the moment a real polearm item is authored, not silently omitted.
	"Double Thrust": {Name: "Double Thrust", Attrs: []Resonance{Scission}, WeaponType: "polearm", MinSkillLevel: 10},
	"Penta Thrust":  {Name: "Penta Thrust", Attrs: []Resonance{Scission, Impaction}, WeaponType: "polearm", MinSkillLevel: 30},
	"Vorpal Thrust": {Name: "Vorpal Thrust", Attrs: []Resonance{Induration}, WeaponType: "polearm", MinSkillLevel: 55},
	"Skewer":        {Name: "Skewer", Attrs: []Resonance{Gravitation}, WeaponType: "polearm", MinSkillLevel: 80},
	// H2H (MNK's real affinity weapon; also unarmed's real weapon type -- "when we first start
	// out we dont have a weapon so hand to hand should level up and the weapon skill should be
	// combo instead of fast blade")
	"Combo":        {Name: "Combo", Attrs: []Resonance{Impaction}, WeaponType: "h2h", MinSkillLevel: 10},
	"Backhand Blow": {Name: "Backhand Blow", Attrs: []Resonance{Compression}, WeaponType: "h2h", MinSkillLevel: 25},
	"Raging Fists":  {Name: "Raging Fists", Attrs: []Resonance{Liquefaction}, WeaponType: "h2h", MinSkillLevel: 50},
	"Asuran Fists":  {Name: "Asuran Fists", Attrs: []Resonance{Fusion}, WeaponType: "h2h", MinSkillLevel: 90},
}

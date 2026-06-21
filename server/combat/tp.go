// Package combat implements DragonsNShit melee combat mechanics.
//
// # TP (Tactical Points) system — FFXI-inspired
//
// TP accumulates per melee hit and enables weapon skill execution at ≥100 TP.
// It caps at 300 so skilled players can "store" up to 3× the WS threshold.
//
// # Delay units
//
// weaponDelay is expressed in delay-units (du), where 60 du ≈ 1 second of
// swing time.  Representative values:
//
//	Hand-to-hand  80 du  (~1.3 s)
//	1H sword     240 du  (~4.0 s)
//	1H axe       288 du  (~4.8 s)
//	Great sword  480 du  (~8.0 s)
//	Staff        366 du  (~6.1 s)
//	Polearm      402 du  (~6.7 s)
//
// # TP per hit formula
//
//	effective_delay = weaponDelay × (1 − hastePct/100)
//	tp_per_hit      = max(MinTPPerHit, floor(effective_delay / TPDivisor))
//
// Haste is capped at MaxHastePct (80 %) before applying.
// The floor is capped at MinTPPerHit (5) so gear can never reduce TP gain to 0.
//
// # Weapon skill execution
//
// UseWeaponSkill requires Current ≥ TPWSThreshold (100).
// On success TP resets to 0 (base mode; caller may implement TP-carry rules later).
package combat

// TP constants.
const (
	TPCap         = 300  // hard cap on stored TP
	TPWSThreshold = 100  // minimum TP required to execute a weapon skill
	TPDivisor     = 6    // delay-units per TP point at 0 % haste
	MinTPPerHit   = 5    // floor: minimum TP gained per hit regardless of haste
	MaxHastePct   = 80.0 // cap: haste cannot reduce effective delay below 20 %
)

// Representative weapon delay values (delay-units, 60 du ≈ 1 second).
const (
	DelayHtH        = 80  // hand-to-hand
	Delay1HSword    = 240 // one-handed sword
	Delay1HAxe      = 288 // one-handed axe
	DelayGreatSword = 480 // great sword
	DelayStaff      = 366 // staff
	DelayPolearm    = 402 // polearm
)

// TPState tracks one character's current Tactical Points.
type TPState struct {
	Current int // 0 – TPCap
}

// AddTP adds the TP gained from one melee hit and returns the new total.
//
// weaponDelay is the weapon's delay in delay-units (see constants above).
// hastePct is the character's current haste (0–MaxHastePct); values outside
// the range are clamped silently.
func (s *TPState) AddTP(weaponDelay int, hastePct float64) int {
	if hastePct < 0 {
		hastePct = 0
	}
	if hastePct > MaxHastePct {
		hastePct = MaxHastePct
	}
	effective := float64(weaponDelay) * (1.0 - hastePct/100.0)
	gained := int(effective) / TPDivisor
	if gained < MinTPPerHit {
		gained = MinTPPerHit
	}
	s.Current += gained
	if s.Current > TPCap {
		s.Current = TPCap
	}
	return s.Current
}

// CanWeaponSkill reports whether the character has enough TP to use a weapon skill.
func (s *TPState) CanWeaponSkill() bool { return s.Current >= TPWSThreshold }

// UseWeaponSkill executes a weapon skill, resetting TP to 0.
// Returns false without modifying state if Current < TPWSThreshold.
func (s *TPState) UseWeaponSkill() bool {
	if s.Current < TPWSThreshold {
		return false
	}
	s.Current = 0
	return true
}

// ConsumeTP subtracts amount from Current and returns true.
// Returns false without modifying state if Current < amount.
// Used for abilities that cost a specific TP amount other than a full WS.
func (s *TPState) ConsumeTP(amount int) bool {
	if amount < 0 || s.Current < amount {
		return false
	}
	s.Current -= amount
	return true
}

// Reset sets TP to zero (death, zone change, logout).
func (s *TPState) Reset() { s.Current = 0 }

// TPPerHit returns the TP gained by a single hit with the given weapon and haste,
// without modifying any state.  Useful for UI previews and tests.
func TPPerHit(weaponDelay int, hastePct float64) int {
	if hastePct < 0 {
		hastePct = 0
	}
	if hastePct > MaxHastePct {
		hastePct = MaxHastePct
	}
	effective := float64(weaponDelay) * (1.0 - hastePct/100.0)
	tp := int(effective) / TPDivisor
	if tp < MinTPPerHit {
		tp = MinTPPerHit
	}
	return tp
}

// HitsToWS returns the minimum number of hits required to reach TPWSThreshold
// given weaponDelay and hastePct.  Returns 1 if already above threshold.
func HitsToWS(weaponDelay int, hastePct float64) int {
	perHit := TPPerHit(weaponDelay, hastePct)
	if perHit <= 0 {
		return TPWSThreshold // guard; can't happen with MinTPPerHit
	}
	h := TPWSThreshold / perHit
	if TPWSThreshold%perHit != 0 {
		h++
	}
	return h
}

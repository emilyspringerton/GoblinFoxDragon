// Package xp implements the DragonsNShit level progression system.
//
// # Level cap + XP model (FFXI-parity, simplified)
//
// Characters progress from level 1 to 99 (level cap; 119 extension planned).
// Each level has an XP cost — the XP needed to advance from the previous level.
// Costs scale with level^1.8 (FFXI approximation), floored at 75 XP.
//
// Only the mob-tagging party receives XP; distribution across party members
// is handled by the party package (XPSplit). Non-tagger parties receive 0.
package xp

import (
	"errors"
	"math"
)

const (
	MinLevel = 1
	MaxLevel = 99 // extend to 119 in a future sprint

	xpBase     = 100.0
	xpFloor    = 75 // minimum XP cost for any level
	xpExponent = 1.8
)

var (
	ErrAtCap        = errors.New("character is already at level cap")
	ErrInvalidLevel = errors.New("level out of range [1,99]")
)

// XPToLevel returns the XP cost to advance from level (lvl-1) to lvl.
// Returns 0 for lvl == 1 (no cost; starting level).
// Returns ErrInvalidLevel for lvl < 1 or lvl > MaxLevel.
func XPToLevel(lvl int) (int, error) {
	if lvl < 1 || lvl > MaxLevel {
		return 0, ErrInvalidLevel
	}
	if lvl == 1 {
		return 0, nil
	}
	cost := int(xpBase * math.Pow(float64(lvl-1), xpExponent))
	if cost < xpFloor {
		cost = xpFloor
	}
	return cost, nil
}

// ── CharXP ────────────────────────────────────────────────────────────────────

// CharXP tracks a character's current level and XP within that level.
type CharXP struct {
	Level     int
	CurrentXP int // XP earned toward the next level
	TotalXP   int // lifetime XP earned
}

// NewCharXP creates a level-1 character with 0 XP.
func NewCharXP() *CharXP { return &CharXP{Level: MinLevel} }

// AddXP grants xpGain XP to the character.
// Returns true if one or more level-ups occurred.
// Returns ErrAtCap if the character is already at MaxLevel.
func (c *CharXP) AddXP(xpGain int) (leveledUp bool, err error) {
	if c.Level >= MaxLevel {
		return false, ErrAtCap
	}
	c.CurrentXP += xpGain
	c.TotalXP += xpGain

	for c.Level < MaxLevel {
		needed, _ := XPToLevel(c.Level + 1)
		if c.CurrentXP < needed {
			break
		}
		c.CurrentXP -= needed
		c.Level++
		leveledUp = true
	}
	if c.Level >= MaxLevel {
		c.CurrentXP = 0 // cap reached; overflow discarded
	}
	return leveledUp, nil
}

// XPToNextLevel returns XP needed to reach the next level.
// Returns 0 at MaxLevel.
func (c *CharXP) XPToNextLevel() int {
	if c.Level >= MaxLevel {
		return 0
	}
	needed, _ := XPToLevel(c.Level + 1)
	return needed
}

// Progress returns CurrentXP / XPToNextLevel as a fraction (0.0–1.0).
// Returns 1.0 at cap.
func (c *CharXP) Progress() float64 {
	needed := c.XPToNextLevel()
	if needed == 0 {
		return 1.0
	}
	return float64(c.CurrentXP) / float64(needed)
}

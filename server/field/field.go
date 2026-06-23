// Package field implements zone-specific XP bonus books (Field Manuals / Survival Guides).
//
// # Field Manual model (FFXI-parity)
//
// A Field Manual grants an XP bonus percentage for kills in its registered zone.
// Manuals expire at a wall-clock time; after expiry they grant 0 bonus.
// The caller is responsible for providing the current time.
package field

import (
	"errors"
	"time"
)

var ErrManualExpired = errors.New("field manual has expired")

// Manual is a zone-bound XP bonus book.
type Manual struct {
	ID        string
	Name      string
	SceneID   int       // zone where this manual applies
	BonusPct  int       // XP bonus percentage (e.g. 100 = double XP)
	ExpiresAt time.Time // wall-clock expiry; zero = never expires
}

// NewManual creates a Manual that expires at expiresAt.
// Pass time.Time{} (zero) for a manual that never expires.
func NewManual(id, name string, sceneID, bonusPct int, expiresAt time.Time) *Manual {
	return &Manual{
		ID:        id,
		Name:      name,
		SceneID:   sceneID,
		BonusPct:  bonusPct,
		ExpiresAt: expiresAt,
	}
}

// Active reports whether the manual is currently usable (not expired).
func (m *Manual) Active(now time.Time) bool {
	if m.ExpiresAt.IsZero() {
		return true
	}
	return now.Before(m.ExpiresAt)
}

// ApplyBonus returns the XP earned after applying this manual's bonus for a kill in sceneID.
// Returns (baseXP, nil) with bonus applied if the manual is active and matches the scene.
// Returns (baseXP, ErrManualExpired) if expired.
// Returns (baseXP, nil) with no bonus if sceneID doesn't match.
func (m *Manual) ApplyBonus(baseXP, sceneID int, now time.Time) (int, error) {
	if !m.Active(now) {
		return baseXP, ErrManualExpired
	}
	if m.SceneID != sceneID {
		return baseXP, nil
	}
	bonus := baseXP * m.BonusPct / 100
	return baseXP + bonus, nil
}

// ── Multi-manual application ──────────────────────────────────────────────────

// ApplyAll applies all active, matching manuals from a slice, stacking additively.
// Expired manuals are skipped (no error returned; they simply don't contribute).
// Only manuals whose SceneID matches sceneID are applied.
func ApplyAll(baseXP, sceneID int, now time.Time, manuals []*Manual) int {
	total := baseXP
	for _, m := range manuals {
		if !m.Active(now) || m.SceneID != sceneID {
			continue
		}
		bonus := baseXP * m.BonusPct / 100
		total += bonus
	}
	return total
}

// ── Presets ───────────────────────────────────────────────────────────────────

// MeadowSurvivalGuide returns a 30-minute Field Manual for scene 0 (Meadow) with 100% XP bonus.
func MeadowSurvivalGuide(now time.Time) *Manual {
	return NewManual("manual-meadow-survival", "Meadow Survival Guide", 0, 100, now.Add(30*time.Minute))
}

// SwampFieldManual returns a 30-minute Field Manual for scene 3 (Swampville) with 100% XP bonus.
func SwampFieldManual(now time.Time) *Manual {
	return NewManual("manual-swamp-field", "Swamp Field Manual", 3, 100, now.Add(30*time.Minute))
}

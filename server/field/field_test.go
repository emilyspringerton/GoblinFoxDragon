package field

import (
	"testing"
	"time"
)

var now = time.Now()

func activeManual(scene, bonusPct int) *Manual {
	return NewManual("test", "Test Manual", scene, bonusPct, now.Add(time.Hour))
}

func expiredManual(scene, bonusPct int) *Manual {
	return NewManual("test-exp", "Expired Manual", scene, bonusPct, now.Add(-time.Second))
}

// ── Active ────────────────────────────────────────────────────────────────────

func TestActive_WithinExpiry(t *testing.T) {
	m := activeManual(0, 100)
	if !m.Active(now) {
		t.Error("active manual should report Active=true")
	}
}

func TestActive_Expired(t *testing.T) {
	m := expiredManual(0, 100)
	if m.Active(now) {
		t.Error("expired manual should report Active=false")
	}
}

func TestActive_NeverExpires(t *testing.T) {
	m := NewManual("x", "x", 0, 100, time.Time{})
	if !m.Active(now.Add(100 * 24 * time.Hour)) {
		t.Error("zero-expiry manual should always be active")
	}
}

// ── ApplyBonus ────────────────────────────────────────────────────────────────

func TestApplyBonus_DoublesXP(t *testing.T) {
	m := activeManual(0, 100) // 100% bonus
	got, err := m.ApplyBonus(100, 0, now)
	if err != nil {
		t.Fatalf("ApplyBonus: %v", err)
	}
	if got != 200 {
		t.Errorf("100%% bonus on 100xp: got %d, want 200", got)
	}
}

func TestApplyBonus_ExpiredReturnsBase(t *testing.T) {
	m := expiredManual(0, 100)
	got, err := m.ApplyBonus(100, 0, now)
	if err != ErrManualExpired {
		t.Errorf("expired: got %v, want ErrManualExpired", err)
	}
	if got != 100 {
		t.Errorf("expired should return base XP: got %d, want 100", got)
	}
}

func TestApplyBonus_WrongScene(t *testing.T) {
	m := activeManual(1, 100) // scene 1
	got, err := m.ApplyBonus(100, 0, now) // scene 0
	if err != nil {
		t.Fatalf("wrong scene: %v", err)
	}
	if got != 100 {
		t.Errorf("wrong scene should return base XP: got %d, want 100", got)
	}
}

func TestApplyBonus_50PctBonus(t *testing.T) {
	m := activeManual(0, 50) // 50% bonus
	got, _ := m.ApplyBonus(200, 0, now)
	if got != 300 {
		t.Errorf("50%% on 200xp: got %d, want 300", got)
	}
}

func TestApplyBonus_ZeroBaseXP(t *testing.T) {
	m := activeManual(0, 100)
	got, _ := m.ApplyBonus(0, 0, now)
	if got != 0 {
		t.Errorf("0 base xp: got %d, want 0", got)
	}
}

// ── ApplyAll ──────────────────────────────────────────────────────────────────

func TestApplyAll_StacksMultiple(t *testing.T) {
	manuals := []*Manual{
		activeManual(0, 100), // +100
		activeManual(0, 50),  // +50
	}
	got := ApplyAll(100, 0, now, manuals)
	// 100 + 100 + 50 = 250
	if got != 250 {
		t.Errorf("stacked bonuses: got %d, want 250", got)
	}
}

func TestApplyAll_SkipsExpired(t *testing.T) {
	manuals := []*Manual{
		activeManual(0, 100),
		expiredManual(0, 100), // should be skipped
	}
	got := ApplyAll(100, 0, now, manuals)
	if got != 200 {
		t.Errorf("skip expired: got %d, want 200", got)
	}
}

func TestApplyAll_SkipsWrongScene(t *testing.T) {
	manuals := []*Manual{
		activeManual(1, 100), // scene 1 — skip
		activeManual(0, 50),  // scene 0 — apply
	}
	got := ApplyAll(100, 0, now, manuals)
	if got != 150 {
		t.Errorf("wrong-scene skip: got %d, want 150", got)
	}
}

func TestApplyAll_EmptyList(t *testing.T) {
	got := ApplyAll(100, 0, now, nil)
	if got != 100 {
		t.Errorf("empty: got %d, want 100", got)
	}
}

// ── Presets ───────────────────────────────────────────────────────────────────

func TestMeadowSurvivalGuide_Scene0(t *testing.T) {
	m := MeadowSurvivalGuide(now)
	if m.SceneID != 0 {
		t.Errorf("MeadowSurvivalGuide SceneID: got %d, want 0", m.SceneID)
	}
	if !m.Active(now) {
		t.Error("fresh MeadowSurvivalGuide should be active")
	}
}

func TestSwampFieldManual_Scene3(t *testing.T) {
	m := SwampFieldManual(now)
	if m.SceneID != 3 {
		t.Errorf("SwampFieldManual SceneID: got %d, want 3", m.SceneID)
	}
	if !m.Active(now) {
		t.Error("fresh SwampFieldManual should be active")
	}
}

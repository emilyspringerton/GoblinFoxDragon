package homepoint

import (
	"testing"

	"dragonsnshit/server/mob"
)

var testPos = mob.Pos{X: 10, Y: 0, Z: 5}
var testCrystal = func() { }

func newKO(xp int) *State {
	s := NewState(xp)
	s.KO()
	return s
}

// ── SetHome ───────────────────────────────────────────────────────────────────

func TestSetHome_SetsHome(t *testing.T) {
	s := NewState(500)
	s.SetHome(0, testPos)
	if !s.HasHome() {
		t.Error("HasHome should be true after SetHome")
	}
	if s.Home.SceneID != 0 {
		t.Errorf("SceneID: got %d, want 0", s.Home.SceneID)
	}
}

func TestSetHome_Replaces(t *testing.T) {
	s := NewState(500)
	s.SetHome(0, testPos)
	s.SetHome(3, mob.Pos{X: 50, Y: 1, Z: 50})
	if s.Home.SceneID != 3 {
		t.Errorf("second SetHome should override: got SceneID %d, want 3", s.Home.SceneID)
	}
}

// ── ReturnHome ────────────────────────────────────────────────────────────────

func TestReturnHome_DeductsXP(t *testing.T) {
	s := NewState(1000)
	s.SetHome(0, testPos)
	s.KO()
	_, penalty, err := s.ReturnHome()
	if err != nil {
		t.Fatalf("ReturnHome: %v", err)
	}
	expected := 1000 * DefaultXPPenaltyPct / 100 // 80
	if penalty != expected {
		t.Errorf("penalty: got %d, want %d", penalty, expected)
	}
	if s.CurrentXP != 1000-expected {
		t.Errorf("XP after return: got %d, want %d", s.CurrentXP, 1000-expected)
	}
}

func TestReturnHome_ClearsKO(t *testing.T) {
	s := newKO(500)
	s.SetHome(0, testPos)
	_, _, err := s.ReturnHome()
	if err != nil {
		t.Fatalf("ReturnHome: %v", err)
	}
	if s.IsKO {
		t.Error("IsKO should be false after ReturnHome")
	}
}

func TestReturnHome_NoHomePoint(t *testing.T) {
	s := newKO(500)
	_, _, err := s.ReturnHome()
	if err != ErrNoHomePoint {
		t.Errorf("no home: got %v, want ErrNoHomePoint", err)
	}
}

func TestReturnHome_NotKO(t *testing.T) {
	s := NewState(500)
	s.SetHome(0, testPos)
	_, _, err := s.ReturnHome()
	if err != ErrNotKO {
		t.Errorf("alive: got %v, want ErrNotKO", err)
	}
}

func TestReturnHome_ZeroXPNoPanic(t *testing.T) {
	s := newKO(0)
	s.SetHome(0, testPos)
	_, penalty, err := s.ReturnHome()
	if err != nil {
		t.Fatalf("ReturnHome 0xp: %v", err)
	}
	if penalty != 0 {
		t.Errorf("0xp penalty should be 0: got %d", penalty)
	}
	if s.CurrentXP != 0 {
		t.Errorf("XP after 0xp return: got %d, want 0", s.CurrentXP)
	}
}

func TestReturnHome_ReturnsCrystalLocation(t *testing.T) {
	s := newKO(500)
	s.SetHome(2, testPos)
	crystal, _, err := s.ReturnHome()
	if err != nil {
		t.Fatalf("ReturnHome: %v", err)
	}
	if crystal.SceneID != 2 || crystal.Pos != testPos {
		t.Errorf("crystal: got sceneID=%d pos=%v, want sceneID=2 pos=%v", crystal.SceneID, crystal.Pos, testPos)
	}
}

func TestReturnHome_IdempotentKO(t *testing.T) {
	s := NewState(1000)
	s.SetHome(0, testPos)
	s.KO()
	_, _, _ = s.ReturnHome()
	// calling again after Return while alive should error
	s.KO()
	s.SetHome(1, testPos)
	_, penalty, err := s.ReturnHome()
	if err != nil {
		t.Fatalf("second return: %v", err)
	}
	// XP after first return: 1000-80=920; after second: 920-73=847
	if penalty == 0 && s.CurrentXP != 920 {
		t.Errorf("second return XP: got %d, want 847 approx", s.CurrentXP)
	}
}

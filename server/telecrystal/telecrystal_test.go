package telecrystal

import (
	"errors"
	"testing"
)

func TestRegistry_SixCrystals(t *testing.T) {
	if len(Registry) != 6 {
		t.Errorf("Registry: got %d, want 6", len(Registry))
	}
}

func TestLookup_Found(t *testing.T) {
	c, err := Lookup("TELECRYSTAL_ID_TOWN_TO_MINES")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if c.TargetScene != SceneCaves {
		t.Errorf("target scene: got %d, want %d (SceneCaves)", c.TargetScene, SceneCaves)
	}
}

func TestLookup_NotFound(t *testing.T) {
	_, err := Lookup("bogus-id")
	if !errors.Is(err, ErrUnknownCrystal) {
		t.Errorf("unknown: got %v, want ErrUnknownCrystal", err)
	}
}

func TestInScene_Meadow(t *testing.T) {
	cs := InScene(SceneMeadow)
	if len(cs) != 2 {
		t.Errorf("meadow crystals: got %d, want 2", len(cs))
	}
}

func TestInScene_Caves(t *testing.T) {
	cs := InScene(SceneCaves)
	if len(cs) != 3 {
		t.Errorf("caves crystals: got %d, want 3", len(cs))
	}
}

func TestInScene_Swamp(t *testing.T) {
	cs := InScene(SceneSwamp)
	if len(cs) != 1 {
		t.Errorf("swamp crystals: got %d, want 1", len(cs))
	}
}

func TestInScene_Hills_Empty(t *testing.T) {
	cs := InScene(SceneHills)
	if len(cs) != 0 {
		t.Errorf("hills crystals: got %d, want 0", len(cs))
	}
}

func TestInRange_InsideRadius(t *testing.T) {
	c, _ := Lookup("TELECRYSTAL_ID_TOWN_TO_MINES")
	// crystal at (270,0,60) radius 14 — player at (272,0,58) dist≈2.8
	if !c.InRange(Vec3{272, 0, 58}) {
		t.Error("InRange: inside radius returned false")
	}
}

func TestInRange_OutsideRadius(t *testing.T) {
	c, _ := Lookup("TELECRYSTAL_ID_TOWN_TO_MINES")
	// crystal at (270,0,60) radius 14 — player at (290,0,60) dist=20
	if c.InRange(Vec3{290, 0, 60}) {
		t.Error("InRange: outside radius returned true")
	}
}

func TestInRange_ExactlyOnEdge(t *testing.T) {
	c, _ := Lookup("TELECRYSTAL_ID_TOWN_TO_MINES")
	// radius=14, move exactly 14 units in X
	edge := Vec3{270 + 14, 0, 60}
	if !c.InRange(edge) {
		t.Error("InRange: exactly on edge (=radius) should be in range")
	}
}

func TestValidate_Valid(t *testing.T) {
	c, err := Validate("TELECRYSTAL_ID_TOWN_TO_MINES", SceneMeadow, Vec3{270, 0, 60}, 2000)
	if err != nil {
		t.Fatalf("Validate valid: %v", err)
	}
	if c.CastCost != 1000 {
		t.Errorf("CastCost: got %d, want 1000", c.CastCost)
	}
}

func TestValidate_WrongScene(t *testing.T) {
	_, err := Validate("TELECRYSTAL_ID_TOWN_TO_MINES", SceneCaves, Vec3{}, 2000)
	if !errors.Is(err, ErrWrongScene) {
		t.Errorf("wrong scene: got %v, want ErrWrongScene", err)
	}
}

func TestValidate_InsufficientGold(t *testing.T) {
	_, err := Validate("TELECRYSTAL_ID_TOWN_TO_MINES", SceneMeadow, Vec3{}, 500)
	if !errors.Is(err, ErrInsufficientGold) {
		t.Errorf("insufficient gold: got %v, want ErrInsufficientGold", err)
	}
}

func TestValidate_UnknownCrystal(t *testing.T) {
	_, err := Validate("bad-id", SceneMeadow, Vec3{}, 9999)
	if !errors.Is(err, ErrUnknownCrystal) {
		t.Errorf("unknown crystal: got %v, want ErrUnknownCrystal", err)
	}
}

func TestDist2D_Zero(t *testing.T) {
	c, _ := Lookup("TELECRYSTAL_ID_TOWN_TO_MINES")
	d := c.Dist2D(c.Position)
	if d > 0.001 {
		t.Errorf("Dist2D same pos: got %f, want ~0", d)
	}
}

func TestDist2D_Known(t *testing.T) {
	c, _ := Lookup("TELECRYSTAL_ID_TOWN_TO_MINES") // at (270,0,60)
	d := c.Dist2D(Vec3{270, 5, 57}) // dx=0 dz=-3 → d=3 (Y ignored)
	if d < 2.9 || d > 3.1 {
		t.Errorf("Dist2D: got %f, want ~3.0", d)
	}
}

func TestCrystals_AllIDs_Unique(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Registry {
		if seen[c.ID] {
			t.Errorf("duplicate crystal ID: %q", c.ID)
		}
		seen[c.ID] = true
	}
}

func TestCrystals_AllHavePrompt(t *testing.T) {
	for _, c := range Registry {
		if c.Prompt == "" {
			t.Errorf("crystal %q has empty Prompt", c.ID)
		}
	}
}

func TestCrystals_AllHavePositiveCost(t *testing.T) {
	for _, c := range Registry {
		if c.CastCost <= 0 {
			t.Errorf("crystal %q has non-positive CastCost %d", c.ID, c.CastCost)
		}
	}
}

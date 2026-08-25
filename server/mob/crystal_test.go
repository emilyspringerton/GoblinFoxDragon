package mob

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTestSeed(t *testing.T, seed crystalSeed) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "crystal-seed-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(seed); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func TestMeadowCrystalSpawns_EmptyPath(t *testing.T) {
	mobs, err := MeadowCrystalSpawns("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mobs != nil {
		t.Errorf("got %d mobs, want nil for empty path", len(mobs))
	}
}

func TestMeadowCrystalSpawns_NonexistentFile(t *testing.T) {
	mobs, err := MeadowCrystalSpawns(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err != nil {
		t.Fatalf("unexpected error for missing file (should be a soft no-op): %v", err)
	}
	if mobs != nil {
		t.Errorf("got %d mobs, want nil for missing file", len(mobs))
	}
}

func TestMeadowCrystalSpawns_MalformedFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "bad-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("not valid json")
	f.Close()

	_, err = MeadowCrystalSpawns(f.Name())
	if err == nil {
		t.Error("expected a real error for malformed JSON, got nil")
	}
}

func TestMeadowCrystalSpawns_RealSeed(t *testing.T) {
	seed := crystalSeed{
		Tick:   100,
		Width:  44,
		Height: 44,
		Goblins: []crystalNPC{
			{X: 0, Y: 0, Kind: "raider"},
			{X: 43, Y: 43, Kind: "scavenger"},
			{X: 21, Y: 21, Kind: "unknown-kind"}, // must fall back gracefully
		},
		Foxes: []crystalNPC{
			{X: 10, Y: 10, Kind: "courier"},
		},
	}
	path := writeTestSeed(t, seed)

	mobs, err := MeadowCrystalSpawns(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mobs) != 4 {
		t.Fatalf("got %d mobs, want 4 (3 goblins + 1 fox)", len(mobs))
	}

	// All spawned into the Meadow (zone 0).
	for _, m := range mobs {
		if m.SceneID != 0 {
			t.Errorf("mob %s SceneID=%d, want 0 (Meadow)", m.ID, m.SceneID)
		}
	}

	// Goblin (0,0) should map to the grid's negative corner, and (43,43) to
	// the positive corner -- confirms the coordinate mapping isn't a no-op
	// or a constant, and that it's centered (not all-positive/all-negative).
	if mobs[0].Pos.X >= 0 || mobs[0].Pos.Z >= 0 {
		t.Errorf("grid (0,0) mapped to (%.1f,%.1f), want both negative", mobs[0].Pos.X, mobs[0].Pos.Z)
	}
	if mobs[1].Pos.X <= 0 || mobs[1].Pos.Z <= 0 {
		t.Errorf("grid (43,43) mapped to (%.1f,%.1f), want both positive", mobs[1].Pos.X, mobs[1].Pos.Z)
	}

	// Raider gets real combat stats (aggro range > 0); unknown kind falls
	// back to scavenger's (non-aggressive) stats rather than zero-value/crash.
	if mobs[0].Kind != "goblin-raider" || mobs[0].AggroRange <= 0 {
		t.Errorf("raider: Kind=%q AggroRange=%.1f, want goblin-raider with AggroRange>0", mobs[0].Kind, mobs[0].AggroRange)
	}
	if mobs[2].Kind != "goblin-unknown-kind" || mobs[2].HP != goblinStatsByKind["scavenger"].hp {
		t.Errorf("unknown-kind goblin didn't fall back to scavenger stats: Kind=%q HP=%d", mobs[2].Kind, mobs[2].HP)
	}

	// Fox is non-aggressive wildlife flavor.
	if mobs[3].Kind != "fox-courier" || mobs[3].AggroRange != 0 {
		t.Errorf("fox: Kind=%q AggroRange=%.1f, want fox-courier with AggroRange=0", mobs[3].Kind, mobs[3].AggroRange)
	}

	// IDs are unique (no collisions between the two goblins with the same
	// index-adjacent pattern, or between goblin/fox ID namespaces).
	seen := map[string]bool{}
	for _, m := range mobs {
		if seen[m.ID] {
			t.Errorf("duplicate mob ID: %s", m.ID)
		}
		seen[m.ID] = true
	}
}

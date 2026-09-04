package zone

import "testing"

func TestCellFor_OriginIsRoughlyCentered(t *testing.T) {
	cell, err := CellFor(0, Pos{X: 0, Y: 2, Z: 0})
	if err != nil {
		t.Fatalf("CellFor: %v", err)
	}
	// Meadow spans -40..40 in 10-unit cells (8 columns: A-H), so the origin (0,0) lands in
	// column index 4 (E) row index 4 (row 5).
	if cell != "E-5" {
		t.Fatalf("expected origin to land in E-5, got %q", cell)
	}
}

func TestCellFor_RealMeadowWormPosition(t *testing.T) {
	// server/mob/worm.go's own MeadowWormSpawns places a real worm at (35, 2, 0).
	cell, err := CellFor(0, Pos{X: 35, Y: 2, Z: 0})
	if err != nil {
		t.Fatalf("CellFor: %v", err)
	}
	// Column: (35 - (-40)) / 10 = 7.5 -> index 7 -> "H". Row: (0 - (-40))/10 = 4 -> row 5.
	if cell != "H-5" {
		t.Fatalf("expected H-5 for the real east-edge worm, got %q", cell)
	}
}

func TestCellFor_UnknownZone_Errors(t *testing.T) {
	if _, err := CellFor(999, Pos{}); err == nil {
		t.Fatal("expected an error for an unregistered zone")
	}
}

func TestCenterOf_IsCellFors_RealInverse(t *testing.T) {
	for _, zoneID := range []int{0, 1, 2, 3, 4} {
		for _, cell := range []string{"A-1", "E-5", "H-8"} {
			center, err := CenterOf(zoneID, cell)
			if err != nil {
				// Not every cell exists in every zone's own real extent -- that's fine, this
				// test only checks the round trip for cells that DO resolve.
				continue
			}
			roundTrip, err := CellFor(zoneID, center)
			if err != nil {
				t.Fatalf("zone %d cell %s: CellFor on its own center errored: %v", zoneID, cell, err)
			}
			if roundTrip != cell {
				t.Fatalf("zone %d: CenterOf(%s) -> CellFor round-trip gave %s, not the original cell", zoneID, cell, roundTrip)
			}
		}
	}
}

func TestCenterOf_InvalidCellFormat_Errors(t *testing.T) {
	if _, err := CenterOf(0, "not-a-cell-7"); err == nil {
		t.Fatal("expected an error for a malformed cell")
	}
	if _, err := CenterOf(0, "I0"); err == nil {
		t.Fatal("expected an error for a cell missing the separator")
	}
}

func TestColumnLetter_MultiCharacterBeyondZ(t *testing.T) {
	// Real, honest edge case: no zone today is wide enough to need this, but the scheme should
	// still be correct if one ever is (same base-26 rule spreadsheet columns use).
	if got := columnLetter(26); got != "AA" {
		t.Fatalf("columnLetter(26) = %q, want AA", got)
	}
	idx, err := columnIndex("AA")
	if err != nil || idx != 26 {
		t.Fatalf("columnIndex(AA) = %d, %v, want 26, nil", idx, err)
	}
}

func TestCellFor_RealHillsWolfPosition(t *testing.T) {
	// server/mob/hills.go's own HillsSpawns places a real wolf at (42, 11, 0) -- right at the
	// registered bound's own edge.
	cell, err := CellFor(1, Pos{X: 42, Y: 11, Z: 0})
	if err != nil {
		t.Fatalf("CellFor: %v", err)
	}
	if cell == "" {
		t.Fatal("expected a real cell label even at the zone's own outer edge")
	}
}

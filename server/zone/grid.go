package zone

import (
	"fmt"
	"strconv"
	"strings"

	"dragonsnshit/server/mob"
)

// Pos is an alias for mob.Pos -- this package's own real world-position type, reused here
// rather than duplicated (checked directly: server/mob does not import server/zone, so this
// import direction is safe).
type Pos = mob.Pos

// grid.go — Phase 1 of MOB_SPAWN_NORTHSTAR.md (kanban GFD-MOBSPAWN-001, founder real-time: "if
// you split the map up into a grid you can have a coordinates system like I-7 like ffxi uses").
// Gives every zone a real bounding box on the X/Z plane and derives an FFXI-style letter-number
// cell label ("I-7") from any world position, plus the inverse (a cell's center point, for
// spawning). Y (height) is deliberately not part of the grid -- FFXI's own real map grid is
// purely 2-D, and every zone here is built on a flat-ish ground plane already (checked directly
// against the Y values baked into every *Spawns() function across server/mob).

// GridCellSize is the width/depth of one grid cell, in the same world units every mob spawn
// position already uses. Chosen from the real zone extents below: Meadow/Hills/Swampville all
// span roughly ±35-42 units, so a 10-unit cell gives a real, FFXI-scale grid (roughly 7-9 cells
// per axis) rather than either a single giant cell or hundreds of tiny ones.
const GridCellSize = 10.0

// ZoneBounds is a zone's real X/Z extent, used to derive its grid. Values below are grounded in
// the actual spawn positions baked into server/mob's own *Spawns() functions (checked directly,
// not guessed), padded slightly so a mob at the literal edge of its ring still gets a real cell.
type ZoneBounds struct {
	MinX, MaxX float64
	MinZ, MaxZ float64
}

// zoneBoundsByID mirrors DefaultZones' own real zone roster. New Handington (zone 4) is a real,
// honest approximation -- no explicit town-bounds constant exists anywhere in
// apps2/battlegrounds_gui today (checked directly), only the Worm Hut's own position (10, 30)
// and the founder's own "doubled the size of the town" note -- padded generously rather than
// guessed tight, since an oversized bound just means a few unused outer cells, not a bug.
var zoneBoundsByID = map[int]ZoneBounds{
	0: {MinX: -40, MaxX: 40, MinZ: -40, MaxZ: 40},  // Meadow -- MeadowWormSpawns spans ±35
	1: {MinX: -45, MaxX: 45, MinZ: -45, MaxZ: 45},  // Hills -- HillsSpawns spans ±42
	2: {MinX: -25, MaxX: 25, MinZ: -55, MaxZ: 20},  // Caves -- CavesSpawns spans X ±20, Z +18..-50 (narrows away from the entrance)
	3: {MinX: -45, MaxX: 45, MinZ: -45, MaxZ: 45},  // Swampville -- SwampvilleSpawns spans ±42
	4: {MinX: -60, MaxX: 60, MinZ: -60, MaxZ: 60},  // New Handington -- approximate, see doc comment above
}

// BoundsFor returns the real bounding box for a zone, and whether one is registered.
func BoundsFor(zoneID int) (ZoneBounds, bool) {
	b, ok := zoneBoundsByID[zoneID]
	return b, ok
}

// columnLetter renders a 0-indexed column as FFXI-style letters: A, B, ... Z, AA, AB, ...
// (the same base-26 scheme spreadsheet columns use, since FFXI's own real grids never exceed Z
// but this stays correct if a future zone's bounds ever did).
func columnLetter(col int) string {
	if col < 0 {
		col = 0
	}
	s := ""
	for {
		s = string(rune('A'+col%26)) + s
		col = col/26 - 1
		if col < 0 {
			break
		}
	}
	return s
}

// columnIndex is columnLetter's inverse.
func columnIndex(letters string) (int, error) {
	letters = strings.ToUpper(letters)
	col := 0
	for _, c := range letters {
		if c < 'A' || c > 'Z' {
			return 0, fmt.Errorf("zone: invalid column letters %q", letters)
		}
		col = col*26 + int(c-'A') + 1
	}
	return col - 1, nil
}

// CellFor derives the FFXI-style grid cell label ("I-7") for a world position within a zone.
// A position outside the zone's registered bounds still gets a real (if out-of-range) label
// rather than an error -- clamping would silently hide a mob spawned outside its own zone's
// expected extent, a real bug worth surfacing instead.
func CellFor(zoneID int, pos Pos) (string, error) {
	b, ok := BoundsFor(zoneID)
	if !ok {
		return "", fmt.Errorf("zone: no bounds registered for zone %d", zoneID)
	}
	col := int((pos.X - b.MinX) / GridCellSize)
	row := int((pos.Z - b.MinZ) / GridCellSize)
	return fmt.Sprintf("%s-%d", columnLetter(col), row+1), nil
}

// CenterOf is CellFor's inverse: the real world position at the center of a named grid cell,
// e.g. for a spawn-table row that names "I-7" instead of a raw X/Z. Y is always 0 -- callers own
// their own ground-height lookup, same as every *Spawns() constructor already does today.
func CenterOf(zoneID int, cell string) (Pos, error) {
	b, ok := BoundsFor(zoneID)
	if !ok {
		return Pos{}, fmt.Errorf("zone: no bounds registered for zone %d", zoneID)
	}
	parts := strings.SplitN(cell, "-", 2)
	if len(parts) != 2 {
		return Pos{}, fmt.Errorf("zone: invalid cell %q, want LETTER-NUMBER (e.g. I-7)", cell)
	}
	col, err := columnIndex(parts[0])
	if err != nil {
		return Pos{}, err
	}
	row, err := strconv.Atoi(parts[1])
	if err != nil || row < 1 {
		return Pos{}, fmt.Errorf("zone: invalid cell row %q, want a positive integer", parts[1])
	}
	return Pos{
		X: b.MinX + (float64(col)+0.5)*GridCellSize,
		Y: 0,
		Z: b.MinZ + (float64(row-1)+0.5)*GridCellSize,
	}, nil
}

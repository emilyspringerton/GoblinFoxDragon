// Package cartography implements player zone map discovery for DragonsNShit MUD.
//
// As players explore zones, their Atlas records which zones they've visited.
// The ExitMap function builds an ASCII map showing visited zones by name and
// unvisited connected zones as "???".
package cartography

import (
	"fmt"
	"sort"
	"strings"
)

// Atlas tracks the zones a player has visited.
type Atlas struct {
	visited map[int]bool
}

// NewAtlas creates an empty Atlas.
func NewAtlas() *Atlas {
	return &Atlas{visited: make(map[int]bool)}
}

// Visit marks zoneID as visited. Idempotent.
func (a *Atlas) Visit(zoneID int) {
	a.visited[zoneID] = true
}

// Has reports whether the player has visited zoneID.
func (a *Atlas) Has(zoneID int) bool {
	return a.visited[zoneID]
}

// KnownZones returns the sorted list of visited zone IDs.
func (a *Atlas) KnownZones() []int {
	out := make([]int, 0, len(a.visited))
	for id := range a.visited {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}

// Count returns the number of visited zones.
func (a *Atlas) Count() int {
	return len(a.visited)
}

// ExitMap builds a compact ASCII map of the player's known world.
// exits: the full world exit table (zoneID → direction → zoneID).
// zoneNames: optional human-readable name for each zone ID; omitted zones use "Zone N".
// current: the player's current zone ID (shown with a marker).
//
// Output format (example with 4 zones, visited zones 0+1):
//
//	┌────────────────────────────────┐
//	│  ZONE MAP                      │
//	│  Meadow [0] ← YOU ARE HERE     │
//	│   north → Hills [1]            │
//	│   south → ??? [2]              │
//	│   east  → ??? [3]              │
//	│  Hills [1]                     │
//	│   south → Meadow [0]           │
//	└────────────────────────────────┘
func (a *Atlas) ExitMap(exits map[int]map[string]int, zoneNames map[int]string, current int) string {
	if len(a.visited) == 0 {
		return "(no zones mapped yet)"
	}
	name := func(id int) string {
		if n, ok := zoneNames[id]; ok && n != "" {
			return n
		}
		return fmt.Sprintf("Zone%d", id)
	}
	// Collect directions sorted for stable output.
	sortedDirs := func(m map[string]int) []string {
		dirs := make([]string, 0, len(m))
		for d := range m {
			dirs = append(dirs, d)
		}
		sort.Strings(dirs)
		return dirs
	}

	var sb strings.Builder
	sb.WriteString("┌──────────────────────────────────────┐\r\n")
	sb.WriteString("│  ZONE MAP                            │\r\n")

	zones := a.KnownZones()
	for _, zid := range zones {
		hereMarker := ""
		if zid == current {
			hereMarker = " ← YOU ARE HERE"
		}
		sb.WriteString(fmt.Sprintf("│  %s [%d]%s\r\n", name(zid), zid, hereMarker))
		if exs, ok := exits[zid]; ok {
			for _, dir := range sortedDirs(exs) {
				dest := exs[dir]
				destLabel := "???"
				if a.Has(dest) {
					destLabel = name(dest)
				}
				sb.WriteString(fmt.Sprintf("│   %-6s → %s [%d]\r\n", dir, destLabel, dest))
			}
		}
	}
	sb.WriteString("└──────────────────────────────────────┘")
	return sb.String()
}

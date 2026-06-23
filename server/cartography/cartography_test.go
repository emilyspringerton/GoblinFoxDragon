package cartography

import (
	"strings"
	"testing"
)

var testExits = map[int]map[string]int{
	0: {"north": 1, "south": 2, "east": 3},
	1: {"south": 0},
	2: {"north": 0},
	3: {"west": 0},
}

var testNames = map[int]string{
	0: "Meadow",
	1: "Hills",
	2: "Caves",
	3: "Swamp",
}

func TestNewAtlasEmpty(t *testing.T) {
	a := NewAtlas()
	if a.Count() != 0 {
		t.Errorf("new atlas should be empty, got count %d", a.Count())
	}
}

func TestVisitAndHas(t *testing.T) {
	a := NewAtlas()
	a.Visit(0)
	if !a.Has(0) {
		t.Error("zone 0 should be visited")
	}
	if a.Has(1) {
		t.Error("zone 1 should not be visited")
	}
}

func TestVisitIdempotent(t *testing.T) {
	a := NewAtlas()
	a.Visit(0)
	a.Visit(0)
	if a.Count() != 1 {
		t.Errorf("double visit should count as 1, got %d", a.Count())
	}
}

func TestKnownZonesSorted(t *testing.T) {
	a := NewAtlas()
	a.Visit(3)
	a.Visit(1)
	a.Visit(0)
	zones := a.KnownZones()
	if len(zones) != 3 {
		t.Fatalf("expected 3 zones, got %d", len(zones))
	}
	if zones[0] != 0 || zones[1] != 1 || zones[2] != 3 {
		t.Errorf("expected [0,1,3] got %v", zones)
	}
}

func TestCountMatches(t *testing.T) {
	a := NewAtlas()
	a.Visit(0)
	a.Visit(1)
	a.Visit(5)
	if a.Count() != 3 {
		t.Errorf("expected 3, got %d", a.Count())
	}
}

func TestExitMapEmpty(t *testing.T) {
	a := NewAtlas()
	out := a.ExitMap(testExits, testNames, 0)
	if out != "(no zones mapped yet)" {
		t.Errorf("expected no-zones message, got %q", out)
	}
}

func TestExitMapContainsCurrentZone(t *testing.T) {
	a := NewAtlas()
	a.Visit(0)
	out := a.ExitMap(testExits, testNames, 0)
	if !strings.Contains(out, "YOU ARE HERE") {
		t.Error("map should show YOU ARE HERE for current zone")
	}
	if !strings.Contains(out, "Meadow") {
		t.Error("map should show current zone name")
	}
}

func TestExitMapShowsUnknownAsQM(t *testing.T) {
	a := NewAtlas()
	a.Visit(0) // knows Meadow, not Hills/Caves/Swamp
	out := a.ExitMap(testExits, testNames, 0)
	if !strings.Contains(out, "???") {
		t.Error("map should show ??? for unvisited connected zones")
	}
}

func TestExitMapShowsKnownDestination(t *testing.T) {
	a := NewAtlas()
	a.Visit(0)
	a.Visit(1) // Hills now known
	out := a.ExitMap(testExits, testNames, 0)
	if !strings.Contains(out, "Hills") {
		t.Error("known destination Hills should appear in map")
	}
}

func TestExitMapNoCurrentMarkerForOtherZone(t *testing.T) {
	a := NewAtlas()
	a.Visit(0)
	a.Visit(1)
	out := a.ExitMap(testExits, testNames, 1) // current = Hills
	lines := strings.Split(out, "\n")
	for _, l := range lines {
		if strings.Contains(l, "Meadow") && strings.Contains(l, "YOU ARE HERE") {
			t.Error("Meadow should not be marked as current when player is in Hills")
		}
	}
}

func TestExitMapFallbackZoneName(t *testing.T) {
	a := NewAtlas()
	a.Visit(99)
	out := a.ExitMap(map[int]map[string]int{99: {}}, map[int]string{}, 99)
	if !strings.Contains(out, "Zone99") {
		t.Errorf("unknown zone should show as Zone99, got:\n%s", out)
	}
}

// Package conquest implements the DragonsNShit region-control / outpost system.
//
// # Conquest model (FFXI-parity)
//
// The world is divided into Regions, each controlled by one of three nations
// (San d'Oria, Bastok, Windurst — or "neutral" until points accumulate).
// Nations earn points by killing monsters or completing objectives in a region.
// At the weekly tick the nation with the most points takes control.
// Ties are resolved in favour of the incumbent controller (no change).
//
// Points reset to zero after each weekly evaluation.
package conquest

import "errors"

// Nation identifies a player nation / faction.
type Nation int

const (
	NationNeutral Nation = 0
	NationSandoria Nation = 1
	NationBastok   Nation = 2
	NationWindurst Nation = 3
)

var nationNames = map[Nation]string{
	NationNeutral:  "Neutral",
	NationSandoria: "San d'Oria",
	NationBastok:   "Bastok",
	NationWindurst: "Windurst",
}

// NationName returns the display name for a nation.
func NationName(n Nation) string {
	if s, ok := nationNames[n]; ok {
		return s
	}
	return "Unknown"
}

// AllNations returns the three playable nations.
func AllNations() []Nation { return []Nation{NationSandoria, NationBastok, NationWindurst} }

var (
	ErrUnknownRegion = errors.New("unknown region")
	ErrUnknownNation = errors.New("unknown nation")
)

// Region is a conquest-eligible area of the world.
// Not goroutine-safe; callers synchronise via the server tick loop.
type Region struct {
	ID         int
	Name       string
	Controller Nation            // current controlling nation (NationNeutral if unclaimed)
	Points     map[Nation]int    // accumulated points since last Tick
}

// NewRegion creates a Region with no controller and zeroed points.
func NewRegion(id int, name string) *Region {
	return &Region{
		ID:         id,
		Name:       name,
		Controller: NationNeutral,
		Points:     map[Nation]int{NationSandoria: 0, NationBastok: 0, NationWindurst: 0},
	}
}

// AddPoints adds delta points for the given nation in this region.
// delta may be negative (e.g. conquest point drain from events).
// Points are clamped at 0 on the low end.
func (r *Region) AddPoints(nation Nation, delta int) error {
	if _, ok := nationNames[nation]; !ok {
		return ErrUnknownNation
	}
	r.Points[nation] += delta
	if r.Points[nation] < 0 {
		r.Points[nation] = 0
	}
	return nil
}

// Tick evaluates weekly conquest for this region:
//   - The nation with the most points takes control.
//   - Ties preserve the current Controller (incumbent wins).
//   - Points are reset to zero after evaluation.
//
// Returns the new Controller (may be unchanged).
func (r *Region) Tick() Nation {
	// Find maximum points earned by any nation.
	maxPts := 0
	for _, n := range AllNations() {
		if r.Points[n] > maxPts {
			maxPts = r.Points[n]
		}
	}

	resetPts := func() {
		for n := range r.Points {
			r.Points[n] = 0
		}
	}

	// No one scored — keep incumbent.
	if maxPts == 0 {
		resetPts()
		return r.Controller
	}

	// Incumbent retains if tied for max (FFXI parity: no change on tie).
	if r.Controller != NationNeutral && r.Points[r.Controller] == maxPts {
		resetPts()
		return r.Controller
	}

	// First nation in deterministic order (Sandoria→Bastok→Windurst) with max pts wins.
	for _, n := range AllNations() {
		if r.Points[n] == maxPts {
			r.Controller = n
			resetPts()
			return n
		}
	}

	// unreachable
	return r.Controller
}

// ── Conquest Map ──────────────────────────────────────────────────────────────

// Map holds all conquest regions.
// Not goroutine-safe.
type Map struct {
	regions map[int]*Region
}

// NewMap returns an empty conquest map.
func NewMap() *Map { return &Map{regions: make(map[int]*Region)} }

// AddRegion registers a region. Panics on duplicate ID.
func (m *Map) AddRegion(r *Region) {
	if _, exists := m.regions[r.ID]; exists {
		panic("duplicate region ID")
	}
	m.regions[r.ID] = r
}

// Get returns a region by ID.
func (m *Map) Get(id int) (*Region, bool) {
	r, ok := m.regions[id]
	return r, ok
}

// AddPoints adds points to the named region for nation.
func (m *Map) AddPoints(regionID int, nation Nation, delta int) error {
	r, ok := m.regions[regionID]
	if !ok {
		return ErrUnknownRegion
	}
	return r.AddPoints(nation, delta)
}

// TickAll runs weekly conquest evaluation across all regions.
// Returns a map of regionID → new Controller.
func (m *Map) TickAll() map[int]Nation {
	result := make(map[int]Nation, len(m.regions))
	for id, r := range m.regions {
		result[id] = r.Tick()
	}
	return result
}

// Scoreboard returns point totals per nation across all regions (for display).
func (m *Map) Scoreboard() map[Nation]int {
	totals := map[Nation]int{NationSandoria: 0, NationBastok: 0, NationWindurst: 0}
	for _, r := range m.regions {
		for n, pts := range r.Points {
			totals[n] += pts
		}
	}
	return totals
}

// RegionCount returns the number of regions controlled by each nation.
func (m *Map) RegionCount() map[Nation]int {
	counts := map[Nation]int{}
	for _, r := range m.regions {
		counts[r.Controller]++
	}
	return counts
}

// DefaultRegions returns the starting conquest regions matching the GFD zone IDs.
func DefaultRegions() []*Region {
	return []*Region{
		NewRegion(0, "Ronfaure"),  // Meadow zone
		NewRegion(1, "Gustaberg"), // Hills zone
		NewRegion(2, "Jugner"),    // Caves zone
		NewRegion(3, "Sarutabaruta"), // Swampville zone
	}
}

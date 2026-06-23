// Package fame implements FFXI-parity nation reputation tracking for DragonsNShit.
//
// Players earn fame points in each nation by completing quests and other activities.
// Fame points accumulate into a Rank (0–5) that gates advanced quest content.
//
// Thresholds (points → rank):
//
//	0  → Rank 0 (Unknown)
//	50 → Rank 1 (Known)
//	200 → Rank 2 (Liked)
//	500 → Rank 3 (Trusted)
//	1000 → Rank 4 (Admired)
//	2000 → Rank 5 (Revered)
package fame

import "errors"

// Nation identifies one of the three city-states.
type Nation int

const (
	Neutral  Nation = iota
	Sandoria        // San d'Oria (elven kingdom)
	Bastok          // Bastok (industrial republic)
	Windurst        // Windurst (tarutaru federation)
)

var nationNames = map[Nation]string{
	Neutral:  "Neutral",
	Sandoria: "San d'Oria",
	Bastok:   "Bastok",
	Windurst: "Windurst",
}

// NationName returns the display name for a Nation.
func NationName(n Nation) string {
	if name, ok := nationNames[n]; ok {
		return name
	}
	return "Unknown"
}

// AllNations returns the three playable city-state nations in canonical order.
func AllNations() []Nation {
	return []Nation{Sandoria, Bastok, Windurst}
}

// rankThresholds maps minimum points to rank tier.
var rankThresholds = []struct {
	points int
	rank   int
	label  string
}{
	{2000, 5, "Revered"},
	{1000, 4, "Admired"},
	{500, 3, "Trusted"},
	{200, 2, "Liked"},
	{50, 1, "Known"},
	{0, 0, "Unknown"},
}

// Rank returns the fame rank (0–5) for the given point total.
func Rank(points int) int {
	for _, t := range rankThresholds {
		if points >= t.points {
			return t.rank
		}
	}
	return 0
}

// RankLabel returns the display label for a fame rank.
func RankLabel(rank int) string {
	for _, t := range rankThresholds {
		if t.rank == rank {
			return t.label
		}
	}
	return "Unknown"
}

// NextRankThreshold returns the minimum points needed for the next rank above current,
// or -1 if already at maximum rank.
func NextRankThreshold(currentPoints int) int {
	cur := Rank(currentPoints)
	if cur >= 5 {
		return -1
	}
	for i := len(rankThresholds) - 1; i >= 0; i-- {
		if rankThresholds[i].rank == cur+1 {
			return rankThresholds[i].points
		}
	}
	return -1
}

var ErrUnknownNation = errors.New("unknown nation")

// Store tracks fame points earned in each nation.
type Store struct {
	points map[Nation]int
}

// NewStore creates an empty fame Store.
func NewStore() *Store {
	return &Store{points: make(map[Nation]int)}
}

// Earn adds pts fame to nation n (pts must be positive; negative is a no-op).
func (s *Store) Earn(n Nation, pts int) {
	if pts <= 0 {
		return
	}
	s.points[n] += pts
}

// TotalPoints returns the accumulated fame points for nation n.
func (s *Store) TotalPoints(n Nation) int {
	return s.points[n]
}

// Rank returns the fame rank (0–5) for nation n.
func (s *Store) Rank(n Nation) int {
	return Rank(s.points[n])
}

// RankLabel returns the display label for the player's rank in nation n.
func (s *Store) RankLabel(n Nation) string {
	return RankLabel(s.Rank(n))
}

// MeetsRank reports whether the player's fame in nation n is at least minRank.
func (s *Store) MeetsRank(n Nation, minRank int) bool {
	return s.Rank(n) >= minRank
}

// NextThreshold returns points needed for the next rank in n, or -1 at max.
func (s *Store) NextThreshold(n Nation) int {
	return NextRankThreshold(s.points[n])
}

// Summary returns a slice of NationSummary for all nations.
func (s *Store) Summary() []NationSummary {
	nations := AllNations()
	out := make([]NationSummary, len(nations))
	for i, n := range nations {
		pts := s.points[n]
		out[i] = NationSummary{
			Nation:    n,
			Name:      NationName(n),
			Points:    pts,
			Rank:      Rank(pts),
			RankLabel: RankLabel(Rank(pts)),
			Next:      NextRankThreshold(pts),
		}
	}
	return out
}

// NationSummary is the display-ready reputation record for one nation.
type NationSummary struct {
	Nation    Nation
	Name      string
	Points    int
	Rank      int
	RankLabel string
	Next      int // points to next rank; -1 = maxed
}

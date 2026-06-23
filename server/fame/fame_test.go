package fame

import (
	"testing"
)

// ── Rank thresholds ───────────────────────────────────────────────────────────

func TestRankZeroAtZeroPoints(t *testing.T) {
	if Rank(0) != 0 {
		t.Errorf("expected rank 0 at 0 points, got %d", Rank(0))
	}
}

func TestRankThresholds(t *testing.T) {
	cases := []struct {
		pts  int
		want int
	}{
		{0, 0},
		{49, 0},
		{50, 1},
		{199, 1},
		{200, 2},
		{499, 2},
		{500, 3},
		{999, 3},
		{1000, 4},
		{1999, 4},
		{2000, 5},
		{9999, 5},
	}
	for _, tc := range cases {
		if got := Rank(tc.pts); got != tc.want {
			t.Errorf("Rank(%d) = %d, want %d", tc.pts, got, tc.want)
		}
	}
}

// ── RankLabel ─────────────────────────────────────────────────────────────────

func TestRankLabelKnown(t *testing.T) {
	if RankLabel(1) != "Known" {
		t.Errorf("expected Known, got %q", RankLabel(1))
	}
}

func TestRankLabelRevered(t *testing.T) {
	if RankLabel(5) != "Revered" {
		t.Errorf("expected Revered, got %q", RankLabel(5))
	}
}

// ── NextRankThreshold ─────────────────────────────────────────────────────────

func TestNextRankThresholdFromZero(t *testing.T) {
	next := NextRankThreshold(0)
	if next != 50 {
		t.Errorf("expected 50 from 0 pts, got %d", next)
	}
}

func TestNextRankThresholdAtMax(t *testing.T) {
	next := NextRankThreshold(2000)
	if next != -1 {
		t.Errorf("expected -1 at max rank, got %d", next)
	}
}

// ── Store.Earn + TotalPoints ──────────────────────────────────────────────────

func TestStoreEarnAdds(t *testing.T) {
	s := NewStore()
	s.Earn(Bastok, 30)
	s.Earn(Bastok, 25)
	if s.TotalPoints(Bastok) != 55 {
		t.Errorf("expected 55, got %d", s.TotalPoints(Bastok))
	}
}

func TestStoreEarnNegativeIsNoop(t *testing.T) {
	s := NewStore()
	s.Earn(Sandoria, 100)
	s.Earn(Sandoria, -50)
	if s.TotalPoints(Sandoria) != 100 {
		t.Errorf("negative earn should not decrease points, got %d", s.TotalPoints(Sandoria))
	}
}

func TestStoreEarnZeroIsNoop(t *testing.T) {
	s := NewStore()
	s.Earn(Windurst, 0)
	if s.TotalPoints(Windurst) != 0 {
		t.Errorf("zero earn should leave points unchanged, got %d", s.TotalPoints(Windurst))
	}
}

func TestStoreNationsAreIndependent(t *testing.T) {
	s := NewStore()
	s.Earn(Bastok, 500)
	if s.TotalPoints(Sandoria) != 0 {
		t.Error("Bastok fame should not affect Sandoria")
	}
}

// ── Store.Rank / MeetsRank ────────────────────────────────────────────────────

func TestStoreMeetsRank(t *testing.T) {
	s := NewStore()
	s.Earn(Bastok, 200)
	if !s.MeetsRank(Bastok, 2) {
		t.Error("200 pts should meet rank 2")
	}
	if s.MeetsRank(Bastok, 3) {
		t.Error("200 pts should not meet rank 3")
	}
}

func TestStoreRankProgressesToMax(t *testing.T) {
	s := NewStore()
	s.Earn(Sandoria, 2000)
	if s.Rank(Sandoria) != 5 {
		t.Errorf("2000 pts should give rank 5, got %d", s.Rank(Sandoria))
	}
}

// ── Summary ───────────────────────────────────────────────────────────────────

func TestStoreSummaryLength(t *testing.T) {
	s := NewStore()
	sum := s.Summary()
	if len(sum) != len(AllNations()) {
		t.Errorf("summary should have %d entries, got %d", len(AllNations()), len(sum))
	}
}

func TestStoreSummaryPointsCorrect(t *testing.T) {
	s := NewStore()
	s.Earn(Windurst, 350)
	for _, ns := range s.Summary() {
		if ns.Nation == Windurst {
			if ns.Points != 350 || ns.Rank != 2 || ns.RankLabel != "Liked" {
				t.Errorf("Windurst summary wrong: %+v", ns)
			}
			return
		}
	}
	t.Error("Windurst not found in summary")
}

// ── NationName ────────────────────────────────────────────────────────────────

func TestNationNameKnown(t *testing.T) {
	if NationName(Bastok) != "Bastok" {
		t.Errorf("expected Bastok, got %q", NationName(Bastok))
	}
}

func TestNationNameUnknown(t *testing.T) {
	if NationName(Nation(99)) != "Unknown" {
		t.Errorf("expected Unknown for out-of-range nation")
	}
}

package duel

import (
	"errors"
	"testing"
	"time"
)

func now() time.Time { return time.Now() }

func TestChallengeBasic(t *testing.T) {
	m := NewManager()
	if err := m.Challenge("alice", "bob", now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.InDuel("alice") {
		t.Error("alice should be in duel after challenge")
	}
}

func TestChallengeSelf(t *testing.T) {
	m := NewManager()
	if err := m.Challenge("alice", "alice", now()); !errors.Is(err, ErrSelfDuel) {
		t.Errorf("expected ErrSelfDuel got %v", err)
	}
}

func TestChallengeAlreadyDueling(t *testing.T) {
	m := NewManager()
	m.Challenge("alice", "bob", now())
	if err := m.Challenge("alice", "charlie", now()); !errors.Is(err, ErrAlreadyDueling) {
		t.Errorf("expected ErrAlreadyDueling got %v", err)
	}
}

func TestChallengeDefenderAlreadyDueling(t *testing.T) {
	m := NewManager()
	m.Challenge("alice", "bob", now())
	if err := m.Challenge("charlie", "bob", now()); !errors.Is(err, ErrAlreadyDueling) {
		t.Errorf("expected ErrAlreadyDueling got %v", err)
	}
}

func TestPendingForFindsChallenge(t *testing.T) {
	m := NewManager()
	m.Challenge("alice", "bob", now())
	st := m.PendingFor("bob")
	if st == nil {
		t.Fatal("expected pending challenge for bob")
	}
	if st.Challenger != "alice" {
		t.Errorf("expected challenger alice, got %s", st.Challenger)
	}
}

func TestAcceptTransitionsToActive(t *testing.T) {
	n := now()
	m := NewManager()
	m.Challenge("alice", "bob", n)
	st, err := m.Accept("bob", n.Add(time.Second))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Phase != PhaseActive {
		t.Errorf("expected Active, got %v", st.Phase)
	}
	if !m.InDuel("alice") || !m.InDuel("bob") {
		t.Error("both players should be in duel after accept")
	}
}

func TestAcceptNoPending(t *testing.T) {
	m := NewManager()
	if _, err := m.Accept("bob", now()); !errors.Is(err, ErrNoPendingDuel) {
		t.Errorf("expected ErrNoPendingDuel got %v", err)
	}
}

func TestAcceptExpired(t *testing.T) {
	n := time.Now().Add(-2 * ChallengeTimeout)
	m := NewManager()
	m.Challenge("alice", "bob", n)
	_, err := m.Accept("bob", time.Now())
	if !errors.Is(err, ErrChallengeExpired) {
		t.Errorf("expected ErrChallengeExpired got %v", err)
	}
}

func TestForfeitActive(t *testing.T) {
	n := now()
	m := NewManager()
	m.Challenge("alice", "bob", n)
	m.Accept("bob", n.Add(time.Second))
	st, err := m.Forfeit("alice", n.Add(2*time.Second))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Winner != "bob" {
		t.Errorf("expected bob to win on alice forfeit, got %s", st.Winner)
	}
	if m.InDuel("alice") || m.InDuel("bob") {
		t.Error("neither player should be in duel after forfeit")
	}
}

func TestForfeitPendingChallenge(t *testing.T) {
	m := NewManager()
	m.Challenge("alice", "bob", now())
	st, err := m.Forfeit("alice", now().Add(time.Second))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Phase != PhaseDone {
		t.Errorf("expected Done, got %v", st.Phase)
	}
}

func TestForfeitNotInDuel(t *testing.T) {
	m := NewManager()
	if _, err := m.Forfeit("alice", now()); !errors.Is(err, ErrNotInDuel) {
		t.Errorf("expected ErrNotInDuel got %v", err)
	}
}

func TestReportHPChallengerDies(t *testing.T) {
	n := now()
	m := NewManager()
	m.Challenge("alice", "bob", n)
	st, _ := m.Accept("bob", n.Add(time.Second))
	winner, done := m.ReportHP(st, 0, 50)
	if !done {
		t.Error("expected duel done when challenger HP=0")
	}
	if winner != "bob" {
		t.Errorf("expected bob to win, got %s", winner)
	}
}

func TestReportHPDefenderDies(t *testing.T) {
	n := now()
	m := NewManager()
	m.Challenge("alice", "bob", n)
	st, _ := m.Accept("bob", n.Add(time.Second))
	winner, done := m.ReportHP(st, 50, 0)
	if !done {
		t.Error("expected duel done when defender HP=0")
	}
	if winner != "alice" {
		t.Errorf("expected alice to win, got %s", winner)
	}
}

func TestReportHPBothAlive(t *testing.T) {
	n := now()
	m := NewManager()
	m.Challenge("alice", "bob", n)
	st, _ := m.Accept("bob", n.Add(time.Second))
	winner, done := m.ReportHP(st, 50, 50)
	if done {
		t.Error("expected duel not done when both alive")
	}
	if winner != "" {
		t.Errorf("expected no winner, got %s", winner)
	}
}

func TestRatingDefault(t *testing.T) {
	m := NewManager()
	if m.Rating("nobody") != 1000 {
		t.Error("default rating should be 1000")
	}
}

func TestRatingAfterWin(t *testing.T) {
	n := now()
	m := NewManager()
	m.Challenge("alice", "bob", n)
	st, _ := m.Accept("bob", n.Add(time.Second))
	m.ReportHP(st, 50, 0) // alice wins
	if m.Rating("alice") != 1000+WinRating {
		t.Errorf("expected %d, got %d", 1000+WinRating, m.Rating("alice"))
	}
	if m.Rating("bob") != 1000+LossRating {
		t.Errorf("expected %d, got %d", 1000+LossRating, m.Rating("bob"))
	}
}

func TestTopN(t *testing.T) {
	m := NewManager()
	m.Ratings["alice"] = 100
	m.Ratings["bob"] = 50
	m.Ratings["charlie"] = 200
	top := m.TopN(2)
	if len(top) != 2 || top[0] != "charlie" {
		t.Errorf("expected charlie first, got %v", top)
	}
}

func TestExpirePending(t *testing.T) {
	n := time.Now().Add(-2 * ChallengeTimeout)
	m := NewManager()
	m.Challenge("alice", "bob", n)
	expired := m.ExpirePending(time.Now())
	if len(expired) != 1 {
		t.Errorf("expected 1 expired, got %d", len(expired))
	}
	if m.InDuel("alice") {
		t.Error("alice should not be in duel after expiry")
	}
}

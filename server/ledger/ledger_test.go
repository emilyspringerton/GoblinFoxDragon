package ledger

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

func newLedger() *Ledger { return NewLedger() }

func append1(l *Ledger, verb VerbType, foID, actor string, now time.Time) Receipt {
	return l.Append(verb, foID, actor, "", "", now)
}

// ── Append ────────────────────────────────────────────────────────────────────

func TestAppendSetsSeq(t *testing.T) {
	l := newLedger()
	r1 := append1(l, VerbClaimed, "fo-1", "crew-a", t0)
	r2 := append1(l, VerbContested, "fo-1", "crew-b", t0)
	if r1.Seq != 1 || r2.Seq != 2 {
		t.Errorf("expected seq 1,2 got %d,%d", r1.Seq, r2.Seq)
	}
}

func TestAppendIncrementsLen(t *testing.T) {
	l := newLedger()
	append1(l, VerbClaimed, "fo-1", "crew-a", t0)
	append1(l, VerbFlipped, "fo-1", "crew-b", t0)
	// Flipped adds 1 base + possibly 1 exploit (no exploit yet, only 1 flip)
	if l.Len() < 2 {
		t.Errorf("expected at least 2 entries, got %d", l.Len())
	}
}

// ── Query ─────────────────────────────────────────────────────────────────────

func TestAll(t *testing.T) {
	l := newLedger()
	append1(l, VerbClaimed, "fo-1", "crew-a", t0)
	append1(l, VerbClaimed, "fo-2", "crew-b", t0)
	all := l.All()
	if len(all) != 2 {
		t.Errorf("expected 2 receipts, got %d", len(all))
	}
}

func TestByFO(t *testing.T) {
	l := newLedger()
	append1(l, VerbClaimed, "fo-1", "crew-a", t0)
	append1(l, VerbClaimed, "fo-2", "crew-b", t0)
	append1(l, VerbContested, "fo-1", "crew-b", t0)
	by1 := l.ByFO("fo-1")
	if len(by1) != 2 {
		t.Errorf("expected 2 receipts for fo-1, got %d", len(by1))
	}
}

func TestByActor(t *testing.T) {
	l := newLedger()
	append1(l, VerbClaimed, "fo-1", "crew-a", t0)
	append1(l, VerbClaimed, "fo-2", "crew-a", t0)
	append1(l, VerbClaimed, "fo-3", "crew-b", t0)
	byA := l.ByActor("crew-a")
	if len(byA) != 2 {
		t.Errorf("expected 2 for crew-a, got %d", len(byA))
	}
}

func TestByVerb(t *testing.T) {
	l := newLedger()
	append1(l, VerbClaimed, "fo-1", "crew-a", t0)
	append1(l, VerbClaimed, "fo-2", "crew-b", t0)
	append1(l, VerbContested, "fo-1", "crew-b", t0)
	claimed := l.ByVerb(VerbClaimed)
	if len(claimed) != 2 {
		t.Errorf("expected 2 CLAIMED receipts, got %d", len(claimed))
	}
}

func TestSince(t *testing.T) {
	l := newLedger()
	append1(l, VerbClaimed, "fo-1", "crew-a", t0)
	later := t0.Add(5 * time.Second)
	append1(l, VerbContested, "fo-1", "crew-b", later)
	since := l.Since(later)
	if len(since) != 1 {
		t.Errorf("expected 1 receipt since later, got %d", len(since))
	}
	if since[0].Verb != VerbContested {
		t.Errorf("expected CONTESTED, got %q", since[0].Verb)
	}
}

// ── ReceiptBurst ──────────────────────────────────────────────────────────────

func TestReceiptBurstWindowFilter(t *testing.T) {
	l := newLedger()
	// Old receipt — outside burst window.
	oldTime := t0.Add(-BurstWindow - time.Second)
	append1(l, VerbClaimed, "fo-1", "crew-a", oldTime)
	// Recent receipt — inside burst window.
	append1(l, VerbContested, "fo-1", "crew-b", t0)

	burst := l.ReceiptBurst("fo-1", t0)
	if len(burst) != 1 {
		t.Errorf("expected 1 receipt in burst window, got %d", len(burst))
	}
	if burst[0].Verb != VerbContested {
		t.Errorf("expected CONTESTED in burst, got %q", burst[0].Verb)
	}
}

func TestReceiptBurstFOFilter(t *testing.T) {
	l := newLedger()
	append1(l, VerbClaimed, "fo-1", "crew-a", t0)
	append1(l, VerbClaimed, "fo-2", "crew-b", t0)
	burst := l.ReceiptBurst("fo-1", t0)
	if len(burst) != 1 || burst[0].FOID != "fo-1" {
		t.Error("burst should only return receipts for requested FO")
	}
}

// ── Anti-exploit ──────────────────────────────────────────────────────────────

func TestFlipScoreCounts(t *testing.T) {
	l := newLedger()
	for i := 0; i < 3; i++ {
		append1(l, VerbFlipped, "fo-1", "crew-a", t0)
	}
	score := l.FlipScore("crew-a", t0)
	if score != 3 {
		t.Errorf("expected flip score 3, got %d", score)
	}
}

func TestFlipScoreIgnoresOldFlips(t *testing.T) {
	l := newLedger()
	oldTime := t0.Add(-AnalysisWindow - time.Second)
	for i := 0; i < ExploitThreshold+2; i++ {
		append1(l, VerbFlipped, "fo-1", "crew-a", oldTime)
	}
	score := l.FlipScore("crew-a", t0) // check at t0; old flips should be outside window
	if score != 0 {
		t.Errorf("old flips should not count, got score=%d", score)
	}
}

func TestExploitFlagFires(t *testing.T) {
	l := newLedger()
	// Fire ExploitThreshold flips in rapid succession.
	for i := 0; i < ExploitThreshold; i++ {
		append1(l, VerbFlipped, "fo-1", "crew-a", t0)
	}
	suspicious := l.ByVerb(VerbSuspicious)
	if len(suspicious) == 0 {
		t.Error("expected SUSPICIOUS_PATTERN flag after exploit threshold")
	}
	if suspicious[0].Actor != "crew-a" {
		t.Errorf("expected actor crew-a in suspicious flag, got %q", suspicious[0].Actor)
	}
}

func TestExploitFlagNotFiredBeforeThreshold(t *testing.T) {
	l := newLedger()
	for i := 0; i < ExploitThreshold-1; i++ {
		append1(l, VerbFlipped, "fo-1", "crew-a", t0)
	}
	if len(l.ByVerb(VerbSuspicious)) != 0 {
		t.Error("should not flag before threshold")
	}
}

func TestPruneFlipLog(t *testing.T) {
	l := newLedger()
	oldTime := t0.Add(-AnalysisWindow - time.Second)
	append1(l, VerbFlipped, "fo-1", "crew-a", oldTime)
	l.PruneFlipLog(t0)
	if l.FlipScore("crew-a", t0) != 0 {
		t.Error("flip score should be 0 after pruning old entries")
	}
}

// ── Receipt immutability ──────────────────────────────────────────────────────

func TestAllReturnsCopy(t *testing.T) {
	l := newLedger()
	append1(l, VerbClaimed, "fo-1", "crew-a", t0)
	snap := l.All()
	snap[0].Actor = "mutated"
	all2 := l.All()
	if all2[0].Actor == "mutated" {
		t.Error("All() should return a copy that does not modify the ledger")
	}
}

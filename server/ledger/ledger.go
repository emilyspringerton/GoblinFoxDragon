// Package ledger implements the TRAPX mocumentary Receipt ledger.
//
// Every meaningful game action produces a Receipt — a diegetic log line that
// forms the "mocumentary record" of what happened in the city. The ledger is
// append-only. Receipts are never deleted.
//
// # Anti-exploit scoring
//
// Rapid-flip sequences (multiple FO flips in short windows) indicate likely
// exploit/griefing behaviour. The ledger tracks per-crew flip timestamps and
// scores a burst pattern:
//
//	PatternScore = number of flips in the last AnalysisWindow seconds
//
// When PatternScore >= ExploitThreshold, a SUSPICIOUS_PATTERN flag is appended.
//
// # ReceiptBurst
//
// ReceiptBurst returns all receipts in the last BurstWindow seconds for a given
// FO. K9 dogs use this to generate per-FO audit reports.
package ledger

import (
	"fmt"
	"time"
)

const (
	ExploitThreshold = 4             // flips/minute considered suspicious
	AnalysisWindow   = 60 * time.Second
	BurstWindow      = 30 * time.Second
)

// VerbType groups receipt verbs by category.
type VerbType string

const (
	// FO custody verbs (from fieldoffice package).
	VerbClaimed          VerbType = "CLAIMED"
	VerbContested        VerbType = "CONTESTED"
	VerbFlipped          VerbType = "FLIPPED"
	VerbDefended         VerbType = "DEFENDED"
	VerbContainmentStart VerbType = "CONTAINMENT_START"
	VerbContainmentEnd   VerbType = "CONTAINMENT_END"
	VerbObjective        VerbType = "OBJECTIVE_COMPLETE"

	// K9 verbs.
	VerbDogDeployed    VerbType = "DOG_DEPLOYED"
	VerbDogMark        VerbType = "DOG_MARK"
	VerbDogCustodyLock VerbType = "DOG_CUSTODY_LOCK"
	VerbDogRogueDeath  VerbType = "DOG_ROGUE_DEATH"

	// City simulation verbs.
	VerbRogueSwarm    VerbType = "ROGUE_SWARM"
	VerbScarWritten   VerbType = "SCAR_WRITTEN"
	VerbCrownProtocol VerbType = "CROWN_PROTOCOL"
	VerbBirdCorrect   VerbType = "BIRD_CORRECTION"

	// Anti-exploit.
	VerbSuspicious VerbType = "SUSPICIOUS_PATTERN"
)

// Receipt is an immutable diegetic log line.
type Receipt struct {
	Seq      int       // monotonically increasing sequence number
	At       time.Time
	FOID     string   // Field Office ID (empty for city-wide events)
	Verb     VerbType
	Actor    string   // crew ID (empty for automated events)
	Subject  string   // target crew ID or entity
	Metadata string   // arbitrary detail string
}

func (r Receipt) String() string {
	return fmt.Sprintf("[%05d %s] %s @%s actor=%s subj=%s %s",
		r.Seq, r.At.Format("15:04:05"), r.Verb, r.FOID, r.Actor, r.Subject, r.Metadata)
}

// Ledger is an append-only diegetic action log.
// Not goroutine-safe; callers synchronise via the server tick loop.
type Ledger struct {
	entries []Receipt
	seq     int

	// flip tracking per crew, for anti-exploit scoring.
	flipLog map[string][]time.Time // crewID → flip timestamps
}

// NewLedger creates an empty Ledger.
func NewLedger() *Ledger {
	return &Ledger{flipLog: make(map[string][]time.Time)}
}

// Append adds a Receipt to the ledger. The Seq field is assigned automatically.
// Returns the assigned Receipt (with Seq set).
func (l *Ledger) Append(verb VerbType, foID, actor, subject, metadata string, now time.Time) Receipt {
	l.seq++
	r := Receipt{
		Seq: l.seq, At: now,
		FOID: foID, Verb: verb,
		Actor: actor, Subject: subject, Metadata: metadata,
	}
	l.entries = append(l.entries, r)

	// Track flips for anti-exploit analysis.
	if verb == VerbFlipped {
		l.flipLog[actor] = append(l.flipLog[actor], now)
		if flag, ok := l.detectExploit(actor, now); ok {
			l.seq++
			flag.Seq = l.seq
			l.entries = append(l.entries, flag)
		}
	}

	return r
}

// Len returns the total number of receipts in the ledger (including exploit flags).
func (l *Ledger) Len() int { return len(l.entries) }

// All returns a snapshot of all receipts in order.
func (l *Ledger) All() []Receipt {
	out := make([]Receipt, len(l.entries))
	copy(out, l.entries)
	return out
}

// ByFO returns all receipts for a given FO ID.
func (l *Ledger) ByFO(foID string) []Receipt {
	var out []Receipt
	for _, r := range l.entries {
		if r.FOID == foID {
			out = append(out, r)
		}
	}
	return out
}

// ByActor returns all receipts for a given actor crew ID.
func (l *Ledger) ByActor(actor string) []Receipt {
	var out []Receipt
	for _, r := range l.entries {
		if r.Actor == actor {
			out = append(out, r)
		}
	}
	return out
}

// ByVerb returns all receipts matching the given verb.
func (l *Ledger) ByVerb(verb VerbType) []Receipt {
	var out []Receipt
	for _, r := range l.entries {
		if r.Verb == verb {
			out = append(out, r)
		}
	}
	return out
}

// ReceiptBurst returns all receipts for a given FO within the last BurstWindow.
// Used by K9 dogs to generate per-FO audit reports.
func (l *Ledger) ReceiptBurst(foID string, now time.Time) []Receipt {
	cutoff := now.Add(-BurstWindow)
	var out []Receipt
	for _, r := range l.entries {
		if r.FOID == foID && !r.At.Before(cutoff) {
			out = append(out, r)
		}
	}
	return out
}

// Since returns all receipts added on or after the given time.
func (l *Ledger) Since(t time.Time) []Receipt {
	var out []Receipt
	for _, r := range l.entries {
		if !r.At.Before(t) {
			out = append(out, r)
		}
	}
	return out
}

// FlipScore returns how many flips crewID has recorded in the last AnalysisWindow.
func (l *Ledger) FlipScore(crewID string, now time.Time) int {
	cutoff := now.Add(-AnalysisWindow)
	count := 0
	for _, ts := range l.flipLog[crewID] {
		if !ts.Before(cutoff) {
			count++
		}
	}
	return count
}

// detectExploit checks if the actor's recent flip log exceeds ExploitThreshold.
// Returns a SUSPICIOUS_PATTERN receipt and true if the threshold is exceeded.
func (l *Ledger) detectExploit(actor string, now time.Time) (Receipt, bool) {
	score := l.FlipScore(actor, now)
	if score < ExploitThreshold {
		return Receipt{}, false
	}
	return Receipt{
		At: now, Verb: VerbSuspicious, Actor: actor,
		Metadata: fmt.Sprintf("flip_score=%d window=%s threshold=%d",
			score, AnalysisWindow, ExploitThreshold),
	}, true
}

// PruneFlipLog removes flip timestamps older than AnalysisWindow to bound memory.
// Should be called periodically (e.g. every minute) by the game loop.
func (l *Ledger) PruneFlipLog(now time.Time) {
	cutoff := now.Add(-AnalysisWindow)
	for crew, timestamps := range l.flipLog {
		live := timestamps[:0]
		for _, ts := range timestamps {
			if !ts.Before(cutoff) {
				live = append(live, ts)
			}
		}
		l.flipLog[crew] = live
	}
}

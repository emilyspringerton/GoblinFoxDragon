package ledger

import (
	"fmt"
	"strings"
	"time"
)

// tyler_bridge.go: TYLER episode receipts → TRAPX ledger bridge (S123-02).
//
// TYLER lore artifacts and episode events are filed as ledger receipts so they
// appear in the same unified feed as TRAPX FO events. The Dragon ACT phase
// calls TYLERPostEpisodeReceipt to record canonical TYLER events; the in-game
// CAST terminal (scene 203, Cairngorms Archive) calls TYLERLoreFragments to
// render browsable lore documents.

// TYLEREpisode identifies a TYLER episode/timeline event.
type TYLEREpisode struct {
	ID         string // e.g. "VS0-01", "S1E03-migration"
	Title      string
	Timeline   string // "present" or branch ID
	DistrictID string // TRAPX district where event manifested
	Summary    string // one-sentence summary for the ledger receipt
}

// TYLERPostEpisodeReceipt records a TYLER episode event in the unified ledger.
func TYLERPostEpisodeReceipt(l *Ledger, ep TYLEREpisode, actor string, now time.Time) Receipt {
	return l.Append(
		VerbTYLEREpisodeReceipt,
		ep.DistrictID,
		actor,
		ep.ID,
		fmt.Sprintf("title=%q timeline=%s summary=%s", ep.Title, ep.Timeline, ep.Summary),
		now,
	)
}

// TYLERPostMigrationEvent records a Migration Event (equivalent to a Rogue Swarm in TYLER timeline).
func TYLERPostMigrationEvent(l *Ledger, fromBranch, toBranch, districtID, actor string, now time.Time) Receipt {
	return l.Append(
		VerbTYLERMigrationEvent,
		districtID,
		actor,
		"",
		fmt.Sprintf("from=%s to=%s", fromBranch, toBranch),
		now,
	)
}

// TYLERPostFieldActivation records when a TYLER Field entity activates in a district.
func TYLERPostFieldActivation(l *Ledger, entityName, districtID, actor string, now time.Time) Receipt {
	return l.Append(
		VerbTYLERFieldActivation,
		districtID,
		actor,
		entityName,
		"",
		now,
	)
}

// TYLERPostArchiveEntry records a Cairngorms Archive discovery (lore read).
func TYLERPostArchiveEntry(l *Ledger, docID, title, reader string, now time.Time) Receipt {
	return l.Append(
		VerbTYLERArchiveEntry,
		"district-underground", // Cairngorms is scene 203, underground district
		reader,
		docID,
		fmt.Sprintf("title=%q", title),
		now,
	)
}

// TYLERLoreDoc is a CAST terminal browsable lore document from the Cairngorms Archive.
type TYLERLoreDoc struct {
	ID      string
	Title   string
	Scene   int    // which CAST terminal (scene 203 = Cairngorms, 204 = Vatican, etc.)
	Body    string // multi-line lore text (MUD terminal renders this verbatim)
}

// TYLERLoreDocs contains the canonical in-game CAST terminal lore fragments.
// Browsable via `cast read <doc-id>` command in CAST terminal scenes.
var TYLERLoreDocs = []TYLERLoreDoc{
	{
		ID:    "eastwind-archive-001",
		Title: "Eastwind Archive: Incident Log, Year 0",
		Scene: 203,
		Body: strings.Join([]string{
			"[EASTWIND ARCHIVE — INCIDENT LOG — YEAR 0]",
			"",
			"0001: Jiangshi faction establishes first Detroit Apartment FO. No contester.",
			"0002: First Rogue Swarm event recorded. Emily system logs: ORIGIN UNKNOWN.",
			"0003: Migration Event opens branch: PRESENT → VS0-DIVERGENCE.",
			"0004: Scar written in Detroit Apartment. Text: 'She came back.'",
			"0005: Heikegani faction claims Osaka Underport dock-side FO.",
			"       This is the last entry before the Year 0 archive lock.",
			"",
			"[END OF ACCESSIBLE RECORD]",
		}, "\r\n"),
	},
	{
		ID:    "jiangshi-project-memos-001",
		Title: "Jiangshi Project: Internal Memo #1 — The Building",
		Scene: 200,
		Body: strings.Join([]string{
			"[JIANGSHI PROJECT — INTERNAL MEMO #1]",
			"",
			"RE: The building at Detroit Apartment",
			"",
			"The building is not a headquarters. It is a listening post.",
			"Jiangshi does not recruit. We recognise.",
			"",
			"The tenants who have lived here since before the Rogue Swarms —",
			"they are the archive. Their routines are the data.",
			"",
			"Do not let the CAST terminal in 3F be decommissioned.",
			"It was the first terminal to log a Dragon observation.",
			"The data predates Emily OS by two years.",
			"",
			"[MEMO ENDS]",
		}, "\r\n"),
	},
	{
		ID:    "shell-parliament-ledger-001",
		Title: "Shell Parliament: Ledger Entry — Heikegani Tithe",
		Scene: 205,
		Body: strings.Join([]string{
			"[SHELL PARLIAMENT LEDGER — HEIKEGANI TITHE RECORD]",
			"",
			"District: Osaka Underport",
			"Period: Cycle 4–7",
			"",
			"Item          Quantity   Destination",
			"────────────────────────────────────────",
			"Bulkhead data   38 reels   Cairngorms",
			"Tide-mark keys   3 sets    Vatican Corridors",
			"Crab-carapace    12 units  Kuroshio Coast",
			"",
			"Note: Shipment 38B was intercepted by a Rogue Swarm event",
			"at the dock-south channel. Dragon observation logged.",
			"Parliament voted to resume. No coverage deal offered.",
			"",
			"[LEDGER ENTRY SEALED]",
		}, "\r\n"),
	},
	{
		ID:    "field-activation-logs-001",
		Title: "Field Activation Log — Dragon Event: Cycle 22",
		Scene: 203,
		Body: strings.Join([]string{
			"[FIELD ACTIVATION LOG — DRAGON EVENT: CYCLE 22]",
			"",
			"TYPE:         media_spike",
			"DISTRICT:     district-commercial (Osaka Convenience Store)",
			"ATTENTION:    847 / 1000",
			"ACTOR:        EMILY_PRIME",
			"ARCHETYPE:    THE_FIELD E2",
			"REASON:       attn>800&&!IsUnderAudit",
			"",
			"Archetype corridor: THE_FIELD — spirit_stack: RESOLVE_DOUBT — Δφ: 27°",
			"",
			"CONSEQUENCE:  Coverage drone density +2.",
			"              Channel 11 broadcast team entered district.",
			"              Myth seeded: 'The Dragon moved through here once.",
			"              The streetlights changed color and stayed that way.'",
			"",
			"[LOG FILED BY DRAGON ACT — CYCLE 22]",
		}, "\r\n"),
	},
}

// TYLERLoreDocByID returns the lore doc for a given ID, or nil if not found.
func TYLERLoreDocByID(id string) *TYLERLoreDoc {
	for i := range TYLERLoreDocs {
		if TYLERLoreDocs[i].ID == id {
			return &TYLERLoreDocs[i]
		}
	}
	return nil
}

// TYLERLoreDocsForScene returns all lore docs accessible from a given scene ID.
func TYLERLoreDocsForScene(sceneID int) []TYLERLoreDoc {
	var out []TYLERLoreDoc
	for _, d := range TYLERLoreDocs {
		if d.Scene == sceneID {
			out = append(out, d)
		}
	}
	return out
}

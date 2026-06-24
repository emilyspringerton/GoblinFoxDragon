package campaign_test

import (
	"testing"
	"time"

	"dragonsnshit/server/campaign"
	"dragonsnshit/server/conquest"
)

var (
	startT = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	dur24h = campaign.DefaultCycleDuration
)

func newBattle(t *testing.T, nodes ...*campaign.CampaignNode) *campaign.Battle {
	t.Helper()
	b := campaign.NewBattle(startT, dur24h)
	for _, n := range nodes {
		b.AddNode(n)
	}
	return b
}

func node(id string, scene int) *campaign.CampaignNode {
	return &campaign.CampaignNode{ID: id, SceneID: scene, Holder: conquest.NationNeutral}
}

func enrollAll(t *testing.T, b *campaign.Battle) {
	t.Helper()
	b.Enroll("p1", conquest.NationSandoria)
	b.Enroll("p2", conquest.NationBastok)
	b.Enroll("p3", conquest.NationWindurst)
}

// TestNewBattleHasNoNodes verifies empty battle starts correctly.
func TestNewBattleHasNoNodes(t *testing.T) {
	b := campaign.NewBattle(startT, dur24h)
	if b.NodeCount() != 0 {
		t.Errorf("want 0 nodes, got %d", b.NodeCount())
	}
	if b.IsActive() {
		t.Error("want inactive before start")
	}
}

// TestAddNode adds nodes and checks count.
func TestAddNodeIncrementsCount(t *testing.T) {
	b := newBattle(t, node("n1", 0), node("n2", 1))
	if b.NodeCount() != 2 {
		t.Errorf("want 2 nodes, got %d", b.NodeCount())
	}
}

// TestEnrollNeutralReturnsError verifies neutral players cannot join.
func TestEnrollNeutralReturnsError(t *testing.T) {
	b := newBattle(t)
	if err := b.Enroll("p0", conquest.NationNeutral); err == nil {
		t.Error("want error for neutral enroll")
	}
}

// TestEnrollAndIsEnrolled round-trips enrollment.
func TestEnrollAndIsEnrolled(t *testing.T) {
	b := newBattle(t)
	b.Enroll("p1", conquest.NationBastok)
	if !b.IsEnrolled("p1") {
		t.Error("want p1 enrolled")
	}
	if b.IsEnrolled("p2") {
		t.Error("want p2 not enrolled")
	}
}

// TestRecordKillUnenrolledIsNoop verifies unenrolled players have no effect.
func TestRecordKillUnenrolledIsNoop(t *testing.T) {
	b := newBattle(t, node("n0", 0))
	nodeID := b.RecordKill("ghost", 0)
	if nodeID != "" {
		t.Errorf("want no node change, got %q", nodeID)
	}
}

// TestRecordKillNoNodeInScene returns "" if no node in scene.
func TestRecordKillNoNodeInScene(t *testing.T) {
	b := newBattle(t, node("n0", 0))
	b.Enroll("p1", conquest.NationSandoria)
	if id := b.RecordKill("p1", 99); id != "" {
		t.Errorf("want no node for scene 99, got %q", id)
	}
}

// TestRecordKillShiftsHPAtThreshold verifies HP shifts after KillsPerHPShift kills.
func TestRecordKillShiftsHPAtThreshold(t *testing.T) {
	b := newBattle(t, node("n0", 0))
	b.Enroll("p1", conquest.NationSandoria)
	// KillsPerHPShift-1 kills: no shift yet.
	for i := 0; i < campaign.KillsPerHPShift-1; i++ {
		if id := b.RecordKill("p1", 0); id != "" {
			t.Errorf("early shift at kill %d", i+1)
		}
	}
	// 10th kill: HP shifts.
	id := b.RecordKill("p1", 0)
	if id != "n0" {
		t.Errorf("want n0 changed, got %q", id)
	}
	snap, _ := b.NodeSnapshot("n0")
	if snap.HP != campaign.DefaultNodeHP-1 {
		t.Errorf("HP: want %d, got %d", campaign.DefaultNodeHP-1, snap.HP)
	}
}

// TestRecordKillCapture verifies attacker captures node when HP hits 0.
func TestRecordKillCapture(t *testing.T) {
	b := newBattle(t, &campaign.CampaignNode{ID: "n0", SceneID: 0, Holder: conquest.NationNeutral, HP: campaign.KillsPerHPShift})
	b.AddNode(&campaign.CampaignNode{}) // add a dummy to avoid re-use
	// Re-build with exact HP.
	b2 := campaign.NewBattle(startT, dur24h)
	b2.AddNode(&campaign.CampaignNode{ID: "low", SceneID: 0, Holder: conquest.NationNeutral, HP: 1})
	b2.Enroll("attacker", conquest.NationBastok)
	// One batch of KillsPerHPShift kills → HP -1 → 0 → Bastok captures.
	var lastID string
	for i := 0; i < campaign.KillsPerHPShift; i++ {
		lastID = b2.RecordKill("attacker", 0)
	}
	if lastID != "low" {
		t.Fatalf("want capture of low, got %q", lastID)
	}
	snap, _ := b2.NodeSnapshot("low")
	if snap.Holder != conquest.NationBastok {
		t.Errorf("holder: want Bastok, got %d", snap.Holder)
	}
	if snap.HP != campaign.DefaultNodeHP {
		t.Errorf("HP after capture: want %d, got %d", campaign.DefaultNodeHP, snap.HP)
	}
}

// TestRecordKillDefenderRestoresHP verifies holder kills restore HP.
func TestRecordKillDefenderRestoresHP(t *testing.T) {
	b := campaign.NewBattle(startT, dur24h)
	b.AddNode(&campaign.CampaignNode{ID: "hold", SceneID: 0, Holder: conquest.NationSandoria, HP: campaign.DefaultNodeHP - 5})
	b.Enroll("defender", conquest.NationSandoria)
	for i := 0; i < campaign.KillsPerHPShift; i++ {
		b.RecordKill("defender", 0)
	}
	snap, _ := b.NodeSnapshot("hold")
	if snap.HP != campaign.DefaultNodeHP-4 {
		t.Errorf("defender HP restore: want %d, got %d", campaign.DefaultNodeHP-4, snap.HP)
	}
}

// TestTickActivatesBattleAtStart verifies IsActive flips when start time passes.
func TestTickActivatesBattleAtStart(t *testing.T) {
	b := newBattle(t)
	if b.IsActive() {
		t.Fatal("want inactive before start")
	}
	b.Tick(startT)
	if !b.IsActive() {
		t.Error("want active at start time")
	}
}

// TestTickEndsBattleAndReturnsResult verifies cycle ends and returns result.
func TestTickEndsBattleAndReturnsResult(t *testing.T) {
	b := campaign.NewBattle(startT, dur24h)
	b.AddNode(&campaign.CampaignNode{ID: "n1", SceneID: 0, Holder: conquest.NationSandoria, HP: 50})
	b.AddNode(&campaign.CampaignNode{ID: "n2", SceneID: 1, Holder: conquest.NationBastok, HP: 50})
	b.Tick(startT) // activate
	ended, result := b.Tick(startT.Add(dur24h))
	if !ended {
		t.Fatal("want ended = true")
	}
	// Sandoria and Bastok each hold 1 node → tie → NationNeutral winner.
	if result.Winner != conquest.NationNeutral {
		t.Errorf("tie winner: want Neutral, got %d", result.Winner)
	}
	if result.NodesByNation[conquest.NationSandoria] != 1 {
		t.Errorf("sandoria nodes: want 1, got %d", result.NodesByNation[conquest.NationSandoria])
	}
}

// TestCycleWinnerClearMajority verifies correct winner with clear majority.
func TestCycleWinnerClearMajority(t *testing.T) {
	b := campaign.NewBattle(startT, dur24h)
	for i, holder := range []conquest.Nation{
		conquest.NationWindurst, conquest.NationWindurst, conquest.NationWindurst,
		conquest.NationSandoria, conquest.NationBastok,
	} {
		b.AddNode(&campaign.CampaignNode{
			ID:      string(rune('a' + i)),
			SceneID: i,
			Holder:  holder,
			HP:      50,
		})
	}
	b.Tick(startT) // activate
	_, result := b.Tick(startT.Add(dur24h))
	if result.Winner != conquest.NationWindurst {
		t.Errorf("winner: want Windurst, got %d", result.Winner)
	}
}

// TestDefaultNodes returns exactly 10 nodes.
func TestDefaultNodes(t *testing.T) {
	nodes := campaign.DefaultNodes()
	if len(nodes) != 10 {
		t.Errorf("want 10 nodes, got %d", len(nodes))
	}
}

// TestDefaultNodesAllNeutral verifies all default nodes start unclaimed.
func TestDefaultNodesAllNeutral(t *testing.T) {
	for _, n := range campaign.DefaultNodes() {
		if n.Holder != conquest.NationNeutral {
			t.Errorf("node %s: want neutral holder, got %d", n.ID, n.Holder)
		}
	}
}

// TestAllNodesSnapshot verifies AllNodes returns the correct count.
func TestAllNodesSnapshot(t *testing.T) {
	b := newBattle(t)
	for _, n := range campaign.DefaultNodes() {
		b.AddNode(n)
	}
	all := b.AllNodes()
	if len(all) != campaign.NodeCount {
		t.Errorf("want %d nodes, got %d", campaign.NodeCount, len(all))
	}
}

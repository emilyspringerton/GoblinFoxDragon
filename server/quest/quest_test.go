package quest

import (
	"testing"
)

func starterBank() *Bank {
	return NewBank(StarterQuests)
}

func TestStarterQuestCount(t *testing.T) {
	b := starterBank()
	if got := len(b.All()); got != 5 {
		t.Fatalf("expected 5 starter quests, got %d", got)
	}
}

func TestBankGetKnown(t *testing.T) {
	b := starterBank()
	q, err := b.Get("gather-iron-ore")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.RewardFlow != 100 {
		t.Errorf("expected 100 flow reward, got %d", q.RewardFlow)
	}
}

func TestBankGetUnknown(t *testing.T) {
	b := starterBank()
	if _, err := b.Get("no-such-quest"); err != ErrUnknownQuest {
		t.Errorf("expected ErrUnknownQuest got %v", err)
	}
}

func TestBankForNPC(t *testing.T) {
	b := starterBank()
	guildmasterQuests := b.ForNPC("guildmaster")
	if len(guildmasterQuests) != 2 {
		t.Errorf("expected 2 guildmaster quests, got %d", len(guildmasterQuests))
	}
	scoutQuests := b.ForNPC("scout")
	if len(scoutQuests) != 2 {
		t.Errorf("expected 2 scout quests, got %d", len(scoutQuests))
	}
	merchantQuests := b.ForNPC("merchant")
	if len(merchantQuests) != 1 {
		t.Errorf("expected 1 merchant quest, got %d", len(merchantQuests))
	}
}

func TestAcceptQuest(t *testing.T) {
	b := starterBank()
	j := NewJournal()
	q, _ := b.Get("swamp-patrol")
	if err := j.Accept(q); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := j.Active["swamp-patrol"]; !ok {
		t.Error("quest should be active after accept")
	}
}

func TestAcceptAlreadyActive(t *testing.T) {
	b := starterBank()
	j := NewJournal()
	q, _ := b.Get("swamp-patrol")
	j.Accept(q)
	if err := j.Accept(q); err != ErrAlreadyActive {
		t.Errorf("expected ErrAlreadyActive got %v", err)
	}
}

func TestAcceptAlreadyDone(t *testing.T) {
	b := starterBank()
	j := NewJournal()
	q, _ := b.Get("crisis-volunteer")
	j.Accept(q)
	j.Completed["crisis-volunteer"] = true
	delete(j.Active, "crisis-volunteer")
	if err := j.Accept(q); err != ErrAlreadyDone {
		t.Errorf("expected ErrAlreadyDone got %v", err)
	}
}

func TestRecordKillProgressesQuest(t *testing.T) {
	b := starterBank()
	j := NewJournal()
	q, _ := b.Get("swamp-patrol")
	j.Accept(q)
	updated := j.RecordKill("leech")
	if len(updated) != 1 || updated[0] != "swamp-patrol" {
		t.Errorf("expected swamp-patrol updated, got %v", updated)
	}
	if j.Active["swamp-patrol"].KillProgress["leech"] != 1 {
		t.Error("kill count should be 1 after one RecordKill")
	}
}

func TestRecordKillNoMatch(t *testing.T) {
	b := starterBank()
	j := NewJournal()
	q, _ := b.Get("swamp-patrol")
	j.Accept(q)
	updated := j.RecordKill("wolf")
	if len(updated) != 0 {
		t.Errorf("expected no update for non-matching mob, got %v", updated)
	}
}

func TestIsMetKillsNotDone(t *testing.T) {
	b := starterBank()
	j := NewJournal()
	q, _ := b.Get("swamp-patrol")
	j.Accept(q)
	ok, err := j.IsMet("swamp-patrol", map[string]int{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("should not be met with 0 kills")
	}
}

func TestIsMetKillsDone(t *testing.T) {
	b := starterBank()
	j := NewJournal()
	q, _ := b.Get("swamp-patrol")
	j.Accept(q)
	for i := 0; i < 5; i++ {
		j.RecordKill("leech")
	}
	ok, err := j.IsMet("swamp-patrol", map[string]int{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("should be met after 5 leech kills")
	}
}

func TestIsMetItemsNotOwned(t *testing.T) {
	b := starterBank()
	j := NewJournal()
	q, _ := b.Get("gather-iron-ore")
	j.Accept(q)
	ok, err := j.IsMet("gather-iron-ore", map[string]int{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("should not be met with 0 iron ore")
	}
}

func TestIsMetItemsSufficient(t *testing.T) {
	b := starterBank()
	j := NewJournal()
	q, _ := b.Get("gather-iron-ore")
	j.Accept(q)
	ok, err := j.IsMet("gather-iron-ore", map[string]int{"iron-ore": 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("should be met with 3 iron ore")
	}
}

func TestIsMetNotActive(t *testing.T) {
	j := NewJournal()
	_, err := j.IsMet("swamp-patrol", map[string]int{})
	if err != ErrNotActive {
		t.Errorf("expected ErrNotActive got %v", err)
	}
}

func TestTurnInSuccess(t *testing.T) {
	b := starterBank()
	j := NewJournal()
	q, _ := b.Get("gather-iron-ore")
	j.Accept(q)
	res, err := j.TurnIn(b, "gather-iron-ore", map[string]int{"iron-ore": 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Flow != 100 {
		t.Errorf("expected 100 flow, got %d", res.Flow)
	}
	if res.Item != "" {
		t.Errorf("expected no item reward, got %q", res.Item)
	}
	if !j.Completed["gather-iron-ore"] {
		t.Error("quest should be marked complete after turn-in")
	}
	if _, still := j.Active["gather-iron-ore"]; still {
		t.Error("quest should be removed from active after turn-in")
	}
}

func TestTurnInItemReward(t *testing.T) {
	b := starterBank()
	j := NewJournal()
	q, _ := b.Get("defeat-king-worm")
	j.Accept(q)
	j.RecordKill("king-worm")
	res, err := j.TurnIn(b, "defeat-king-worm", map[string]int{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Item != "iron-sword" {
		t.Errorf("expected iron-sword item reward, got %q", res.Item)
	}
}

func TestTurnInFameReward(t *testing.T) {
	b := starterBank()
	j := NewJournal()
	q, _ := b.Get("gather-iron-ore")
	j.Accept(q)
	res, err := j.TurnIn(b, "gather-iron-ore", map[string]int{"iron-ore": 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.RewardFame != 30 {
		t.Errorf("expected 30 fame, got %d", res.RewardFame)
	}
	if res.FameNation != 1 {
		t.Errorf("expected FameNation=1 (Sandoria), got %d", res.FameNation)
	}
}

func TestTurnInNotActive(t *testing.T) {
	b := starterBank()
	j := NewJournal()
	_, err := j.TurnIn(b, "swamp-patrol", map[string]int{})
	if err != ErrNotActive {
		t.Errorf("expected ErrNotActive got %v", err)
	}
}

func TestTurnInNotComplete(t *testing.T) {
	b := starterBank()
	j := NewJournal()
	q, _ := b.Get("swamp-patrol")
	j.Accept(q)
	// Only 2 kills, need 5
	j.RecordKill("leech")
	j.RecordKill("leech")
	_, err := j.TurnIn(b, "swamp-patrol", map[string]int{})
	if err != ErrNotComplete {
		t.Errorf("expected ErrNotComplete got %v", err)
	}
}

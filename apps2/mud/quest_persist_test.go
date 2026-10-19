package main

import (
	"path/filepath"
	"testing"

	"dragonsnshit/server/quest"
)

// A quest accepted and part-way through must survive a "logout" (save), a process restart (a new
// store reading the same file), and still be completable afterwards.
func TestQuestJournalSurvivesLogoutAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quests.json")
	q, err := questBank.Get("owl-10-mousewright")
	if err != nil {
		t.Fatal(err)
	}
	pellets, _ := questBank.Get("owl-10-pellets")

	s1 := newQuestStore(path)
	j := s1.load("char-1")
	if len(j.Active) != 0 {
		t.Fatalf("new character should start with an empty journal")
	}
	if err := j.Accept(q); err != nil {
		t.Fatal(err)
	}
	if err := j.Accept(pellets); err != nil {
		t.Fatal(err)
	}
	inv := map[string]int{"owl-pellet": 4}
	if res, err := j.TurnIn(questBank, "owl-10-pellets", inv); err != nil || res.Flow == 0 {
		t.Fatalf("turn-in: %v %+v", err, res)
	}
	s1.save("char-1", j)

	s2 := newQuestStore(path) // simulated server restart
	got := s2.load("char-1")
	if !got.Completed["owl-10-pellets"] {
		t.Errorf("completed quest forgotten: %+v", got.Completed)
	}
	if _, ok := got.Active["owl-10-mousewright"]; !ok {
		t.Fatalf("active quest forgotten: %+v", got.Active)
	}
	if other := s2.load("char-2"); len(other.Active) != 0 || len(other.Completed) != 0 {
		t.Errorf("journals leaked between characters")
	}
	got.RecordKill("Barnaby Mousewright")
	if ok, _ := got.IsMet("owl-10-mousewright", map[string]int{}); !ok {
		t.Errorf("restored quest should be completable after the kill")
	}
	if _, err := got.TurnIn(questBank, "owl-10-mousewright", map[string]int{}); err != nil {
		t.Errorf("turn-in after restore: %v", err)
	}
	var _ *quest.Journal = got
}

// Drives the real command handlers for an IDUNA-bound slot: accept -> (restart) -> quest is back.
// A guest slot (no charIDBySlot entry) must not write anything.
func TestQuestCommandsPersistForBoundCharactersOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quests.json")
	oldStore, oldGW := mudQuestStore, gw
	mudQuestStore = newQuestStore(path)
	gw = &world{charIDBySlot: map[string]string{"bound": "char-9"}}
	t.Cleanup(func() { mudQuestStore, gw = oldStore, oldGW })

	bound := newTestPlayer()
	bound.slot = "bound"
	bound.questJournal = mudQuestStore.load("char-9")
	cmdQuestAccept(bound, "owl-10-mousewright")

	guest := newTestPlayer()
	guest.slot = "guest"
	guest.questJournal = quest.NewJournal()
	cmdQuestAccept(guest, "owl-10-pellets")

	restarted := newQuestStore(path)
	if _, ok := restarted.load("char-9").Active["owl-10-mousewright"]; !ok {
		t.Errorf("bound character's accepted quest was not persisted")
	}
	if len(restarted.data) != 1 {
		t.Errorf("guest quests must not be persisted, store has %d entries", len(restarted.data))
	}
}

package main

import (
	"strings"
	"testing"

	"dragonsnshit/server/quest"
)

func newTestJournal() *quest.Journal { return quest.NewJournal() }

// Every quest this project adds must have a real giver standing in the world and a reward that is a
// real row of the item database.
func TestAllAddedQuestsAreWellFormed(t *testing.T) {
	slugs := map[string]bool{}
	for _, it := range loadRawItems(t) {
		slugs[slugOf(it.Name)] = true
	}
	prefixes := []string{"owl-", "reach-", "start-", "bounty-"}
	n := 0
	for _, q := range questBank.All() {
		ours := false
		for _, p := range prefixes {
			ours = ours || strings.HasPrefix(q.ID, p)
		}
		if !ours {
			continue
		}
		n++
		if npcByID(q.GiverNPCID) == nil {
			t.Errorf("%s: giver %q does not exist", q.ID, q.GiverNPCID)
		}
		if !slugs[q.RewardItem] {
			t.Errorf("%s: reward %q is not in the item database", q.ID, q.RewardItem)
		}
		if len(q.RequireItems) == 0 && len(q.RequireKills) == 0 {
			t.Errorf("%s asks for nothing", q.ID)
		}
		for it := range q.RequireItems {
			if !slugs[it] && !strings.HasPrefix(it, "fish-") {
				t.Errorf("%s: required item %q is not a real item", q.ID, it)
			}
		}
		if q.RewardFlow <= 0 {
			t.Errorf("%s pays nothing", q.ID)
		}
	}
	if n < 80 {
		t.Errorf("expected a rich quest list, found %d added quests", n)
	}
	// every zone with an NPC gives at least one quest
	for _, z := range []int{0, 1, 2, 3, 4, 10, 11, 12, 13, 14, 15, 16, 17, 18, 20, 21, 22, 23, 24, 25, 26, 28, 29, 30, 31, 200, 201, 202, 203, 204, 205, 206, 207} {
		found := false
		for i := range npcs {
			if npcs[i].ZoneID != z {
				continue
			}
			if len(questBank.ForNPC(npcs[i].ID)) > 0 {
				found = true
			}
		}
		if !found {
			t.Errorf("zone %d has no NPC that gives a quest", z)
		}
	}
}

func TestBountiesTargetTheRealHNMs(t *testing.T) {
	names := map[string]bool{}
	for _, d := range hnmDefs {
		names[d.Name] = true
	}
	count := 0
	for _, q := range questBank.All() {
		if !strings.HasPrefix(q.ID, "bounty-") {
			continue
		}
		count++
		for kind := range q.RequireKills {
			if !names[kind] {
				t.Errorf("%s targets %q which is not an HNM", q.ID, kind)
			}
		}
	}
	if count != len(hnmDefs) {
		t.Errorf("want one bounty per HNM (%d), got %d", len(hnmDefs), count)
	}
}

// Handing in a quest takes the items it asked for.
func TestTurningInAQuestTakesTheItems(t *testing.T) {
	p, _ := craftWorld(t)
	p.questJournal = newTestJournal()
	p.inventory["copper-ingot"] = 4
	cmdQuestAccept(p, "start-elder-ingots")
	cmdQuestTurnIn(p, "start-elder-ingots")
	if p.inventory["copper-ingot"] != 1 {
		t.Errorf("3 ingots should have been taken, %d left", p.inventory["copper-ingot"])
	}
	if p.inventory["copper-knife"] != 1 {
		t.Errorf("reward not granted: %v", p.inventory)
	}
	// a kill quest takes nothing
	p.inventory["x"] = 2
	cmdQuestAccept(p, "start-elder-worms")
	for i := 0; i < 8; i++ {
		p.questJournal.RecordKill("worm")
	}
	cmdQuestTurnIn(p, "start-elder-worms")
	if p.inventory["x"] != 2 || p.inventory["iron-shortsword"] != 1 {
		t.Errorf("kill quest reward/side effects wrong: %v", p.inventory)
	}
}

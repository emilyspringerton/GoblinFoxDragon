package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"dragonsnshit/server/quest"
)

// questStore persists each character's quest journal (active quests with kill/item progress, plus
// completed quests) to var/mud-quests.json, keyed by IDUNA character_id. Before this existed the
// journal was rebuilt empty with quest.NewJournal() on every login, so all quest progress was lost
// on logout. var/ is the PVC mount in the gfd image, so this survives redeploys like
// var/mud-chars.json does. Guests (no character_id) are deliberately not persisted.
type questStore struct {
	path string
	mu   sync.Mutex
	data map[string]*quest.Journal
}

var mudQuestStore = newQuestStore("var/mud-quests.json")

func newQuestStore(path string) *questStore {
	s := &questStore{path: path, data: make(map[string]*quest.Journal)}
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &s.data)
	}
	return s
}

// load returns a deep copy of the character's saved journal, or a fresh empty one.
func (s *questStore) load(charID string) *quest.Journal {
	s.mu.Lock()
	defer s.mu.Unlock()
	j := quest.NewJournal()
	saved, ok := s.data[charID]
	if !ok || saved == nil {
		return j
	}
	raw, err := json.Marshal(saved)
	if err != nil || json.Unmarshal(raw, j) != nil {
		return quest.NewJournal()
	}
	if j.Active == nil {
		j.Active = make(map[string]*quest.State)
	}
	if j.Completed == nil {
		j.Completed = make(map[string]bool)
	}
	return j
}

// save records the journal for charID and writes the file atomically. Best-effort: a write
// failure must never block gameplay or a disconnect.
func (s *questStore) save(charID string, j *quest.Journal) {
	if charID == "" || j == nil {
		return
	}
	raw, err := json.Marshal(j)
	if err != nil {
		return
	}
	snap := quest.NewJournal()
	if json.Unmarshal(raw, snap) != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[charID] = snap
	out, err := json.Marshal(s.data)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.path), 0755)
	tmp := s.path + ".tmp"
	if os.WriteFile(tmp, out, 0644) == nil {
		_ = os.Rename(tmp, s.path)
	}
}

// saveQuestJournal persists p's journal if p is bound to a real character. Callers already hold
// gw.mu (command handlers do), which is what makes the charIDBySlot read safe.
func saveQuestJournal(p *player) {
	if gw == nil || p == nil {
		return
	}
	if charID, ok := gw.charIDBySlot[p.slot]; ok {
		mudQuestStore.save(charID, p.questJournal)
	}
}

// Package quest implements the NPC quest system for DragonsNShit MUD.
//
// NPCs offer quests. Players accept quests, progress through kill/item requirements,
// then turn in for gil and item rewards.
package quest

import "errors"

// Quest is a quest definition, authored at startup.
type Quest struct {
	ID           string
	Title        string
	Desc         string
	GiverNPCID   string
	RequireItems map[string]int // itemID → quantity
	RequireKills map[string]int // mob kind → count
	RewardGil    int
	RewardItem   string // "" if no item reward
}

// State is one player's progress through a quest.
type State struct {
	QuestID      string
	RequireKills map[string]int // original kill requirements (static)
	RequireItems map[string]int // original item requirements (static)
	KillProgress map[string]int // mob kind → kills so far
	ItemProgress map[string]int // itemID → quantity collected
	Complete     bool
}

// Errors.
var (
	ErrUnknownQuest = errors.New("unknown quest")
	ErrAlreadyActive = errors.New("quest already active")
	ErrAlreadyDone   = errors.New("quest already completed")
	ErrNotActive     = errors.New("quest not active")
	ErrNotComplete   = errors.New("quest requirements not met")
)

// StarterQuests are the 5 starter quests available from NPCs.
var StarterQuests = []*Quest{
	{
		ID:           "gather-iron-ore",
		Title:        "Gather Iron Ore",
		Desc:         "Bring 3 Iron Ore to the Guildmaster in the Meadow.",
		GiverNPCID:   "guildmaster",
		RequireItems: map[string]int{"iron-ore": 3},
		RequireKills: map[string]int{},
		RewardGil:    100,
		RewardItem:   "",
	},
	{
		ID:           "defeat-king-worm",
		Title:        "Defeat the King Worm",
		Desc:         "Hunt down the King Worm NM in the Swamp.",
		GiverNPCID:   "guildmaster",
		RequireItems: map[string]int{},
		RequireKills: map[string]int{"king-worm": 1},
		RewardGil:    500,
		RewardItem:   "iron-sword",
	},
	{
		ID:           "merchants-request",
		Title:        "The Merchant's Request",
		Desc:         "Bring 5 Fire Crystals to the Merchant in the Hills.",
		GiverNPCID:   "merchant",
		RequireItems: map[string]int{"fire-crystal": 5},
		RequireKills: map[string]int{},
		RewardGil:    200,
		RewardItem:   "",
	},
	{
		ID:           "swamp-patrol",
		Title:        "Swamp Patrol",
		Desc:         "Exterminate 5 leeches in the Swamp.",
		GiverNPCID:   "scout",
		RequireItems: map[string]int{},
		RequireKills: map[string]int{"leech": 5},
		RewardGil:    300,
		RewardItem:   "leather-belt",
	},
	{
		ID:           "crisis-volunteer",
		Title:        "Crisis Volunteer",
		Desc:         "Collect a Crisis Shard from the World Crisis event and bring it to the Scout.",
		GiverNPCID:   "scout",
		RequireItems: map[string]int{"crisis-shard": 1},
		RequireKills: map[string]int{},
		RewardGil:    400,
		RewardItem:   "echo-drop",
	},
}

// Bank is a indexed set of quest definitions, keyed by quest ID.
type Bank struct {
	quests map[string]*Quest
}

// NewBank creates a Bank from a slice of quest definitions.
func NewBank(quests []*Quest) *Bank {
	b := &Bank{quests: make(map[string]*Quest, len(quests))}
	for _, q := range quests {
		b.quests[q.ID] = q
	}
	return b
}

// Get returns the Quest for the given ID, or ErrUnknownQuest.
func (b *Bank) Get(id string) (*Quest, error) {
	q, ok := b.quests[id]
	if !ok {
		return nil, ErrUnknownQuest
	}
	return q, nil
}

// All returns all quests in the bank.
func (b *Bank) All() []*Quest {
	out := make([]*Quest, 0, len(b.quests))
	for _, q := range b.quests {
		out = append(out, q)
	}
	return out
}

// ForNPC returns all quests whose GiverNPCID matches npcID.
func (b *Bank) ForNPC(npcID string) []*Quest {
	var out []*Quest
	for _, q := range b.quests {
		if q.GiverNPCID == npcID {
			out = append(out, q)
		}
	}
	return out
}

// Journal tracks all quest states for one player.
type Journal struct {
	Active    map[string]*State // questID → in-progress state
	Completed map[string]bool   // questID → done
}

// NewJournal creates an empty Journal.
func NewJournal() *Journal {
	return &Journal{
		Active:    make(map[string]*State),
		Completed: make(map[string]bool),
	}
}

// Accept adds a quest to the Active map. Returns ErrAlreadyActive or ErrAlreadyDone if
// the quest has already been accepted or completed.
func (j *Journal) Accept(q *Quest) error {
	if j.Completed[q.ID] {
		return ErrAlreadyDone
	}
	if _, ok := j.Active[q.ID]; ok {
		return ErrAlreadyActive
	}
	kills := make(map[string]int, len(q.RequireKills))
	items := make(map[string]int, len(q.RequireItems))
	for k := range q.RequireKills {
		kills[k] = 0
	}
	for k := range q.RequireItems {
		items[k] = 0
	}
	j.Active[q.ID] = &State{
		QuestID:      q.ID,
		RequireKills: q.RequireKills,
		RequireItems: q.RequireItems,
		KillProgress: kills,
		ItemProgress: items,
	}
	return nil
}

// RecordKill increments kill progress for the given mob kind across all active quests
// that require it. Returns the list of quest IDs that had their kill count updated.
func (j *Journal) RecordKill(mobKind string) []string {
	var updated []string
	for id, st := range j.Active {
		if _, needed := st.KillProgress[mobKind]; needed {
			st.KillProgress[mobKind]++
			updated = append(updated, id)
		}
	}
	return updated
}

// RecordItem updates item progress for the given item across all active quests.
// inventory is a map of itemID → quantity the player currently holds.
func (j *Journal) RecordItem(itemID string, qty int) []string {
	var updated []string
	for id, st := range j.Active {
		if needed, ok := st.ItemProgress[itemID]; ok {
			if qty < needed {
				st.ItemProgress[itemID] = qty
			} else {
				st.ItemProgress[itemID] = qty
			}
			updated = append(updated, id)
		}
	}
	return updated
}

// IsMet returns true if the player's active quest state satisfies all requirements.
// inventory is the player's current item quantities.
func (j *Journal) IsMet(questID string, inventory map[string]int) (bool, error) {
	st, ok := j.Active[questID]
	if !ok {
		return false, ErrNotActive
	}
	for kind, need := range st.RequireKills {
		if st.KillProgress[kind] < need {
			return false, nil
		}
	}
	for itemID, need := range st.RequireItems {
		if inventory[itemID] < need {
			return false, nil
		}
	}
	return true, nil
}

// TurnIn completes a quest. Returns ErrNotActive, ErrNotComplete, or nil on success.
// On success, the quest is moved to Completed and removed from Active.
// Returns (rewardGil, rewardItem).
func (j *Journal) TurnIn(b *Bank, questID string, inventory map[string]int) (gil int, item string, err error) {
	q, qErr := b.Get(questID)
	if qErr != nil {
		return 0, "", qErr
	}
	st, ok := j.Active[questID]
	if !ok {
		return 0, "", ErrNotActive
	}
	// Check kills.
	for kind, need := range q.RequireKills {
		if st.KillProgress[kind] < need {
			return 0, "", ErrNotComplete
		}
	}
	// Check items.
	for itemID, need := range q.RequireItems {
		if inventory[itemID] < need {
			return 0, "", ErrNotComplete
		}
	}
	j.Completed[questID] = true
	delete(j.Active, questID)
	return q.RewardGil, q.RewardItem, nil
}

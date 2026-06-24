// Package campaign implements the DragonsNShit weekly Campaign Battle event.
//
// # Campaign Battle model
//
// Once per week, all three nations contest 10 campaign nodes spread across
// the 8 TRAPX scenes. Each node has a holder nation and HP. Players join the
// campaign for their faction; every 10 combat kills in a node's scene shifts
// node HP by -1 (favouring the attacker) or +1 (defending the holder).
//
// At cycle end (24h), the nation with the most nodes wins the week and earns
// a prestige bonus via the conquest system. Individual winners earn a
// campaign medal.
package campaign

import (
	"errors"
	"sync"
	"time"

	"dragonsnshit/server/conquest"
)

const (
	// DefaultNodeHP is the starting HP for each campaign node.
	DefaultNodeHP = 100
	// KillsPerHPShift is the number of kills required to shift a node's HP by 1.
	KillsPerHPShift = 10
	// DefaultCycleDuration is 24h.
	DefaultCycleDuration = 24 * time.Hour
	// NodeCount is the number of campaign nodes per event.
	NodeCount = 10
)

// CampaignNode is one contested location in a Campaign Battle cycle.
type CampaignNode struct {
	ID       string         // unique node ID
	SceneID  int            // TRAPX scene this node belongs to
	Holder   conquest.Nation // current holding nation (changes when HP reaches 0)
	HP       int            // node HP; reduced by attackers, restored on capture
	killsBuf map[conquest.Nation]int // buffered kills since last HP shift
}

// Battle is one active Campaign Battle cycle. Thread-safe.
type Battle struct {
	mu        sync.RWMutex
	nodes     map[string]*CampaignNode // node ID → node
	enrolled  map[string]conquest.Nation // player slot → faction
	startsAt  time.Time
	endsAt    time.Time
	active    bool
	cycleWinner conquest.Nation // set when cycle ends; NationNeutral = tie/pending
}

// NewBattle creates a Campaign Battle starting at startsAt with the given cycle duration.
func NewBattle(startsAt time.Time, duration time.Duration) *Battle {
	b := &Battle{
		nodes:    make(map[string]*CampaignNode),
		enrolled: make(map[string]conquest.Nation),
		startsAt: startsAt,
		endsAt:   startsAt.Add(duration),
		active:   false,
	}
	return b
}

// AddNode registers a campaign node. Must be called before the battle starts.
func (b *Battle) AddNode(n *CampaignNode) {
	if n.HP == 0 {
		n.HP = DefaultNodeHP
	}
	if n.killsBuf == nil {
		n.killsBuf = make(map[conquest.Nation]int)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nodes[n.ID] = n
}

// Enroll registers a player (by slot) for their faction. Non-fatal if already enrolled.
func (b *Battle) Enroll(slot string, nation conquest.Nation) error {
	if nation == conquest.NationNeutral {
		return errors.New("campaign: neutral players cannot join campaign battle")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.enrolled[slot] = nation
	return nil
}

// IsEnrolled reports whether the player is enrolled.
func (b *Battle) IsEnrolled(slot string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, ok := b.enrolled[slot]
	return ok
}

// RecordKill records that an enrolled player in sceneID killed one combat mob.
// Every KillsPerHPShift kills by a non-holder nation reduces the node's HP by 1.
// When HP hits 0, the attacker captures the node (HP resets to DefaultNodeHP).
// Holder kills restore 1 HP (up to DefaultNodeHP).
// Returns the node ID that changed, or "" if no change.
func (b *Battle) RecordKill(slot string, sceneID int) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	nation, enrolled := b.enrolled[slot]
	if !enrolled {
		return ""
	}
	// Find node in this scene.
	var node *CampaignNode
	for _, n := range b.nodes {
		if n.SceneID == sceneID {
			node = n
			break
		}
	}
	if node == nil {
		return ""
	}
	node.killsBuf[nation]++
	if node.killsBuf[nation] < KillsPerHPShift {
		return ""
	}
	node.killsBuf[nation] = 0

	if nation == node.Holder {
		// Holder defends: restore HP.
		node.HP++
		if node.HP > DefaultNodeHP {
			node.HP = DefaultNodeHP
		}
	} else {
		// Attacker presses.
		node.HP--
		if node.HP <= 0 {
			// Capture: attacker takes the node.
			node.Holder = nation
			node.HP = DefaultNodeHP
		}
	}
	return node.ID
}

// Tick should be called periodically. If endsAt has passed, it resolves the cycle.
// Returns (true, CycleResult) if the cycle just ended; (false, zero) otherwise.
func (b *Battle) Tick(now time.Time) (ended bool, result CycleResult) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.active && !now.Before(b.endsAt) {
		b.active = false
		result = b.computeResult()
		b.cycleWinner = result.Winner
		return true, result
	}
	if !b.active && !now.Before(b.startsAt) && now.Before(b.endsAt) {
		b.active = true
	}
	return false, CycleResult{}
}

// IsActive reports whether the battle is currently in progress.
func (b *Battle) IsActive() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.active
}

// NodeCount returns the number of registered nodes.
func (b *Battle) NodeCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.nodes)
}

// NodeSnapshot returns a copy of the node for read-only display.
func (b *Battle) NodeSnapshot(nodeID string) (CampaignNode, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	n, ok := b.nodes[nodeID]
	if !ok {
		return CampaignNode{}, false
	}
	return *n, true
}

// AllNodes returns a snapshot of all nodes.
func (b *Battle) AllNodes() []CampaignNode {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]CampaignNode, 0, len(b.nodes))
	for _, n := range b.nodes {
		out = append(out, *n)
	}
	return out
}

// CycleResult summarises the outcome of a completed Campaign Battle.
type CycleResult struct {
	Winner     conquest.Nation          // nation with most nodes; NationNeutral = tie
	NodesByNation map[conquest.Nation]int // nation → node count at cycle end
	Participants  int                     // total enrolled players
}

func (b *Battle) computeResult() CycleResult {
	counts := map[conquest.Nation]int{}
	for _, n := range b.nodes {
		if n.Holder != conquest.NationNeutral {
			counts[n.Holder]++
		}
	}
	best := conquest.NationNeutral
	bestCount := -1
	tie := false
	for nation, count := range counts {
		if count > bestCount {
			best = nation
			bestCount = count
			tie = false
		} else if count == bestCount {
			tie = true
		}
	}
	if tie {
		best = conquest.NationNeutral
	}
	return CycleResult{
		Winner:        best,
		NodesByNation: counts,
		Participants:  len(b.enrolled),
	}
}

// DefaultNodes returns the 10 default campaign nodes spread across 8 TRAPX scenes (0-7).
func DefaultNodes() []*CampaignNode {
	// Scenes 0-7 (TRAPX); nodes 0-9 distributed: scenes 0,1,2,3 get 2 nodes each;
	// scenes 4-7 get 0 (contested territory concentrated in core zones).
	// Holder starts as NationNeutral (unclaimed).
	return []*CampaignNode{
		{ID: "cn-0a", SceneID: 0, Holder: conquest.NationNeutral, HP: DefaultNodeHP},
		{ID: "cn-0b", SceneID: 0, Holder: conquest.NationNeutral, HP: DefaultNodeHP},
		{ID: "cn-1a", SceneID: 1, Holder: conquest.NationNeutral, HP: DefaultNodeHP},
		{ID: "cn-1b", SceneID: 1, Holder: conquest.NationNeutral, HP: DefaultNodeHP},
		{ID: "cn-2a", SceneID: 2, Holder: conquest.NationNeutral, HP: DefaultNodeHP},
		{ID: "cn-2b", SceneID: 2, Holder: conquest.NationNeutral, HP: DefaultNodeHP},
		{ID: "cn-3a", SceneID: 3, Holder: conquest.NationNeutral, HP: DefaultNodeHP},
		{ID: "cn-3b", SceneID: 3, Holder: conquest.NationNeutral, HP: DefaultNodeHP},
		{ID: "cn-4a", SceneID: 4, Holder: conquest.NationNeutral, HP: DefaultNodeHP},
		{ID: "cn-5a", SceneID: 5, Holder: conquest.NationNeutral, HP: DefaultNodeHP},
	}
}

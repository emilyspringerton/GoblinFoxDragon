// Package npcattention implements per-NPC stealth awareness for disguise gameplay.
//
// Inspired by Hitman: Codename 47 / Blood Money mechanics.
//
// # Core model
//
// Every NPC has a [NPCWatch] per player in the same scene. The watch tracks
// a Suspicion value [0, 100]. Suspicion climbs when the NPC has line-of-sight
// to a player in a compromising state; it decays when the player is out of
// sight or fully accepted (matching disguise, not running, no weapon).
//
// Suspicion crosses three thresholds:
//
//	SuspiciousAt (35): NPC narrows eyes, comments, does not yet act.
//	AlertedAt    (65): NPC moves to investigate or calls for a guard.
//	HostileAt   (100): NPC draws weapon / shouts for backup.
//
// # Disguise system
//
// A player can equip a [Disguise] (guard uniform, civilian clothes, target
// staff livery, etc.). Each disguise has a faction string. NPCs also carry a
// faction string. When the player's disguise faction matches the NPC's
// faction, the NPC is an "enforcer": they see through the disguise much faster
// than outsiders do. A civilian in a guard uniform passes guards; another guard
// will spot them eventually.
//
// # Witness system
//
// If a player kills or knocks out someone while this NPC has LOS, the NPC
// becomes a Witness. Witnesses jump directly to AlertedAt and will call for
// backup unless silenced. Eliminating a witness (SilenceWitness) removes the
// witness flag and resets suspicion.
//
// # Tick loop
//
// The game server calls [Scene.Tick] each server tick with the current state
// of all players in the scene. Tick returns [AwarenessEvent] slices describing
// threshold crossings. The server converts these to game effects (NPC speech,
// backup calls, hostile state changes).
//
// Note: this package shares no code with server/attention (Field Office macro
// attention). They are orthogonal systems.
package npcattention

import (
	"fmt"
	"math"
	"time"
)

// ── Awareness ─────────────────────────────────────────────────────────────────

// Awareness is the NPC's current state toward a specific player.
type Awareness int

const (
	AwarenessUnaware    Awareness = iota // no knowledge of the player
	AwarenessSuspicious                  // noticed; building a case; doesn't act yet
	AwarenessAlerted                     // actively investigating; may call backup
	AwarenessHostile                     // draws weapon; calls for help
)

func (a Awareness) String() string {
	switch a {
	case AwarenessUnaware:
		return "unaware"
	case AwarenessSuspicious:
		return "suspicious"
	case AwarenessAlerted:
		return "alerted"
	case AwarenessHostile:
		return "hostile"
	default:
		return "unknown"
	}
}

const (
	SuspiciousAt = 35.0
	AlertedAt    = 65.0
	HostileAt    = 100.0

	// Decay rate (suspicion units / second) when player is out of LOS or safe.
	BaseDecayRate = 4.0

	// Suspicion gain multipliers.
	GainNoDisguise        = 25.0 // per second, no disguise at all
	GainWrongFaction      = 15.0 // per second, wearing wrong faction's disguise
	GainSameFaction       = 8.0  // per second, enforcer effect (same faction as disguise)
	GainRunning           = 4.0  // added on top of base gain per second
	GainVisibleWeapon     = 6.0  // added on top of base gain per second
	GainBodyNearby        = 20.0 // one-time spike per body in LOS

	// Witness suspicion floor: witnesses always stay at AlertedAt or above.
	WitnessFloor = AlertedAt
)

// ── Disguise ──────────────────────────────────────────────────────────────────

// Faction is a named group (guard, civilian, target-staff, crime-boss, etc.).
// The empty string means no faction / no disguise.
type Faction string

const (
	FactionNone      Faction = ""
	FactionCivilian  Faction = "civilian"
	FactionGuard     Faction = "guard"
	FactionCrimeBoss Faction = "crime-boss"
	FactionCultist   Faction = "cultist"
	FactionServant   Faction = "servant"
	FactionMerchant  Faction = "merchant"
)

// Disguise is the player's current identity.
type Disguise struct {
	Faction        Faction
	WeaponVisible  bool // carrying large weapon visibly (pistol concealed = false)
	Running        bool // player is currently running
}

// AcceptedBy reports whether this disguise would be accepted at face value by
// an NPC of npcFaction. Returns true if the NPC would not gain extra suspicion.
// Even when accepted, close proximity over time still builds slow suspicion.
func (d Disguise) AcceptedBy(npcFaction Faction) bool {
	if d.Faction == FactionNone {
		return false // no disguise = never accepted
	}
	// Same faction = enforcer; NOT accepted (they know their own colleagues)
	return d.Faction != npcFaction
}

// ── AwarenessEvent ────────────────────────────────────────────────────────────

// EventVerb classifies an awareness event.
type EventVerb string

const (
	VerbBecameSuspicious EventVerb = "SUSPICIOUS"    // crossed SuspiciousAt
	VerbBecameAlerted    EventVerb = "ALERTED"        // crossed AlertedAt
	VerbBecameHostile    EventVerb = "HOSTILE"        // reached HostileAt
	VerbCalmedDown       EventVerb = "CALMED"         // dropped back below SuspiciousAt
	VerbWitnessAlerted   EventVerb = "WITNESS_ALERTED" // saw a crime; hard alerted
)

// AwarenessEvent is fired when an NPC crosses an awareness threshold for a player.
type AwarenessEvent struct {
	NPCID    string
	PlayerID string
	Verb     EventVerb
	Suspicion float64
	At       time.Time
}

func (e AwarenessEvent) String() string {
	return fmt.Sprintf("[%s] npc:%s → player:%s %s (%.0f)",
		e.At.Format("15:04:05"), e.NPCID, e.PlayerID, e.Verb, e.Suspicion)
}

// ── NPCWatch ──────────────────────────────────────────────────────────────────

// NPCWatch tracks one NPC's suspicion toward one player.
type NPCWatch struct {
	PlayerID  string
	Suspicion float64   // [0, 100]
	Awareness Awareness
	IsWitness bool      // saw a crime; won't calm below WitnessFloor
	LastSeen  time.Time // zero = never seen
}

// ── NPC ───────────────────────────────────────────────────────────────────────

// NPC is a single non-player character with stealth awareness state.
// Not goroutine-safe; callers synchronise via server tick loop.
type NPC struct {
	ID      string
	Faction Faction
	SceneID int

	// watches maps playerID → watch state.
	watches map[string]*NPCWatch
}

// NewNPC creates a new NPC.
func NewNPC(id string, faction Faction, sceneID int) *NPC {
	return &NPC{
		ID:      id,
		Faction: faction,
		SceneID: sceneID,
		watches: make(map[string]*NPCWatch),
	}
}

// watchFor returns (or creates) the watch state for a player.
func (n *NPC) watchFor(playerID string) *NPCWatch {
	w, ok := n.watches[playerID]
	if !ok {
		w = &NPCWatch{PlayerID: playerID}
		n.watches[playerID] = w
	}
	return w
}

// Tick advances this NPC's awareness state for one player.
//
// hasLOS must be true if the NPC can see the player this tick.
// disguise describes the player's current state.
// bodiesInLOS is the count of bodies the NPC can see near the player this tick.
// dt is the elapsed seconds since the last tick.
func (n *NPC) Tick(
	playerID string,
	hasLOS bool,
	disguise Disguise,
	bodiesInLOS int,
	dt float64,
	now time.Time,
) []AwarenessEvent {
	w := n.watchFor(playerID)
	prevAwareness := w.Awareness

	if !hasLOS {
		// Out of sight — decay toward 0 (or witness floor if applicable).
		floor := 0.0
		if w.IsWitness {
			floor = WitnessFloor
		}
		w.Suspicion = math.Max(w.Suspicion-BaseDecayRate*dt, floor)
	} else {
		w.LastSeen = now
		gain := n.computeGain(disguise) * dt
		// One-time body spike (already per-tick; callers should gate this to avoid spam)
		if bodiesInLOS > 0 {
			gain += GainBodyNearby * float64(bodiesInLOS)
		}
		w.Suspicion = math.Min(w.Suspicion+gain, HostileAt)
	}

	// Update awareness state.
	switch {
	case w.Suspicion >= HostileAt:
		w.Awareness = AwarenessHostile
	case w.Suspicion >= AlertedAt:
		w.Awareness = AwarenessAlerted
	case w.Suspicion >= SuspiciousAt:
		w.Awareness = AwarenessSuspicious
	default:
		w.Awareness = AwarenessUnaware
	}

	return n.emitEvents(w, prevAwareness, now)
}

// computeGain calculates the suspicion gain per second for the given disguise.
func (n *NPC) computeGain(d Disguise) float64 {
	var gain float64
	switch {
	case d.Faction == FactionNone:
		gain = GainNoDisguise
	case d.Faction == n.Faction:
		// Same faction: enforcer — suspicious but not immediately hostile.
		gain = GainSameFaction
	case d.AcceptedBy(n.Faction):
		// Different-faction disguise accepted by this NPC — low passive gain.
		gain = 1.0
	default:
		gain = GainWrongFaction
	}
	if d.Running {
		gain += GainRunning
	}
	if d.WeaponVisible {
		gain += GainVisibleWeapon
	}
	return gain
}

// WitnessKill marks this NPC as having witnessed the player commit a crime.
// Immediately jumps suspicion to WitnessFloor and sets IsWitness = true.
func (n *NPC) WitnessKill(playerID string, now time.Time) []AwarenessEvent {
	w := n.watchFor(playerID)
	prev := w.Awareness
	if w.Suspicion < WitnessFloor {
		w.Suspicion = WitnessFloor
	}
	w.IsWitness = true
	w.Awareness = AwarenessAlerted
	w.LastSeen = now

	var events []AwarenessEvent
	if prev < AwarenessAlerted {
		events = append(events, AwarenessEvent{
			NPCID: n.ID, PlayerID: playerID,
			Verb:      VerbWitnessAlerted,
			Suspicion: w.Suspicion, At: now,
		})
	}
	events = append(events, n.emitEvents(w, prev, now)...)
	return events
}

// SilenceWitness removes the witness flag (NPC was knocked out, killed, or
// bribed) and resets suspicion to 0. If this NPC was not a witness, no-op.
func (n *NPC) SilenceWitness(playerID string) {
	w, ok := n.watches[playerID]
	if !ok {
		return
	}
	w.IsWitness = false
	w.Suspicion = 0
	w.Awareness = AwarenessUnaware
}

// WatchState returns a snapshot of the watch state for playerID.
func (n *NPC) WatchState(playerID string) NPCWatch {
	if w, ok := n.watches[playerID]; ok {
		return *w
	}
	return NPCWatch{PlayerID: playerID}
}

// HighestAwareness returns the most elevated awareness state this NPC has
// toward any player, along with the playerID responsible.
func (n *NPC) HighestAwareness() (Awareness, string) {
	best := AwarenessUnaware
	bestPlayer := ""
	for pid, w := range n.watches {
		if w.Awareness > best {
			best = w.Awareness
			bestPlayer = pid
		}
	}
	return best, bestPlayer
}

func (n *NPC) emitEvents(w *NPCWatch, prev Awareness, now time.Time) []AwarenessEvent {
	if w.Awareness == prev {
		return nil
	}
	var verb EventVerb
	switch {
	case w.Awareness > prev:
		switch w.Awareness {
		case AwarenessSuspicious:
			verb = VerbBecameSuspicious
		case AwarenessAlerted:
			verb = VerbBecameAlerted
		case AwarenessHostile:
			verb = VerbBecameHostile
		}
	case w.Awareness < prev && w.Awareness == AwarenessUnaware:
		verb = VerbCalmedDown
	default:
		return nil
	}
	return []AwarenessEvent{{
		NPCID:    n.ID,
		PlayerID: w.PlayerID,
		Verb:     verb,
		Suspicion: w.Suspicion,
		At:       now,
	}}
}

// ── Scene ─────────────────────────────────────────────────────────────────────

// PlayerState describes a single player's current stealth state in the scene.
type PlayerState struct {
	PlayerID    string
	Disguise    Disguise
	SceneID     int
	BodiesInLOS int // number of bodies this player is standing near that NPCs can see
}

// LOSFunc is a callback the server provides to report whether a specific NPC
// has line-of-sight to a specific player. The server owns spatial queries;
// this package does not.
type LOSFunc func(npcID, playerID string) bool

// Scene manages all NPCs in a single scene and ticks them together.
type Scene struct {
	SceneID int
	npcs    map[string]*NPC
}

// NewScene creates an empty scene.
func NewScene(sceneID int) *Scene {
	return &Scene{SceneID: sceneID, npcs: make(map[string]*NPC)}
}

// AddNPC registers an NPC in this scene.
func (s *Scene) AddNPC(npc *NPC) { s.npcs[npc.ID] = npc }

// GetNPC returns the NPC with the given ID, or nil.
func (s *Scene) GetNPC(id string) *NPC { return s.npcs[id] }

// Tick advances all NPCs in the scene by dt seconds.
// players: all players currently in this scene.
// los: LOS oracle (the server provides spatial queries).
// Returns all awareness events generated across all NPCs.
func (s *Scene) Tick(players []PlayerState, los LOSFunc, dt float64, now time.Time) []AwarenessEvent {
	var all []AwarenessEvent
	for _, npc := range s.npcs {
		if npc.SceneID != s.SceneID {
			continue
		}
		for _, p := range players {
			if p.SceneID != s.SceneID {
				continue
			}
			hasLOS := los(npc.ID, p.PlayerID)
			events := npc.Tick(p.PlayerID, hasLOS, p.Disguise, p.BodiesInLOS, dt, now)
			all = append(all, events...)
		}
	}
	return all
}

// AlertLevel returns the highest awareness level among all NPCs toward any player.
func (s *Scene) AlertLevel() Awareness {
	best := AwarenessUnaware
	for _, npc := range s.npcs {
		a, _ := npc.HighestAwareness()
		if a > best {
			best = a
		}
	}
	return best
}

// WitnessKill broadcasts a witnessed kill to all NPCs in this scene that have
// LOS to the player at the moment of the kill.
func (s *Scene) WitnessKill(playerID string, los LOSFunc, now time.Time) []AwarenessEvent {
	var all []AwarenessEvent
	for _, npc := range s.npcs {
		if !los(npc.ID, playerID) {
			continue
		}
		events := npc.WitnessKill(playerID, now)
		all = append(all, events...)
	}
	return all
}

// WitnessCount returns the number of NPCs that are current witnesses to the given player.
func (s *Scene) WitnessCount(playerID string) int {
	n := 0
	for _, npc := range s.npcs {
		if npc.watches[playerID] != nil && npc.watches[playerID].IsWitness {
			n++
		}
	}
	return n
}

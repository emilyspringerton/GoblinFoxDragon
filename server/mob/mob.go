// Package mob implements server-authoritative mob entities for DragonsNShit.
//
// # Mob lifecycle
//
//	Spawn → Idle → [player enters aggro range] → Pursuing → [in melee] → attacking
//	                                                       → [leash exceeded] → Returning → Idle
//	Any state → [HP ≤ 0] → Dead
//
// # Tagging
//
// The first character to deal damage to a mob claims the tag. The tag is permanent
// for the mob's lifetime — subsequent hitters do not steal it. On death, the tagger
// is credited for XP and loot rights. This mirrors FFXI's claim system.
//
// # Auto-attack
//
// Both mobs and players auto-attack on a swing timer:
//   - Mob:    calls back via Tick() events; damage delivered to player slot.
//   - Player: caller maintains a PlayerCombat per slot; call TickPlayer() each server frame.
package mob

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// Pos is a 3-D world position (avoids import cycle with server/system).
type Pos struct{ X, Y, Z float64 }

func dist(a, b Pos) float64 {
	dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// Dist is the exported form of dist, for callers outside this package (e.g.
// apps2/mud's cmdAttack, which needs to know whether a player is in melee
// range before committing to an approach).
func Dist(a, b Pos) float64 { return dist(a, b) }

func moveToward(from, to Pos, speed float64) Pos {
	d := dist(from, to)
	if d <= speed || d == 0 {
		return to
	}
	t := speed / d
	return Pos{
		X: from.X + (to.X-from.X)*t,
		Y: from.Y + (to.Y-from.Y)*t,
		Z: from.Z + (to.Z-from.Z)*t,
	}
}

// MobState is the AI state of a mob.
type MobState int

const (
	StateIdle       MobState = iota
	StatePursuing            // moving toward aggro target
	StateReturning           // leashed; moving back to home position
	StateDead
	StateBurrowed // underground; untargetable until burrow expires
)

func (s MobState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StatePursuing:
		return "pursuing"
	case StateReturning:
		return "returning"
	case StateDead:
		return "dead"
	case StateBurrowed:
		return "burrowed"
	default:
		return "unknown"
	}
}

// Mob is a single server-authoritative mobile entity.
// All fields are owned by the single server goroutine.
type Mob struct {
	ID       string
	Kind     string // "skeleton", "wolf", "goblin", … -- the real identity used for loot/quest/AI lookups, never overridden by DisplayName
	// DisplayName is a real, optional player-facing name override for a difficulty-tier variant
	// of Kind (GFD-MOBSPAWN-001 Phase 4, founder real-time: "they like to have some way harder
	// mobs pretty close to lower level mobs with the same model and texture just a different
	// name"). Empty means "just show Kind," the pre-existing behavior. Deliberately does NOT
	// change Kind itself -- loot (dropsForMob), quest tracking, and AI dispatch all still key off
	// the real base Kind, so a variant inherits its base's drop table automatically rather than
	// needing its own (the founder's own real decision on this).
	DisplayName string
	SceneID  int
	Pos      Pos
	HomePos  Pos // spawn anchor; leash resets here
	HP       int
	MaxHP    int
	State    MobState
	AggroSlot   string // slot key of current aggro target; "" = none
	TaggerSlot  string // slot key of first hitter; "" = untagged
	TaggedAt    time.Time
	AggroRange  float64
	LeashRange  float64
	MeleeRange  float64
	MoveSpeed   float64 // units per second
	SwingDelay  time.Duration
	MeleeDamage int
	lastSwing   time.Time
	DiedAt      time.Time

	// Burrow fields — non-zero enables periodic burrowing (e.g. worms).
	BurrowInterval time.Duration // how often to burrow; 0 = never
	BurrowDuration time.Duration // how long to stay underground
	lastBurrow     time.Time     // when the last burrow began (zero = never burrowed)
	burrowEnd      time.Time     // when the current burrow finishes
}

func (m *Mob) alive() bool     { return m.State != StateDead }
func (m *Mob) targetable() bool { return m.alive() && m.State != StateBurrowed }

// Label returns the real, player-facing name for this mob -- DisplayName if a difficulty-tier
// variant set one (see the field's own doc comment), otherwise the plain Kind, matching every
// caller's pre-existing behavior before variants existed.
func (m *Mob) Label() string {
	if m.DisplayName != "" {
		return m.DisplayName
	}
	return m.Kind
}

// EventKind classifies a combat event returned by Tick.
type EventKind int

const (
	EvtMobAggro    EventKind = iota // mob targeted a player
	EvtMobAttack                    // mob swung at player (damage delivered)
	EvtMobReset                     // mob leashed back to home
	EvtMobDied                      // mob died; Tagger = who gets credit
	EvtMobBurrow                    // mob began burrowing (untargetable)
	EvtMobSurface                   // mob surfaced from burrow
)

// Event is a single combat occurrence produced by Tick or Hit.
type Event struct {
	Kind   EventKind
	MobID  string
	Slot   string // player slot involved
	Damage int    // set for EvtMobAttack
	Tagger string // set for EvtMobDied
}

// HitResult is returned by Hit.
type HitResult struct {
	Dealt  int  // damage actually applied (capped at remaining HP)
	Died   bool // true if this hit killed the mob
	Tagged bool // true if this hit set the tag (first damaging hit)
}

// PlayerCombat tracks one player's auto-attack state.
// The server main loop maintains one per connected slot.
type PlayerCombat struct {
	TargetMobID string
	SwingDelay  time.Duration // weapon speed; default DefaultPlayerSwingDelay
	BaseDamage  int           // base melee damage
	MeleeRange  float64       // melee reach
	lastSwing   time.Time
}

const (
	DefaultPlayerSwingDelay = 2 * time.Second
	DefaultPlayerDamage     = 20
	DefaultPlayerMeleeRange = 3.0
	DefaultMobMoveSpeed     = 5.0  // units/sec
	DefaultMobMeleeRange    = 2.0
	DefaultMobSwingDelay    = 3 * time.Second
	DefaultMobDamage        = 10
	DefaultAggroRange       = 20.0
	DefaultLeashRange       = 50.0
)

var (
	ErrMobNotFound = errors.New("mob not found")
	ErrMobDead     = errors.New("mob is dead")
	ErrNoTarget    = errors.New("no target set")
	ErrOutOfRange  = errors.New("out of range")
)

// Registry holds all mobs for a zone/scene set.
type Registry struct {
	mobs map[string]*Mob
}

// New returns an empty Registry.
func New() *Registry { return &Registry{mobs: make(map[string]*Mob)} }

// Spawn adds a new mob. Returns error if ID is already taken.
func (reg *Registry) Spawn(m Mob) error {
	if _, ok := reg.mobs[m.ID]; ok {
		return fmt.Errorf("mob %q already exists", m.ID)
	}
	if m.AggroRange == 0 {
		m.AggroRange = DefaultAggroRange
	}
	if m.LeashRange == 0 {
		m.LeashRange = DefaultLeashRange
	}
	if m.MeleeRange == 0 {
		m.MeleeRange = DefaultMobMeleeRange
	}
	if m.MoveSpeed == 0 {
		m.MoveSpeed = DefaultMobMoveSpeed
	}
	if m.SwingDelay == 0 {
		m.SwingDelay = DefaultMobSwingDelay
	}
	if m.MeleeDamage == 0 {
		m.MeleeDamage = DefaultMobDamage
	}
	reg.mobs[m.ID] = &m
	return nil
}

// Get returns a mob by ID.
func (reg *Registry) Get(id string) (*Mob, bool) {
	m, ok := reg.mobs[id]
	return m, ok
}

// All returns a snapshot of all mob IDs.
func (reg *Registry) All() []string {
	out := make([]string, 0, len(reg.mobs))
	for id := range reg.mobs {
		out = append(out, id)
	}
	return out
}

// Hit applies damage from a player to a mob and updates tagging.
// Returns HitResult and any events (EvtMobDied).
func (reg *Registry) Hit(mobID, attackerSlot string, damage int) (HitResult, []Event, error) {
	m, ok := reg.mobs[mobID]
	if !ok {
		return HitResult{}, nil, ErrMobNotFound
	}
	if !m.alive() {
		return HitResult{}, nil, ErrMobDead
	}
	if !m.targetable() {
		return HitResult{}, nil, ErrMobDead // burrowed: same error class, unhittable
	}

	res := HitResult{}

	// Tag on first damaging hit.
	if m.TaggerSlot == "" && damage > 0 {
		m.TaggerSlot = attackerSlot
		m.TaggedAt = time.Now()
		res.Tagged = true
	}

	// Apply damage.
	actual := damage
	if actual > m.HP {
		actual = m.HP
	}
	m.HP -= actual
	res.Dealt = actual

	var events []Event
	if m.HP <= 0 {
		m.State = StateDead
		m.DiedAt = time.Now()
		res.Died = true
		events = append(events, Event{
			Kind:   EvtMobDied,
			MobID:  m.ID,
			Slot:   attackerSlot,
			Tagger: m.TaggerSlot,
		})
	} else if m.AggroSlot == "" {
		// First hit aggroes the mob if it wasn't already engaged.
		m.AggroSlot = attackerSlot
		m.State = StatePursuing
		events = append(events, Event{Kind: EvtMobAggro, MobID: m.ID, Slot: attackerSlot})
	}

	return res, events, nil
}

// PlayerPositions is the snapshot of player positions passed to Tick.
type PlayerPositions struct {
	Slot    string
	SceneID int
	Pos     Pos
}

// Tick advances all mob AI by dt seconds, using player positions for aggro/chase.
// Returns all events that occurred this tick (mob attacks, deaths, resets).
func (reg *Registry) Tick(now time.Time, dt float64, players []PlayerPositions) []Event {
	var events []Event
	for _, m := range reg.mobs {
		if !m.alive() {
			continue
		}
		evts := tickMob(m, now, dt, players)
		events = append(events, evts...)
	}
	return events
}

func tickMob(m *Mob, now time.Time, dt float64, players []PlayerPositions) []Event {
	var events []Event

	// Handle burrow surface: if currently burrowed and the burrow has ended, surface.
	if m.State == StateBurrowed {
		if now.After(m.burrowEnd) {
			m.State = StateIdle
			events = append(events, Event{Kind: EvtMobSurface, MobID: m.ID})
		}
		return events // burrowed mobs do nothing else this tick
	}

	// Check whether it's time to burrow (only from Idle).
	if m.BurrowInterval > 0 && m.State == StateIdle {
		triggerTime := m.lastBurrow.Add(m.BurrowInterval)
		if m.lastBurrow.IsZero() {
			// Delay first burrow by one full interval from spawn.
			triggerTime = now.Add(m.BurrowInterval)
			m.lastBurrow = now
		}
		if now.After(triggerTime) {
			m.State = StateBurrowed
			m.lastBurrow = now
			m.burrowEnd = now.Add(m.BurrowDuration)
			m.AggroSlot = "" // drop aggro on burrow
			events = append(events, Event{Kind: EvtMobBurrow, MobID: m.ID})
			return events
		}
	}

	switch m.State {
	case StateIdle:
		// Scan for players in aggro range (same scene).
		for _, p := range players {
			if p.SceneID != m.SceneID {
				continue
			}
			if dist(m.Pos, p.Pos) <= m.AggroRange {
				m.AggroSlot = p.Slot
				m.State = StatePursuing
				events = append(events, Event{Kind: EvtMobAggro, MobID: m.ID, Slot: p.Slot})
				break
			}
		}

	case StatePursuing:
		// Find current aggro target's position.
		targetPos, found := findPlayer(m.AggroSlot, m.SceneID, players)
		if !found {
			// Target left the scene — reset.
			m.AggroSlot = ""
			m.State = StateReturning
			break
		}

		// Leash check.
		if dist(m.HomePos, m.Pos) > m.LeashRange {
			m.AggroSlot = ""
			m.State = StateReturning
			m.HP = m.MaxHP // full heal on leash (standard MMO behaviour)
			events = append(events, Event{Kind: EvtMobReset, MobID: m.ID})
			break
		}

		// Move toward target.
		step := m.MoveSpeed * dt
		m.Pos = moveToward(m.Pos, targetPos, step)

		// Melee check.
		if dist(m.Pos, targetPos) <= m.MeleeRange {
			if now.Sub(m.lastSwing) >= m.SwingDelay {
				m.lastSwing = now
				events = append(events, Event{
					Kind:   EvtMobAttack,
					MobID:  m.ID,
					Slot:   m.AggroSlot,
					Damage: m.MeleeDamage,
				})
			}
		}

	case StateReturning:
		step := m.MoveSpeed * dt
		m.Pos = moveToward(m.Pos, m.HomePos, step)
		if dist(m.Pos, m.HomePos) < 0.1 {
			m.Pos = m.HomePos
			m.State = StateIdle
		}
	}

	return events
}

// TickPlayer processes one player's auto-attack for the current tick.
// playerPos and playerSceneID describe the attacker's current state.
// Returns a HitResult and any events (including EvtMobDied) if a swing landed.
// Returns ErrNoTarget if no target is set, ErrOutOfRange if mob is too far,
// ErrMobDead if target is dead.
func (reg *Registry) TickPlayer(
	slot string,
	combat *PlayerCombat,
	playerPos Pos,
	playerSceneID int,
	now time.Time,
) (HitResult, []Event, error) {
	if combat.TargetMobID == "" {
		return HitResult{}, nil, ErrNoTarget
	}
	if combat.SwingDelay == 0 {
		combat.SwingDelay = DefaultPlayerSwingDelay
	}
	if combat.BaseDamage == 0 {
		combat.BaseDamage = DefaultPlayerDamage
	}
	if combat.MeleeRange == 0 {
		combat.MeleeRange = DefaultPlayerMeleeRange
	}

	// Swing timer gate.
	if now.Sub(combat.lastSwing) < combat.SwingDelay {
		return HitResult{}, nil, nil // not yet time
	}

	m, ok := reg.mobs[combat.TargetMobID]
	if !ok {
		return HitResult{}, nil, ErrMobNotFound
	}
	if !m.alive() {
		combat.TargetMobID = ""
		return HitResult{}, nil, ErrMobDead
	}
	if !m.targetable() {
		return HitResult{}, nil, ErrMobDead // burrowed
	}
	if m.SceneID != playerSceneID {
		return HitResult{}, nil, ErrOutOfRange
	}
	if dist(playerPos, m.Pos) > combat.MeleeRange {
		return HitResult{}, nil, ErrOutOfRange
	}

	combat.lastSwing = now
	return reg.Hit(combat.TargetMobID, slot, combat.BaseDamage)
}

func findPlayer(slot string, sceneID int, players []PlayerPositions) (Pos, bool) {
	for _, p := range players {
		if p.Slot == slot && p.SceneID == sceneID {
			return p.Pos, true
		}
	}
	return Pos{}, false
}

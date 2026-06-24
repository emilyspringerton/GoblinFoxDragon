// Package k9 implements the TRAPX K9 / RoboDog unit system.
//
// K9 units (RoboDogs) are mechanical custody organisms deployed by players who have
// unlocked the K9 Security Doctrine (tech tree Tier 4). Dogs execute custody operations
// on Field Offices. They operate in three modes and consume Battery over time.
//
// # Swarm math (diminishing returns)
//
// Adding more dogs to an FO yields less marginal benefit:
//
//	custody_score += base_score * math.Pow(SwarmDecay, float64(n_active - 1))
//
// SwarmDecay = 0.85 (each additional dog is 85% as effective as the prior one).
// This prevents trivial custody lock by flooding an FO with dogs.
//
// # Modes
//
//	Sentry  — stationary guard; generates slow steady custody score
//	Escort  — follows operator; generates custody score in movement range
//	Audit   — active investigation; generates Receipt events; drains Battery faster
//
// # Abilities (require sufficient dogs on same FO)
//
//	Mark         — flags a target operator; increases Watcher alertness in scene
//	Latch        — attaches to FO; blocks Flip during active Latch (costs 2 Battery/s)
//	HowlBeacon   — summons additional idle dogs from the district pool
//	CustodyLock  — hard FO lock; only executable during active Contest Window
//	ReceiptBurst — emits a burst of Receipt events for the last 30s of FO activity
package k9

import (
	"errors"
	"fmt"
	"math"
	"time"
)

const (
	SwarmDecay         = 0.85  // diminishing-returns factor per additional dog
	MaxActivePerOffice = 8     // hard cap on simultaneous dogs per FO
	BatterySentryDrain = 0.5   // Battery/s in Sentry mode
	BatteryEscortDrain = 0.8   // Battery/s in Escort mode
	BatteryAuditDrain  = 1.5   // Battery/s in Audit mode
	BatteryLatchExtra  = 2.0   // additional Battery/s while Latch is active
	BatteryMax         = 100.0
)

var (
	ErrBatteryDead       = errors.New("k9: battery dead; cannot deploy")
	ErrMaxDogsOnFO       = errors.New("k9: max dogs already on this field office")
	ErrNotOnFO           = errors.New("k9: dog is not assigned to a field office")
	ErrNotContested      = errors.New("k9: CustodyLock requires an active Contest Window")
	ErrAlreadyLatching   = errors.New("k9: dog is already latching")
	ErrNotLatching       = errors.New("k9: dog is not latching")
)

// Mode is the operational mode of a K9 unit.
type Mode int

const (
	ModeSentry Mode = iota
	ModeEscort
	ModeAudit
)

var modeNames = map[Mode]string{
	ModeSentry: "Sentry",
	ModeEscort: "Escort",
	ModeAudit:  "Audit",
}

// ModeName returns the display name for a Mode.
func ModeName(m Mode) string {
	if s, ok := modeNames[m]; ok {
		return s
	}
	return "Unknown"
}

// DogEvent records an action taken by a K9 unit.
type DogEvent struct {
	At    time.Time
	DogID string
	FOID  string // empty if not on an FO
	Verb  string // DEPLOYED, MODE_CHANGED, MARK, LATCH_START, LATCH_END, HOWL_BEACON,
	//              CUSTODY_LOCK, RECEIPT_BURST, BATTERY_LOW, BATTERY_DEAD, TICK
	Detail string
}

func (e DogEvent) String() string {
	loc := e.FOID
	if loc == "" {
		loc = "unassigned"
	}
	return fmt.Sprintf("[%s] K9:%s@%s — %s %s", e.At.Format("15:04:05"), e.DogID, loc, e.Verb, e.Detail)
}

// DogUnit is a single K9 / RoboDog unit.
// Not goroutine-safe; callers synchronise via the server tick loop.
type DogUnit struct {
	ID        string
	FOID      string  // Field Office this dog is assigned to (empty = unassigned)
	OperatorID string // player crew that controls this dog

	Mode    Mode
	Battery float64 // 0.0–100.0
	HeatLevel float64 // 0.0–100.0; rises with activity; affects Watcher alertness contribution

	Latching bool // true while Latch ability is active
}

// NewDog creates a fully-charged DogUnit.
func NewDog(id, operatorID string) *DogUnit {
	return &DogUnit{
		ID:         id,
		OperatorID: operatorID,
		Mode:       ModeSentry,
		Battery:    BatteryMax,
	}
}

// Deploy assigns the dog to a Field Office.
// Fails if battery is dead.
func (d *DogUnit) Deploy(foID string, now time.Time) (DogEvent, error) {
	if d.Battery <= 0 {
		return DogEvent{}, ErrBatteryDead
	}
	d.FOID = foID
	return DogEvent{At: now, DogID: d.ID, FOID: foID, Verb: "DEPLOYED",
		Detail: fmt.Sprintf("mode=%s battery=%.0f", ModeName(d.Mode), d.Battery)}, nil
}

// SetMode changes the operational mode. Cannot change if battery is dead.
func (d *DogUnit) SetMode(m Mode, now time.Time) (DogEvent, error) {
	if d.Battery <= 0 {
		return DogEvent{}, ErrBatteryDead
	}
	d.Mode = m
	return DogEvent{At: now, DogID: d.ID, FOID: d.FOID, Verb: "MODE_CHANGED",
		Detail: ModeName(m)}, nil
}

// Mark flags a target operator, increasing Heat and Watcher alertness contribution.
func (d *DogUnit) Mark(targetID string, now time.Time) (DogEvent, error) {
	if d.FOID == "" {
		return DogEvent{}, ErrNotOnFO
	}
	d.HeatLevel = math.Min(d.HeatLevel+15, 100)
	return DogEvent{At: now, DogID: d.ID, FOID: d.FOID, Verb: "MARK",
		Detail: fmt.Sprintf("target=%s heat=%.0f", targetID, d.HeatLevel)}, nil
}

// StartLatch activates the Latch ability — blocks FO flip while active.
// Drains extra Battery/s. Cannot latch if already latching.
func (d *DogUnit) StartLatch(now time.Time) (DogEvent, error) {
	if d.FOID == "" {
		return DogEvent{}, ErrNotOnFO
	}
	if d.Latching {
		return DogEvent{}, ErrAlreadyLatching
	}
	d.Latching = true
	return DogEvent{At: now, DogID: d.ID, FOID: d.FOID, Verb: "LATCH_START",
		Detail: fmt.Sprintf("battery=%.0f", d.Battery)}, nil
}

// EndLatch deactivates the Latch ability.
func (d *DogUnit) EndLatch(now time.Time) (DogEvent, error) {
	if !d.Latching {
		return DogEvent{}, ErrNotLatching
	}
	d.Latching = false
	return DogEvent{At: now, DogID: d.ID, FOID: d.FOID, Verb: "LATCH_END", Detail: ""}, nil
}

// HowlBeacon emits a summons signal from this dog's FO.
// The pool/roster must provide additional idle dogs; this just fires the event.
func (d *DogUnit) HowlBeacon(now time.Time) (DogEvent, error) {
	if d.FOID == "" {
		return DogEvent{}, ErrNotOnFO
	}
	d.HeatLevel = math.Min(d.HeatLevel+10, 100)
	return DogEvent{At: now, DogID: d.ID, FOID: d.FOID, Verb: "HOWL_BEACON",
		Detail: fmt.Sprintf("heat=%.0f", d.HeatLevel)}, nil
}

// CustodyLock hard-locks the FO against any flip. Only valid during an active Contest Window
// (caller passes isContested=true when the FO's phase is PhaseContested).
func (d *DogUnit) CustodyLock(isContested bool, now time.Time) (DogEvent, error) {
	if d.FOID == "" {
		return DogEvent{}, ErrNotOnFO
	}
	if !isContested {
		return DogEvent{}, ErrNotContested
	}
	d.HeatLevel = math.Min(d.HeatLevel+25, 100)
	return DogEvent{At: now, DogID: d.ID, FOID: d.FOID, Verb: "CUSTODY_LOCK",
		Detail: fmt.Sprintf("heat=%.0f", d.HeatLevel)}, nil
}

// ReceiptBurst emits an event signalling the last 30s of FO activity should be logged.
func (d *DogUnit) ReceiptBurst(now time.Time) (DogEvent, error) {
	if d.FOID == "" {
		return DogEvent{}, ErrNotOnFO
	}
	return DogEvent{At: now, DogID: d.ID, FOID: d.FOID, Verb: "RECEIPT_BURST",
		Detail: "last 30s of FO activity"}, nil
}

// Tick advances the dog by dt. Drains Battery; returns events for low battery or death.
func (d *DogUnit) Tick(dt time.Duration, now time.Time) []DogEvent {
	if d.Battery <= 0 {
		return nil
	}
	drain := batteryDrainRate(d.Mode)
	if d.Latching {
		drain += BatteryLatchExtra
	}
	prev := d.Battery
	d.Battery = math.Max(d.Battery-drain*dt.Seconds(), 0)
	d.HeatLevel = math.Max(d.HeatLevel-0.1*dt.Seconds(), 0) // slow heat decay

	var events []DogEvent
	if prev >= 20 && d.Battery < 20 {
		events = append(events, DogEvent{At: now, DogID: d.ID, FOID: d.FOID, Verb: "BATTERY_LOW",
			Detail: fmt.Sprintf("%.0f%%", d.Battery)})
	}
	if d.Battery == 0 {
		if d.Latching {
			d.Latching = false
		}
		events = append(events, DogEvent{At: now, DogID: d.ID, FOID: d.FOID, Verb: "BATTERY_DEAD", Detail: ""})
	}
	return events
}

// IsAlive returns true if the dog has remaining battery.
func (d *DogUnit) IsAlive() bool { return d.Battery > 0 }

func batteryDrainRate(m Mode) float64 {
	switch m {
	case ModeSentry:
		return BatterySentryDrain
	case ModeEscort:
		return BatteryEscortDrain
	case ModeAudit:
		return BatteryAuditDrain
	}
	return BatterySentryDrain
}

// SwarmCustodyScore computes the aggregate custody score for n active dogs at an FO,
// each with the given base score. Applies 0.85^(n-1) diminishing returns per dog.
//
// Score_total = base * Σ(i=0..n-1) SwarmDecay^i
func SwarmCustodyScore(n int, baseScorePerDog float64) float64 {
	if n <= 0 {
		return 0
	}
	total := 0.0
	for i := 0; i < n; i++ {
		total += baseScorePerDog * math.Pow(SwarmDecay, float64(i))
	}
	return total
}

// Swarm manages a group of K9 units assigned to a single Field Office.
// Not goroutine-safe.
type Swarm struct {
	FOID string
	Dogs []*DogUnit
}

// NewSwarm creates an empty Swarm for the given FO.
func NewSwarm(foID string) *Swarm { return &Swarm{FOID: foID} }

// Add adds a dog to the swarm. Returns an error if the cap is reached or the dog
// has no battery.
func (s *Swarm) Add(d *DogUnit) error {
	if len(s.Dogs) >= MaxActivePerOffice {
		return ErrMaxDogsOnFO
	}
	if !d.IsAlive() {
		return ErrBatteryDead
	}
	s.Dogs = append(s.Dogs, d)
	return nil
}

// ActiveCount returns the number of dogs with remaining battery.
func (s *Swarm) ActiveCount() int {
	count := 0
	for _, d := range s.Dogs {
		if d.IsAlive() {
			count++
		}
	}
	return count
}

// CustodyScore returns the aggregate custody score of the live dogs in the swarm.
// Base score per dog: 10.0.
func (s *Swarm) CustodyScore() float64 {
	return SwarmCustodyScore(s.ActiveCount(), 10.0)
}

// TickAll advances all dogs by dt and returns all generated events.
// Dead dogs are retained in the slice (caller can prune with Prune).
func (s *Swarm) TickAll(dt time.Duration, now time.Time) []DogEvent {
	var all []DogEvent
	for _, d := range s.Dogs {
		all = append(all, d.Tick(dt, now)...)
	}
	return all
}

// Prune removes dogs with dead batteries. Returns count removed.
func (s *Swarm) Prune() int {
	live := s.Dogs[:0]
	removed := 0
	for _, d := range s.Dogs {
		if d.IsAlive() {
			live = append(live, d)
		} else {
			removed++
		}
	}
	s.Dogs = live
	return removed
}

// IsLatched returns true if any dog in the swarm is currently latching the FO.
func (s *Swarm) IsLatched() bool {
	for _, d := range s.Dogs {
		if d.IsAlive() && d.Latching {
			return true
		}
	}
	return false
}

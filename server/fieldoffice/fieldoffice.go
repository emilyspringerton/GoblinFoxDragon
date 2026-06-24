// Package fieldoffice implements the TRAPX Field Office custody system.
//
// A Field Office (FO) is a capturable custody node in the TRAPX city world.
// Held FOs generate Flow (resources) and accumulate Pressure (attention risk).
// Ownership changes through a timed Contest Window. K9 swarms execute custody
// flips. Every state transition emits a Receipt for the mocumentary log.
//
// # Phase model
//
//	Unclaimed → Held (Claim)
//	Held → Contested (Contest)
//	Contested → Held (Flip or Defend)
//	Contested → Containment (Rogue Swarm trigger)
//	Containment → Held (containment objectives complete)
//
// # Knobs (configurable via Options)
//
//	FlipWindowsPerDay  int           — how many Contest Windows open per day
//	ContestDuration    time.Duration — length of a Contest Window
//	FlowBaseRate       float64       — Flow units generated per second when Held
//	PressureRate       float64       — Pressure added per second proportional to Flow
package fieldoffice

import (
	"errors"
	"fmt"
	"time"
)

// Phase is the current state of a Field Office.
type Phase int

const (
	PhaseUnclaimed   Phase = iota // no crew holds the FO
	PhaseHeld                     // a crew holds the FO; Flow is accumulating
	PhaseContested                // Contest Window is open; FO can be flipped
	PhaseContainment              // Rogue Swarm active; FO locked until objectives complete
)

var phaseNames = map[Phase]string{
	PhaseUnclaimed:   "Unclaimed",
	PhaseHeld:        "Held",
	PhaseContested:   "Contested",
	PhaseContainment: "Containment",
}

// PhaseName returns the display name for a Phase.
func PhaseName(p Phase) string {
	if s, ok := phaseNames[p]; ok {
		return s
	}
	return "Unknown"
}

var (
	ErrAlreadyClaimed   = errors.New("field office is already claimed")
	ErrNotHeld          = errors.New("field office is not currently held")
	ErrNotContested     = errors.New("field office is not in contest")
	ErrSelfFlip         = errors.New("contesting crew already holds this office")
	ErrFlipWindowClosed = errors.New("flip window is not open")
	ErrInContainment    = errors.New("field office is in containment; no custody changes allowed")
)

// Receipt records a diegetic event emitted by a Field Office state change.
type Receipt struct {
	At      time.Time
	FOID    string
	Verb    string // CLAIMED, CONTESTED, FLIPPED, DEFENDED, CONTAINMENT_START, CONTAINMENT_END
	Actor   string // crew ID or empty
	Subject string // crew ID (prior holder) or empty
	Detail  string
}

func (r Receipt) String() string {
	if r.Actor != "" && r.Subject != "" {
		return fmt.Sprintf("[%s] %s — %s by %s (was: %s) %s",
			r.At.Format("15:04:05"), r.FOID, r.Verb, r.Actor, r.Subject, r.Detail)
	}
	if r.Actor != "" {
		return fmt.Sprintf("[%s] %s — %s by %s %s",
			r.At.Format("15:04:05"), r.FOID, r.Verb, r.Actor, r.Detail)
	}
	return fmt.Sprintf("[%s] %s — %s %s", r.At.Format("15:04:05"), r.FOID, r.Verb, r.Detail)
}

// Options configures a FieldOffice at creation time.
type Options struct {
	FlipWindowsPerDay int           // how many Contest Windows per 24-hour period (default 2)
	ContestDuration   time.Duration // length of each Contest Window (default 5 min)
	FlowBaseRate      float64       // Flow units per second when Held (default 1.0)
	PressureRate      float64       // Pressure added per second per unit of Flow (default 0.1)
}

func defaultOptions() Options {
	return Options{
		FlipWindowsPerDay: 2,
		ContestDuration:   5 * time.Minute,
		FlowBaseRate:      1.0,
		PressureRate:      0.1,
	}
}

// FieldOffice is a single capturable custody node.
// Not goroutine-safe; callers synchronise via the server tick loop.
type FieldOffice struct {
	ID       string
	SceneID  int
	X, Y, Z float32

	Phase     Phase
	HolderID  string  // crew holding the FO (empty if Unclaimed)
	Flow      float64 // accumulated Flow since last claim
	Pressure  float64 // accumulated Pressure

	// contest state
	ContestStart  time.Time
	ContestEnd    time.Time
	ContesterID   string // crew contesting

	// containment state
	ObjectivesComplete int
	ObjectivesRequired int // set by the caller on Rogue Swarm activation

	opts Options
}

// New creates a Field Office with the given ID and scene position.
func New(id string, sceneID int, x, y, z float32, opts *Options) *FieldOffice {
	o := defaultOptions()
	if opts != nil {
		if opts.FlipWindowsPerDay > 0 {
			o.FlipWindowsPerDay = opts.FlipWindowsPerDay
		}
		if opts.ContestDuration > 0 {
			o.ContestDuration = opts.ContestDuration
		}
		if opts.FlowBaseRate > 0 {
			o.FlowBaseRate = opts.FlowBaseRate
		}
		if opts.PressureRate > 0 {
			o.PressureRate = opts.PressureRate
		}
	}
	return &FieldOffice{
		ID: id, SceneID: sceneID,
		X: x, Y: y, Z: z,
		Phase:              PhaseUnclaimed,
		ObjectivesRequired: 3,
		opts:               o,
	}
}

// FlipWindowOpen returns true if the given time falls within a Contest Window.
// Windows are evenly distributed across 24 hours.
func (fo *FieldOffice) FlipWindowOpen(now time.Time) bool {
	if fo.opts.FlipWindowsPerDay <= 0 {
		return false
	}
	period := 24 * time.Hour / time.Duration(fo.opts.FlipWindowsPerDay)
	elapsed := time.Duration(now.Unix()%int64(period.Seconds())) * time.Second
	return elapsed < fo.opts.ContestDuration
}

// Claim transitions an Unclaimed FO to Held by crewID.
func (fo *FieldOffice) Claim(crewID string, now time.Time) (Receipt, error) {
	if fo.Phase == PhaseContainment {
		return Receipt{}, ErrInContainment
	}
	if fo.Phase != PhaseUnclaimed {
		return Receipt{}, ErrAlreadyClaimed
	}
	fo.Phase = PhaseHeld
	fo.HolderID = crewID
	fo.Flow = 0
	fo.Pressure = 0
	return Receipt{
		At: now, FOID: fo.ID, Verb: "CLAIMED", Actor: crewID,
		Detail: fmt.Sprintf("Scene %d (%.0f,%.0f,%.0f)", fo.SceneID, fo.X, fo.Y, fo.Z),
	}, nil
}

// Contest opens a Contest Window against a Held FO.
// contesterID must differ from the current holder.
// The flip window must be open (unless overridden by a forced contest).
func (fo *FieldOffice) Contest(contesterID string, now time.Time, forceWindow bool) (Receipt, error) {
	if fo.Phase == PhaseContainment {
		return Receipt{}, ErrInContainment
	}
	if fo.Phase != PhaseHeld {
		return Receipt{}, ErrNotHeld
	}
	if contesterID == fo.HolderID {
		return Receipt{}, ErrSelfFlip
	}
	if !forceWindow && !fo.FlipWindowOpen(now) {
		return Receipt{}, ErrFlipWindowClosed
	}
	fo.Phase = PhaseContested
	fo.ContestStart = now
	fo.ContestEnd = now.Add(fo.opts.ContestDuration)
	fo.ContesterID = contesterID
	return Receipt{
		At: now, FOID: fo.ID, Verb: "CONTESTED",
		Actor: contesterID, Subject: fo.HolderID,
		Detail: fmt.Sprintf("Window closes at %s", fo.ContestEnd.Format("15:04:05")),
	}, nil
}

// Flip resolves an active Contest in favour of the contester.
// Must be called before ContestEnd; caller is responsible for timing.
func (fo *FieldOffice) Flip(now time.Time) (Receipt, error) {
	if fo.Phase != PhaseContested {
		return Receipt{}, ErrNotContested
	}
	prior := fo.HolderID
	fo.HolderID = fo.ContesterID
	fo.ContesterID = ""
	fo.Phase = PhaseHeld
	fo.Flow = 0
	fo.Pressure = 0
	return Receipt{
		At: now, FOID: fo.ID, Verb: "FLIPPED",
		Actor: fo.HolderID, Subject: prior,
		Detail: "Custody transferred",
	}, nil
}

// Defend resolves an active Contest in favour of the current holder.
// Called when the Contest Window expires without a successful flip.
func (fo *FieldOffice) Defend(now time.Time) (Receipt, error) {
	if fo.Phase != PhaseContested {
		return Receipt{}, ErrNotContested
	}
	prior := fo.ContesterID
	fo.ContesterID = ""
	fo.Phase = PhaseHeld
	return Receipt{
		At: now, FOID: fo.ID, Verb: "DEFENDED",
		Actor: fo.HolderID, Subject: prior,
		Detail: "Contest expired; holder retained",
	}, nil
}

// StartContainment transitions the FO into Containment phase (Rogue Swarm active).
// No custody changes are possible until containment is resolved.
func (fo *FieldOffice) StartContainment(objectivesRequired int, now time.Time) Receipt {
	fo.Phase = PhaseContainment
	fo.ObjectivesComplete = 0
	if objectivesRequired > 0 {
		fo.ObjectivesRequired = objectivesRequired
	}
	fo.ContesterID = ""
	return Receipt{
		At: now, FOID: fo.ID, Verb: "CONTAINMENT_START",
		Detail: fmt.Sprintf("%d objectives required", fo.ObjectivesRequired),
	}
}

// CompleteObjective records progress toward ending Containment.
// Returns (true, Receipt) when all objectives are complete and Containment ends.
func (fo *FieldOffice) CompleteObjective(now time.Time) (bool, Receipt) {
	if fo.Phase != PhaseContainment {
		return false, Receipt{}
	}
	fo.ObjectivesComplete++
	if fo.ObjectivesComplete >= fo.ObjectivesRequired {
		fo.Phase = PhaseHeld
		return true, Receipt{
			At: now, FOID: fo.ID, Verb: "CONTAINMENT_END",
			Actor: fo.HolderID,
			Detail: fmt.Sprintf("All %d objectives complete; custody restored", fo.ObjectivesRequired),
		}
	}
	return false, Receipt{
		At: now, FOID: fo.ID, Verb: "OBJECTIVE_COMPLETE",
		Detail: fmt.Sprintf("%d/%d objectives", fo.ObjectivesComplete, fo.ObjectivesRequired),
	}
}

// Tick advances the simulation by dt. Accumulates Flow and Pressure when Held.
// If the Contest Window has expired during a contested phase, automatically defends.
// Returns any auto-generated receipts (may be empty).
func (fo *FieldOffice) Tick(now time.Time, dt time.Duration) []Receipt {
	var receipts []Receipt

	// Auto-defend if contest window expired.
	if fo.Phase == PhaseContested && now.After(fo.ContestEnd) {
		r, _ := fo.Defend(now)
		receipts = append(receipts, r)
	}

	// Accumulate Flow + Pressure when Held.
	if fo.Phase == PhaseHeld {
		secs := dt.Seconds()
		fo.Flow += fo.opts.FlowBaseRate * secs
		fo.Pressure += fo.opts.PressureRate * fo.Flow * secs
	}

	return receipts
}

// ContestTimeRemaining returns how much time is left in the current Contest Window.
// Returns 0 if not contested.
func (fo *FieldOffice) ContestTimeRemaining(now time.Time) time.Duration {
	if fo.Phase != PhaseContested {
		return 0
	}
	rem := fo.ContestEnd.Sub(now)
	if rem < 0 {
		return 0
	}
	return rem
}

// Summary returns a human-readable one-line description of the FO's current state.
func (fo *FieldOffice) Summary() string {
	switch fo.Phase {
	case PhaseUnclaimed:
		return fmt.Sprintf("Field Office %s [UNCLAIMED] Scene:%d", fo.ID, fo.SceneID)
	case PhaseHeld:
		return fmt.Sprintf("Field Office %s [HELD by %s] Flow:%.1f Pressure:%.1f",
			fo.ID, fo.HolderID, fo.Flow, fo.Pressure)
	case PhaseContested:
		return fmt.Sprintf("Field Office %s [CONTESTED: %s vs %s] Window closes %s",
			fo.ID, fo.ContesterID, fo.HolderID, fo.ContestEnd.Format("15:04:05"))
	case PhaseContainment:
		return fmt.Sprintf("Field Office %s [CONTAINMENT %d/%d objectives] Holder:%s",
			fo.ID, fo.ObjectivesComplete, fo.ObjectivesRequired, fo.HolderID)
	}
	return fmt.Sprintf("Field Office %s [unknown phase]", fo.ID)
}

// Registry manages a collection of Field Offices for a scene cluster.
type Registry struct {
	offices map[string]*FieldOffice
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{offices: make(map[string]*FieldOffice)}
}

// Add registers a FieldOffice. Panics if the ID is already registered.
func (r *Registry) Add(fo *FieldOffice) {
	if _, exists := r.offices[fo.ID]; exists {
		panic(fmt.Sprintf("fieldoffice: duplicate ID %q", fo.ID))
	}
	r.offices[fo.ID] = fo
}

// Get returns the FieldOffice with the given ID, or false if not found.
func (r *Registry) Get(id string) (*FieldOffice, bool) {
	fo, ok := r.offices[id]
	return fo, ok
}

// All returns all registered Field Offices in arbitrary order.
func (r *Registry) All() []*FieldOffice {
	out := make([]*FieldOffice, 0, len(r.offices))
	for _, fo := range r.offices {
		out = append(out, fo)
	}
	return out
}

// TickAll advances all offices and returns all generated receipts.
func (r *Registry) TickAll(now time.Time, dt time.Duration) []Receipt {
	var all []Receipt
	for _, fo := range r.offices {
		all = append(all, fo.Tick(now, dt)...)
	}
	return all
}

// DefaultFieldOffices returns a starter set of Field Offices for the TRAPX city
// district cluster (scene IDs 200–204).
func DefaultFieldOffices(opts *Options) []*FieldOffice {
	return []*FieldOffice{
		New("fo-residential-1",  200, 10, 0, 10, opts),
		New("fo-residential-2",  200, -10, 0, -10, opts),
		New("fo-commercial-1",   201, 20, 0, 5, opts),
		New("fo-industrial-1",   202, 0, 0, 30, opts),
		New("fo-underground-1",  203, 5, -5, 5, opts),
		New("fo-abandoned-1",    204, -20, 0, 20, opts),
	}
}

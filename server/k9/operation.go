// operation.go — K9 Merciless Operation doctrine.
//
// The Merciless Operation is the highest-escalation custody action in TRAPX.
// It is a 4-phase process that locks a Field Office from all flip attempts
// and culminates in a permanent Scar on the district.
//
// # Phases
//
//	Phase 1  Encirclement  — surround FO; requires 2+ dogs deployed; 3 min
//	Phase 2  Isolation     — cut external contact; prevents Flip for 3 min
//	Phase 3  CustodyLock  — hard lock; counterplay lanes open; 3 min
//	Phase 4  Resolution   — close out; writes district Scar
//
// Phase transition: each phase lasts PhaseDuration (3 min). Advance via AdvancePhase().
// Phase 1→2 requires at least MinDogsForPhase2 dogs on the FO.
//
// # Counterplay lanes
//
//	BirdCorrection — reduces TechPressure by BirdCorrectionReduction (150)
//	ScarBurn       — removes the most recent scar from the district (via scar.Registry)
//	FlipWindow     — forces a Contest Window open; FO can be reclaimed before Phase 4
//
// # MUD command
//
//	merciless-op <fo-id>   — initiate a Merciless Operation on the given FO

package k9

import (
	"errors"
	"fmt"
	"time"
)

const (
	PhaseDuration           = 3 * time.Minute
	MinDogsForPhase2        = 2
	BirdCorrectionReduction = 150.0
)

// Phase represents the current stage of a Merciless Operation.
type Phase int

const (
	PhaseEncirclement Phase = iota + 1
	PhaseIsolation
	PhaseCustodyLock
	PhaseResolution
)

var phaseNames = map[Phase]string{
	PhaseEncirclement: "Encirclement",
	PhaseIsolation:    "Isolation",
	PhaseCustodyLock:  "CustodyLock",
	PhaseResolution:   "Resolution",
}

// PhaseName returns the display name for a Phase.
func PhaseName(p Phase) string {
	if s, ok := phaseNames[p]; ok {
		return s
	}
	return "Unknown"
}

var (
	ErrOperationNotActive     = errors.New("k9: no active Merciless Operation on this FO")
	ErrOperationAlreadyActive = errors.New("k9: Merciless Operation already active on this FO")
	ErrNotEnoughDogs          = errors.New("k9: Phase 2 requires at least 2 dogs deployed on the FO")
	ErrPhaseNotComplete       = errors.New("k9: current phase duration not yet elapsed")
	ErrAlreadyResolved        = errors.New("k9: operation already at Resolution phase")
	ErrCounterplayUsed        = errors.New("k9: counterplay lane already used this operation")
)

// OpEvent records a significant action within a Merciless Operation.
type OpEvent struct {
	At     time.Time
	FOID   string
	Verb   string // PHASE_ADVANCED, BIRD_CORRECTION, SCAR_BURN, FLIP_WINDOW, RESOLVED
	Detail string
}

func (e OpEvent) String() string {
	return fmt.Sprintf("[%s] MercilessOp:%s — %s %s", e.At.Format("15:04:05"), e.FOID, e.Verb, e.Detail)
}

// MercilessOp tracks the state of an active Merciless Operation on one FO.
type MercilessOp struct {
	FOID       string
	Phase      Phase
	StartAt    time.Time // when Phase 1 started
	PhaseStart time.Time // when the current phase started

	// Counterplay flags — each lane can only be used once per operation.
	BirdCorrectionUsed bool
	ScarBurnUsed       bool
	FlipWindowUsed     bool

	// FlipWindowOpen is set by UseFlipWindow and cleared when the window expires.
	FlipWindowOpen  bool
	FlipWindowUntil time.Time

	Events []OpEvent
}

// Registry tracks all active Merciless Operations, keyed by FO ID.
// Not goroutine-safe; callers synchronise via the server tick loop.
type OpRegistry struct {
	ops map[string]*MercilessOp
}

// NewOpRegistry returns an empty OpRegistry.
func NewOpRegistry() *OpRegistry {
	return &OpRegistry{ops: make(map[string]*MercilessOp)}
}

// Initiate starts a new Merciless Operation on the given FO.
func (r *OpRegistry) Initiate(foID string, now time.Time) (*MercilessOp, error) {
	if _, exists := r.ops[foID]; exists {
		return nil, ErrOperationAlreadyActive
	}
	op := &MercilessOp{
		FOID:       foID,
		Phase:      PhaseEncirclement,
		StartAt:    now,
		PhaseStart: now,
	}
	op.Events = append(op.Events, OpEvent{
		At: now, FOID: foID, Verb: "PHASE_ADVANCED",
		Detail: fmt.Sprintf("phase=%s", PhaseName(PhaseEncirclement)),
	})
	r.ops[foID] = op
	return op, nil
}

// Get returns the active operation for the given FO, or nil.
func (r *OpRegistry) Get(foID string) *MercilessOp {
	return r.ops[foID]
}

// AdvancePhase moves the operation to the next phase.
// Enforces: (a) phase duration elapsed, (b) dog count check for Phase 1→2.
// Returns the new phase and a transition event.
//
// Callers provide the current dog count on the FO and the current time.
func (r *OpRegistry) AdvancePhase(foID string, dogCount int, now time.Time) (Phase, OpEvent, error) {
	op := r.ops[foID]
	if op == nil {
		return 0, OpEvent{}, ErrOperationNotActive
	}
	if op.Phase == PhaseResolution {
		return 0, OpEvent{}, ErrAlreadyResolved
	}
	if now.Before(op.PhaseStart.Add(PhaseDuration)) {
		return 0, OpEvent{}, ErrPhaseNotComplete
	}
	// Phase 1→2 requires minimum dogs.
	if op.Phase == PhaseEncirclement && dogCount < MinDogsForPhase2 {
		return 0, OpEvent{}, ErrNotEnoughDogs
	}

	op.Phase++
	op.PhaseStart = now
	ev := OpEvent{
		At: now, FOID: foID, Verb: "PHASE_ADVANCED",
		Detail: fmt.Sprintf("phase=%s dogs=%d", PhaseName(op.Phase), dogCount),
	}
	op.Events = append(op.Events, ev)
	return op.Phase, ev, nil
}

// IsFlipBlocked reports whether the FO flip is blocked by this operation.
// Flip is blocked during Isolation and CustodyLock unless FlipWindow is open.
func (r *OpRegistry) IsFlipBlocked(foID string, now time.Time) bool {
	op := r.ops[foID]
	if op == nil {
		return false
	}
	if op.FlipWindowOpen && now.Before(op.FlipWindowUntil) {
		return false // counterplay opened the window
	}
	return op.Phase == PhaseIsolation || op.Phase == PhaseCustodyLock
}

// UseBirdCorrection applies the BirdCorrection counterplay lane.
// Returns the TechPressure reduction value and an event.
func (r *OpRegistry) UseBirdCorrection(foID string, now time.Time) (float64, OpEvent, error) {
	op := r.ops[foID]
	if op == nil {
		return 0, OpEvent{}, ErrOperationNotActive
	}
	if op.BirdCorrectionUsed {
		return 0, OpEvent{}, ErrCounterplayUsed
	}
	op.BirdCorrectionUsed = true
	ev := OpEvent{
		At: now, FOID: foID, Verb: "BIRD_CORRECTION",
		Detail: fmt.Sprintf("TechPressure -%.0f", BirdCorrectionReduction),
	}
	op.Events = append(op.Events, ev)
	return BirdCorrectionReduction, ev, nil
}

// UseScarBurn applies the ScarBurn counterplay lane.
// Removes the most recent scar from the district via the provided scarRemover.
// scarRemover signature: func(districtID string) (removedDetail string, ok bool)
func (r *OpRegistry) UseScarBurn(foID, districtID string, scarRemover func(string) (string, bool), now time.Time) (OpEvent, error) {
	op := r.ops[foID]
	if op == nil {
		return OpEvent{}, ErrOperationNotActive
	}
	if op.ScarBurnUsed {
		return OpEvent{}, ErrCounterplayUsed
	}
	removed, ok := scarRemover(districtID)
	if !ok {
		return OpEvent{}, errors.New("k9: no scar to burn in district " + districtID)
	}
	op.ScarBurnUsed = true
	ev := OpEvent{
		At: now, FOID: foID, Verb: "SCAR_BURN",
		Detail: fmt.Sprintf("removed scar from %s: %s", districtID, removed),
	}
	op.Events = append(op.Events, ev)
	return ev, nil
}

// UseFlipWindow opens a forced Contest Window for the FO for windowDur.
// During this window, IsFlipBlocked returns false, enabling reclaim.
func (r *OpRegistry) UseFlipWindow(foID string, windowDur time.Duration, now time.Time) (OpEvent, error) {
	op := r.ops[foID]
	if op == nil {
		return OpEvent{}, ErrOperationNotActive
	}
	if op.FlipWindowUsed {
		return OpEvent{}, ErrCounterplayUsed
	}
	op.FlipWindowUsed = true
	op.FlipWindowOpen = true
	op.FlipWindowUntil = now.Add(windowDur)
	ev := OpEvent{
		At: now, FOID: foID, Verb: "FLIP_WINDOW",
		Detail: fmt.Sprintf("contest window open until %s", op.FlipWindowUntil.Format("15:04:05")),
	}
	op.Events = append(op.Events, ev)
	return ev, nil
}

// Resolve completes the operation and removes it from the registry.
// Caller is responsible for writing the district Scar (use scar.Registry.Append
// with CauseMercilessOp) after this returns successfully.
func (r *OpRegistry) Resolve(foID string, now time.Time) (OpEvent, error) {
	op := r.ops[foID]
	if op == nil {
		return OpEvent{}, ErrOperationNotActive
	}
	if op.Phase != PhaseResolution {
		return OpEvent{}, fmt.Errorf("k9: cannot resolve: operation is in phase %s (need Resolution)", PhaseName(op.Phase))
	}
	ev := OpEvent{
		At: now, FOID: foID, Verb: "RESOLVED",
		Detail: fmt.Sprintf("district scar written; duration=%s", now.Sub(op.StartAt).Round(time.Second)),
	}
	op.Events = append(op.Events, ev)
	delete(r.ops, foID)
	return ev, nil
}

// ActiveOps returns the FO IDs of all active operations.
func (r *OpRegistry) ActiveOps() []string {
	out := make([]string, 0, len(r.ops))
	for id := range r.ops {
		out = append(out, id)
	}
	return out
}

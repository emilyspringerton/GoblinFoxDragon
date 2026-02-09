// Source [8] - Validator Staking & Slashing
package pol

import "sync"

type ValidatorStatus int

const (
	StatusActive ValidatorStatus = iota
	StatusJailed
	StatusSlashed
)

type Validator struct {
	Address        string
	Stake          uint64
	Status         ValidatorStatus
	BlocksSigned   uint64
	SlashingEvents []SlashingEvent
	mu             sync.RWMutex
}

type SlashingEvent struct {
	Height        uint64
	Reason        string // "double_sign", "downtime"
	AmountSlashed uint64
}

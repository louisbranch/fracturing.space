package action

// RollState captures authoritative roll metadata for causal replay checks.
type RollState struct {
	RequestID string
	SessionID string
	Outcome   string
}

// State captures causal action-replay state used by command-time invariants.
//
// Only causal action events mutate this state:
// - action.roll_resolved
// - action.outcome_applied
//
// Non-causal narrative/audit events must not mutate it.
type State struct {
	Rolls           map[uint64]RollState
	AppliedOutcomes map[uint64]struct{}
}

package session

// State captures the replayed live-session context for command routing.
//
// The command engine uses this aggregate to enforce gate and spotlight behavior
// before allowing session-scoped actions to proceed.
type State struct {
	// Started indicates whether a start command has been accepted for this campaign session.
	Started bool
	// Ended indicates whether the active session lifecycle has been concluded.
	Ended bool
	// SessionID is the canonical identifier used to scope session-local commands.
	SessionID string
	// Name is a human-facing label for the running session.
	Name string
	// GateOpen blocks non-allowed commands while adjudication is paused.
	GateOpen bool
	// GateID identifies the active gate when GateOpen is true.
	GateID string
	// SpotlightType tracks which entity type currently holds initiative context.
	SpotlightType string
	// SpotlightCharacterID tracks the focused character in spotlight workflows.
	SpotlightCharacterID string
}

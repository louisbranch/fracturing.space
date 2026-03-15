package session

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"

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
	SessionID ids.SessionID
	// Name is a human-facing label for the running session.
	Name string
	// GateOpen blocks non-allowed commands while adjudication is paused.
	GateOpen bool
	// GateID identifies the active gate when GateOpen is true.
	GateID ids.GateID
	// GateType captures the active gate workflow type while a gate is open.
	GateType string
	// GateMetadataJSON preserves normalized workflow metadata for replay-owned
	// gate response validation.
	GateMetadataJSON []byte
	// SpotlightType tracks which entity type currently holds initiative context.
	SpotlightType string
	// SpotlightCharacterID tracks the focused character in spotlight workflows.
	SpotlightCharacterID ids.CharacterID
	// ActiveSceneID tracks which scene currently owns in-character interaction.
	ActiveSceneID ids.SceneID
	// GMAuthorityParticipantID identifies the GM participant that currently owns
	// the next GM decision point for the session.
	GMAuthorityParticipantID ids.ParticipantID
	// OOCPaused reports whether the session is currently paused for out-of-character discussion.
	OOCPaused bool
	// OOCReadyParticipants tracks which participants have marked ready to resume.
	OOCReadyParticipants map[ids.ParticipantID]bool
	// AITurnStatus tracks whether the current GM-owned moment has an AI turn queued,
	// running, or failed.
	AITurnStatus AITurnStatus
	// AITurnToken deduplicates retries and worker queue requests for the same GM turn.
	AITurnToken string
	// AITurnOwnerParticipantID records which GM participant owns the queued AI turn.
	AITurnOwnerParticipantID ids.ParticipantID
	// AITurnSourceEventType records which interaction-owned transition created the current turn.
	AITurnSourceEventType string
	// AITurnSourceSceneID records the scene context for the current queued AI turn.
	AITurnSourceSceneID ids.SceneID
	// AITurnSourcePhaseID records the scene phase context for the current queued AI turn.
	AITurnSourcePhaseID string
	// AITurnLastError captures the latest orchestration failure for retryable AI turns.
	AITurnLastError string
}

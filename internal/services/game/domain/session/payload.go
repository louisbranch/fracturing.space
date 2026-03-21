package session

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"

// StartPayload captures the payload for session.start commands and session.started events.
type StartPayload struct {
	SessionID   ids.SessionID `json:"session_id"`
	SessionName string        `json:"session_name,omitempty"`
}

// EndPayload captures the payload for session.end commands and session.ended events.
type EndPayload struct {
	SessionID ids.SessionID `json:"session_id"`
}

// SpotlightSetPayload captures the payload for session.spotlight_set events.
type SpotlightSetPayload struct {
	SpotlightType string          `json:"spotlight_type"`
	CharacterID   ids.CharacterID `json:"character_id,omitempty"`
}

// SpotlightClearedPayload captures the payload for session.spotlight_cleared events.
type SpotlightClearedPayload struct {
	Reason string `json:"reason,omitempty"`
}

// SceneActivatedPayload captures the payload for session.scene_activated events.
type SceneActivatedPayload struct {
	SessionID     ids.SessionID `json:"session_id"`
	ActiveSceneID ids.SceneID   `json:"active_scene_id"`
}

// GMAuthoritySetPayload captures the payload for session.gm_authority_set events.
type GMAuthoritySetPayload struct {
	SessionID     ids.SessionID     `json:"session_id"`
	ParticipantID ids.ParticipantID `json:"participant_id"`
}

// OOCOpenedPayload captures the payload for session.ooc_opened events.
type OOCOpenedPayload struct {
	SessionID                ids.SessionID     `json:"session_id"`
	RequestedByParticipantID ids.ParticipantID `json:"requested_by_participant_id,omitempty"`
	Reason                   string            `json:"reason,omitempty"`
	InterruptedSceneID       ids.SceneID       `json:"interrupted_scene_id,omitempty"`
	InterruptedPhaseID       string            `json:"interrupted_phase_id,omitempty"`
	InterruptedPhaseStatus   string            `json:"interrupted_phase_status,omitempty"`
}

// OOCPostedPayload captures the payload for session.ooc_posted events.
type OOCPostedPayload struct {
	SessionID     ids.SessionID     `json:"session_id"`
	PostID        string            `json:"post_id"`
	ParticipantID ids.ParticipantID `json:"participant_id"`
	Body          string            `json:"body"`
}

// OOCReadyMarkedPayload captures the payload for session.ooc_ready_marked events.
type OOCReadyMarkedPayload struct {
	SessionID     ids.SessionID     `json:"session_id"`
	ParticipantID ids.ParticipantID `json:"participant_id"`
}

// OOCReadyClearedPayload captures the payload for session.ooc_ready_cleared events.
type OOCReadyClearedPayload struct {
	SessionID     ids.SessionID     `json:"session_id"`
	ParticipantID ids.ParticipantID `json:"participant_id"`
}

// OOCClosedPayload captures the payload for session.ooc_closed events.
type OOCClosedPayload struct {
	SessionID ids.SessionID `json:"session_id"`
	Reason    string        `json:"reason,omitempty"`
}

// OOCResolvedPayload clears the post-OOC resolution gate once the
// GM has explicitly chosen how the interrupted scene should continue.
type OOCResolvedPayload struct {
	SessionID  ids.SessionID `json:"session_id"`
	Resolution string        `json:"resolution,omitempty"`
}

// AITurnQueuedPayload captures the payload for session.ai_turn_queued events.
type AITurnQueuedPayload struct {
	SessionID          ids.SessionID     `json:"session_id"`
	TurnToken          string            `json:"turn_token"`
	OwnerParticipantID ids.ParticipantID `json:"owner_participant_id"`
	SourceEventType    string            `json:"source_event_type,omitempty"`
	SourceSceneID      ids.SceneID       `json:"source_scene_id,omitempty"`
	SourcePhaseID      string            `json:"source_phase_id,omitempty"`
}

// AITurnRunningPayload captures the payload for session.ai_turn_running events.
type AITurnRunningPayload struct {
	SessionID ids.SessionID `json:"session_id"`
	TurnToken string        `json:"turn_token"`
}

// AITurnFailedPayload captures the payload for session.ai_turn_failed events.
type AITurnFailedPayload struct {
	SessionID ids.SessionID `json:"session_id"`
	TurnToken string        `json:"turn_token"`
	LastError string        `json:"last_error,omitempty"`
}

// AITurnClearedPayload captures the payload for session.ai_turn_cleared events.
type AITurnClearedPayload struct {
	SessionID ids.SessionID `json:"session_id"`
	TurnToken string        `json:"turn_token,omitempty"`
	Reason    string        `json:"reason,omitempty"`
}

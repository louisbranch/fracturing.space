package event

// CampaignCreatedPayload captures the payload for campaign.created events.
type CampaignCreatedPayload struct {
	Name        string `json:"name"`
	GameSystem  string `json:"game_system"`
	GmMode      string `json:"gm_mode"`
	ThemePrompt string `json:"theme_prompt,omitempty"`
}

// CampaignForkedPayload captures the payload for campaign.forked events.
type CampaignForkedPayload struct {
	ParentCampaignID string `json:"parent_campaign_id"`
	ForkEventSeq     uint64 `json:"fork_event_seq"`
	OriginCampaignID string `json:"origin_campaign_id"`
	CopyParticipants bool   `json:"copy_participants"`
}

// CampaignStatusChangedPayload captures the payload for campaign.status_changed events.
type CampaignStatusChangedPayload struct {
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
}

// CampaignUpdatedPayload captures the payload for campaign.updated events.
type CampaignUpdatedPayload struct {
	Fields map[string]any `json:"fields"`
}

// ParticipantJoinedPayload captures the payload for participant.joined events.
type ParticipantJoinedPayload struct {
	ParticipantID string `json:"participant_id"`
	DisplayName   string `json:"display_name"`
	Role          string `json:"role"`
	Controller    string `json:"controller"`
	IsOwner       bool   `json:"is_owner"`
}

// ParticipantLeftPayload captures the payload for participant.left events.
type ParticipantLeftPayload struct {
	ParticipantID string `json:"participant_id"`
	Reason        string `json:"reason,omitempty"`
}

// ParticipantUpdatedPayload captures the payload for participant.updated events.
type ParticipantUpdatedPayload struct {
	ParticipantID string         `json:"participant_id"`
	Fields        map[string]any `json:"fields"`
}

// CharacterCreatedPayload captures the payload for character.created events.
type CharacterCreatedPayload struct {
	CharacterID string `json:"character_id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Notes       string `json:"notes,omitempty"`
}

// CharacterDeletedPayload captures the payload for character.deleted events.
type CharacterDeletedPayload struct {
	CharacterID string `json:"character_id"`
	Reason      string `json:"reason,omitempty"`
}

// CharacterUpdatedPayload captures the payload for character.updated events.
type CharacterUpdatedPayload struct {
	CharacterID string         `json:"character_id"`
	Fields      map[string]any `json:"fields"`
}

// ProfileUpdatedPayload captures the payload for character.profile_updated events.
type ProfileUpdatedPayload struct {
	CharacterID string `json:"character_id"`
	// Core profile fields
	HpMax *int `json:"hp_max,omitempty"`
	// SystemProfile holds game-system-specific profile updates.
	SystemProfile map[string]any `json:"system_profile,omitempty"`
}

// ControllerAssignedPayload captures the payload for character.controller_assigned events.
type ControllerAssignedPayload struct {
	CharacterID string `json:"character_id"`
	// IsGM indicates if the GM controls this character.
	IsGM bool `json:"is_gm"`
	// ParticipantID is set if a specific participant controls the character.
	ParticipantID string `json:"participant_id,omitempty"`
}

// CharacterStateChangedPayload captures the payload for snapshot character state changed events.
type CharacterStateChangedPayload struct {
	CharacterID string `json:"character_id"`
	// Core state fields (before/after for audit)
	HpBefore *int `json:"hp_before,omitempty"`
	HpAfter  *int `json:"hp_after,omitempty"`
	// SystemState holds game-system-specific state changes.
	SystemState map[string]any `json:"system_state,omitempty"`
}

// GMFearChangedPayload captures the payload for snapshot GM fear changed events.
type GMFearChangedPayload struct {
	Before int    `json:"before"`
	After  int    `json:"after"`
	Reason string `json:"reason,omitempty"`
}

// SessionStartedPayload captures the payload for session.started events.
type SessionStartedPayload struct {
	SessionID   string `json:"session_id"`
	SessionName string `json:"session_name,omitempty"`
}

// SessionEndedPayload captures the payload for session.ended events.
type SessionEndedPayload struct {
	SessionID string `json:"session_id"`
}

// RollResolvedPayload captures the payload for action.roll_resolved events.
type RollResolvedPayload struct {
	RequestID string `json:"request_id"`
	RollSeq   uint64 `json:"roll_seq"`
	// Results holds the dice roll results.
	Results map[string]any `json:"results"`
	// Outcome holds the determined outcome (e.g., "success", "failure").
	Outcome string `json:"outcome,omitempty"`
	// SystemData holds game-system-specific resolution data.
	SystemData map[string]any `json:"system_data,omitempty"`
}

// OutcomeAppliedChange captures a single applied change.
type OutcomeAppliedChange struct {
	CharacterID string `json:"character_id,omitempty"`
	Field       string `json:"field"`
	Before      int    `json:"before"`
	After       int    `json:"after"`
}

// OutcomeAppliedPayload captures the payload for action.outcome_applied events.
type OutcomeAppliedPayload struct {
	RequestID            string                 `json:"request_id"`
	RollSeq              uint64                 `json:"roll_seq"`
	Targets              []string               `json:"targets"`
	RequiresComplication bool                   `json:"requires_complication"`
	AppliedChanges       []OutcomeAppliedChange `json:"applied_changes,omitempty"`
}

// OutcomeRejectedPayload captures the payload for action.outcome_rejected events.
type OutcomeRejectedPayload struct {
	RequestID  string `json:"request_id"`
	RollSeq    uint64 `json:"roll_seq"`
	ReasonCode string `json:"reason_code"`
	Message    string `json:"message,omitempty"`
}

// NoteAddedPayload captures the payload for action.note_added events.
type NoteAddedPayload struct {
	Content     string `json:"content"`
	CharacterID string `json:"character_id,omitempty"`
}

package event

// CampaignCreatedPayload captures the payload for campaign.created events.
type CampaignCreatedPayload struct {
	Name         string `json:"name"`
	Locale       string `json:"locale"`
	GameSystem   string `json:"game_system"`
	GmMode       string `json:"gm_mode"`
	Intent       string `json:"intent,omitempty"`
	AccessPolicy string `json:"access_policy,omitempty"`
	ThemePrompt  string `json:"theme_prompt,omitempty"`
}

// CampaignForkedPayload captures the payload for campaign.forked events.
type CampaignForkedPayload struct {
	ParentCampaignID string `json:"parent_campaign_id"`
	ForkEventSeq     uint64 `json:"fork_event_seq"`
	OriginCampaignID string `json:"origin_campaign_id"`
	CopyParticipants bool   `json:"copy_participants"`
}

// CampaignUpdatedPayload captures the payload for campaign.updated events.
type CampaignUpdatedPayload struct {
	Fields map[string]any `json:"fields"`
}

// ParticipantJoinedPayload captures the payload for participant.joined events.
type ParticipantJoinedPayload struct {
	ParticipantID  string `json:"participant_id"`
	UserID         string `json:"user_id"`
	DisplayName    string `json:"display_name"`
	Role           string `json:"role"`
	Controller     string `json:"controller"`
	CampaignAccess string `json:"campaign_access"`
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

// ParticipantBoundPayload captures the payload for participant.bound events.
type ParticipantBoundPayload struct {
	ParticipantID string `json:"participant_id"`
	UserID        string `json:"user_id"`
}

// ParticipantUnboundPayload captures the payload for participant.unbound events.
type ParticipantUnboundPayload struct {
	ParticipantID string `json:"participant_id"`
	UserID        string `json:"user_id"`
	Reason        string `json:"reason,omitempty"`
}

// SeatReassignedPayload captures the payload for seat.reassigned events.
type SeatReassignedPayload struct {
	ParticipantID string `json:"participant_id"`
	PriorUserID   string `json:"prior_user_id"`
	UserID        string `json:"user_id"`
	Reason        string `json:"reason,omitempty"`
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
	// SystemProfile holds game-system-specific profile updates.
	SystemProfile map[string]any `json:"system_profile,omitempty"`
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

// SessionGateOpenedPayload captures the payload for session.gate_opened events.
type SessionGateOpenedPayload struct {
	GateID   string         `json:"gate_id"`
	GateType string         `json:"gate_type"`
	Reason   string         `json:"reason,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// SessionGateResolvedPayload captures the payload for session.gate_resolved events.
type SessionGateResolvedPayload struct {
	GateID     string         `json:"gate_id"`
	Decision   string         `json:"decision,omitempty"`
	Resolution map[string]any `json:"resolution,omitempty"`
}

// SessionGateAbandonedPayload captures the payload for session.gate_abandoned events.
type SessionGateAbandonedPayload struct {
	GateID string `json:"gate_id"`
	Reason string `json:"reason,omitempty"`
}

// SessionSpotlightSetPayload captures the payload for session.spotlight_set events.
type SessionSpotlightSetPayload struct {
	SpotlightType string `json:"spotlight_type"`
	CharacterID   string `json:"character_id,omitempty"`
}

// SessionSpotlightClearedPayload captures the payload for session.spotlight_cleared events.
type SessionSpotlightClearedPayload struct {
	Reason string `json:"reason,omitempty"`
}

// InviteCreatedPayload captures the payload for invite.created events.
type InviteCreatedPayload struct {
	InviteID               string `json:"invite_id"`
	ParticipantID          string `json:"participant_id"`
	RecipientUserID        string `json:"recipient_user_id,omitempty"`
	Status                 string `json:"status"`
	CreatedByParticipantID string `json:"created_by_participant_id,omitempty"`
}

// InviteUpdatedPayload captures the payload for invite.updated events.
type InviteUpdatedPayload struct {
	InviteID string `json:"invite_id"`
	Status   string `json:"status"`
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

// InviteClaimedPayload captures the payload for invite.claimed events.
type InviteClaimedPayload struct {
	InviteID      string `json:"invite_id"`
	ParticipantID string `json:"participant_id"`
	UserID        string `json:"user_id"`
	JWTID         string `json:"jti"`
}

// InviteRevokedPayload captures the payload for invite.revoked events.
type InviteRevokedPayload struct {
	InviteID string `json:"invite_id"`
}

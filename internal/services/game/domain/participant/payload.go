package participant

// JoinPayload captures the payload for participant.join commands and participant.joined events.
type JoinPayload struct {
	ParticipantID  string `json:"participant_id"`
	UserID         string `json:"user_id"`
	Name           string `json:"name"`
	Role           string `json:"role"`
	Controller     string `json:"controller"`
	CampaignAccess string `json:"campaign_access"`
	AvatarSetID    string `json:"avatar_set_id,omitempty"`
	AvatarAssetID  string `json:"avatar_asset_id,omitempty"`
}

// UpdatePayload captures the payload for participant.update commands and participant.updated events.
type UpdatePayload struct {
	ParticipantID string            `json:"participant_id"`
	Fields        map[string]string `json:"fields"`
}

// LeavePayload captures the payload for participant.leave commands and participant.left events.
type LeavePayload struct {
	ParticipantID string `json:"participant_id"`
	Reason        string `json:"reason,omitempty"`
}

// BindPayload captures the payload for participant.bind commands and participant.bound events.
type BindPayload struct {
	ParticipantID string `json:"participant_id"`
	UserID        string `json:"user_id"`
}

// UnbindPayload captures the payload for participant.unbind commands and participant.unbound events.
type UnbindPayload struct {
	ParticipantID string `json:"participant_id"`
	UserID        string `json:"user_id,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

// SeatReassignPayload captures the payload for seat.reassign commands and seat.reassigned events.
type SeatReassignPayload struct {
	ParticipantID string `json:"participant_id"`
	PriorUserID   string `json:"prior_user_id,omitempty"`
	UserID        string `json:"user_id"`
	Reason        string `json:"reason,omitempty"`
}

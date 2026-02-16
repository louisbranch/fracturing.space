package invite

// CreatePayload captures the payload for invite.create commands and invite.created events.
type CreatePayload struct {
	InviteID               string `json:"invite_id"`
	ParticipantID          string `json:"participant_id"`
	RecipientUserID        string `json:"recipient_user_id,omitempty"`
	CreatedByParticipantID string `json:"created_by_participant_id,omitempty"`
	Status                 string `json:"status"`
}

// ClaimPayload captures the payload for invite.claim commands and invite.claimed events.
type ClaimPayload struct {
	InviteID      string `json:"invite_id"`
	ParticipantID string `json:"participant_id"`
	UserID        string `json:"user_id"`
	JWTID         string `json:"jti"`
}

// RevokePayload captures the payload for invite.revoke commands and invite.revoked events.
type RevokePayload struct {
	InviteID string `json:"invite_id"`
}

// UpdatePayload captures the payload for invite.update commands and invite.updated events.
type UpdatePayload struct {
	InviteID string `json:"invite_id"`
	Status   string `json:"status"`
}

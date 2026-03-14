package invite

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"

// CreatePayload captures the payload for invite.create commands and invite.created events.
type CreatePayload struct {
	InviteID               ids.InviteID      `json:"invite_id"`
	ParticipantID          ids.ParticipantID `json:"participant_id"`
	RecipientUserID        ids.UserID        `json:"recipient_user_id,omitempty"`
	CreatedByParticipantID ids.ParticipantID `json:"created_by_participant_id,omitempty"`
	Status                 string            `json:"status"`
}

// ClaimPayload captures the payload for invite.claim commands and invite.claimed events.
type ClaimPayload struct {
	InviteID      ids.InviteID      `json:"invite_id"`
	ParticipantID ids.ParticipantID `json:"participant_id"`
	UserID        ids.UserID        `json:"user_id"`
	JWTID         string            `json:"jti"`
}

// DeclinePayload captures the payload for invite.decline commands and invite.declined events.
type DeclinePayload struct {
	InviteID ids.InviteID `json:"invite_id"`
	UserID   ids.UserID   `json:"user_id"`
}

// RevokePayload captures the payload for invite.revoke commands and invite.revoked events.
type RevokePayload struct {
	InviteID ids.InviteID `json:"invite_id"`
}

// UpdatePayload captures the payload for invite.update commands and invite.updated events.
type UpdatePayload struct {
	InviteID ids.InviteID `json:"invite_id"`
	Status   string       `json:"status"`
}

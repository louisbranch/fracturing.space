package inviteclaimworkflow

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"

// ClaimBindPayload captures the workflow input for atomic participant bind and
// invite claim orchestration.
type ClaimBindPayload struct {
	InviteID      ids.InviteID      `json:"invite_id"`
	ParticipantID ids.ParticipantID `json:"participant_id"`
	UserID        ids.UserID        `json:"user_id"`
	JWTID         string            `json:"jti"`
}

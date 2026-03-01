package invite

import (
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

const (
	CommandTypeCreate command.Type = "invite.create"
	CommandTypeClaim  command.Type = "invite.claim"
	CommandTypeRevoke command.Type = "invite.revoke"
	CommandTypeUpdate command.Type = "invite.update"
	EventTypeCreated  event.Type   = "invite.created"
	EventTypeClaimed  event.Type   = "invite.claimed"
	EventTypeRevoked  event.Type   = "invite.revoked"
	EventTypeUpdated  event.Type   = "invite.updated"

	statusPending = "pending"
	statusClaimed = "claimed"
	statusRevoked = "revoked"

	rejectionCodeInviteAlreadyExists     = "INVITE_ALREADY_EXISTS"
	rejectionCodeInviteIDRequired        = "INVITE_ID_REQUIRED"
	rejectionCodeInviteParticipantNeeded = "INVITE_PARTICIPANT_ID_REQUIRED"
	rejectionCodeInviteNotCreated        = "INVITE_NOT_CREATED"
	rejectionCodeInviteStatusInvalid     = "INVITE_STATUS_INVALID"
	rejectionCodeInviteUserIDRequired    = "INVITE_USER_ID_REQUIRED"
	rejectionCodeInviteJWTRequired       = "INVITE_JTI_REQUIRED"
)

// Decide returns the decision for an invite command against current state.
//
// Invite flow is intentionally strict because it gates who can participate in a
// campaign. Each transition emits an immutable state event that can be audited
// and replayed for investigation or migration.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	switch cmd.Type {
	case CommandTypeCreate:
		return decideCreate(state, cmd, now)
	case CommandTypeClaim:
		return decideClaim(state, cmd, now)
	case CommandTypeRevoke:
		return decideRevoke(state, cmd, now)
	case CommandTypeUpdate:
		return decideUpdate(state, cmd, now)
	default:
		return command.Reject(command.Rejection{Code: "COMMAND_TYPE_UNSUPPORTED", Message: fmt.Sprintf("command type %s is not supported by invite decider", cmd.Type)})
	}
}

func decideCreate(state State, cmd command.Command, now func() time.Time) command.Decision {
	if state.Created {
		return command.Reject(command.Rejection{Code: rejectionCodeInviteAlreadyExists, Message: "invite already exists"})
	}
	// create always derives entity routing from payload for deterministic identity.
	cmd.EntityID = ""
	cmd.EntityType = ""
	return module.DecideFunc(
		cmd,
		EventTypeCreated,
		"invite",
		func(payload *CreatePayload) string {
			return payload.InviteID
		},
		func(payload *CreatePayload, _ func() time.Time) *command.Rejection {
			payload.InviteID = strings.TrimSpace(payload.InviteID)
			if payload.InviteID == "" {
				return &command.Rejection{Code: rejectionCodeInviteIDRequired, Message: "invite id is required"}
			}
			payload.ParticipantID = strings.TrimSpace(payload.ParticipantID)
			if payload.ParticipantID == "" {
				return &command.Rejection{Code: rejectionCodeInviteParticipantNeeded, Message: "participant id is required"}
			}
			payload.RecipientUserID = strings.TrimSpace(payload.RecipientUserID)
			payload.CreatedByParticipantID = strings.TrimSpace(payload.CreatedByParticipantID)
			payload.Status = statusPending
			return nil
		},
		now,
	)
}

func decideClaim(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{Code: rejectionCodeInviteNotCreated, Message: "invite not created"})
	}
	if state.Status != "" && state.Status != statusPending {
		return command.Reject(command.Rejection{Code: rejectionCodeInviteStatusInvalid, Message: "invite status is invalid"})
	}
	// claim always derives entity routing from payload for deterministic identity.
	cmd.EntityID = ""
	cmd.EntityType = ""
	return module.DecideFunc(
		cmd,
		EventTypeClaimed,
		"invite",
		func(payload *ClaimPayload) string {
			return payload.InviteID
		},
		func(payload *ClaimPayload, _ func() time.Time) *command.Rejection {
			payload.InviteID = strings.TrimSpace(payload.InviteID)
			if payload.InviteID == "" {
				return &command.Rejection{Code: rejectionCodeInviteIDRequired, Message: "invite id is required"}
			}
			payload.ParticipantID = strings.TrimSpace(payload.ParticipantID)
			if payload.ParticipantID == "" {
				return &command.Rejection{Code: rejectionCodeInviteParticipantNeeded, Message: "participant id is required"}
			}
			payload.UserID = strings.TrimSpace(payload.UserID)
			if payload.UserID == "" {
				return &command.Rejection{Code: rejectionCodeInviteUserIDRequired, Message: "user id is required"}
			}
			payload.JWTID = strings.TrimSpace(payload.JWTID)
			if payload.JWTID == "" {
				return &command.Rejection{Code: rejectionCodeInviteJWTRequired, Message: "jti is required"}
			}
			return nil
		},
		now,
	)
}

func decideRevoke(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{Code: rejectionCodeInviteNotCreated, Message: "invite not created"})
	}
	if state.Status == statusClaimed || state.Status == statusRevoked {
		return command.Reject(command.Rejection{Code: rejectionCodeInviteStatusInvalid, Message: "invite status is invalid"})
	}
	return module.DecideFunc(
		cmd,
		EventTypeRevoked,
		"invite",
		func(payload *RevokePayload) string {
			return payload.InviteID
		},
		func(payload *RevokePayload, _ func() time.Time) *command.Rejection {
			payload.InviteID = strings.TrimSpace(payload.InviteID)
			if payload.InviteID == "" {
				return &command.Rejection{Code: rejectionCodeInviteIDRequired, Message: "invite id is required"}
			}
			return nil
		},
		now,
	)
}

func decideUpdate(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{Code: rejectionCodeInviteNotCreated, Message: "invite not created"})
	}
	return module.DecideFunc(
		cmd,
		EventTypeUpdated,
		"invite",
		func(payload *UpdatePayload) string {
			return payload.InviteID
		},
		func(payload *UpdatePayload, _ func() time.Time) *command.Rejection {
			payload.InviteID = strings.TrimSpace(payload.InviteID)
			if payload.InviteID == "" {
				return &command.Rejection{Code: rejectionCodeInviteIDRequired, Message: "invite id is required"}
			}
			status, ok := normalizeStatusLabel(payload.Status)
			if !ok {
				return &command.Rejection{Code: rejectionCodeInviteStatusInvalid, Message: "invite status is invalid"}
			}
			payload.Status = status
			return nil
		},
		now,
	)
}

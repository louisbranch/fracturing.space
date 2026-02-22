package invite

import (
	"encoding/json"
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
		if state.Created {
			return command.Reject(command.Rejection{Code: rejectionCodeInviteAlreadyExists, Message: "invite already exists"})
		}
		var payload CreatePayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{Code: "PAYLOAD_DECODE_FAILED", Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
		}
		inviteID := strings.TrimSpace(payload.InviteID)
		if inviteID == "" {
			return command.Reject(command.Rejection{Code: rejectionCodeInviteIDRequired, Message: "invite id is required"})
		}
		participantID := strings.TrimSpace(payload.ParticipantID)
		if participantID == "" {
			return command.Reject(command.Rejection{Code: rejectionCodeInviteParticipantNeeded, Message: "participant id is required"})
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := CreatePayload{
			InviteID:               inviteID,
			ParticipantID:          participantID,
			RecipientUserID:        strings.TrimSpace(payload.RecipientUserID),
			CreatedByParticipantID: strings.TrimSpace(payload.CreatedByParticipantID),
			Status:                 statusPending,
		}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		return command.Accept(command.NewEvent(cmd, EventTypeCreated, "invite", inviteID, payloadJSON, now().UTC()))

	case CommandTypeClaim:
		if !state.Created {
			return command.Reject(command.Rejection{Code: rejectionCodeInviteNotCreated, Message: "invite not created"})
		}
		if state.Status != "" && state.Status != statusPending {
			return command.Reject(command.Rejection{Code: rejectionCodeInviteStatusInvalid, Message: "invite status is invalid"})
		}
		var payload ClaimPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{Code: "PAYLOAD_DECODE_FAILED", Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
		}
		inviteID := strings.TrimSpace(payload.InviteID)
		if inviteID == "" {
			return command.Reject(command.Rejection{Code: rejectionCodeInviteIDRequired, Message: "invite id is required"})
		}
		participantID := strings.TrimSpace(payload.ParticipantID)
		if participantID == "" {
			return command.Reject(command.Rejection{Code: rejectionCodeInviteParticipantNeeded, Message: "participant id is required"})
		}
		userID := strings.TrimSpace(payload.UserID)
		if userID == "" {
			return command.Reject(command.Rejection{Code: rejectionCodeInviteUserIDRequired, Message: "user id is required"})
		}
		jwtID := strings.TrimSpace(payload.JWTID)
		if jwtID == "" {
			return command.Reject(command.Rejection{Code: rejectionCodeInviteJWTRequired, Message: "jti is required"})
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := ClaimPayload{
			InviteID:      inviteID,
			ParticipantID: participantID,
			UserID:        userID,
			JWTID:         jwtID,
		}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		return command.Accept(command.NewEvent(cmd, EventTypeClaimed, "invite", inviteID, payloadJSON, now().UTC()))

	case CommandTypeRevoke:
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

	case CommandTypeUpdate:
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

	default:
		return command.Reject(command.Rejection{Code: "COMMAND_TYPE_UNSUPPORTED", Message: fmt.Sprintf("command type %s is not supported by invite decider", cmd.Type)})
	}
}

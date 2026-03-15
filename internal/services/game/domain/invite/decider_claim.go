package invite

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideClaim(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{Code: rejectionCodeInviteNotCreated, Message: "invite not created"})
	}
	if state.Status != "" && state.Status != statusPending {
		return command.Reject(command.Rejection{Code: rejectionCodeInviteStatusInvalid, Message: "invite status is invalid"})
	}
	cmd.EntityID = ""
	cmd.EntityType = ""
	return module.DecideFunc(
		cmd,
		EventTypeClaimed,
		"invite",
		func(payload *ClaimPayload) string {
			return payload.InviteID.String()
		},
		func(payload *ClaimPayload, _ func() time.Time) *command.Rejection {
			payload.InviteID = ids.InviteID(strings.TrimSpace(payload.InviteID.String()))
			if payload.InviteID == "" {
				return &command.Rejection{Code: rejectionCodeInviteIDRequired, Message: "invite id is required"}
			}
			payload.ParticipantID = ids.ParticipantID(strings.TrimSpace(payload.ParticipantID.String()))
			if payload.ParticipantID == "" {
				return &command.Rejection{Code: rejectionCodeInviteParticipantNeeded, Message: "participant id is required"}
			}
			payload.UserID = ids.UserID(strings.TrimSpace(payload.UserID.String()))
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

func decideDecline(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{Code: rejectionCodeInviteNotCreated, Message: "invite not created"})
	}
	if state.Status != "" && state.Status != statusPending {
		return command.Reject(command.Rejection{Code: rejectionCodeInviteStatusInvalid, Message: "invite status is invalid"})
	}
	return module.DecideFunc(
		cmd,
		EventTypeDeclined,
		"invite",
		func(payload *DeclinePayload) string {
			return payload.InviteID.String()
		},
		func(payload *DeclinePayload, _ func() time.Time) *command.Rejection {
			payload.InviteID = ids.InviteID(strings.TrimSpace(payload.InviteID.String()))
			if payload.InviteID == "" {
				return &command.Rejection{Code: rejectionCodeInviteIDRequired, Message: "invite id is required"}
			}
			payload.UserID = ids.UserID(strings.TrimSpace(payload.UserID.String()))
			if payload.UserID == "" {
				return &command.Rejection{Code: rejectionCodeInviteUserIDRequired, Message: "user id is required"}
			}
			return nil
		},
		now,
	)
}

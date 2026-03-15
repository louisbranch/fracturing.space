package invite

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideCreate(state State, cmd command.Command, now func() time.Time) command.Decision {
	if state.Created {
		return command.Reject(command.Rejection{Code: rejectionCodeInviteAlreadyExists, Message: "invite already exists"})
	}
	cmd.EntityID = ""
	cmd.EntityType = ""
	return module.DecideFunc(
		cmd,
		EventTypeCreated,
		"invite",
		func(payload *CreatePayload) string {
			return payload.InviteID.String()
		},
		func(payload *CreatePayload, _ func() time.Time) *command.Rejection {
			payload.InviteID = ids.InviteID(strings.TrimSpace(payload.InviteID.String()))
			if payload.InviteID == "" {
				return &command.Rejection{Code: rejectionCodeInviteIDRequired, Message: "invite id is required"}
			}
			payload.ParticipantID = ids.ParticipantID(strings.TrimSpace(payload.ParticipantID.String()))
			if payload.ParticipantID == "" {
				return &command.Rejection{Code: rejectionCodeInviteParticipantNeeded, Message: "participant id is required"}
			}
			payload.RecipientUserID = ids.UserID(strings.TrimSpace(payload.RecipientUserID.String()))
			payload.CreatedByParticipantID = ids.ParticipantID(strings.TrimSpace(payload.CreatedByParticipantID.String()))
			payload.Status = statusPending
			return nil
		},
		now,
	)
}

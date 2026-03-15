package invite

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideRevoke(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{Code: rejectionCodeInviteNotCreated, Message: "invite not created"})
	}
	if state.Status == statusClaimed || state.Status == statusDeclined || state.Status == statusRevoked {
		return command.Reject(command.Rejection{Code: rejectionCodeInviteStatusInvalid, Message: "invite status is invalid"})
	}
	return module.DecideFunc(
		cmd,
		EventTypeRevoked,
		"invite",
		func(payload *RevokePayload) string {
			return payload.InviteID.String()
		},
		func(payload *RevokePayload, _ func() time.Time) *command.Rejection {
			payload.InviteID = ids.InviteID(strings.TrimSpace(payload.InviteID.String()))
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
			return payload.InviteID.String()
		},
		func(payload *UpdatePayload, _ func() time.Time) *command.Rejection {
			payload.InviteID = ids.InviteID(strings.TrimSpace(payload.InviteID.String()))
			if payload.InviteID == "" {
				return &command.Rejection{Code: rejectionCodeInviteIDRequired, Message: "invite id is required"}
			}
			status, ok := NormalizeStatusLabel(payload.Status)
			if !ok {
				return &command.Rejection{Code: rejectionCodeInviteStatusInvalid, Message: "invite status is invalid"}
			}
			payload.Status = status
			return nil
		},
		now,
	)
}

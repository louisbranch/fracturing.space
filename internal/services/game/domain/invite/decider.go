package invite

import (
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	CommandTypeCreate  command.Type = "invite.create"
	CommandTypeClaim   command.Type = "invite.claim"
	CommandTypeDecline command.Type = "invite.decline"
	CommandTypeRevoke  command.Type = "invite.revoke"
	CommandTypeUpdate  command.Type = "invite.update"
	EventTypeCreated   event.Type   = "invite.created"
	EventTypeClaimed   event.Type   = "invite.claimed"
	EventTypeDeclined  event.Type   = "invite.declined"
	EventTypeRevoked   event.Type   = "invite.revoked"
	EventTypeUpdated   event.Type   = "invite.updated"

	statusPending  = "pending"
	statusClaimed  = "claimed"
	statusDeclined = "declined"
	statusRevoked  = "revoked"

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
	case CommandTypeDecline:
		return decideDecline(state, cmd, now)
	case CommandTypeRevoke:
		return decideRevoke(state, cmd, now)
	case CommandTypeUpdate:
		return decideUpdate(state, cmd, now)
	default:
		return command.Reject(command.Rejection{Code: command.RejectionCodeCommandTypeUnsupported, Message: fmt.Sprintf("command type %s is not supported by invite decider", cmd.Type)})
	}
}

package participant

import (
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	CommandTypeJoin         command.Type = "participant.join"
	CommandTypeUpdate       command.Type = "participant.update"
	CommandTypeLeave        command.Type = "participant.leave"
	CommandTypeBind         command.Type = "participant.bind"
	CommandTypeUnbind       command.Type = "participant.unbind"
	CommandTypeSeatReassign command.Type = "participant.seat.reassign"
	EventTypeJoined         event.Type   = "participant.joined"
	EventTypeUpdated        event.Type   = "participant.updated"
	EventTypeLeft           event.Type   = "participant.left"
	EventTypeBound          event.Type   = "participant.bound"
	EventTypeUnbound        event.Type   = "participant.unbound"
	EventTypeSeatReassigned event.Type   = "participant.seat_reassigned"

	rejectionCodeParticipantAlreadyJoined      = "PARTICIPANT_ALREADY_JOINED"
	rejectionCodeParticipantNotJoined          = "PARTICIPANT_NOT_JOINED"
	rejectionCodeParticipantIDRequired         = "PARTICIPANT_ID_REQUIRED"
	rejectionCodeParticipantNameEmpty          = "PARTICIPANT_NAME_EMPTY"
	rejectionCodeParticipantRoleInvalid        = "PARTICIPANT_INVALID_ROLE"
	rejectionCodeParticipantControllerInvalid  = "PARTICIPANT_INVALID_CONTROLLER"
	rejectionCodeParticipantAccessInvalid      = "PARTICIPANT_INVALID_CAMPAIGN_ACCESS"
	rejectionCodeParticipantAvatarSetInvalid   = "PARTICIPANT_INVALID_AVATAR_SET"
	rejectionCodeParticipantAvatarAssetInvalid = "PARTICIPANT_INVALID_AVATAR_ASSET"
	rejectionCodeParticipantUpdateEmpty        = "PARTICIPANT_UPDATE_EMPTY"
	rejectionCodeParticipantUpdateFieldInvalid = "PARTICIPANT_UPDATE_FIELD_INVALID"
	rejectionCodeParticipantAlreadyClaimed     = "PARTICIPANT_ALREADY_CLAIMED"
	rejectionCodeParticipantUserIDRequired     = "PARTICIPANT_USER_ID_REQUIRED"
	rejectionCodeParticipantUserIDMismatch     = "PARTICIPANT_USER_ID_MISMATCH"
	rejectionCodeParticipantAIRoleRequired     = "PARTICIPANT_AI_ROLE_REQUIRED"
	rejectionCodeParticipantAIAccessRequired   = "PARTICIPANT_AI_ACCESS_REQUIRED"
	rejectionCodeParticipantAIUserIDForbidden  = "PARTICIPANT_AI_USER_ID_FORBIDDEN"
	rejectionCodeParticipantAIIdentityLocked   = "PARTICIPANT_AI_IDENTITY_LOCKED"
)

// RejectionCodes returns all rejection code strings used by the participant
// decider. Used by startup validators to detect cross-domain collisions.
func RejectionCodes() []string {
	return []string{
		rejectionCodeParticipantAlreadyJoined,
		rejectionCodeParticipantNotJoined,
		rejectionCodeParticipantIDRequired,
		rejectionCodeParticipantNameEmpty,
		rejectionCodeParticipantRoleInvalid,
		rejectionCodeParticipantControllerInvalid,
		rejectionCodeParticipantAccessInvalid,
		rejectionCodeParticipantAvatarSetInvalid,
		rejectionCodeParticipantAvatarAssetInvalid,
		rejectionCodeParticipantUpdateEmpty,
		rejectionCodeParticipantUpdateFieldInvalid,
		rejectionCodeParticipantAlreadyClaimed,
		rejectionCodeParticipantUserIDRequired,
		rejectionCodeParticipantUserIDMismatch,
		rejectionCodeParticipantAIRoleRequired,
		rejectionCodeParticipantAIAccessRequired,
		rejectionCodeParticipantAIUserIDForbidden,
		rejectionCodeParticipantAIIdentityLocked,
	}
}

// Decide returns the decision for a participant command against current state.
//
// Participant commands define membership and authorization context. This decider keeps
// that context explicit by emitting identity/role/capability changes as immutable
// events rather than mutating shared storage directly.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	switch cmd.Type {
	case CommandTypeJoin:
		return decideJoin(state, cmd, now)
	case CommandTypeUpdate:
		return decideUpdate(state, cmd, now)
	case CommandTypeLeave:
		return decideLeave(state, cmd, now)
	case CommandTypeBind:
		return decideBind(state, cmd, now)
	case CommandTypeUnbind:
		return decideUnbind(state, cmd, now)
	case CommandTypeSeatReassign:
		return decideSeatReassign(state, cmd, now)
	default:
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodeCommandTypeUnsupported,
			Message: fmt.Sprintf("command type %s is not supported by participant decider", cmd.Type),
		})
	}
}

package character

import (
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	CommandTypeCreate command.Type = "character.create"
	CommandTypeUpdate command.Type = "character.update"
	CommandTypeDelete command.Type = "character.delete"
	EventTypeCreated  event.Type   = "character.created"
	EventTypeUpdated  event.Type   = "character.updated"
	EventTypeDeleted  event.Type   = "character.deleted"

	rejectionCodeCharacterAlreadyExists      = "CHARACTER_ALREADY_EXISTS"
	rejectionCodeCharacterIDRequired         = "CHARACTER_ID_REQUIRED"
	rejectionCodeCharacterNameEmpty          = "CHARACTER_NAME_EMPTY"
	rejectionCodeCharacterKindInvalid        = "CHARACTER_KIND_INVALID"
	rejectionCodeCharacterAvatarSetInvalid   = "CHARACTER_INVALID_AVATAR_SET"
	rejectionCodeCharacterAvatarAssetInvalid = "CHARACTER_INVALID_AVATAR_ASSET"
	rejectionCodeCharacterNotCreated         = "CHARACTER_NOT_CREATED"
	rejectionCodeCharacterUpdateEmpty        = "CHARACTER_UPDATE_EMPTY"
	rejectionCodeCharacterUpdateFieldInvalid = "CHARACTER_UPDATE_FIELD_INVALID"
	rejectionCodeCharacterAliasesInvalid     = "CHARACTER_ALIASES_INVALID"
	rejectionCodeCharacterOwnerParticipantID = "CHARACTER_OWNER_PARTICIPANT_ID_REQUIRED"
)

// RejectionCodes returns all rejection code strings used by the character
// decider. Used by startup validators to detect cross-domain collisions.
func RejectionCodes() []string {
	return []string{
		rejectionCodeCharacterAlreadyExists,
		rejectionCodeCharacterIDRequired,
		rejectionCodeCharacterNameEmpty,
		rejectionCodeCharacterKindInvalid,
		rejectionCodeCharacterAvatarSetInvalid,
		rejectionCodeCharacterAvatarAssetInvalid,
		rejectionCodeCharacterNotCreated,
		rejectionCodeCharacterUpdateEmpty,
		rejectionCodeCharacterUpdateFieldInvalid,
		rejectionCodeCharacterAliasesInvalid,
		rejectionCodeCharacterOwnerParticipantID,
	}
}

// Decide returns the decision for a character command against current state.
//
// Character changes are intentionally event-driven so ownership and profile edits
// can be replayed and projected consistently across tools and clients.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	now = command.NowFunc(now)

	switch cmd.Type {
	case CommandTypeCreate:
		return decideCreate(state, cmd, now)

	case CommandTypeUpdate:
		return decideUpdate(state, cmd, now)

	case CommandTypeDelete:
		return decideDelete(state, cmd, now)

	default:
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodeCommandTypeUnsupported,
			Message: fmt.Sprintf("command type %s is not supported by character decider", cmd.Type),
		})
	}
}

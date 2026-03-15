package campaign

import (
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	CommandTypeCreate                 command.Type = "campaign.create"
	CommandTypeCreateWithParticipants command.Type = "campaign.create_with_participants"
	CommandTypeUpdate                 command.Type = "campaign.update"
	CommandTypeAIBind                 command.Type = "campaign.ai_bind"
	CommandTypeAIUnbind               command.Type = "campaign.ai_unbind"
	CommandTypeAIAuthRotate           command.Type = "campaign.ai_auth_rotate"
	CommandTypeFork                   command.Type = "campaign.fork"
	CommandTypeEnd                    command.Type = "campaign.end"
	CommandTypeArchive                command.Type = "campaign.archive"
	CommandTypeRestore                command.Type = "campaign.restore"
	EventTypeCreated                  event.Type   = "campaign.created"
	EventTypeUpdated                  event.Type   = "campaign.updated"
	EventTypeAIBound                  event.Type   = "campaign.ai_bound"
	EventTypeAIUnbound                event.Type   = "campaign.ai_unbound"
	EventTypeAIAuthRotated            event.Type   = "campaign.ai_auth_rotated"
	EventTypeForked                   event.Type   = "campaign.forked"

	rejectionCodeCampaignAlreadyExists        = "CAMPAIGN_ALREADY_EXISTS"
	rejectionCodeCampaignNotCreated           = "CAMPAIGN_NOT_CREATED"
	rejectionCodeCampaignNameEmpty            = "CAMPAIGN_NAME_EMPTY"
	rejectionCodeCampaignGameSystemInvalid    = "CAMPAIGN_INVALID_GAME_SYSTEM"
	rejectionCodeCampaignGmModeInvalid        = "CAMPAIGN_INVALID_GM_MODE"
	rejectionCodeCampaignUpdateEmpty          = "CAMPAIGN_UPDATE_EMPTY"
	rejectionCodeCampaignStatusInvalid        = "CAMPAIGN_INVALID_STATUS"
	rejectionCodeCampaignStatusTransition     = "CAMPAIGN_INVALID_STATUS_TRANSITION"
	rejectionCodeCampaignUpdateFieldInvalid   = "CAMPAIGN_UPDATE_FIELD_INVALID"
	rejectionCodeCampaignLocaleInvalid        = "CAMPAIGN_LOCALE_INVALID"
	rejectionCodeCampaignCoverAssetInvalid    = "CAMPAIGN_COVER_ASSET_INVALID"
	rejectionCodeCampaignCoverSetInvalid      = "CAMPAIGN_COVER_SET_INVALID"
	rejectionCodeCampaignAIAgentIDRequired    = "CAMPAIGN_AI_AGENT_ID_REQUIRED"
	rejectionCodeCampaignParticipantsEmpty    = "CAMPAIGN_PARTICIPANTS_REQUIRED"
	rejectionCodeCampaignParticipantDuplicate = "CAMPAIGN_PARTICIPANT_DUPLICATE"
	rejectionCodeCommandTypeUnsupported       = command.RejectionCodeCommandTypeUnsupported
	rejectionCodePayloadDecodeFailed          = command.RejectionCodePayloadDecodeFailed
)

// Decide returns the decision for a campaign command against current state.
//
// This function is the campaign policy hub: it normalizes command payloads,
// enforces legal transitions, and emits immutable events that can be replayed
// to reproduce the same campaign state.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	switch cmd.Type {
	case CommandTypeCreate:
		return decideCreate(state, cmd, now)
	case CommandTypeUpdate:
		return decideUpdate(state, cmd, now)
	case CommandTypeAIBind:
		return decideAIBind(state, cmd, now)
	case CommandTypeAIUnbind:
		return decideAIUnbind(state, cmd, now)
	case CommandTypeAIAuthRotate:
		return decideAIAuthRotate(state, cmd, now)
	case CommandTypeFork:
		return decideFork(state, cmd, now)
	case CommandTypeEnd, CommandTypeArchive, CommandTypeRestore:
		return decideLifecycleStatus(state, cmd, now)
	default:
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCommandTypeUnsupported,
			Message: fmt.Sprintf("command type %s is not supported by campaign decider", cmd.Type),
		})
	}
}

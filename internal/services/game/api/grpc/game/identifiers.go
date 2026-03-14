package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	commandTypeActionOutcomeApply             command.Type = commandids.ActionOutcomeApply
	commandTypeActionOutcomeReject            command.Type = commandids.ActionOutcomeReject
	commandTypeActionRollResolve              command.Type = commandids.ActionRollResolve
	commandTypeCampaignArchive                command.Type = commandids.CampaignArchive
	commandTypeCampaignAIBind                 command.Type = commandids.CampaignAIBind
	commandTypeCampaignAIUnbind               command.Type = commandids.CampaignAIUnbind
	commandTypeCampaignAIAuthRotate           command.Type = commandids.CampaignAIAuthRotate
	commandTypeCampaignCreate                 command.Type = commandids.CampaignCreate
	commandTypeCampaignCreateWithParticipants command.Type = commandids.CampaignCreateWithParticipants
	commandTypeCampaignEnd                    command.Type = commandids.CampaignEnd
	commandTypeCampaignFork                   command.Type = commandids.CampaignFork
	commandTypeCampaignRestore                command.Type = commandids.CampaignRestore
	commandTypeCampaignUpdate                 command.Type = commandids.CampaignUpdate
	commandTypeCharacterCreate                command.Type = commandids.CharacterCreate
	commandTypeCharacterDelete                command.Type = commandids.CharacterDelete
	commandTypeCharacterUpdate                command.Type = commandids.CharacterUpdate
	commandTypeInviteClaim                    command.Type = commandids.InviteClaim
	commandTypeInviteCreate                   command.Type = commandids.InviteCreate
	commandTypeInviteDecline                  command.Type = commandids.InviteDecline
	commandTypeInviteRevoke                   command.Type = commandids.InviteRevoke
	commandTypeParticipantBind                command.Type = commandids.ParticipantBind
	commandTypeParticipantJoin                command.Type = commandids.ParticipantJoin
	commandTypeParticipantLeave               command.Type = commandids.ParticipantLeave
	commandTypeParticipantUpdate              command.Type = commandids.ParticipantUpdate
	commandTypeSessionEnd                     command.Type = commandids.SessionEnd
	commandTypeSessionGateAbandon             command.Type = commandids.SessionGateAbandon
	commandTypeSessionGateOpen                command.Type = commandids.SessionGateOpen
	commandTypeSessionGateRespond             command.Type = commandids.SessionGateRespond
	commandTypeSessionGateResolve             command.Type = commandids.SessionGateResolve
	commandTypeSessionSpotlightClear          command.Type = commandids.SessionSpotlightClear
	commandTypeSessionSpotlightSet            command.Type = commandids.SessionSpotlightSet
	commandTypeSessionStart                   command.Type = commandids.SessionStart
	commandTypeStoryNoteAdd                   command.Type = commandids.StoryNoteAdd
	commandTypeSceneCreate                    command.Type = commandids.SceneCreate
	commandTypeSceneUpdate                    command.Type = commandids.SceneUpdate
	commandTypeSceneEnd                       command.Type = commandids.SceneEnd
	commandTypeSceneCharacterAdd              command.Type = commandids.SceneCharacterAdd
	commandTypeSceneCharacterRemove           command.Type = commandids.SceneCharacterRemove
	commandTypeSceneCharacterTransfer         command.Type = commandids.SceneCharacterTransfer
	commandTypeSceneTransition                command.Type = commandids.SceneTransition
	commandTypeSceneGateOpen                  command.Type = commandids.SceneGateOpen
	commandTypeSceneGateResolve               command.Type = commandids.SceneGateResolve
	commandTypeSceneGateAbandon               command.Type = commandids.SceneGateAbandon
	commandTypeSceneSpotlightSet              command.Type = commandids.SceneSpotlightSet
	commandTypeSceneSpotlightClear            command.Type = commandids.SceneSpotlightClear

	commandTypeDaggerheartCharacterStatePatch     command.Type = commandids.DaggerheartCharacterStatePatch
	commandTypeDaggerheartCharacterProfileReplace command.Type = commandids.DaggerheartCharacterProfileReplace
	commandTypeDaggerheartCharacterProfileDelete  command.Type = commandids.DaggerheartCharacterProfileDelete
	commandTypeDaggerheartConditionChange         command.Type = commandids.DaggerheartConditionChange
	commandTypeDaggerheartGMFearSet               command.Type = commandids.DaggerheartGMFearSet
)

const (
	eventTypeActionOutcomeApplied             event.Type = "action.outcome_applied"
	eventTypeActionOutcomeRejected            event.Type = "action.outcome_rejected"
	eventTypeActionRollResolved               event.Type = "action.roll_resolved"
	eventTypeCampaignCreated                  event.Type = "campaign.created"
	eventTypeCampaignAIBound                  event.Type = "campaign.ai_bound"
	eventTypeCampaignAIUnbound                event.Type = "campaign.ai_unbound"
	eventTypeCampaignAIAuthRotated            event.Type = "campaign.ai_auth_rotated"
	eventTypeCampaignForked                   event.Type = "campaign.forked"
	eventTypeCharacterCreated                 event.Type = "character.created"
	eventTypeCharacterDeleted                 event.Type = "character.deleted"
	eventTypeCharacterUpdated                 event.Type = "character.updated"
	eventTypeInviteClaimed                    event.Type = "invite.claimed"
	eventTypeInviteDeclined                   event.Type = "invite.declined"
	eventTypeParticipantJoined                event.Type = "participant.joined"
	eventTypeParticipantLeft                  event.Type = "participant.left"
	eventTypeParticipantUpdated               event.Type = "participant.updated"
	eventTypeStoryNoteAdded                   event.Type = "story.note_added"
	eventTypeDaggerheartCharacterStatePatched event.Type = "sys.daggerheart.character_state_patched"
)

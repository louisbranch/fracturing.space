package handler

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// Command type aliases.
const (
	CommandTypeActionOutcomeApply             command.Type = commandids.ActionOutcomeApply
	CommandTypeActionOutcomeReject            command.Type = commandids.ActionOutcomeReject
	CommandTypeActionRollResolve              command.Type = commandids.ActionRollResolve
	CommandTypeCampaignArchive                command.Type = commandids.CampaignArchive
	CommandTypeCampaignAIBind                 command.Type = commandids.CampaignAIBind
	CommandTypeCampaignAIUnbind               command.Type = commandids.CampaignAIUnbind
	CommandTypeCampaignAIAuthRotate           command.Type = commandids.CampaignAIAuthRotate
	CommandTypeCampaignCreate                 command.Type = commandids.CampaignCreate
	CommandTypeCampaignCreateWithParticipants command.Type = commandids.CampaignCreateWithParticipants
	CommandTypeCampaignEnd                    command.Type = commandids.CampaignEnd
	CommandTypeCampaignFork                   command.Type = commandids.CampaignFork
	CommandTypeCampaignRestore                command.Type = commandids.CampaignRestore
	CommandTypeCampaignUpdate                 command.Type = commandids.CampaignUpdate
	CommandTypeCharacterCreate                command.Type = commandids.CharacterCreate
	CommandTypeCharacterDelete                command.Type = commandids.CharacterDelete
	CommandTypeCharacterUpdate                command.Type = commandids.CharacterUpdate
	CommandTypeInviteClaim                    command.Type = commandids.InviteClaim
	CommandTypeInviteCreate                   command.Type = commandids.InviteCreate
	CommandTypeInviteDecline                  command.Type = commandids.InviteDecline
	CommandTypeInviteRevoke                   command.Type = commandids.InviteRevoke
	CommandTypeParticipantBind                command.Type = commandids.ParticipantBind
	CommandTypeParticipantJoin                command.Type = commandids.ParticipantJoin
	CommandTypeParticipantLeave               command.Type = commandids.ParticipantLeave
	CommandTypeParticipantSeatReassign        command.Type = commandids.ParticipantSeatReassign
	CommandTypeParticipantUpdate              command.Type = commandids.ParticipantUpdate
	CommandTypeSessionEnd                     command.Type = commandids.SessionEnd
	CommandTypeSessionGateAbandon             command.Type = commandids.SessionGateAbandon
	CommandTypeSessionGateOpen                command.Type = commandids.SessionGateOpen
	CommandTypeSessionGateRespond             command.Type = commandids.SessionGateRespond
	CommandTypeSessionGateResolve             command.Type = commandids.SessionGateResolve
	CommandTypeSessionSpotlightClear          command.Type = commandids.SessionSpotlightClear
	CommandTypeSessionSpotlightSet            command.Type = commandids.SessionSpotlightSet
	CommandTypeSessionStart                   command.Type = commandids.SessionStart
	CommandTypeStoryNoteAdd                   command.Type = commandids.StoryNoteAdd
	CommandTypeSceneCreate                    command.Type = commandids.SceneCreate
	CommandTypeSceneUpdate                    command.Type = commandids.SceneUpdate
	CommandTypeSceneEnd                       command.Type = commandids.SceneEnd
	CommandTypeSceneCharacterAdd              command.Type = commandids.SceneCharacterAdd
	CommandTypeSceneCharacterRemove           command.Type = commandids.SceneCharacterRemove
	CommandTypeSceneCharacterTransfer         command.Type = commandids.SceneCharacterTransfer
	CommandTypeSceneTransition                command.Type = commandids.SceneTransition
	CommandTypeSceneGateOpen                  command.Type = commandids.SceneGateOpen
	CommandTypeSceneGateResolve               command.Type = commandids.SceneGateResolve
	CommandTypeSceneGateAbandon               command.Type = commandids.SceneGateAbandon
	CommandTypeSceneSpotlightSet              command.Type = commandids.SceneSpotlightSet
	CommandTypeSceneSpotlightClear            command.Type = commandids.SceneSpotlightClear

	CommandTypeDaggerheartCharacterStatePatch     command.Type = commandids.DaggerheartCharacterStatePatch
	CommandTypeDaggerheartCharacterProfileReplace command.Type = commandids.DaggerheartCharacterProfileReplace
	CommandTypeDaggerheartCharacterProfileDelete  command.Type = commandids.DaggerheartCharacterProfileDelete
	CommandTypeDaggerheartConditionChange         command.Type = commandids.DaggerheartConditionChange
	CommandTypeDaggerheartGMFearSet               command.Type = commandids.DaggerheartGMFearSet
)

// Event type aliases.
const (
	EventTypeActionOutcomeApplied             event.Type = "action.outcome_applied"
	EventTypeActionOutcomeRejected            event.Type = "action.outcome_rejected"
	EventTypeActionRollResolved               event.Type = "action.roll_resolved"
	EventTypeCampaignCreated                  event.Type = "campaign.created"
	EventTypeCampaignAIBound                  event.Type = "campaign.ai_bound"
	EventTypeCampaignAIUnbound                event.Type = "campaign.ai_unbound"
	EventTypeCampaignAIAuthRotated            event.Type = "campaign.ai_auth_rotated"
	EventTypeCampaignForked                   event.Type = "campaign.forked"
	EventTypeCharacterCreated                 event.Type = "character.created"
	EventTypeCharacterDeleted                 event.Type = "character.deleted"
	EventTypeCharacterUpdated                 event.Type = "character.updated"
	EventTypeInviteClaimed                    event.Type = "invite.claimed"
	EventTypeInviteDeclined                   event.Type = "invite.declined"
	EventTypeParticipantJoined                event.Type = "participant.joined"
	EventTypeParticipantLeft                  event.Type = "participant.left"
	EventTypeParticipantUpdated               event.Type = "participant.updated"
	EventTypeStoryNoteAdded                   event.Type = "story.note_added"
	EventTypeDaggerheartCharacterStatePatched event.Type = "sys.daggerheart.character_state_patched"
)

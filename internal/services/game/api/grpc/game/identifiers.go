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
	commandTypeCampaignCreate                 command.Type = commandids.CampaignCreate
	commandTypeCampaignEnd                    command.Type = commandids.CampaignEnd
	commandTypeCampaignFork                   command.Type = commandids.CampaignFork
	commandTypeCampaignRestore                command.Type = commandids.CampaignRestore
	commandTypeCampaignUpdate                 command.Type = commandids.CampaignUpdate
	commandTypeCharacterCreate                command.Type = commandids.CharacterCreate
	commandTypeCharacterDelete                command.Type = commandids.CharacterDelete
	commandTypeCharacterProfileUpdate         command.Type = commandids.CharacterProfileUpdate
	commandTypeCharacterUpdate                command.Type = commandids.CharacterUpdate
	commandTypeInviteClaim                    command.Type = commandids.InviteClaim
	commandTypeInviteCreate                   command.Type = commandids.InviteCreate
	commandTypeInviteRevoke                   command.Type = commandids.InviteRevoke
	commandTypeParticipantBind                command.Type = commandids.ParticipantBind
	commandTypeParticipantJoin                command.Type = commandids.ParticipantJoin
	commandTypeParticipantLeave               command.Type = commandids.ParticipantLeave
	commandTypeParticipantUpdate              command.Type = commandids.ParticipantUpdate
	commandTypeSessionEnd                     command.Type = commandids.SessionEnd
	commandTypeSessionGateAbandon             command.Type = commandids.SessionGateAbandon
	commandTypeSessionGateOpen                command.Type = commandids.SessionGateOpen
	commandTypeSessionGateResolve             command.Type = commandids.SessionGateResolve
	commandTypeSessionSpotlightClear          command.Type = commandids.SessionSpotlightClear
	commandTypeSessionSpotlightSet            command.Type = commandids.SessionSpotlightSet
	commandTypeSessionStart                   command.Type = commandids.SessionStart
	commandTypeStoryNoteAdd                   command.Type = commandids.StoryNoteAdd
	commandTypeDaggerheartCharacterStatePatch command.Type = commandids.DaggerheartCharacterStatePatch
	commandTypeDaggerheartConditionChange     command.Type = commandids.DaggerheartConditionChange
	commandTypeDaggerheartGMFearSet           command.Type = commandids.DaggerheartGMFearSet
)

const (
	eventTypeActionOutcomeApplied             event.Type = "action.outcome_applied"
	eventTypeActionOutcomeRejected            event.Type = "action.outcome_rejected"
	eventTypeActionRollResolved               event.Type = "action.roll_resolved"
	eventTypeCampaignCreated                  event.Type = "campaign.created"
	eventTypeCampaignForked                   event.Type = "campaign.forked"
	eventTypeCharacterCreated                 event.Type = "character.created"
	eventTypeCharacterDeleted                 event.Type = "character.deleted"
	eventTypeCharacterUpdated                 event.Type = "character.updated"
	eventTypeInviteClaimed                    event.Type = "invite.claimed"
	eventTypeParticipantJoined                event.Type = "participant.joined"
	eventTypeParticipantLeft                  event.Type = "participant.left"
	eventTypeParticipantUpdated               event.Type = "participant.updated"
	eventTypeStoryNoteAdded                   event.Type = "story.note_added"
	eventTypeDaggerheartCharacterStatePatched event.Type = "sys.daggerheart.character_state_patched"
)

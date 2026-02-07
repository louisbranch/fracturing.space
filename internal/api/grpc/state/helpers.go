package state

import (
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/state/v1"
	"github.com/louisbranch/fracturing.space/internal/state/campaign"
	"github.com/louisbranch/fracturing.space/internal/state/character"
	"github.com/louisbranch/fracturing.space/internal/state/participant"
	"github.com/louisbranch/fracturing.space/internal/state/session"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Campaign proto conversion helpers

func campaignToProto(c campaign.Campaign) *statev1.Campaign {
	return &statev1.Campaign{
		Id:               c.ID,
		Name:             c.Name,
		System:           gameSystemToProto(c.System),
		Status:           campaignStatusToProto(c.Status),
		GmMode:           gmModeToProto(c.GmMode),
		ParticipantCount: int32(c.ParticipantCount),
		CharacterCount:   int32(c.CharacterCount),
		ThemePrompt:      c.ThemePrompt,
		CreatedAt:        timestamppb.New(c.CreatedAt),
		LastActivityAt:   timestamppb.New(c.LastActivityAt),
		UpdatedAt:        timestamppb.New(c.UpdatedAt),
		CompletedAt:      timestampOrNil(c.CompletedAt),
		ArchivedAt:       timestampOrNil(c.ArchivedAt),
	}
}

func campaignStatusToProto(status campaign.CampaignStatus) statev1.CampaignStatus {
	switch status {
	case campaign.CampaignStatusDraft:
		return statev1.CampaignStatus_DRAFT
	case campaign.CampaignStatusActive:
		return statev1.CampaignStatus_ACTIVE
	case campaign.CampaignStatusCompleted:
		return statev1.CampaignStatus_COMPLETED
	case campaign.CampaignStatusArchived:
		return statev1.CampaignStatus_ARCHIVED
	default:
		return statev1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED
	}
}

func gmModeFromProto(mode statev1.GmMode) campaign.GmMode {
	switch mode {
	case statev1.GmMode_HUMAN:
		return campaign.GmModeHuman
	case statev1.GmMode_AI:
		return campaign.GmModeAI
	case statev1.GmMode_HYBRID:
		return campaign.GmModeHybrid
	default:
		return campaign.GmModeUnspecified
	}
}

func gmModeToProto(mode campaign.GmMode) statev1.GmMode {
	switch mode {
	case campaign.GmModeHuman:
		return statev1.GmMode_HUMAN
	case campaign.GmModeAI:
		return statev1.GmMode_AI
	case campaign.GmModeHybrid:
		return statev1.GmMode_HYBRID
	default:
		return statev1.GmMode_GM_MODE_UNSPECIFIED
	}
}

func gameSystemToProto(system commonv1.GameSystem) commonv1.GameSystem {
	return system
}

func gameSystemFromProto(system commonv1.GameSystem) commonv1.GameSystem {
	return system
}

// Participant proto conversion helpers

func participantToProto(p participant.Participant) *statev1.Participant {
	return &statev1.Participant{
		Id:          p.ID,
		CampaignId:  p.CampaignID,
		DisplayName: p.DisplayName,
		Role:        participantRoleToProto(p.Role),
		Controller:  controllerToProto(p.Controller),
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
}

func participantRoleFromProto(role statev1.ParticipantRole) participant.ParticipantRole {
	switch role {
	case statev1.ParticipantRole_GM:
		return participant.ParticipantRoleGM
	case statev1.ParticipantRole_PLAYER:
		return participant.ParticipantRolePlayer
	default:
		return participant.ParticipantRoleUnspecified
	}
}

func participantRoleToProto(role participant.ParticipantRole) statev1.ParticipantRole {
	switch role {
	case participant.ParticipantRoleGM:
		return statev1.ParticipantRole_GM
	case participant.ParticipantRolePlayer:
		return statev1.ParticipantRole_PLAYER
	default:
		return statev1.ParticipantRole_ROLE_UNSPECIFIED
	}
}

func controllerFromProto(controller statev1.Controller) participant.Controller {
	switch controller {
	case statev1.Controller_CONTROLLER_HUMAN:
		return participant.ControllerHuman
	case statev1.Controller_CONTROLLER_AI:
		return participant.ControllerAI
	default:
		return participant.ControllerUnspecified
	}
}

func controllerToProto(controller participant.Controller) statev1.Controller {
	switch controller {
	case participant.ControllerHuman:
		return statev1.Controller_CONTROLLER_HUMAN
	case participant.ControllerAI:
		return statev1.Controller_CONTROLLER_AI
	default:
		return statev1.Controller_CONTROLLER_UNSPECIFIED
	}
}

// Character proto conversion helpers

func characterToProto(ch character.Character) *statev1.Character {
	return &statev1.Character{
		Id:         ch.ID,
		CampaignId: ch.CampaignID,
		Name:       ch.Name,
		Kind:       characterKindToProto(ch.Kind),
		Notes:      ch.Notes,
		CreatedAt:  timestamppb.New(ch.CreatedAt),
		UpdatedAt:  timestamppb.New(ch.UpdatedAt),
	}
}

func characterKindFromProto(kind statev1.CharacterKind) character.CharacterKind {
	switch kind {
	case statev1.CharacterKind_PC:
		return character.CharacterKindPC
	case statev1.CharacterKind_NPC:
		return character.CharacterKindNPC
	default:
		return character.CharacterKindUnspecified
	}
}

func characterKindToProto(kind character.CharacterKind) statev1.CharacterKind {
	switch kind {
	case character.CharacterKindPC:
		return statev1.CharacterKind_PC
	case character.CharacterKindNPC:
		return statev1.CharacterKind_NPC
	default:
		return statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED
	}
}

func characterControllerFromProto(pb *statev1.CharacterController) (character.CharacterController, error) {
	if pb == nil {
		return character.CharacterController{}, character.ErrInvalidCharacterController
	}

	switch c := pb.GetController().(type) {
	case *statev1.CharacterController_Gm:
		if c.Gm == nil {
			return character.CharacterController{}, character.ErrInvalidCharacterController
		}
		return character.NewGmController(), nil
	case *statev1.CharacterController_Participant:
		if c.Participant == nil {
			return character.CharacterController{}, character.ErrInvalidCharacterController
		}
		return character.NewParticipantController(c.Participant.GetParticipantId())
	default:
		return character.CharacterController{}, character.ErrInvalidCharacterController
	}
}

func characterControllerToProto(ctrl character.CharacterController) *statev1.CharacterController {
	if ctrl.IsGM {
		return &statev1.CharacterController{
			Controller: &statev1.CharacterController_Gm{
				Gm: &statev1.GmController{},
			},
		}
	}
	return &statev1.CharacterController{
		Controller: &statev1.CharacterController_Participant{
			Participant: &statev1.ParticipantController{
				ParticipantId: ctrl.ParticipantID,
			},
		},
	}
}

// Session proto conversion helpers

func sessionToProto(sess session.Session) *statev1.Session {
	pb := &statev1.Session{
		Id:         sess.ID,
		CampaignId: sess.CampaignID,
		Name:       sess.Name,
		Status:     sessionStatusToProto(sess.Status),
		StartedAt:  timestamppb.New(sess.StartedAt),
		UpdatedAt:  timestamppb.New(sess.UpdatedAt),
	}
	if sess.EndedAt != nil {
		pb.EndedAt = timestamppb.New(*sess.EndedAt)
	}
	return pb
}

func sessionStatusToProto(status session.SessionStatus) statev1.SessionStatus {
	switch status {
	case session.SessionStatusActive:
		return statev1.SessionStatus_SESSION_ACTIVE
	case session.SessionStatusEnded:
		return statev1.SessionStatus_SESSION_ENDED
	default:
		return statev1.SessionStatus_SESSION_STATUS_UNSPECIFIED
	}
}

// Timestamp helpers

func timestampOrNil(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(value.UTC())
}

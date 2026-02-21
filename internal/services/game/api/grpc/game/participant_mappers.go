package game

import (
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Participant proto conversion helpers.
func participantToProto(p storage.ParticipantRecord) *campaignv1.Participant {
	return &campaignv1.Participant{
		Id:             p.ID,
		CampaignId:     p.CampaignID,
		UserId:         p.UserID,
		Name:           p.Name,
		Role:           participantRoleToProto(p.Role),
		CampaignAccess: campaignAccessToProto(p.CampaignAccess),
		Controller:     controllerToProto(p.Controller),
		AvatarSetId:    p.AvatarSetID,
		AvatarAssetId:  p.AvatarAssetID,
		CreatedAt:      timestamppb.New(p.CreatedAt),
		UpdatedAt:      timestamppb.New(p.UpdatedAt),
	}
}

func participantRoleFromProto(role campaignv1.ParticipantRole) participant.Role {
	switch role {
	case campaignv1.ParticipantRole_GM:
		return participant.RoleGM
	case campaignv1.ParticipantRole_PLAYER:
		return participant.RolePlayer
	default:
		return participant.RoleUnspecified
	}
}

func participantRoleToProto(role participant.Role) campaignv1.ParticipantRole {
	switch role {
	case participant.RoleGM:
		return campaignv1.ParticipantRole_GM
	case participant.RolePlayer:
		return campaignv1.ParticipantRole_PLAYER
	default:
		return campaignv1.ParticipantRole_ROLE_UNSPECIFIED
	}
}

func controllerFromProto(controller campaignv1.Controller) participant.Controller {
	switch controller {
	case campaignv1.Controller_CONTROLLER_HUMAN:
		return participant.ControllerHuman
	case campaignv1.Controller_CONTROLLER_AI:
		return participant.ControllerAI
	default:
		return participant.ControllerUnspecified
	}
}

func controllerToProto(controller participant.Controller) campaignv1.Controller {
	switch controller {
	case participant.ControllerHuman:
		return campaignv1.Controller_CONTROLLER_HUMAN
	case participant.ControllerAI:
		return campaignv1.Controller_CONTROLLER_AI
	default:
		return campaignv1.Controller_CONTROLLER_UNSPECIFIED
	}
}

func campaignAccessFromProto(access campaignv1.CampaignAccess) participant.CampaignAccess {
	switch access {
	case campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER:
		return participant.CampaignAccessMember
	case campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER:
		return participant.CampaignAccessManager
	case campaignv1.CampaignAccess_CAMPAIGN_ACCESS_OWNER:
		return participant.CampaignAccessOwner
	default:
		return participant.CampaignAccessUnspecified
	}
}

func campaignAccessToProto(access participant.CampaignAccess) campaignv1.CampaignAccess {
	switch access {
	case participant.CampaignAccessMember:
		return campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER
	case participant.CampaignAccessManager:
		return campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER
	case participant.CampaignAccessOwner:
		return campaignv1.CampaignAccess_CAMPAIGN_ACCESS_OWNER
	default:
		return campaignv1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED
	}
}

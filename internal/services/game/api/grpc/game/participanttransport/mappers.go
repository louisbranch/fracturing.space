package participanttransport

import (
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ParticipantToProto converts a participant projection record into its protobuf
// read model.
func ParticipantToProto(record storage.ParticipantRecord) *campaignv1.Participant {
	return &campaignv1.Participant{
		Id:             record.ID,
		CampaignId:     record.CampaignID,
		UserId:         record.UserID,
		Name:           record.Name,
		Role:           RoleToProto(record.Role),
		CampaignAccess: CampaignAccessToProto(record.CampaignAccess),
		Controller:     ControllerToProto(record.Controller),
		AvatarSetId:    record.AvatarSetID,
		AvatarAssetId:  record.AvatarAssetID,
		Pronouns:       sharedpronouns.ToProto(record.Pronouns),
		CreatedAt:      timestamppb.New(record.CreatedAt),
		UpdatedAt:      timestamppb.New(record.UpdatedAt),
	}
}

// RoleFromProto converts a protobuf participant role to the domain value.
func RoleFromProto(role campaignv1.ParticipantRole) participant.Role {
	switch role {
	case campaignv1.ParticipantRole_GM:
		return participant.RoleGM
	case campaignv1.ParticipantRole_PLAYER:
		return participant.RolePlayer
	default:
		return participant.RoleUnspecified
	}
}

// RoleToProto converts a domain participant role to the protobuf enum.
func RoleToProto(role participant.Role) campaignv1.ParticipantRole {
	switch role {
	case participant.RoleGM:
		return campaignv1.ParticipantRole_GM
	case participant.RolePlayer:
		return campaignv1.ParticipantRole_PLAYER
	default:
		return campaignv1.ParticipantRole_ROLE_UNSPECIFIED
	}
}

// ControllerFromProto converts a protobuf participant controller to the domain
// value.
func ControllerFromProto(controller campaignv1.Controller) participant.Controller {
	switch controller {
	case campaignv1.Controller_CONTROLLER_HUMAN:
		return participant.ControllerHuman
	case campaignv1.Controller_CONTROLLER_AI:
		return participant.ControllerAI
	default:
		return participant.ControllerUnspecified
	}
}

// ControllerToProto converts a domain participant controller to the protobuf
// enum.
func ControllerToProto(controller participant.Controller) campaignv1.Controller {
	switch controller {
	case participant.ControllerHuman:
		return campaignv1.Controller_CONTROLLER_HUMAN
	case participant.ControllerAI:
		return campaignv1.Controller_CONTROLLER_AI
	default:
		return campaignv1.Controller_CONTROLLER_UNSPECIFIED
	}
}

// CampaignAccessFromProto converts a protobuf campaign-access enum to the
// domain value.
func CampaignAccessFromProto(access campaignv1.CampaignAccess) participant.CampaignAccess {
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

// CampaignAccessToProto converts a domain campaign-access enum to the protobuf
// value.
func CampaignAccessToProto(access participant.CampaignAccess) campaignv1.CampaignAccess {
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

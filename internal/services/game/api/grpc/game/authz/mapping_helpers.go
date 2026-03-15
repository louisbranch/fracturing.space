package authz

import (
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// CanResponse builds a CanResponse proto from the given decision components.
func CanResponse(allowed bool, reasonCode string, actor storage.ParticipantRecord) *campaignv1.CanResponse {
	return &campaignv1.CanResponse{
		Allowed:             allowed,
		ReasonCode:          strings.TrimSpace(reasonCode),
		ActorCampaignAccess: campaignAccessToProto(actor.CampaignAccess),
		ActorParticipantId:  strings.TrimSpace(actor.ID),
	}
}

// ActionFromProto maps proto action values to domain authz actions.
func ActionFromProto(action campaignv1.AuthorizationAction) (domainauthz.Action, bool) {
	switch action {
	case campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_READ:
		return domainauthz.ActionRead, true
	case campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE:
		return domainauthz.ActionManage, true
	case campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE:
		return domainauthz.ActionMutate, true
	case campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_TRANSFER_OWNERSHIP:
		return domainauthz.ActionTransferOwnership, true
	default:
		return domainauthz.ActionUnspecified, false
	}
}

// campaignAccessFromProto converts a protobuf campaign-access enum to the
// domain value. Local copy avoids importing entity-scoped transport packages.
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

// campaignAccessToProto converts a domain campaign-access enum to the protobuf
// value. Local copy avoids importing entity-scoped transport packages.
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

// ResourceFromProto maps proto resource values to domain authz resources.
func ResourceFromProto(resource campaignv1.AuthorizationResource) (domainauthz.Resource, bool) {
	switch resource {
	case campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CAMPAIGN:
		return domainauthz.ResourceCampaign, true
	case campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT:
		return domainauthz.ResourceParticipant, true
	case campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_INVITE:
		return domainauthz.ResourceInvite, true
	case campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_SESSION:
		return domainauthz.ResourceSession, true
	case campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER:
		return domainauthz.ResourceCharacter, true
	default:
		return domainauthz.ResourceUnspecified, false
	}
}

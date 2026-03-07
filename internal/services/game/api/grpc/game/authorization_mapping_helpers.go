package game

import (
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func canResponse(allowed bool, reasonCode string, actor storage.ParticipantRecord) *campaignv1.CanResponse {
	return &campaignv1.CanResponse{
		Allowed:             allowed,
		ReasonCode:          strings.TrimSpace(reasonCode),
		ActorCampaignAccess: campaignAccessToProto(actor.CampaignAccess),
		ActorParticipantId:  strings.TrimSpace(actor.ID),
	}
}

// authorizationActionFromProto maps proto action values to domain authz actions.
func authorizationActionFromProto(action campaignv1.AuthorizationAction) (domainauthz.Action, bool) {
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

// authorizationResourceFromProto maps proto resource values to domain authz resources.
func authorizationResourceFromProto(resource campaignv1.AuthorizationResource) (domainauthz.Resource, bool) {
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

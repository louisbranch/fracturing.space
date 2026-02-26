package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthorizationService implements the game.v1.AuthorizationService gRPC API.
type AuthorizationService struct {
	campaignv1.UnimplementedAuthorizationServiceServer
	stores Stores
}

// NewAuthorizationService creates an AuthorizationService with default dependencies.
func NewAuthorizationService(stores Stores) *AuthorizationService {
	return &AuthorizationService{stores: stores}
}

// Can evaluates whether the request actor can perform action/resource in campaign.
func (s *AuthorizationService) Can(ctx context.Context, in *campaignv1.CanRequest) (*campaignv1.CanResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "authorization request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	action, ok := authorizationActionFromProto(in.GetAction())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "authorization action is required")
	}
	resource, ok := authorizationResourceFromProto(in.GetResource())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "authorization resource is required")
	}
	capability, ok := domainauthz.CapabilityFromActionResource(action, resource)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "unsupported authorization action/resource combination")
	}

	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	actor, reasonCode, err := authorizePolicyActor(ctx, s.stores, capability, campaignRecord)
	if err != nil {
		emitAuthzDecisionTelemetry(ctx, s.stores.Audit, campaignID, capability, authzDecisionDeny, reasonCode, actor, err, nil)
		if status.Code(err) == codes.PermissionDenied {
			return canResponse(false, reasonCode, actor), nil
		}
		return nil, err
	}

	extraAttributes := map[string]any{}
	if capability == domainauthz.CapabilityMutateCharacters {
		ownerParticipantID, evaluateOwnership, resolveErr := resolveCanCharacterOwnerParticipantID(ctx, s.stores, campaignID, in.GetTarget())
		if resolveErr != nil {
			emitAuthzDecisionTelemetry(ctx, s.stores.Audit, campaignID, capability, authzDecisionDeny, domainauthz.ReasonErrorOwnerResolution, actor, resolveErr, nil)
			return nil, resolveErr
		}
		if evaluateOwnership {
			characterID := strings.TrimSpace(in.GetTarget().GetResourceId())
			decision := domainauthz.CanCharacterMutation(actor.CampaignAccess, actor.ID, ownerParticipantID)
			extraAttributes["character_id"] = characterID
			extraAttributes["owner_participant_id"] = ownerParticipantID
			if !decision.Allowed {
				authErr := status.Error(codes.PermissionDenied, "participant lacks permission")
				emitAuthzDecisionTelemetry(ctx, s.stores.Audit, campaignID, capability, authzDecisionDeny, decision.ReasonCode, actor, authErr, extraAttributes)
				return canResponse(false, decision.ReasonCode, actor), nil
			}
			reasonCode = decision.ReasonCode
		}
	}

	emitAuthzDecisionTelemetry(
		ctx,
		s.stores.Audit,
		campaignID,
		capability,
		authzDecisionForReason(reasonCode),
		reasonCode,
		actor,
		nil,
		extraAttributes,
	)
	return canResponse(true, reasonCode, actor), nil
}

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

// resolveCanCharacterOwnerParticipantID resolves owner context for optional
// character ownership authorization checks.
func resolveCanCharacterOwnerParticipantID(
	ctx context.Context,
	stores Stores,
	campaignID string,
	target *campaignv1.AuthorizationTarget,
) (string, bool, error) {
	if target == nil {
		return "", false, nil
	}
	ownerParticipantID := strings.TrimSpace(target.GetOwnerParticipantId())
	if ownerParticipantID != "" {
		return ownerParticipantID, true, nil
	}
	characterID := strings.TrimSpace(target.GetResourceId())
	if characterID == "" {
		return "", false, nil
	}
	ownerParticipantID, err := resolveCharacterMutationOwnerParticipantID(ctx, stores, campaignID, characterID)
	if err != nil {
		return "", false, err
	}
	return ownerParticipantID, true, nil
}

package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
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

	extraAttributes := authzExtraAttributesForReason(ctx, reasonCode)
	if reasonCode != authzReasonAllowAdminOverride {
		if capability == domainauthz.CapabilityMutateCharacters {
			ownerParticipantID, evaluateOwnership, resolveErr := resolveCanCharacterOwnerParticipantID(ctx, s.stores, campaignID, in.GetTarget())
			if resolveErr != nil {
				emitAuthzDecisionTelemetry(ctx, s.stores.Audit, campaignID, capability, authzDecisionDeny, domainauthz.ReasonErrorOwnerResolution, actor, resolveErr, nil)
				return nil, resolveErr
			}
			if evaluateOwnership {
				characterID := ""
				if target := in.GetTarget(); target != nil {
					characterID = strings.TrimSpace(target.GetResourceId())
				}
				characterAttributes := map[string]any{
					"character_id":         characterID,
					"owner_participant_id": ownerParticipantID,
				}
				extraAttributes = mergeAuthzAttributes(extraAttributes, characterAttributes)
				decision := domainauthz.CanCharacterMutation(actor.CampaignAccess, actor.ID, ownerParticipantID)
				if !decision.Allowed {
					authErr := status.Error(codes.PermissionDenied, "participant lacks permission")
					emitAuthzDecisionTelemetry(ctx, s.stores.Audit, campaignID, capability, authzDecisionDeny, decision.ReasonCode, actor, authErr, extraAttributes)
					return canResponse(false, decision.ReasonCode, actor), nil
				}
				reasonCode = decision.ReasonCode
			}
		}

		if capability == domainauthz.CapabilityManageParticipants {
			decision, participantAttributes, evaluated, evaluationErr := evaluateCanParticipantGovernanceTarget(
				ctx,
				s.stores,
				campaignID,
				actor,
				in.GetTarget(),
			)
			if evaluationErr != nil {
				emitAuthzDecisionTelemetry(ctx, s.stores.Audit, campaignID, capability, authzDecisionDeny, authzReasonErrorActorLoad, actor, evaluationErr, participantAttributes)
				return nil, evaluationErr
			}
			if evaluated {
				extraAttributes = mergeAuthzAttributes(extraAttributes, participantAttributes)
				if !decision.Allowed {
					authErr := status.Error(codes.PermissionDenied, "participant lacks permission")
					emitAuthzDecisionTelemetry(ctx, s.stores.Audit, campaignID, capability, authzDecisionDeny, decision.ReasonCode, actor, authErr, extraAttributes)
					return canResponse(false, decision.ReasonCode, actor), nil
				}
				reasonCode = decision.ReasonCode
			}
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

// BatchCan evaluates whether the request actor can perform each batch item
// action/resource in campaign.
func (s *AuthorizationService) BatchCan(ctx context.Context, in *campaignv1.BatchCanRequest) (*campaignv1.BatchCanResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "batch authorization request is required")
	}
	checks := in.GetChecks()
	if len(checks) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one batch authorization check is required")
	}

	results := make([]*campaignv1.BatchCanResult, 0, len(checks))
	for idx, check := range checks {
		if check == nil {
			return nil, status.Errorf(codes.InvalidArgument, "batch authorization check at index %d is required", idx)
		}

		resp, err := s.Can(ctx, &campaignv1.CanRequest{
			CampaignId: strings.TrimSpace(check.GetCampaignId()),
			Action:     check.GetAction(),
			Resource:   check.GetResource(),
			Target:     check.GetTarget(),
		})
		if err != nil {
			return nil, err
		}

		results = append(results, &campaignv1.BatchCanResult{
			CheckId:             strings.TrimSpace(check.GetCheckId()),
			Allowed:             resp.GetAllowed(),
			ReasonCode:          strings.TrimSpace(resp.GetReasonCode()),
			ActorCampaignAccess: resp.GetActorCampaignAccess(),
			ActorParticipantId:  strings.TrimSpace(resp.GetActorParticipantId()),
		})
	}

	return &campaignv1.BatchCanResponse{Results: results}, nil
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

func evaluateCanParticipantGovernanceTarget(
	ctx context.Context,
	stores Stores,
	campaignID string,
	actor storage.ParticipantRecord,
	target *campaignv1.AuthorizationTarget,
) (domainauthz.PolicyDecision, map[string]any, bool, error) {
	if target == nil {
		return domainauthz.PolicyDecision{}, nil, false, nil
	}

	targetParticipantID := strings.TrimSpace(target.GetTargetParticipantId())
	if targetParticipantID == "" {
		targetParticipantID = strings.TrimSpace(target.GetResourceId())
	}
	targetAccess := campaignAccessFromProto(target.GetTargetCampaignAccess())
	requestedAccess := campaignAccessFromProto(target.GetRequestedCampaignAccess())
	participantOperation := target.GetParticipantOperation()

	if targetParticipantID != "" && targetAccess == participant.CampaignAccessUnspecified && stores.Participant != nil {
		targetRecord, err := stores.Participant.GetParticipant(ctx, campaignID, targetParticipantID)
		if err != nil {
			if err != storage.ErrNotFound {
				return domainauthz.PolicyDecision{}, nil, false, status.Errorf(codes.Internal, "load target participant: %v", err)
			}
		} else {
			targetAccess = targetRecord.CampaignAccess
		}
	}

	extraAttributes := map[string]any{}
	if targetParticipantID != "" {
		extraAttributes["target_participant_id"] = targetParticipantID
	}
	if targetAccess != participant.CampaignAccessUnspecified {
		extraAttributes["target_campaign_access"] = strings.TrimSpace(string(targetAccess))
	}
	if requestedAccess != participant.CampaignAccessUnspecified {
		extraAttributes["requested_campaign_access"] = strings.TrimSpace(string(requestedAccess))
	}
	if label := participantGovernanceOperationLabel(participantOperation); label != "" {
		extraAttributes["participant_operation"] = label
	}

	decision := domainauthz.CanParticipantMutation(actor.CampaignAccess, targetAccess)
	if !decision.Allowed {
		return decision, extraAttributes, true, nil
	}

	if participantOperation == campaignv1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_REMOVE {
		if targetParticipantID == "" {
			if len(extraAttributes) == 0 {
				return domainauthz.PolicyDecision{}, nil, false, nil
			}
			return decision, extraAttributes, true, nil
		}
		ownerCount, err := countCampaignOwners(ctx, stores.Participant, campaignID)
		if err != nil {
			return domainauthz.PolicyDecision{}, extraAttributes, false, err
		}
		targetOwnsActiveCharacters, err := participantOwnsActiveCharacters(ctx, stores.Character, campaignID, targetParticipantID)
		if err != nil {
			return domainauthz.PolicyDecision{}, extraAttributes, false, err
		}
		extraAttributes["target_owns_active_characters"] = targetOwnsActiveCharacters
		decision = domainauthz.CanParticipantRemovalWithOwnedResources(
			actor.CampaignAccess,
			targetAccess,
			ownerCount,
			targetOwnsActiveCharacters,
		)
		return decision, extraAttributes, true, nil
	}
	if participantOperation == campaignv1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_ACCESS_CHANGE && requestedAccess == participant.CampaignAccessUnspecified {
		return domainauthz.PolicyDecision{}, extraAttributes, false, status.Error(codes.InvalidArgument, "requested campaign access is required for access-change operation")
	}

	if requestedAccess == participant.CampaignAccessUnspecified {
		if len(extraAttributes) == 0 {
			return domainauthz.PolicyDecision{}, nil, false, nil
		}
		return decision, extraAttributes, true, nil
	}

	ownerCount, err := countCampaignOwners(ctx, stores.Participant, campaignID)
	if err != nil {
		return domainauthz.PolicyDecision{}, extraAttributes, false, err
	}

	decision = domainauthz.CanParticipantAccessChange(
		actor.CampaignAccess,
		targetAccess,
		requestedAccess,
		ownerCount,
	)
	return decision, extraAttributes, true, nil
}

func participantGovernanceOperationLabel(operation campaignv1.ParticipantGovernanceOperation) string {
	switch operation {
	case campaignv1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_MUTATE:
		return "mutate"
	case campaignv1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_ACCESS_CHANGE:
		return "access_change"
	case campaignv1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_REMOVE:
		return "remove"
	default:
		return ""
	}
}

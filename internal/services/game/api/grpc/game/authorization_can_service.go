package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

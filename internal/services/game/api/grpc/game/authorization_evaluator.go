package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authorizationEvaluatorStores struct {
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Character   storage.CharacterStore
	Audit       storage.AuditEventStore
}

type authorizationEvaluator struct {
	stores authorizationEvaluatorStores
}

func newAuthorizationEvaluator(stores Stores) authorizationEvaluator {
	return authorizationEvaluator{
		stores: authorizationEvaluatorStores{
			Campaign:    stores.Campaign,
			Participant: stores.Participant,
			Character:   stores.Character,
			Audit:       stores.Audit,
		},
	}
}

func (e authorizationEvaluator) Evaluate(ctx context.Context, in *campaignv1.CanRequest) (*campaignv1.CanResponse, error) {
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

	campaignRecord, err := e.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	actor, reasonCode, err := authorizePolicyActorWithParticipantStore(ctx, e.stores.Participant, capability, campaignRecord)
	if err != nil {
		emitAuthzDecisionTelemetry(ctx, authzDecisionEvent{
			Store:      e.stores.Audit,
			CampaignID: campaignID,
			Capability: capability,
			Decision:   authzDecisionDeny,
			ReasonCode: reasonCode,
			Actor:      actor,
			Err:        err,
		})
		if status.Code(err) == codes.PermissionDenied {
			return canResponse(false, reasonCode, actor), nil
		}
		return nil, err
	}

	extraAttributes := authzExtraAttributesForReason(ctx, reasonCode)
	if reasonCode != authzReasonAllowAdminOverride {
		if capability == domainauthz.CapabilityMutateCharacters {
			ownerParticipantID, evaluateOwnership, resolveErr := resolveCanCharacterOwnerParticipantIDWithCharacterStore(ctx, e.stores.Character, campaignID, in.GetTarget())
			if resolveErr != nil {
				emitAuthzDecisionTelemetry(ctx, authzDecisionEvent{
					Store:      e.stores.Audit,
					CampaignID: campaignID,
					Capability: capability,
					Decision:   authzDecisionDeny,
					ReasonCode: domainauthz.ReasonErrorOwnerResolution,
					Actor:      actor,
					Err:        resolveErr,
				})
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
				decision := domainauthz.CanCharacterMutation(actor.CampaignAccess, ids.ParticipantID(actor.ID), ids.ParticipantID(ownerParticipantID))
				if !decision.Allowed {
					authErr := status.Error(codes.PermissionDenied, "participant lacks permission")
					emitAuthzDecisionTelemetry(ctx, authzDecisionEvent{
						Store:           e.stores.Audit,
						CampaignID:      campaignID,
						Capability:      capability,
						Decision:        authzDecisionDeny,
						ReasonCode:      decision.ReasonCode,
						Actor:           actor,
						Err:             authErr,
						ExtraAttributes: extraAttributes,
					})
					return canResponse(false, decision.ReasonCode, actor), nil
				}
				reasonCode = decision.ReasonCode
			}
		}

		if capability == domainauthz.CapabilityManageParticipants {
			decision, participantAttributes, evaluated, evaluationErr := evaluateCanParticipantGovernanceTargetWithStores(
				ctx,
				e.stores.Participant,
				e.stores.Character,
				campaignID,
				actor,
				in.GetTarget(),
			)
			if evaluationErr != nil {
				emitAuthzDecisionTelemetry(ctx, authzDecisionEvent{
					Store:           e.stores.Audit,
					CampaignID:      campaignID,
					Capability:      capability,
					Decision:        authzDecisionDeny,
					ReasonCode:      authzReasonErrorActorLoad,
					Actor:           actor,
					Err:             evaluationErr,
					ExtraAttributes: participantAttributes,
				})
				return nil, evaluationErr
			}
			if evaluated {
				extraAttributes = mergeAuthzAttributes(extraAttributes, participantAttributes)
				if !decision.Allowed {
					authErr := status.Error(codes.PermissionDenied, "participant lacks permission")
					emitAuthzDecisionTelemetry(ctx, authzDecisionEvent{
						Store:           e.stores.Audit,
						CampaignID:      campaignID,
						Capability:      capability,
						Decision:        authzDecisionDeny,
						ReasonCode:      decision.ReasonCode,
						Actor:           actor,
						Err:             authErr,
						ExtraAttributes: extraAttributes,
					})
					return canResponse(false, decision.ReasonCode, actor), nil
				}
				reasonCode = decision.ReasonCode
			}
		}
	}

	emitAuthzDecisionTelemetry(ctx, authzDecisionEvent{
		Store:           e.stores.Audit,
		CampaignID:      campaignID,
		Capability:      capability,
		Decision:        authzDecisionForReason(reasonCode),
		ReasonCode:      reasonCode,
		Actor:           actor,
		ExtraAttributes: extraAttributes,
	})
	return canResponse(true, reasonCode, actor), nil
}

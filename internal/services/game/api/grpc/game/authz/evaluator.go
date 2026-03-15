package authz

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

// EvaluatorStores holds the stores required by the authorization evaluator.
type EvaluatorStores struct {
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Character   storage.CharacterStore
	Audit       storage.AuditEventStore
}

// Evaluator implements authorization policy evaluation for the Can/BatchCan
// gRPC API surface.
type Evaluator struct {
	stores EvaluatorStores
}

// NewEvaluator constructs an Evaluator with the given store dependencies.
func NewEvaluator(stores EvaluatorStores) Evaluator {
	return Evaluator{stores: stores}
}

// Evaluate checks whether the calling actor can perform the requested
// action/resource combination in the target campaign.
func (e Evaluator) Evaluate(ctx context.Context, in *campaignv1.CanRequest) (*campaignv1.CanResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "authorization request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	action, ok := ActionFromProto(in.GetAction())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "authorization action is required")
	}
	resource, ok := ResourceFromProto(in.GetResource())
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

	actor, reasonCode, err := AuthorizePolicyActorWithParticipantStore(ctx, e.stores.Participant, capability, campaignRecord)
	if err != nil {
		EmitDecisionTelemetry(ctx, DecisionEvent{
			Store:      e.stores.Audit,
			CampaignID: campaignID,
			Capability: capability,
			Decision:   DecisionDeny,
			ReasonCode: reasonCode,
			Actor:      actor,
			Err:        err,
		})
		if status.Code(err) == codes.PermissionDenied {
			return CanResponse(false, reasonCode, actor), nil
		}
		return nil, err
	}

	extraAttributes := ExtraAttributesForReason(ctx, reasonCode)
	if reasonCode != ReasonAllowAdminOverride {
		if capability == domainauthz.CapabilityMutateCharacters() {
			ownerParticipantID, evaluateOwnership, resolveErr := ResolveCanCharacterOwnerParticipantIDWithCharacterStore(ctx, e.stores.Character, campaignID, in.GetTarget())
			if resolveErr != nil {
				EmitDecisionTelemetry(ctx, DecisionEvent{
					Store:      e.stores.Audit,
					CampaignID: campaignID,
					Capability: capability,
					Decision:   DecisionDeny,
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
				extraAttributes = MergeAttributes(extraAttributes, characterAttributes)
				decision := domainauthz.CanCharacterMutation(actor.CampaignAccess, ids.ParticipantID(actor.ID), ids.ParticipantID(ownerParticipantID))
				if !decision.Allowed {
					authErr := status.Error(codes.PermissionDenied, "participant lacks permission")
					EmitDecisionTelemetry(ctx, DecisionEvent{
						Store:           e.stores.Audit,
						CampaignID:      campaignID,
						Capability:      capability,
						Decision:        DecisionDeny,
						ReasonCode:      decision.ReasonCode,
						Actor:           actor,
						Err:             authErr,
						ExtraAttributes: extraAttributes,
					})
					return CanResponse(false, decision.ReasonCode, actor), nil
				}
				reasonCode = decision.ReasonCode
			}
		}

		if capability == domainauthz.CapabilityManageParticipants() {
			decision, participantAttributes, evaluated, evaluationErr := EvaluateCanParticipantGovernanceTargetWithStores(
				ctx,
				e.stores.Participant,
				e.stores.Character,
				campaignID,
				actor,
				in.GetTarget(),
			)
			if evaluationErr != nil {
				EmitDecisionTelemetry(ctx, DecisionEvent{
					Store:           e.stores.Audit,
					CampaignID:      campaignID,
					Capability:      capability,
					Decision:        DecisionDeny,
					ReasonCode:      ReasonErrorActorLoad,
					Actor:           actor,
					Err:             evaluationErr,
					ExtraAttributes: participantAttributes,
				})
				return nil, evaluationErr
			}
			if evaluated {
				extraAttributes = MergeAttributes(extraAttributes, participantAttributes)
				if !decision.Allowed {
					authErr := status.Error(codes.PermissionDenied, "participant lacks permission")
					EmitDecisionTelemetry(ctx, DecisionEvent{
						Store:           e.stores.Audit,
						CampaignID:      campaignID,
						Capability:      capability,
						Decision:        DecisionDeny,
						ReasonCode:      decision.ReasonCode,
						Actor:           actor,
						Err:             authErr,
						ExtraAttributes: extraAttributes,
					})
					return CanResponse(false, decision.ReasonCode, actor), nil
				}
				reasonCode = decision.ReasonCode
			}
		}
	}

	EmitDecisionTelemetry(ctx, DecisionEvent{
		Store:           e.stores.Audit,
		CampaignID:      campaignID,
		Capability:      capability,
		Decision:        DecisionForReason(reasonCode),
		ReasonCode:      reasonCode,
		Actor:           actor,
		ExtraAttributes: extraAttributes,
	})
	return CanResponse(true, reasonCode, actor), nil
}

package authz

import (
	"context"
	"strings"

	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PolicyDeps holds the focused store subset needed for authorization policy
// enforcement. Entity applications embed this instead of a root transport bag.
type PolicyDeps struct {
	Participant storage.ParticipantStore
	Character   storage.CharacterStore
	Audit       storage.AuditEventStore
}

// RequirePolicy ensures the participant has access for the requested action.
func RequirePolicy(ctx context.Context, deps PolicyDeps, capability domainauthz.Capability, campaignRecord storage.CampaignRecord) error {
	_, err := RequirePolicyActor(ctx, deps, capability, campaignRecord)
	return err
}

// RequireReadPolicy ensures the actor can access campaign-scoped reads.
func RequireReadPolicy(ctx context.Context, deps PolicyDeps, campaignRecord storage.CampaignRecord) error {
	return RequirePolicy(ctx, deps, domainauthz.CapabilityReadCampaign(), campaignRecord)
}

// RequirePolicyActor ensures access and returns the resolved participant actor.
func RequirePolicyActor(
	ctx context.Context,
	deps PolicyDeps,
	capability domainauthz.Capability,
	campaignRecord storage.CampaignRecord,
) (storage.ParticipantRecord, error) {
	actor, reasonCode, err := AuthorizePolicyActorWithParticipantStore(ctx, deps.Participant, capability, campaignRecord)
	if err != nil {
		EmitDecisionTelemetry(ctx, DecisionEvent{
			Store:      deps.Audit,
			CampaignID: campaignRecord.ID,
			Capability: capability,
			Decision:   DecisionDeny,
			ReasonCode: reasonCode,
			Actor:      actor,
			Err:        err,
		})
		return storage.ParticipantRecord{}, err
	}
	EmitDecisionTelemetry(ctx, DecisionEvent{
		Store:           deps.Audit,
		CampaignID:      campaignRecord.ID,
		Capability:      capability,
		Decision:        DecisionForReason(reasonCode),
		ReasonCode:      reasonCode,
		Actor:           actor,
		ExtraAttributes: ExtraAttributesForReason(ctx, reasonCode),
	})
	return actor, nil
}

// characterMutationResult holds the outcome of evaluateCharacterMutationDecision
// so RequireCharacterMutationPolicy can emit a single telemetry call.
type characterMutationResult struct {
	actor           storage.ParticipantRecord
	decision        string
	reasonCode      string
	extraAttributes map[string]any
	err             error
}

// evaluateCharacterMutationDecision contains the pure authorization logic for
// character mutations. It returns a result struct with no side effects (no
// telemetry, no audit writes).
func evaluateCharacterMutationDecision(
	ctx context.Context,
	deps PolicyDeps,
	campaignRecord storage.CampaignRecord,
	characterID string,
) characterMutationResult {
	characterAttributes := map[string]any{
		"character_id": strings.TrimSpace(characterID),
	}

	actor, reasonCode, err := AuthorizePolicyActorWithParticipantStore(ctx, deps.Participant, domainauthz.CapabilityMutateCharacters(), campaignRecord)
	if err != nil {
		return characterMutationResult{
			actor:           actor,
			decision:        DecisionDeny,
			reasonCode:      reasonCode,
			extraAttributes: characterAttributes,
			err:             err,
		}
	}

	decision := DecisionForReason(reasonCode)
	if decision == DecisionOverride {
		return characterMutationResult{
			actor:           actor,
			decision:        decision,
			reasonCode:      reasonCode,
			extraAttributes: MergeAttributes(characterAttributes, ExtraAttributesForReason(ctx, reasonCode)),
		}
	}

	if reasonCode == ReasonAllowAccessLevel && actor.CampaignAccess != participant.CampaignAccessMember {
		return characterMutationResult{
			actor:           actor,
			decision:        DecisionAllow,
			reasonCode:      reasonCode,
			extraAttributes: characterAttributes,
		}
	}

	ownerParticipantID, err := ResolveCharacterMutationOwnerParticipantIDFromStore(ctx, deps.Character, campaignRecord.ID, characterID)
	if err != nil {
		return characterMutationResult{
			actor:           actor,
			decision:        DecisionDeny,
			reasonCode:      ReasonErrorOwnerResolution,
			extraAttributes: characterAttributes,
			err:             err,
		}
	}

	ownershipDecision := domainauthz.CanCharacterMutation(actor.CampaignAccess, ids.ParticipantID(actor.ID), ids.ParticipantID(ownerParticipantID))
	if !ownershipDecision.Allowed {
		return characterMutationResult{
			actor:      actor,
			decision:   DecisionDeny,
			reasonCode: ownershipDecision.ReasonCode,
			extraAttributes: map[string]any{
				"character_id":         characterAttributes["character_id"],
				"owner_participant_id": ownerParticipantID,
			},
			err: status.Error(codes.PermissionDenied, "participant lacks permission"),
		}
	}

	return characterMutationResult{
		actor:           actor,
		decision:        DecisionAllow,
		reasonCode:      ownershipDecision.ReasonCode,
		extraAttributes: characterAttributes,
	}
}

// RequireCharacterMutationPolicy enforces role policy and owner-only mutation
// for members.
func RequireCharacterMutationPolicy(
	ctx context.Context,
	deps PolicyDeps,
	campaignRecord storage.CampaignRecord,
	characterID string,
) (storage.ParticipantRecord, error) {
	result := evaluateCharacterMutationDecision(ctx, deps, campaignRecord, characterID)
	EmitDecisionTelemetry(ctx, DecisionEvent{
		Store:           deps.Audit,
		CampaignID:      campaignRecord.ID,
		Capability:      domainauthz.CapabilityMutateCharacters(),
		Decision:        result.decision,
		ReasonCode:      result.reasonCode,
		Actor:           result.actor,
		Err:             result.err,
		ExtraAttributes: result.extraAttributes,
	})
	if result.err != nil {
		return storage.ParticipantRecord{}, result.err
	}
	return result.actor, nil
}

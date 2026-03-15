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
// enforcement. Entity applications embed this instead of the full Stores struct.
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
	return RequirePolicy(ctx, deps, domainauthz.CapabilityReadCampaign, campaignRecord)
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

// RequireCharacterMutationPolicy enforces role policy and owner-only mutation
// for members.
func RequireCharacterMutationPolicy(
	ctx context.Context,
	deps PolicyDeps,
	campaignRecord storage.CampaignRecord,
	characterID string,
) (storage.ParticipantRecord, error) {
	actor, reasonCode, err := AuthorizePolicyActorWithParticipantStore(ctx, deps.Participant, domainauthz.CapabilityMutateCharacters, campaignRecord)
	characterAttributes := map[string]any{
		"character_id": strings.TrimSpace(characterID),
	}
	if err != nil {
		EmitDecisionTelemetry(ctx, DecisionEvent{
			Store:           deps.Audit,
			CampaignID:      campaignRecord.ID,
			Capability:      domainauthz.CapabilityMutateCharacters,
			Decision:        DecisionDeny,
			ReasonCode:      reasonCode,
			Actor:           actor,
			Err:             err,
			ExtraAttributes: characterAttributes,
		})
		return storage.ParticipantRecord{}, err
	}
	decision := DecisionForReason(reasonCode)
	overrideAttributes := MergeAttributes(characterAttributes, ExtraAttributesForReason(ctx, reasonCode))
	if decision == DecisionOverride {
		EmitDecisionTelemetry(ctx, DecisionEvent{
			Store:           deps.Audit,
			CampaignID:      campaignRecord.ID,
			Capability:      domainauthz.CapabilityMutateCharacters,
			Decision:        decision,
			ReasonCode:      reasonCode,
			Actor:           actor,
			ExtraAttributes: overrideAttributes,
		})
		return actor, nil
	}
	if reasonCode == ReasonAllowAccessLevel && actor.CampaignAccess != participant.CampaignAccessMember {
		EmitDecisionTelemetry(ctx, DecisionEvent{
			Store:           deps.Audit,
			CampaignID:      campaignRecord.ID,
			Capability:      domainauthz.CapabilityMutateCharacters,
			Decision:        DecisionAllow,
			ReasonCode:      reasonCode,
			Actor:           actor,
			ExtraAttributes: characterAttributes,
		})
		return actor, nil
	}
	ownerParticipantID, err := ResolveCharacterMutationOwnerParticipantIDFromStore(ctx, deps.Character, campaignRecord.ID, characterID)
	if err != nil {
		EmitDecisionTelemetry(ctx, DecisionEvent{
			Store:           deps.Audit,
			CampaignID:      campaignRecord.ID,
			Capability:      domainauthz.CapabilityMutateCharacters,
			Decision:        DecisionDeny,
			ReasonCode:      ReasonErrorOwnerResolution,
			Actor:           actor,
			Err:             err,
			ExtraAttributes: characterAttributes,
		})
		return storage.ParticipantRecord{}, err
	}
	ownershipDecision := domainauthz.CanCharacterMutation(actor.CampaignAccess, ids.ParticipantID(actor.ID), ids.ParticipantID(ownerParticipantID))
	if !ownershipDecision.Allowed {
		err := status.Error(codes.PermissionDenied, "participant lacks permission")
		EmitDecisionTelemetry(ctx, DecisionEvent{
			Store:      deps.Audit,
			CampaignID: campaignRecord.ID,
			Capability: domainauthz.CapabilityMutateCharacters,
			Decision:   DecisionDeny,
			ReasonCode: ownershipDecision.ReasonCode,
			Actor:      actor,
			Err:        err,
			ExtraAttributes: map[string]any{
				"character_id":         characterAttributes["character_id"],
				"owner_participant_id": ownerParticipantID,
			},
		})
		return storage.ParticipantRecord{}, err
	}
	EmitDecisionTelemetry(ctx, DecisionEvent{
		Store:           deps.Audit,
		CampaignID:      campaignRecord.ID,
		Capability:      domainauthz.CapabilityMutateCharacters,
		Decision:        DecisionAllow,
		ReasonCode:      ownershipDecision.ReasonCode,
		Actor:           actor,
		ExtraAttributes: characterAttributes,
	})
	return actor, nil
}

package game

import (
	"context"
	"strings"

	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// requirePolicy ensures the participant has access for the requested action.
func requirePolicy(ctx context.Context, stores Stores, capability domainauthz.Capability, campaignRecord storage.CampaignRecord) error {
	_, err := requirePolicyActor(ctx, stores, capability, campaignRecord)
	return err
}

// requireReadPolicy ensures the actor can access campaign-scoped reads.
func requireReadPolicy(ctx context.Context, stores Stores, campaignRecord storage.CampaignRecord) error {
	return requirePolicy(ctx, stores, domainauthz.CapabilityReadCampaign, campaignRecord)
}

// requirePolicyActor ensures access and returns the resolved participant actor.
func requirePolicyActor(ctx context.Context, stores Stores, capability domainauthz.Capability, campaignRecord storage.CampaignRecord) (storage.ParticipantRecord, error) {
	actor, reasonCode, err := authorizePolicyActor(ctx, stores, capability, campaignRecord)
	if err != nil {
		emitAuthzDecisionTelemetry(ctx, stores.Audit, campaignRecord.ID, capability, authzDecisionDeny, reasonCode, actor, err, nil)
		return storage.ParticipantRecord{}, err
	}
	emitAuthzDecisionTelemetry(
		ctx,
		stores.Audit,
		campaignRecord.ID,
		capability,
		authzDecisionForReason(reasonCode),
		reasonCode,
		actor,
		nil,
		authzExtraAttributesForReason(ctx, reasonCode),
	)
	return actor, nil
}

// requireCharacterMutationPolicy enforces role policy and owner-only mutation for members.
func requireCharacterMutationPolicy(
	ctx context.Context,
	stores Stores,
	campaignRecord storage.CampaignRecord,
	characterID string,
) (storage.ParticipantRecord, error) {
	actor, reasonCode, err := authorizePolicyActor(ctx, stores, domainauthz.CapabilityMutateCharacters, campaignRecord)
	characterAttributes := map[string]any{
		"character_id": strings.TrimSpace(characterID),
	}
	if err != nil {
		emitAuthzDecisionTelemetry(
			ctx,
			stores.Audit,
			campaignRecord.ID,
			domainauthz.CapabilityMutateCharacters,
			authzDecisionDeny,
			reasonCode,
			actor,
			err,
			characterAttributes,
		)
		return storage.ParticipantRecord{}, err
	}
	decision := authzDecisionForReason(reasonCode)
	overrideAttributes := mergeAuthzAttributes(characterAttributes, authzExtraAttributesForReason(ctx, reasonCode))
	if decision == authzDecisionOverride {
		emitAuthzDecisionTelemetry(
			ctx,
			stores.Audit,
			campaignRecord.ID,
			domainauthz.CapabilityMutateCharacters,
			decision,
			reasonCode,
			actor,
			nil,
			overrideAttributes,
		)
		return actor, nil
	}
	if reasonCode == authzReasonAllowAccessLevel && actor.CampaignAccess != participant.CampaignAccessMember {
		emitAuthzDecisionTelemetry(
			ctx,
			stores.Audit,
			campaignRecord.ID,
			domainauthz.CapabilityMutateCharacters,
			authzDecisionAllow,
			reasonCode,
			actor,
			nil,
			characterAttributes,
		)
		return actor, nil
	}
	ownerParticipantID, err := resolveCharacterMutationOwnerParticipantID(ctx, stores, campaignRecord.ID, characterID)
	if err != nil {
		emitAuthzDecisionTelemetry(ctx, stores.Audit, campaignRecord.ID, domainauthz.CapabilityMutateCharacters, authzDecisionDeny, authzReasonErrorOwnerResolution, actor, err, characterAttributes)
		return storage.ParticipantRecord{}, err
	}
	ownershipDecision := domainauthz.CanCharacterMutation(actor.CampaignAccess, actor.ID, ownerParticipantID)
	if !ownershipDecision.Allowed {
		err := status.Error(codes.PermissionDenied, "participant lacks permission")
		emitAuthzDecisionTelemetry(ctx, stores.Audit, campaignRecord.ID, domainauthz.CapabilityMutateCharacters, authzDecisionDeny, ownershipDecision.ReasonCode, actor, err, map[string]any{
			"character_id":         characterAttributes["character_id"],
			"owner_participant_id": ownerParticipantID,
		})
		return storage.ParticipantRecord{}, err
	}
	emitAuthzDecisionTelemetry(ctx, stores.Audit, campaignRecord.ID, domainauthz.CapabilityMutateCharacters, authzDecisionAllow, ownershipDecision.ReasonCode, actor, nil, characterAttributes)
	return actor, nil
}

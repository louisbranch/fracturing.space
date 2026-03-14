package game

import (
	"context"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/participanttransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// resolveCanCharacterOwnerParticipantID resolves owner context for optional
// character ownership authorization checks.
func resolveCanCharacterOwnerParticipantID(
	ctx context.Context,
	stores Stores,
	campaignID string,
	target *campaignv1.AuthorizationTarget,
) (string, bool, error) {
	return resolveCanCharacterOwnerParticipantIDWithCharacterStore(ctx, stores.Character, campaignID, target)
}

func resolveCanCharacterOwnerParticipantIDWithCharacterStore(
	ctx context.Context,
	characters storage.CharacterStore,
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
	ownerParticipantID, err := resolveCharacterMutationOwnerParticipantIDFromStore(ctx, characters, campaignID, characterID)
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
	return evaluateCanParticipantGovernanceTargetWithStores(ctx, stores.Participant, stores.Character, campaignID, actor, target)
}

func evaluateCanParticipantGovernanceTargetWithStores(
	ctx context.Context,
	participants storage.ParticipantStore,
	characters storage.CharacterStore,
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
	targetAccess := participanttransport.CampaignAccessFromProto(target.GetTargetCampaignAccess())
	requestedAccess := participanttransport.CampaignAccessFromProto(target.GetRequestedCampaignAccess())
	participantOperation := target.GetParticipantOperation()

	if targetParticipantID != "" && targetAccess == participant.CampaignAccessUnspecified && participants != nil {
		targetRecord, err := participants.GetParticipant(ctx, campaignID, targetParticipantID)
		if err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				return domainauthz.PolicyDecision{}, nil, false, grpcerror.Internal("load target participant", err)
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
		ownerCount, err := countCampaignOwners(ctx, participants, campaignID)
		if err != nil {
			return domainauthz.PolicyDecision{}, extraAttributes, false, err
		}
		targetOwnsActiveCharacters, err := participantOwnsActiveCharacters(ctx, characters, campaignID, targetParticipantID)
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

	ownerCount, err := countCampaignOwners(ctx, participants, campaignID)
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

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

// countCampaignOwners returns current owner-seat count for invariant checks.
func countCampaignOwners(ctx context.Context, participants storage.ParticipantStore, campaignID string) (int, error) {
	if participants == nil {
		return 0, status.Error(codes.Internal, "participant store is not configured")
	}
	records, err := participants.ListParticipantsByCampaign(ctx, campaignID)
	if err != nil {
		return 0, status.Errorf(codes.Internal, "list participants: %v", err)
	}
	ownerCount := 0
	for _, record := range records {
		if record.CampaignAccess == participant.CampaignAccessOwner {
			ownerCount++
		}
	}
	return ownerCount, nil
}

// participantPolicyDecisionError maps policy reason codes to denial messages.
func participantPolicyDecisionError(reasonCode string) error {
	switch strings.TrimSpace(reasonCode) {
	case domainauthz.ReasonDenyManagerOwnerMutationForbidden:
		return status.Error(codes.PermissionDenied, "manager cannot assign owner access")
	case domainauthz.ReasonDenyTargetIsOwner:
		return status.Error(codes.PermissionDenied, "manager cannot mutate owner participant")
	case domainauthz.ReasonDenyLastOwnerGuard:
		return status.Error(codes.PermissionDenied, "cannot remove or demote final owner")
	case domainauthz.ReasonDenyTargetOwnsActiveCharacters:
		return status.Error(codes.FailedPrecondition, "participant owns active characters; transfer ownership first")
	default:
		return status.Error(codes.PermissionDenied, "participant lacks permission")
	}
}

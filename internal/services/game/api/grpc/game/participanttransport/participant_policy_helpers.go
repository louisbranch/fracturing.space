package participanttransport

import (
	"strings"

	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

package game

import (
	"context"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// policyAction identifies a campaign management action requiring access checks.
type policyAction int

const (
	// policyActionManageParticipants allows managing participants.
	policyActionManageParticipants policyAction = iota + 1
	// policyActionManageInvites allows managing invites.
	policyActionManageInvites
)

// requirePolicy ensures the participant has access for the requested action.
func requirePolicy(ctx context.Context, stores Stores, action policyAction, campaignRecord storage.CampaignRecord) error {
	if stores.Participant == nil {
		return status.Error(codes.Internal, "participant store is not configured")
	}
	actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	if actorID == "" {
		return status.Error(codes.PermissionDenied, "missing participant identity")
	}

	actor, err := stores.Participant.GetParticipant(ctx, campaignRecord.ID, actorID)
	if err != nil {
		if err == storage.ErrNotFound {
			return status.Error(codes.PermissionDenied, "participant lacks permission")
		}
		return status.Errorf(codes.Internal, "load participant: %v", err)
	}
	if !canPerformPolicyAction(action, actor.CampaignAccess) {
		return status.Error(codes.PermissionDenied, "participant lacks permission")
	}
	return nil
}

// canPerformPolicyAction enforces the v0 access model for management actions.
func canPerformPolicyAction(action policyAction, access participant.CampaignAccess) bool {
	if action != policyActionManageParticipants && action != policyActionManageInvites {
		return false
	}
	return access == participant.CampaignAccessOwner || access == participant.CampaignAccessManager
}

package campaign

import (
	"context"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/campaign"
	"github.com/louisbranch/fracturing.space/internal/campaign/policy"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func requirePolicy(ctx context.Context, stores Stores, action policy.Action, campaignRecord campaign.Campaign) error {
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
	if !policy.Can(actor, action, campaignRecord) {
		return status.Error(codes.PermissionDenied, "participant lacks permission")
	}
	return nil
}

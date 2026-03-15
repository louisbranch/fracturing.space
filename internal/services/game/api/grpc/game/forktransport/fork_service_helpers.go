package forktransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/fork"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// calculateDepth calculates the fork depth by walking up the parent chain.
func calculateDepth(ctx context.Context, store storage.CampaignForkStore, campaignID string) int {
	depth := 0
	currentID := campaignID

	for i := 0; i < 100; i++ { // Limit to prevent infinite loops
		metadata, err := store.GetCampaignForkMetadata(ctx, currentID)
		if err != nil || metadata.ParentCampaignID == "" {
			break
		}
		depth++
		currentID = metadata.ParentCampaignID
	}

	return depth
}

// forkPointFromProto converts a proto ForkPoint to domain ForkPoint.
func forkPointFromProto(pb *campaignv1.ForkPoint) fork.ForkPoint {
	if pb == nil {
		return fork.ForkPoint{}
	}
	return fork.ForkPoint{
		EventSeq:  pb.GetEventSeq(),
		SessionID: pb.GetSessionId(),
	}
}

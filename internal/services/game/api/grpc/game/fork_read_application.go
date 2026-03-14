package game

import (
	"context"
	"errors"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a forkApplication) GetLineage(ctx context.Context, campaignID string) (*campaignv1.Lineage, error) {
	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.EnsureStatus(err)
	}
	if err := requireReadPolicyWithDependencies(ctx, a.auth, campaignRecord); err != nil {
		return nil, err
	}

	metadata, err := a.stores.CampaignFork.GetCampaignForkMetadata(ctx, campaignID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, grpcerror.Internal("get fork metadata", err)
	}

	depth := 0
	if metadata.ParentCampaignID != "" {
		depth = calculateDepth(ctx, a.stores.CampaignFork, metadata.ParentCampaignID) + 1
	}

	originID := metadata.OriginCampaignID
	if originID == "" {
		originID = campaignID
	}

	return &campaignv1.Lineage{
		CampaignId:       campaignID,
		ParentCampaignId: metadata.ParentCampaignID,
		ForkEventSeq:     metadata.ForkEventSeq,
		OriginCampaignId: originID,
		Depth:            int32(depth),
	}, nil
}

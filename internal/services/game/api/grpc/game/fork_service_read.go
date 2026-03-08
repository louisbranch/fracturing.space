package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetLineage returns the lineage (ancestry) of a campaign.
func (s *ForkService) GetLineage(ctx context.Context, in *campaignv1.GetLineageRequest) (*campaignv1.GetLineageResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get lineage request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	// Verify campaign exists
	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		if isNotFound(err) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "get campaign: %v", err)
	}
	if err := requireReadPolicy(ctx, s.stores, campaignRecord); err != nil {
		return nil, err
	}

	metadata, err := s.stores.CampaignFork.GetCampaignForkMetadata(ctx, campaignID)
	if err != nil && !isNotFound(err) {
		return nil, status.Errorf(codes.Internal, "get fork metadata: %v", err)
	}

	// Calculate depth by walking up the chain
	depth := 0
	if metadata.ParentCampaignID != "" {
		depth = calculateDepth(ctx, s.stores.CampaignFork, metadata.ParentCampaignID) + 1
	}

	originID := metadata.OriginCampaignID
	if originID == "" {
		originID = campaignID
	}

	return &campaignv1.GetLineageResponse{
		Lineage: &campaignv1.Lineage{
			CampaignId:       campaignID,
			ParentCampaignId: metadata.ParentCampaignID,
			ForkEventSeq:     metadata.ForkEventSeq,
			OriginCampaignId: originID,
			Depth:            int32(depth),
		},
	}, nil
}

// ListForks returns campaigns forked from a given campaign.
func (s *ForkService) ListForks(ctx context.Context, in *campaignv1.ListForksRequest) (*campaignv1.ListForksResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list forks request is required")
	}

	_, err := validate.RequiredID(in.GetSourceCampaignId(), "source campaign id")
	if err != nil {
		return nil, err
	}

	// Listing forks requires querying campaigns by parent_campaign_id,
	// which is not yet implemented in the storage layer.
	return nil, status.Error(codes.Unimplemented, "list forks not yet implemented")
}

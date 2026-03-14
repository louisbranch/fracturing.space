package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/campaigntransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ForkCampaign creates a new campaign by forking an existing campaign at a specific point.
func (s *ForkService) ForkCampaign(ctx context.Context, in *campaignv1.ForkCampaignRequest) (*campaignv1.ForkCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "fork campaign request is required")
	}

	sourceCampaignID, err := validate.RequiredID(in.GetSourceCampaignId(), "source campaign id")
	if err != nil {
		return nil, err
	}

	newCampaign, lineage, forkEventSeq, err := newForkApplication(s).ForkCampaign(ctx, sourceCampaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.ForkCampaignResponse{
		Campaign:     campaigntransport.CampaignToProto(newCampaign),
		Lineage:      lineage,
		ForkEventSeq: forkEventSeq,
	}, nil
}

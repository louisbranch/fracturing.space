package campaigntransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetCampaign returns a campaign metadata record by ID.
// Lifecycle validation and read-policy checks are enforced so one read model
// can serve all transport surfaces (gRPC and web).
func (s *CampaignService) GetCampaign(ctx context.Context, in *campaignv1.GetCampaignRequest) (*campaignv1.GetCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get campaign request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	c, err := s.app.GetCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	return &campaignv1.GetCampaignResponse{
		Campaign: CampaignToProto(c),
	}, nil
}

package campaigntransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListCampaigns returns a page of campaign metadata records.
// Admin override requests are allowed to enumerate campaigns without participant scope.
// Non-admin calls remain participant/user scoped and only return member campaigns.
func (s *CampaignService) ListCampaigns(ctx context.Context, in *campaignv1.ListCampaignsRequest) (*campaignv1.ListCampaignsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list campaigns request is required")
	}

	page, err := s.app.ListCampaigns(ctx, in)
	if err != nil {
		return nil, err
	}

	response := &campaignv1.ListCampaignsResponse{
		NextPageToken: page.nextPageToken,
	}
	if len(page.campaigns) == 0 {
		return response, nil
	}

	response.Campaigns = make([]*campaignv1.Campaign, 0, len(page.campaigns))
	for _, record := range page.campaigns {
		response.Campaigns = append(response.Campaigns, CampaignToProto(record))
	}

	return response, nil
}

package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetCampaignAIAuthState returns current campaign AI authorization state.
func (s *CampaignAIService) GetCampaignAIAuthState(ctx context.Context, in *campaignv1.GetCampaignAIAuthStateRequest) (*campaignv1.GetCampaignAIAuthStateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get campaign ai auth state request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	return s.app.GetCampaignAIAuthState(ctx, campaignID)
}

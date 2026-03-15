package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetCampaignAIBindingUsage returns campaign usage for one bound AI agent.
func (s *CampaignAIService) GetCampaignAIBindingUsage(ctx context.Context, in *campaignv1.GetCampaignAIBindingUsageRequest) (*campaignv1.GetCampaignAIBindingUsageResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get campaign ai binding usage request is required")
	}
	aiAgentID, err := validate.RequiredID(in.GetAiAgentId(), "ai agent id")
	if err != nil {
		return nil, err
	}
	return s.app.GetCampaignAIBindingUsage(ctx, aiAgentID)
}

package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
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
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	bindingReader, ok := s.stores.Campaign.(storage.CampaignAIBindingReader)
	if !ok {
		return nil, status.Error(codes.Internal, "campaign ai binding reader is not configured")
	}

	campaignIDs, err := bindingReader.ListCampaignIDsByAIAgent(ctx, aiAgentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list campaign ids by ai agent: %v", err)
	}
	return &campaignv1.GetCampaignAIBindingUsageResponse{
		ActiveCampaignCount: int32(len(campaignIDs)),
		CampaignIds:         campaignIDs,
	}, nil
}

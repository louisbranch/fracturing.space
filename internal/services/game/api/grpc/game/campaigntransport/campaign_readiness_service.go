package campaigntransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetCampaignSessionReadiness returns deterministic readiness blockers for session start.
func (s *CampaignService) GetCampaignSessionReadiness(ctx context.Context, in *campaignv1.GetCampaignSessionReadinessRequest) (*campaignv1.GetCampaignSessionReadinessResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get campaign session readiness request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	readinessProto, err := s.readiness.GetCampaignSessionReadiness(ctx, campaignID, in.GetLocale())
	if err != nil {
		return nil, err
	}

	return &campaignv1.GetCampaignSessionReadinessResponse{
		Readiness: readinessProto,
	}, nil
}

package campaigntransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/participanttransport"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateCampaign creates a new campaign metadata record.
func (s *CampaignService) CreateCampaign(ctx context.Context, in *campaignv1.CreateCampaignRequest) (*campaignv1.CreateCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create campaign request is required")
	}

	created, owner, err := s.app.CreateCampaign(ctx, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.CreateCampaignResponse{
		Campaign:         CampaignToProto(created),
		OwnerParticipant: participanttransport.ParticipantToProto(owner),
	}, nil
}

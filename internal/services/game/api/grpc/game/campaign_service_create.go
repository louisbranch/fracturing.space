package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateCampaign creates a new campaign metadata record.
func (s *CampaignService) CreateCampaign(ctx context.Context, in *campaignv1.CreateCampaignRequest) (*campaignv1.CreateCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create campaign request is required")
	}

	created, owner, err := newCampaignApplication(s).CreateCampaign(ctx, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.CreateCampaignResponse{
		Campaign:         campaignToProto(created),
		OwnerParticipant: participantToProto(owner),
	}, nil
}

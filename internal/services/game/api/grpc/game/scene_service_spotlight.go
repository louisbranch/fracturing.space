package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SetSceneSpotlight sets the scene spotlight.
func (s *SceneService) SetSceneSpotlight(ctx context.Context, in *campaignv1.SetSceneSpotlightRequest) (*campaignv1.SetSceneSpotlightResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set scene spotlight request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	err = newSceneApplication(s).SetSceneSpotlight(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.SetSceneSpotlightResponse{}, nil
}

// ClearSceneSpotlight clears the scene spotlight.
func (s *SceneService) ClearSceneSpotlight(ctx context.Context, in *campaignv1.ClearSceneSpotlightRequest) (*campaignv1.ClearSceneSpotlightResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "clear scene spotlight request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	err = newSceneApplication(s).ClearSceneSpotlight(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.ClearSceneSpotlightResponse{}, nil
}

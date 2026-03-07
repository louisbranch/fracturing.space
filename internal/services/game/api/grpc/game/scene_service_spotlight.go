package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SetSceneSpotlight sets the scene spotlight.
func (s *SceneService) SetSceneSpotlight(ctx context.Context, in *campaignv1.SetSceneSpotlightRequest) (*campaignv1.SetSceneSpotlightResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set scene spotlight request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	err := newSceneApplication(s).SetSceneSpotlight(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.SetSceneSpotlightResponse{}, nil
}

// ClearSceneSpotlight clears the scene spotlight.
func (s *SceneService) ClearSceneSpotlight(ctx context.Context, in *campaignv1.ClearSceneSpotlightRequest) (*campaignv1.ClearSceneSpotlightResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "clear scene spotlight request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	err := newSceneApplication(s).ClearSceneSpotlight(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.ClearSceneSpotlightResponse{}, nil
}

package sessiontransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetSessionSpotlight returns the current spotlight for a session.
func (s *SessionService) GetSessionSpotlight(ctx context.Context, in *campaignv1.GetSessionSpotlightRequest) (*campaignv1.GetSessionSpotlightResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get session spotlight request is required")
	}
	spotlight, err := newSessionApplication(s).GetSessionSpotlight(ctx, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.GetSessionSpotlightResponse{
		Spotlight: SpotlightToProto(spotlight),
	}, nil
}

// SetSessionSpotlight sets the current spotlight for a session.
func (s *SessionService) SetSessionSpotlight(ctx context.Context, in *campaignv1.SetSessionSpotlightRequest) (*campaignv1.SetSessionSpotlightResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set session spotlight request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	spotlight, err := newSessionApplication(s).SetSessionSpotlight(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.SetSessionSpotlightResponse{Spotlight: SpotlightToProto(spotlight)}, nil
}

// ClearSessionSpotlight clears the spotlight for a session.
func (s *SessionService) ClearSessionSpotlight(ctx context.Context, in *campaignv1.ClearSessionSpotlightRequest) (*campaignv1.ClearSessionSpotlightResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "clear session spotlight request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	spotlight, err := newSessionApplication(s).ClearSessionSpotlight(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.ClearSessionSpotlightResponse{Spotlight: SpotlightToProto(spotlight)}, nil
}

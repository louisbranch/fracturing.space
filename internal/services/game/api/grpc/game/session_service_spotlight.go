package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetSessionSpotlight returns the current spotlight for a session.
func (s *SessionService) GetSessionSpotlight(ctx context.Context, in *campaignv1.GetSessionSpotlightRequest) (*campaignv1.GetSessionSpotlightResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get session spotlight request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireReadPolicy(ctx, s.stores, campaignRecord); err != nil {
		return nil, err
	}
	if _, err := s.stores.Session.GetSession(ctx, campaignID, sessionID); err != nil {
		return nil, handleDomainError(err)
	}

	spotlight, err := s.stores.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	return &campaignv1.GetSessionSpotlightResponse{
		Spotlight: sessionSpotlightToProto(spotlight),
	}, nil
}

// SetSessionSpotlight sets the current spotlight for a session.
func (s *SessionService) SetSessionSpotlight(ctx context.Context, in *campaignv1.SetSessionSpotlightRequest) (*campaignv1.SetSessionSpotlightResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set session spotlight request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	spotlight, err := newSessionApplication(s).SetSessionSpotlight(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.SetSessionSpotlightResponse{Spotlight: sessionSpotlightToProto(spotlight)}, nil
}

// ClearSessionSpotlight clears the spotlight for a session.
func (s *SessionService) ClearSessionSpotlight(ctx context.Context, in *campaignv1.ClearSessionSpotlightRequest) (*campaignv1.ClearSessionSpotlightResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "clear session spotlight request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	spotlight, err := newSessionApplication(s).ClearSessionSpotlight(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.ClearSessionSpotlightResponse{Spotlight: sessionSpotlightToProto(spotlight)}, nil
}

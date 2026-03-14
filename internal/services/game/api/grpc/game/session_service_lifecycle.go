package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StartSession starts a new session for a campaign.
// Enforces at most one ACTIVE session per campaign.
func (s *SessionService) StartSession(ctx context.Context, in *campaignv1.StartSessionRequest) (*campaignv1.StartSessionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "start session request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	sess, err := newSessionApplication(s).StartSession(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.StartSessionResponse{
		Session: sessionToProto(sess),
	}, nil
}

// ListSessions returns a page of session records for a campaign.
func (s *SessionService) ListSessions(ctx context.Context, in *campaignv1.ListSessionsRequest) (*campaignv1.ListSessionsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list sessions request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return nil, err
	}
	if err := requireReadPolicy(ctx, s.stores, campaignRecord); err != nil {
		return nil, err
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListSessionsPageSize,
		Max:     maxListSessionsPageSize,
	})

	page, err := s.stores.Session.ListSessions(ctx, campaignID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, grpcerror.Internal("list sessions", err)
	}

	response := &campaignv1.ListSessionsResponse{
		NextPageToken: page.NextPageToken,
	}
	if len(page.Sessions) == 0 {
		return response, nil
	}

	response.Sessions = make([]*campaignv1.Session, 0, len(page.Sessions))
	for _, sess := range page.Sessions {
		response.Sessions = append(response.Sessions, sessionToProto(sess))
	}

	return response, nil
}

// GetSession returns a session by campaign ID and session ID.
func (s *SessionService) GetSession(ctx context.Context, in *campaignv1.GetSessionRequest) (*campaignv1.GetSessionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get session request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}

	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return nil, err
	}
	if err := requireReadPolicy(ctx, s.stores, campaignRecord); err != nil {
		return nil, err
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, err
	}

	return &campaignv1.GetSessionResponse{
		Session: sessionToProto(sess),
	}, nil
}

// EndSession ends a session by campaign ID and session ID.
func (s *SessionService) EndSession(ctx context.Context, in *campaignv1.EndSessionRequest) (*campaignv1.EndSessionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "end session request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	_, err = validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}

	updated, err := newSessionApplication(s).EndSession(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.EndSessionResponse{
		Session: sessionToProto(updated),
	}, nil
}

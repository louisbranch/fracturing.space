package game

import (
	"context"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultListSessionsPageSize = 10
	maxListSessionsPageSize     = 10
)

// SessionService implements the game.v1.SessionService gRPC API.
type SessionService struct {
	campaignv1.UnimplementedSessionServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

// NewSessionService creates a SessionService with default dependencies.
func NewSessionService(stores Stores) *SessionService {
	return &SessionService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}

// StartSession starts a new session for a campaign.
// Enforces at most one ACTIVE session per campaign.
func (s *SessionService) StartSession(ctx context.Context, in *campaignv1.StartSessionRequest) (*campaignv1.StartSessionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "start session request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	sess, err := newSessionApplication(s).StartSession(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
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

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListSessionsPageSize,
		Max:     maxListSessionsPageSize,
	})

	page, err := s.stores.Session.ListSessions(ctx, campaignID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list sessions: %v", err)
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

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	updated, err := newSessionApplication(s).EndSession(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.EndSessionResponse{
		Session: sessionToProto(updated),
	}, nil
}

// OpenSessionGate opens a session gate that blocks action events until resolved.
func (s *SessionService) OpenSessionGate(ctx context.Context, in *campaignv1.OpenSessionGateRequest) (*campaignv1.OpenSessionGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "open session gate request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	gate, err := newSessionApplication(s).OpenSessionGate(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}
	pbGate, err := sessionGateToProto(gate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "decode session gate: %v", err)
	}

	return &campaignv1.OpenSessionGateResponse{Gate: pbGate}, nil
}

// ResolveSessionGate resolves an open session gate.
func (s *SessionService) ResolveSessionGate(ctx context.Context, in *campaignv1.ResolveSessionGateRequest) (*campaignv1.ResolveSessionGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "resolve session gate request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	gate, err := newSessionApplication(s).ResolveSessionGate(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}
	pbGate, err := sessionGateToProto(gate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "decode session gate: %v", err)
	}

	return &campaignv1.ResolveSessionGateResponse{Gate: pbGate}, nil
}

// AbandonSessionGate abandons an open session gate.
func (s *SessionService) AbandonSessionGate(ctx context.Context, in *campaignv1.AbandonSessionGateRequest) (*campaignv1.AbandonSessionGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "abandon session gate request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	gate, err := newSessionApplication(s).AbandonSessionGate(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}
	pbGate, err := sessionGateToProto(gate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "decode session gate: %v", err)
	}

	return &campaignv1.AbandonSessionGateResponse{Gate: pbGate}, nil
}

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

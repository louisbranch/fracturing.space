package service

import (
	"context"
	"errors"
	"strings"
	"time"

	sessionv1 "github.com/louisbranch/duality-engine/api/gen/go/session/v1"
	"github.com/louisbranch/duality-engine/internal/id"
	sessiondomain "github.com/louisbranch/duality-engine/internal/session/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Stores groups all session-related storage interfaces.
type Stores struct {
	Campaign storage.CampaignStore
	Session  storage.SessionStore
}

// SessionService implements the SessionService gRPC API.
type SessionService struct {
	sessionv1.UnimplementedSessionServiceServer
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
func (s *SessionService) StartSession(ctx context.Context, in *sessionv1.StartSessionRequest) (*sessionv1.StartSessionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "start session request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}

	// Validate campaign_id
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	// Check campaign exists
	_, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "check campaign: %v", err)
	}

	// Check for existing active session
	_, err = s.stores.Session.GetActiveSession(ctx, campaignID)
	if err == nil {
		// Active session exists
		return nil, status.Error(codes.FailedPrecondition, "active session exists")
	}
	if !errors.Is(err, storage.ErrNotFound) {
		// Unexpected error
		return nil, status.Errorf(codes.Internal, "check active session: %v", err)
	}

	// Create session domain object
	session, err := sessiondomain.CreateSession(sessiondomain.CreateSessionInput{
		CampaignID: campaignID,
		Name:       in.GetName(),
	}, s.clock, s.idGenerator)
	if err != nil {
		if errors.Is(err, sessiondomain.ErrEmptyCampaignID) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "create session: %v", err)
	}

	// Persist session and set as active (atomic operation)
	if err := s.stores.Session.PutSession(ctx, session); err != nil {
		if errors.Is(err, storage.ErrActiveSessionExists) {
			return nil, status.Error(codes.FailedPrecondition, "active session exists")
		}
		return nil, status.Errorf(codes.Internal, "persist session: %v", err)
	}

	response := &sessionv1.StartSessionResponse{
		Session: &sessionv1.Session{
			Id:         session.ID,
			CampaignId: session.CampaignID,
			Name:       session.Name,
			Status:     sessionStatusToProto(session.Status),
			StartedAt:  timestamppb.New(session.StartedAt),
			UpdatedAt:  timestamppb.New(session.UpdatedAt),
		},
	}
	if session.EndedAt != nil {
		response.Session.EndedAt = timestamppb.New(*session.EndedAt)
	}

	return response, nil
}

const (
	defaultListSessionsPageSize = 10
	maxListSessionsPageSize     = 10
)

// ListSessions returns a page of session records for a campaign.
func (s *SessionService) ListSessions(ctx context.Context, in *sessionv1.ListSessionsRequest) (*sessionv1.ListSessionsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list sessions request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}

	// Validate campaign exists
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	_, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "check campaign: %v", err)
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = defaultListSessionsPageSize
	}
	if pageSize > maxListSessionsPageSize {
		pageSize = maxListSessionsPageSize
	}

	page, err := s.stores.Session.ListSessions(ctx, campaignID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list sessions: %v", err)
	}

	response := &sessionv1.ListSessionsResponse{
		NextPageToken: page.NextPageToken,
	}
	if len(page.Sessions) == 0 {
		return response, nil
	}

	response.Sessions = make([]*sessionv1.Session, 0, len(page.Sessions))
	for _, session := range page.Sessions {
		sessionProto := &sessionv1.Session{
			Id:         session.ID,
			CampaignId: session.CampaignID,
			Name:       session.Name,
			Status:     sessionStatusToProto(session.Status),
			StartedAt:  timestamppb.New(session.StartedAt),
			UpdatedAt:  timestamppb.New(session.UpdatedAt),
		}
		if session.EndedAt != nil {
			sessionProto.EndedAt = timestamppb.New(*session.EndedAt)
		}
		response.Sessions = append(response.Sessions, sessionProto)
	}

	return response, nil
}

// GetSession returns a session by campaign ID and session ID.
func (s *SessionService) GetSession(ctx context.Context, in *sessionv1.GetSessionRequest) (*sessionv1.GetSessionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get session request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	// Validate campaign exists
	_, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "check campaign: %v", err)
	}

	session, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "get session: %v", err)
	}

	sessionProto := &sessionv1.Session{
		Id:         session.ID,
		CampaignId: session.CampaignID,
		Name:       session.Name,
		Status:     sessionStatusToProto(session.Status),
		StartedAt:  timestamppb.New(session.StartedAt),
		UpdatedAt:  timestamppb.New(session.UpdatedAt),
	}
	if session.EndedAt != nil {
		sessionProto.EndedAt = timestamppb.New(*session.EndedAt)
	}

	response := &sessionv1.GetSessionResponse{
		Session: sessionProto,
	}

	return response, nil
}

// sessionStatusToProto maps a domain session status to the protobuf representation.
func sessionStatusToProto(status sessiondomain.SessionStatus) sessionv1.SessionStatus {
	switch status {
	case sessiondomain.SessionStatusActive:
		return sessionv1.SessionStatus_ACTIVE
	case sessiondomain.SessionStatusPaused:
		return sessionv1.SessionStatus_PAUSED
	case sessiondomain.SessionStatusEnded:
		return sessionv1.SessionStatus_ENDED
	default:
		return sessionv1.SessionStatus_STATUS_UNSPECIFIED
	}
}

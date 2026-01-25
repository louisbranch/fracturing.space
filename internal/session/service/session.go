package service

import (
	"context"
	"errors"
	"strings"
	"time"

	sessionv1 "github.com/louisbranch/duality-engine/api/gen/go/session/v1"
	"github.com/louisbranch/duality-engine/internal/campaign/domain"
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
		idGenerator: domain.NewID,
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
	if err := s.stores.Session.PutSessionWithActivePointer(ctx, session); err != nil {
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

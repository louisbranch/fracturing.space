package game

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
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

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionStart); err != nil {
		return nil, handleDomainError(err)
	}

	switch c.Status {
	case campaign.CampaignStatusDraft, campaign.CampaignStatusActive:
		// Allowed to start a session.
	default:
		return nil, status.Error(codes.FailedPrecondition, "campaign status does not allow session start")
	}

	// Check for existing active session
	_, err = s.stores.Session.GetActiveSession(ctx, campaignID)
	if err == nil {
		return nil, handleDomainError(storage.ErrActiveSessionExists)
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return nil, status.Errorf(codes.Internal, "check active session: %v", err)
	}

	if c.Status == campaign.CampaignStatusDraft {
		updated, err := campaign.TransitionCampaignStatus(
			c,
			campaign.CampaignStatusActive,
			s.clock,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "activate campaign: %v", err)
		}
		if err := s.stores.Campaign.Put(ctx, updated); err != nil {
			return nil, status.Errorf(codes.Internal, "persist campaign status: %v", err)
		}
	}

	sess, err := session.CreateSession(session.CreateSessionInput{
		CampaignID: campaignID,
		Name:       in.GetName(),
	}, s.clock, s.idGenerator)
	if err != nil {
		return nil, handleDomainError(err)
	}

	if err := s.stores.Session.PutSession(ctx, sess); err != nil {
		if errors.Is(err, storage.ErrActiveSessionExists) {
			return nil, handleDomainError(err)
		}
		return nil, status.Errorf(codes.Internal, "persist session: %v", err)
	}

	payload := event.SessionStartedPayload{
		SessionID:   sess.ID,
		SessionName: sess.Name,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	if _, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    sess.StartedAt,
		Type:         event.TypeSessionStarted,
		SessionID:    sess.ID,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "session",
		EntityID:     sess.ID,
		PayloadJSON:  payloadJSON,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
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
	_, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
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

	_, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
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

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	if _, err := s.stores.Campaign.Get(ctx, campaignID); err != nil {
		return nil, handleDomainError(err)
	}

	endedAt := s.clock().UTC()
	sess, transitioned, err := s.stores.Session.EndSession(ctx, campaignID, sessionID, endedAt)
	if err != nil {
		return nil, handleDomainError(err)
	}

	if transitioned {
		payload := event.SessionEndedPayload{
			SessionID: sessionID,
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
		}

		actorID := grpcmeta.ParticipantIDFromContext(ctx)
		actorType := event.ActorTypeSystem
		if actorID != "" {
			actorType = event.ActorTypeParticipant
		}

		if _, err := s.stores.Event.AppendEvent(ctx, event.Event{
			CampaignID:   campaignID,
			Timestamp:    endedAt,
			Type:         event.TypeSessionEnded,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			ActorType:    actorType,
			ActorID:      actorID,
			EntityType:   "session",
			EntityID:     sessionID,
			PayloadJSON:  payloadJSON,
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "append event: %v", err)
		}
	}

	return &campaignv1.EndSessionResponse{
		Session: sessionToProto(sess),
	}, nil
}

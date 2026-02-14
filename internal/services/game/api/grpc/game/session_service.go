package game

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
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

	sess, err := session.CreateSession(session.CreateSessionInput{
		CampaignID: campaignID,
		Name:       in.GetName(),
	}, s.clock, s.idGenerator)
	if err != nil {
		return nil, handleDomainError(err)
	}

	applier := s.stores.Applier()

	if c.Status == campaign.CampaignStatusDraft {
		payload := event.CampaignUpdatedPayload{
			Fields: map[string]any{
				"status": campaignStatusToProto(campaign.CampaignStatusActive).String(),
			},
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

		stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
			CampaignID:   campaignID,
			Timestamp:    sess.StartedAt,
			Type:         event.TypeCampaignUpdated,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			ActorType:    actorType,
			ActorID:      actorID,
			EntityType:   "campaign",
			EntityID:     campaignID,
			PayloadJSON:  payloadJSON,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "append event: %v", err)
		}
		if err := applier.Apply(ctx, stored); err != nil {
			return nil, status.Errorf(codes.Internal, "apply event: %v", err)
		}
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

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
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
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
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
	_, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
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

	current, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if current.Status == session.SessionStatusEnded {
		return &campaignv1.EndSessionResponse{
			Session: sessionToProto(current),
		}, nil
	}

	endedAt := s.clock().UTC()
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

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
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
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := s.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load session: %v", err)
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
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	gateType, err := session.NormalizeGateType(in.GetGateType())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.SessionStatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	if _, err := s.stores.SessionGate.GetOpenSessionGate(ctx, campaignID, sessionID); err == nil {
		return nil, status.Error(codes.FailedPrecondition, "session gate already open")
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, status.Errorf(codes.Internal, "check session gate: %v", err)
	}

	gateID := strings.TrimSpace(in.GetGateId())
	if gateID == "" {
		gateID, err = s.idGenerator()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "generate gate id: %v", err)
		}
	}
	reason := session.NormalizeGateReason(in.GetReason())
	metadata := structToMap(in.GetMetadata())
	if err := validateStructPayload(metadata); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	payload := event.SessionGateOpenedPayload{
		GateID:   gateID,
		GateType: gateType,
		Reason:   reason,
		Metadata: metadata,
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

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeSessionGateOpened,
		SessionID:    sessionID,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "session_gate",
		EntityID:     gateID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := s.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	gate, err := s.stores.SessionGate.GetSessionGate(ctx, campaignID, sessionID, gateID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load session gate: %v", err)
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
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	gateID := strings.TrimSpace(in.GetGateId())
	if gateID == "" {
		return nil, status.Error(codes.InvalidArgument, "gate id is required")
	}

	if _, err := s.stores.Campaign.Get(ctx, campaignID); err != nil {
		return nil, handleDomainError(err)
	}
	if _, err := s.stores.Session.GetSession(ctx, campaignID, sessionID); err != nil {
		return nil, handleDomainError(err)
	}

	gate, err := s.stores.SessionGate.GetSessionGate(ctx, campaignID, sessionID, gateID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if gate.Status != string(session.GateStatusOpen) {
		pbGate, err := sessionGateToProto(gate)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "decode session gate: %v", err)
		}
		return &campaignv1.ResolveSessionGateResponse{Gate: pbGate}, nil
	}

	resolution := structToMap(in.GetResolution())
	if err := validateStructPayload(resolution); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	payload := event.SessionGateResolvedPayload{
		GateID:     gateID,
		Decision:   strings.TrimSpace(in.GetDecision()),
		Resolution: resolution,
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

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeSessionGateResolved,
		SessionID:    sessionID,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "session_gate",
		EntityID:     gateID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := s.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	gate, err = s.stores.SessionGate.GetSessionGate(ctx, campaignID, sessionID, gateID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load session gate: %v", err)
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
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	gateID := strings.TrimSpace(in.GetGateId())
	if gateID == "" {
		return nil, status.Error(codes.InvalidArgument, "gate id is required")
	}

	if _, err := s.stores.Campaign.Get(ctx, campaignID); err != nil {
		return nil, handleDomainError(err)
	}
	if _, err := s.stores.Session.GetSession(ctx, campaignID, sessionID); err != nil {
		return nil, handleDomainError(err)
	}

	gate, err := s.stores.SessionGate.GetSessionGate(ctx, campaignID, sessionID, gateID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if gate.Status != string(session.GateStatusOpen) {
		pbGate, err := sessionGateToProto(gate)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "decode session gate: %v", err)
		}
		return &campaignv1.AbandonSessionGateResponse{Gate: pbGate}, nil
	}

	payload := event.SessionGateAbandonedPayload{
		GateID: gateID,
		Reason: session.NormalizeGateReason(in.GetReason()),
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

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeSessionGateAbandoned,
		SessionID:    sessionID,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "session_gate",
		EntityID:     gateID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := s.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	gate, err = s.stores.SessionGate.GetSessionGate(ctx, campaignID, sessionID, gateID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load session gate: %v", err)
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

	if _, err := s.stores.Campaign.Get(ctx, campaignID); err != nil {
		return nil, handleDomainError(err)
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
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	spotlightType, err := sessionSpotlightTypeFromProto(in.GetType())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if err := session.ValidateSpotlightTarget(spotlightType, characterID); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.SessionStatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	payload := event.SessionSpotlightSetPayload{
		SpotlightType: string(spotlightType),
		CharacterID:   characterID,
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

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeSessionSpotlightSet,
		SessionID:    sessionID,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "session_spotlight",
		EntityID:     sessionID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := s.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	spotlight, err := s.stores.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, sessionID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load session spotlight: %v", err)
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
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	reason := strings.TrimSpace(in.GetReason())

	if _, err := s.stores.Campaign.Get(ctx, campaignID); err != nil {
		return nil, handleDomainError(err)
	}
	if _, err := s.stores.Session.GetSession(ctx, campaignID, sessionID); err != nil {
		return nil, handleDomainError(err)
	}

	spotlight, err := s.stores.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	payload := event.SessionSpotlightClearedPayload{Reason: reason}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeSessionSpotlightCleared,
		SessionID:    sessionID,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "session_spotlight",
		EntityID:     sessionID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := s.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	return &campaignv1.ClearSessionSpotlightResponse{Spotlight: sessionSpotlightToProto(spotlight)}, nil
}

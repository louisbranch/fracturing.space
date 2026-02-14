package game

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type sessionApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

func newSessionApplication(service *SessionService) sessionApplication {
	app := sessionApplication{stores: service.stores, clock: service.clock, idGenerator: service.idGenerator}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}

func (a sessionApplication) StartSession(ctx context.Context, campaignID string, in *campaignv1.StartSessionRequest) (session.Session, error) {
	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return session.Session{}, err
	}

	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionStart); err != nil {
		return session.Session{}, err
	}

	switch c.Status {
	case campaign.CampaignStatusDraft, campaign.CampaignStatusActive:
		// Allowed to start a session.
	default:
		return session.Session{}, status.Error(codes.FailedPrecondition, "campaign status does not allow session start")
	}

	_, err = a.stores.Session.GetActiveSession(ctx, campaignID)
	if err == nil {
		return session.Session{}, storage.ErrActiveSessionExists
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return session.Session{}, status.Errorf(codes.Internal, "check active session: %v", err)
	}

	sess, err := session.CreateSession(session.CreateSessionInput{
		CampaignID: campaignID,
		Name:       in.GetName(),
	}, a.clock, a.idGenerator)
	if err != nil {
		return session.Session{}, err
	}

	applier := a.stores.Applier()

	if c.Status == campaign.CampaignStatusDraft {
		payload := event.CampaignUpdatedPayload{
			Fields: map[string]any{
				"status": campaignStatusToProto(campaign.CampaignStatusActive).String(),
			},
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return session.Session{}, status.Errorf(codes.Internal, "encode payload: %v", err)
		}

		actorID := grpcmeta.ParticipantIDFromContext(ctx)
		actorType := event.ActorTypeSystem
		if actorID != "" {
			actorType = event.ActorTypeParticipant
		}

		stored, err := a.stores.Event.AppendEvent(ctx, event.Event{
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
			return session.Session{}, status.Errorf(codes.Internal, "append event: %v", err)
		}
		if err := applier.Apply(ctx, stored); err != nil {
			return session.Session{}, status.Errorf(codes.Internal, "apply event: %v", err)
		}
	}

	payload := event.SessionStartedPayload{
		SessionID:   sess.ID,
		SessionName: sess.Name,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return session.Session{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := a.stores.Event.AppendEvent(ctx, event.Event{
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
		return session.Session{}, status.Errorf(codes.Internal, "append event: %v", err)
	}
	if err := applier.Apply(ctx, stored); err != nil {
		return session.Session{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	return sess, nil
}

func (a sessionApplication) EndSession(ctx context.Context, campaignID string, in *campaignv1.EndSessionRequest) (session.Session, error) {
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return session.Session{}, status.Error(codes.InvalidArgument, "session id is required")
	}

	if _, err := a.stores.Campaign.Get(ctx, campaignID); err != nil {
		return session.Session{}, err
	}

	current, err := a.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return session.Session{}, err
	}
	if current.Status == session.SessionStatusEnded {
		return current, nil
	}

	endedAt := a.clock().UTC()
	payload := event.SessionEndedPayload{
		SessionID: sessionID,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return session.Session{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := a.stores.Event.AppendEvent(ctx, event.Event{
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
		return session.Session{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := a.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return session.Session{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := a.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return session.Session{}, status.Errorf(codes.Internal, "load session: %v", err)
	}

	return updated, nil
}

func (a sessionApplication) OpenSessionGate(ctx context.Context, campaignID string, in *campaignv1.OpenSessionGateRequest) (storage.SessionGate, error) {
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return storage.SessionGate{}, status.Error(codes.InvalidArgument, "session id is required")
	}
	gateType, err := session.NormalizeGateType(in.GetGateType())
	if err != nil {
		return storage.SessionGate{}, status.Error(codes.InvalidArgument, err.Error())
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.SessionGate{}, err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return storage.SessionGate{}, err
	}
	sess, err := a.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionGate{}, err
	}
	if sess.Status != session.SessionStatusActive {
		return storage.SessionGate{}, status.Error(codes.FailedPrecondition, "session is not active")
	}

	if _, err := a.stores.SessionGate.GetOpenSessionGate(ctx, campaignID, sessionID); err == nil {
		return storage.SessionGate{}, status.Error(codes.FailedPrecondition, "session gate already open")
	} else if !errors.Is(err, storage.ErrNotFound) {
		return storage.SessionGate{}, status.Errorf(codes.Internal, "check session gate: %v", err)
	}

	gateID := strings.TrimSpace(in.GetGateId())
	if gateID == "" {
		gateID, err = a.idGenerator()
		if err != nil {
			return storage.SessionGate{}, status.Errorf(codes.Internal, "generate gate id: %v", err)
		}
	}
	reason := session.NormalizeGateReason(in.GetReason())
	metadata := structToMap(in.GetMetadata())
	if err := validateStructPayload(metadata); err != nil {
		return storage.SessionGate{}, status.Error(codes.InvalidArgument, err.Error())
	}

	payload := event.SessionGateOpenedPayload{
		GateID:   gateID,
		GateType: gateType,
		Reason:   reason,
		Metadata: metadata,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionGate{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := a.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    a.clock().UTC(),
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
		return storage.SessionGate{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := a.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return storage.SessionGate{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	gate, err := a.stores.SessionGate.GetSessionGate(ctx, campaignID, sessionID, gateID)
	if err != nil {
		return storage.SessionGate{}, status.Errorf(codes.Internal, "load session gate: %v", err)
	}

	return gate, nil
}

func (a sessionApplication) ResolveSessionGate(ctx context.Context, campaignID string, in *campaignv1.ResolveSessionGateRequest) (storage.SessionGate, error) {
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return storage.SessionGate{}, status.Error(codes.InvalidArgument, "session id is required")
	}
	gateID := strings.TrimSpace(in.GetGateId())
	if gateID == "" {
		return storage.SessionGate{}, status.Error(codes.InvalidArgument, "gate id is required")
	}

	if _, err := a.stores.Campaign.Get(ctx, campaignID); err != nil {
		return storage.SessionGate{}, err
	}
	if _, err := a.stores.Session.GetSession(ctx, campaignID, sessionID); err != nil {
		return storage.SessionGate{}, err
	}

	gate, err := a.stores.SessionGate.GetSessionGate(ctx, campaignID, sessionID, gateID)
	if err != nil {
		return storage.SessionGate{}, err
	}
	if gate.Status != string(session.GateStatusOpen) {
		return gate, nil
	}

	resolution := structToMap(in.GetResolution())
	if err := validateStructPayload(resolution); err != nil {
		return storage.SessionGate{}, status.Error(codes.InvalidArgument, err.Error())
	}
	payload := event.SessionGateResolvedPayload{
		GateID:     gateID,
		Decision:   strings.TrimSpace(in.GetDecision()),
		Resolution: resolution,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionGate{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := a.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    a.clock().UTC(),
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
		return storage.SessionGate{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := a.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return storage.SessionGate{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := a.stores.SessionGate.GetSessionGate(ctx, campaignID, sessionID, gateID)
	if err != nil {
		return storage.SessionGate{}, status.Errorf(codes.Internal, "load session gate: %v", err)
	}

	return updated, nil
}

func (a sessionApplication) AbandonSessionGate(ctx context.Context, campaignID string, in *campaignv1.AbandonSessionGateRequest) (storage.SessionGate, error) {
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return storage.SessionGate{}, status.Error(codes.InvalidArgument, "session id is required")
	}
	gateID := strings.TrimSpace(in.GetGateId())
	if gateID == "" {
		return storage.SessionGate{}, status.Error(codes.InvalidArgument, "gate id is required")
	}

	if _, err := a.stores.Campaign.Get(ctx, campaignID); err != nil {
		return storage.SessionGate{}, err
	}
	if _, err := a.stores.Session.GetSession(ctx, campaignID, sessionID); err != nil {
		return storage.SessionGate{}, err
	}

	gate, err := a.stores.SessionGate.GetSessionGate(ctx, campaignID, sessionID, gateID)
	if err != nil {
		return storage.SessionGate{}, err
	}
	if gate.Status != string(session.GateStatusOpen) {
		return gate, nil
	}

	payload := event.SessionGateAbandonedPayload{
		GateID: gateID,
		Reason: session.NormalizeGateReason(in.GetReason()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionGate{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := a.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    a.clock().UTC(),
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
		return storage.SessionGate{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := a.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return storage.SessionGate{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := a.stores.SessionGate.GetSessionGate(ctx, campaignID, sessionID, gateID)
	if err != nil {
		return storage.SessionGate{}, status.Errorf(codes.Internal, "load session gate: %v", err)
	}

	return updated, nil
}

func (a sessionApplication) SetSessionSpotlight(ctx context.Context, campaignID string, in *campaignv1.SetSessionSpotlightRequest) (storage.SessionSpotlight, error) {
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return storage.SessionSpotlight{}, status.Error(codes.InvalidArgument, "session id is required")
	}
	spotlightType, err := sessionSpotlightTypeFromProto(in.GetType())
	if err != nil {
		return storage.SessionSpotlight{}, status.Error(codes.InvalidArgument, err.Error())
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if err := session.ValidateSpotlightTarget(spotlightType, characterID); err != nil {
		return storage.SessionSpotlight{}, status.Error(codes.InvalidArgument, err.Error())
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.SessionSpotlight{}, err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return storage.SessionSpotlight{}, err
	}
	sess, err := a.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionSpotlight{}, err
	}
	if sess.Status != session.SessionStatusActive {
		return storage.SessionSpotlight{}, status.Error(codes.FailedPrecondition, "session is not active")
	}

	payload := event.SessionSpotlightSetPayload{
		SpotlightType: string(spotlightType),
		CharacterID:   characterID,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionSpotlight{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := a.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    a.clock().UTC(),
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
		return storage.SessionSpotlight{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := a.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return storage.SessionSpotlight{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	spotlight, err := a.stores.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionSpotlight{}, status.Errorf(codes.Internal, "load session spotlight: %v", err)
	}

	return spotlight, nil
}

func (a sessionApplication) ClearSessionSpotlight(ctx context.Context, campaignID string, in *campaignv1.ClearSessionSpotlightRequest) (storage.SessionSpotlight, error) {
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return storage.SessionSpotlight{}, status.Error(codes.InvalidArgument, "session id is required")
	}
	reason := strings.TrimSpace(in.GetReason())

	if _, err := a.stores.Campaign.Get(ctx, campaignID); err != nil {
		return storage.SessionSpotlight{}, err
	}
	if _, err := a.stores.Session.GetSession(ctx, campaignID, sessionID); err != nil {
		return storage.SessionSpotlight{}, err
	}

	spotlight, err := a.stores.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionSpotlight{}, err
	}

	payload := event.SessionSpotlightClearedPayload{Reason: reason}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionSpotlight{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := a.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    a.clock().UTC(),
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
		return storage.SessionSpotlight{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := a.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return storage.SessionSpotlight{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	return spotlight, nil
}

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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
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

func (a sessionApplication) StartSession(ctx context.Context, campaignID string, in *campaignv1.StartSessionRequest) (storage.SessionRecord, error) {
	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.SessionRecord{}, err
	}

	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionStart); err != nil {
		return storage.SessionRecord{}, err
	}

	switch c.Status {
	case campaign.StatusDraft, campaign.StatusActive:
		// Allowed to start a session.
	default:
		return storage.SessionRecord{}, status.Error(codes.FailedPrecondition, "campaign status does not allow session start")
	}

	_, err = a.stores.Session.GetActiveSession(ctx, campaignID)
	if err == nil {
		return storage.SessionRecord{}, storage.ErrActiveSessionExists
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return storage.SessionRecord{}, status.Errorf(codes.Internal, "check active session: %v", err)
	}

	sessionID, err := a.idGenerator()
	if err != nil {
		return storage.SessionRecord{}, status.Errorf(codes.Internal, "generate session id: %v", err)
	}
	sessionName := strings.TrimSpace(in.GetName())

	applier := a.stores.Applier()
	if a.stores.Domain == nil {
		return storage.SessionRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	if c.Status == campaign.StatusDraft {
		payload := campaign.UpdatePayload{Fields: map[string]string{"status": "active"}}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return storage.SessionRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
		}
		_, err = executeAndApplyDomainCommand(
			ctx,
			a.stores.Domain,
			applier,
			command.Command{
				CampaignID:   campaignID,
				Type:         command.Type("campaign.update"),
				ActorType:    actorType,
				ActorID:      actorID,
				RequestID:    grpcmeta.RequestIDFromContext(ctx),
				InvocationID: grpcmeta.InvocationIDFromContext(ctx),
				EntityType:   "campaign",
				EntityID:     campaignID,
				PayloadJSON:  payloadJSON,
			},
			domainCommandApplyOptions{
				requireEvents:   true,
				missingEventMsg: "campaign.update did not emit an event",
			},
		)
		if err != nil {
			return storage.SessionRecord{}, err
		}
	}

	payload := session.StartPayload{
		SessionID:   sessionID,
		SessionName: sessionName,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Domain,
		applier,
		command.Command{
			CampaignID:   campaignID,
			Type:         command.Type("session.start"),
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session",
			EntityID:     sessionID,
			PayloadJSON:  payloadJSON,
		},
		domainCommandApplyOptions{
			requireEvents:   true,
			missingEventMsg: "session.start did not emit an event",
		},
	)
	if err != nil {
		return storage.SessionRecord{}, err
	}

	sess, err := a.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionRecord{}, status.Errorf(codes.Internal, "load session: %v", err)
	}
	return sess, nil
}

func (a sessionApplication) EndSession(ctx context.Context, campaignID string, in *campaignv1.EndSessionRequest) (storage.SessionRecord, error) {
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return storage.SessionRecord{}, status.Error(codes.InvalidArgument, "session id is required")
	}

	if _, err := a.stores.Campaign.Get(ctx, campaignID); err != nil {
		return storage.SessionRecord{}, err
	}

	current, err := a.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionRecord{}, err
	}
	if current.Status == session.StatusEnded {
		return current, nil
	}
	if a.stores.Domain == nil {
		return storage.SessionRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := session.EndPayload{SessionID: sessionID}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Domain,
		a.stores.Applier(),
		command.Command{
			CampaignID:   campaignID,
			Type:         command.Type("session.end"),
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session",
			EntityID:     sessionID,
			PayloadJSON:  payloadJSON,
		},
		domainCommandApplyOptions{
			requireEvents:   true,
			missingEventMsg: "session.end did not emit an event",
		},
	)
	if err != nil {
		return storage.SessionRecord{}, err
	}

	updated, err := a.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionRecord{}, status.Errorf(codes.Internal, "load session: %v", err)
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
	if sess.Status != session.StatusActive {
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
	if a.stores.Domain == nil {
		return storage.SessionGate{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := session.GateOpenedPayload{
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
	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Domain,
		a.stores.Applier(),
		command.Command{
			CampaignID:   campaignID,
			Type:         command.Type("session.gate_open"),
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session_gate",
			EntityID:     gateID,
			PayloadJSON:  payloadJSON,
		},
		domainCommandApplyOptions{
			requireEvents:   true,
			missingEventMsg: "session.gate_open did not emit an event",
		},
	)
	if err != nil {
		return storage.SessionGate{}, err
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
	if gate.Status != session.GateStatusOpen {
		return gate, nil
	}

	resolution := structToMap(in.GetResolution())
	if err := validateStructPayload(resolution); err != nil {
		return storage.SessionGate{}, status.Error(codes.InvalidArgument, err.Error())
	}
	if a.stores.Domain == nil {
		return storage.SessionGate{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := session.GateResolvedPayload{
		GateID:     gateID,
		Decision:   strings.TrimSpace(in.GetDecision()),
		Resolution: resolution,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionGate{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Domain,
		a.stores.Applier(),
		command.Command{
			CampaignID:   campaignID,
			Type:         command.Type("session.gate_resolve"),
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session_gate",
			EntityID:     gateID,
			PayloadJSON:  payloadJSON,
		},
		domainCommandApplyOptions{
			requireEvents:   true,
			missingEventMsg: "session.gate_resolve did not emit an event",
		},
	)
	if err != nil {
		return storage.SessionGate{}, err
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
	if gate.Status != session.GateStatusOpen {
		return gate, nil
	}
	if a.stores.Domain == nil {
		return storage.SessionGate{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := session.GateAbandonedPayload{
		GateID: gateID,
		Reason: session.NormalizeGateReason(in.GetReason()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionGate{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Domain,
		a.stores.Applier(),
		command.Command{
			CampaignID:   campaignID,
			Type:         command.Type("session.gate_abandon"),
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session_gate",
			EntityID:     gateID,
			PayloadJSON:  payloadJSON,
		},
		domainCommandApplyOptions{
			requireEvents:   true,
			missingEventMsg: "session.gate_abandon did not emit an event",
		},
	)
	if err != nil {
		return storage.SessionGate{}, err
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
	if sess.Status != session.StatusActive {
		return storage.SessionSpotlight{}, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if a.stores.Domain == nil {
		return storage.SessionSpotlight{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := session.SpotlightSetPayload{
		SpotlightType: string(spotlightType),
		CharacterID:   characterID,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionSpotlight{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Domain,
		a.stores.Applier(),
		command.Command{
			CampaignID:   campaignID,
			Type:         command.Type("session.spotlight_set"),
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session",
			EntityID:     sessionID,
			PayloadJSON:  payloadJSON,
		},
		domainCommandApplyOptions{
			applyErr:        domainApplyErrorWithCodePreserve("apply event"),
			requireEvents:   true,
			missingEventMsg: "session.spotlight_set did not emit an event",
		},
	)
	if err != nil {
		return storage.SessionSpotlight{}, err
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
	if a.stores.Domain == nil {
		return storage.SessionSpotlight{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := session.SpotlightClearedPayload{Reason: reason}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionSpotlight{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Domain,
		a.stores.Applier(),
		command.Command{
			CampaignID:   campaignID,
			Type:         command.Type("session.spotlight_clear"),
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session",
			EntityID:     sessionID,
			PayloadJSON:  payloadJSON,
		},
		domainCommandApplyOptions{
			applyErr:        domainApplyErrorWithCodePreserve("apply event"),
			requireEvents:   true,
			missingEventMsg: "session.spotlight_clear did not emit an event",
		},
	)
	if err != nil {
		return storage.SessionSpotlight{}, err
	}

	return spotlight, nil
}

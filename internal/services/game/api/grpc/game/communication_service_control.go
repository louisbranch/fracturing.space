package game

import (
	"context"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const communicationGMHandoffGateType = "gm_handoff"

// OpenCommunicationGate opens a new active-session communication gate for GM-managed control workflows.
func (s *CommunicationService) OpenCommunicationGate(ctx context.Context, in *campaignv1.OpenCommunicationGateRequest) (*campaignv1.OpenCommunicationGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "open communication gate request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	gateType, err := session.NormalizeGateType(in.GetGateType())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	controlMetadata := structToMap(in.GetMetadata())
	if err := validateStructPayload(controlMetadata); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	controlMetadata, err = session.NormalizeGateWorkflowMetadata(gateType, controlMetadata)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	actor, activeSession, openGate, err := s.loadActiveCommunicationGateControlState(ctx, campaignID, domainauthz.CapabilityManageSessions, false)
	if err != nil {
		return nil, err
	}
	if openGate != nil {
		return nil, status.Error(codes.FailedPrecondition, "another session gate is already open")
	}
	gateID, err := nextCommunicationGateID(s.idGenerator)
	if err != nil {
		return nil, err
	}

	payload := session.GateOpenedPayload{
		GateID:   ids.GateID(gateID),
		GateType: gateType,
		Reason:   session.NormalizeGateReason(in.GetReason()),
		Metadata: controlMetadata,
	}
	contextState, err := s.executeCommunicationGateOpen(withIncomingParticipantID(ctx, actor.ID), campaignID, activeSession.ID, payload, "communication.open_gate")
	if err != nil {
		return nil, err
	}
	return &campaignv1.OpenCommunicationGateResponse{Context: contextState}, nil
}

// RequestGMHandoff opens or reuses the active session's GM handoff gate for a participant-driven control action.
func (s *CommunicationService) RequestGMHandoff(ctx context.Context, in *campaignv1.RequestGMHandoffRequest) (*campaignv1.RequestGMHandoffResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request gm handoff request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	controlMetadata := structToMap(in.GetMetadata())
	if err := validateStructPayload(controlMetadata); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	actor, activeSession, openGate, err := s.loadActiveCommunicationGateControlState(ctx, campaignID, domainauthz.CapabilityReadCampaign, true)
	if err != nil {
		return nil, err
	}
	commandCtx := withIncomingParticipantID(ctx, actor.ID)
	if openGate != nil {
		if openGate.GateType != communicationGMHandoffGateType {
			return nil, status.Error(codes.FailedPrecondition, "another session gate is already open")
		}
		contextState, err := s.loadCommunicationContext(commandCtx, campaignID)
		if err != nil {
			return nil, err
		}
		return &campaignv1.RequestGMHandoffResponse{Context: contextState}, nil
	}
	gateID, err := nextCommunicationGateID(s.idGenerator)
	if err != nil {
		return nil, err
	}

	payload := session.GateOpenedPayload{
		GateID:   ids.GateID(gateID),
		GateType: communicationGMHandoffGateType,
		Reason:   session.NormalizeGateReason(in.GetReason()),
		Metadata: controlMetadata,
	}
	contextState, err := s.executeCommunicationGateOpen(commandCtx, campaignID, activeSession.ID, payload, "communication.request_gm_handoff")
	if err != nil {
		return nil, err
	}
	return &campaignv1.RequestGMHandoffResponse{Context: contextState}, nil
}

// ResolveCommunicationGate resolves the active session's current communication gate.
func (s *CommunicationService) ResolveCommunicationGate(ctx context.Context, in *campaignv1.ResolveCommunicationGateRequest) (*campaignv1.ResolveCommunicationGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "resolve communication gate request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	resolution := structToMap(in.GetResolution())
	if err := validateStructPayload(resolution); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	actor, activeSession, openGate, err := s.loadActiveCommunicationGateControlState(ctx, campaignID, domainauthz.CapabilityManageSessions, false)
	if err != nil {
		return nil, err
	}
	if openGate == nil {
		return nil, status.Error(codes.FailedPrecondition, "campaign has no open communication gate")
	}

	payload := session.GateResolvedPayload{
		GateID:     ids.GateID(openGate.GateID),
		Decision:   strings.TrimSpace(in.GetDecision()),
		Resolution: resolution,
	}
	contextState, err := s.executeCommunicationGateResolve(withIncomingParticipantID(ctx, actor.ID), campaignID, activeSession.ID, openGate.GateID, payload, "communication.resolve_gate")
	if err != nil {
		return nil, err
	}
	return &campaignv1.ResolveCommunicationGateResponse{Context: contextState}, nil
}

// RespondToCommunicationGate records one participant response against the active session gate.
func (s *CommunicationService) RespondToCommunicationGate(ctx context.Context, in *campaignv1.RespondToCommunicationGateRequest) (*campaignv1.RespondToCommunicationGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "respond to communication gate request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	responsePayload := structToMap(in.GetResponse())
	if err := validateStructPayload(responsePayload); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	actor, activeSession, openGate, err := s.loadActiveCommunicationGateControlState(ctx, campaignID, domainauthz.CapabilityReadCampaign, true)
	if err != nil {
		return nil, err
	}
	if openGate == nil {
		return nil, status.Error(codes.FailedPrecondition, "campaign has no open communication gate")
	}

	decision, responsePayload, err := session.ValidateGateResponse(
		openGate.GateType,
		openGate.MetadataJSON,
		actor.ID,
		in.GetDecision(),
		responsePayload,
	)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	payload := session.GateResponseRecordedPayload{
		GateID:        ids.GateID(openGate.GateID),
		ParticipantID: ids.ParticipantID(actor.ID),
		Decision:      decision,
		Response:      responsePayload,
	}
	contextState, err := s.executeCommunicationGateResponse(withIncomingParticipantID(ctx, actor.ID), campaignID, activeSession.ID, openGate.GateID, payload, "communication.respond_gate")
	if err != nil {
		return nil, err
	}
	return &campaignv1.RespondToCommunicationGateResponse{Context: contextState}, nil
}

// ResolveGMHandoff resolves the active session's GM handoff gate and returns refreshed communication context.
func (s *CommunicationService) ResolveGMHandoff(ctx context.Context, in *campaignv1.ResolveGMHandoffRequest) (*campaignv1.ResolveGMHandoffResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "resolve gm handoff request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	resolution := structToMap(in.GetResolution())
	if err := validateStructPayload(resolution); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	actor, activeSession, openGate, err := s.loadActiveCommunicationGateControlState(ctx, campaignID, domainauthz.CapabilityManageSessions, false)
	if err != nil {
		return nil, err
	}
	commandCtx := withIncomingParticipantID(ctx, actor.ID)
	if openGate == nil {
		contextState, err := s.loadCommunicationContext(commandCtx, campaignID)
		if err != nil {
			return nil, err
		}
		return &campaignv1.ResolveGMHandoffResponse{Context: contextState}, nil
	}
	if openGate.GateType != communicationGMHandoffGateType {
		return nil, status.Error(codes.FailedPrecondition, "active session gate is not a gm handoff")
	}

	payload := session.GateResolvedPayload{
		GateID:     ids.GateID(openGate.GateID),
		Decision:   strings.TrimSpace(in.GetDecision()),
		Resolution: resolution,
	}
	contextState, err := s.executeCommunicationGateResolve(commandCtx, campaignID, activeSession.ID, openGate.GateID, payload, "communication.resolve_gm_handoff")
	if err != nil {
		return nil, err
	}
	return &campaignv1.ResolveGMHandoffResponse{Context: contextState}, nil
}

// AbandonCommunicationGate abandons the active session's current communication gate.
func (s *CommunicationService) AbandonCommunicationGate(ctx context.Context, in *campaignv1.AbandonCommunicationGateRequest) (*campaignv1.AbandonCommunicationGateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "abandon communication gate request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	actor, activeSession, openGate, err := s.loadActiveCommunicationGateControlState(ctx, campaignID, domainauthz.CapabilityManageSessions, false)
	if err != nil {
		return nil, err
	}
	if openGate == nil {
		return nil, status.Error(codes.FailedPrecondition, "campaign has no open communication gate")
	}

	payload := session.GateAbandonedPayload{
		GateID: ids.GateID(openGate.GateID),
		Reason: session.NormalizeGateReason(in.GetReason()),
	}
	contextState, err := s.executeCommunicationGateAbandon(withIncomingParticipantID(ctx, actor.ID), campaignID, activeSession.ID, openGate.GateID, payload, "communication.abandon_gate")
	if err != nil {
		return nil, err
	}
	return &campaignv1.AbandonCommunicationGateResponse{Context: contextState}, nil
}

// AbandonGMHandoff abandons the active session's GM handoff gate and returns refreshed communication context.
func (s *CommunicationService) AbandonGMHandoff(ctx context.Context, in *campaignv1.AbandonGMHandoffRequest) (*campaignv1.AbandonGMHandoffResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "abandon gm handoff request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	actor, activeSession, openGate, err := s.loadActiveCommunicationGateControlState(ctx, campaignID, domainauthz.CapabilityManageSessions, false)
	if err != nil {
		return nil, err
	}
	commandCtx := withIncomingParticipantID(ctx, actor.ID)
	if openGate == nil {
		contextState, err := s.loadCommunicationContext(commandCtx, campaignID)
		if err != nil {
			return nil, err
		}
		return &campaignv1.AbandonGMHandoffResponse{Context: contextState}, nil
	}
	if openGate.GateType != communicationGMHandoffGateType {
		return nil, status.Error(codes.FailedPrecondition, "active session gate is not a gm handoff")
	}

	payload := session.GateAbandonedPayload{
		GateID: ids.GateID(openGate.GateID),
		Reason: session.NormalizeGateReason(in.GetReason()),
	}
	contextState, err := s.executeCommunicationGateAbandon(commandCtx, campaignID, activeSession.ID, openGate.GateID, payload, "communication.abandon_gm_handoff")
	if err != nil {
		return nil, err
	}
	return &campaignv1.AbandonGMHandoffResponse{Context: contextState}, nil
}

// loadActiveCommunicationGateControlState resolves the actor, active session, and current open gate for communication control flows.
func (s *CommunicationService) loadActiveCommunicationGateControlState(
	ctx context.Context,
	campaignID string,
	capability domainauthz.Capability,
	requireSessionAction bool,
) (storage.ParticipantRecord, *storage.SessionRecord, *storage.SessionGate, error) {
	if s.stores.Campaign == nil {
		return storage.ParticipantRecord{}, nil, nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.SessionGate == nil {
		return storage.ParticipantRecord{}, nil, nil, status.Error(codes.Internal, "session gate store is not configured")
	}

	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, nil, nil, err
	}
	actor, err := requirePolicyActor(ctx, s.stores, capability, campaignRecord)
	if err != nil {
		return storage.ParticipantRecord{}, nil, nil, err
	}
	if requireSessionAction {
		if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpSessionAction); err != nil {
			return storage.ParticipantRecord{}, nil, nil, err
		}
	}

	activeSession, err := s.loadActiveSession(ctx, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, nil, nil, err
	}
	if activeSession == nil {
		return storage.ParticipantRecord{}, nil, nil, status.Error(codes.FailedPrecondition, "campaign has no active session")
	}

	openGate, err := s.stores.SessionGate.GetOpenSessionGate(ctx, campaignID, activeSession.ID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return actor, activeSession, nil, nil
		}
		return storage.ParticipantRecord{}, nil, nil, status.Errorf(codes.Internal, "get open session gate: %v", err)
	}
	return actor, activeSession, &openGate, nil
}

// executeCommunicationGateOpen reuses the session gate write path so communication control remains a thin boundary over authoritative session events.
func (s *CommunicationService) executeCommunicationGateOpen(
	ctx context.Context,
	campaignID string,
	sessionID string,
	payload session.GateOpenedPayload,
	requireEventsLabel string,
) (*campaignv1.CommunicationContext, error) {
	return executeSessionGateCommandAndLoad(
		ctx,
		s.stores.Write,
		s.stores.Applier(),
		commandTypeSessionGateOpen,
		campaignID,
		sessionID,
		string(payload.GateID),
		payload,
		requireEventsLabel,
		func(ctx context.Context) (*campaignv1.CommunicationContext, error) {
			return s.loadCommunicationContext(ctx, campaignID)
		},
	)
}

// executeCommunicationGateResolve reuses the session gate resolve write path while keeping refreshed communication context as the only response contract.
func (s *CommunicationService) executeCommunicationGateResolve(
	ctx context.Context,
	campaignID string,
	sessionID string,
	gateID string,
	payload session.GateResolvedPayload,
	requireEventsLabel string,
) (*campaignv1.CommunicationContext, error) {
	return executeSessionGateCommandAndLoad(
		ctx,
		s.stores.Write,
		s.stores.Applier(),
		commandTypeSessionGateResolve,
		campaignID,
		sessionID,
		gateID,
		payload,
		requireEventsLabel,
		func(ctx context.Context) (*campaignv1.CommunicationContext, error) {
			return s.loadCommunicationContext(ctx, campaignID)
		},
	)
}

// executeCommunicationGateResponse records participant response state on the active gate while keeping communication consumers on projection-backed reads.
func (s *CommunicationService) executeCommunicationGateResponse(
	ctx context.Context,
	campaignID string,
	sessionID string,
	gateID string,
	payload session.GateResponseRecordedPayload,
	requireEventsLabel string,
) (*campaignv1.CommunicationContext, error) {
	return executeSessionGateCommandAndLoad(
		ctx,
		s.stores.Write,
		s.stores.Applier(),
		commandTypeSessionGateRespond,
		campaignID,
		sessionID,
		gateID,
		payload,
		requireEventsLabel,
		func(ctx context.Context) (*campaignv1.CommunicationContext, error) {
			return s.loadCommunicationContext(ctx, campaignID)
		},
	)
}

// executeCommunicationGateAbandon reuses the session gate abandon write path while hiding session-specific identifiers from communication consumers.
func (s *CommunicationService) executeCommunicationGateAbandon(
	ctx context.Context,
	campaignID string,
	sessionID string,
	gateID string,
	payload session.GateAbandonedPayload,
	requireEventsLabel string,
) (*campaignv1.CommunicationContext, error) {
	return executeSessionGateCommandAndLoad(
		ctx,
		s.stores.Write,
		s.stores.Applier(),
		commandTypeSessionGateAbandon,
		campaignID,
		sessionID,
		gateID,
		payload,
		requireEventsLabel,
		func(ctx context.Context) (*campaignv1.CommunicationContext, error) {
			return s.loadCommunicationContext(ctx, campaignID)
		},
	)
}

// nextCommunicationGateID resolves a communication gate identifier while keeping ID generation injectable for tests.
func nextCommunicationGateID(idGenerator func() (string, error)) (string, error) {
	gateID, err := idGenerator()
	if err != nil {
		return "", status.Errorf(codes.Internal, "generate gate id: %v", err)
	}
	return gateID, nil
}

// loadCommunicationContext reuses the canonical read path after control mutations so chat/web see the same projection-backed view.
func (s *CommunicationService) loadCommunicationContext(ctx context.Context, campaignID string) (*campaignv1.CommunicationContext, error) {
	resp, err := s.GetCommunicationContext(ctx, &campaignv1.GetCommunicationContextRequest{CampaignId: campaignID})
	if err != nil {
		return nil, err
	}
	return resp.GetContext(), nil
}

// withIncomingParticipantID ensures write commands are attributed to the resolved participant even when the caller authenticated by user-id only.
func withIncomingParticipantID(ctx context.Context, participantID string) context.Context {
	participantID = strings.TrimSpace(participantID)
	if participantID == "" {
		return ctx
	}
	md, _ := metadata.FromIncomingContext(ctx)
	if md == nil {
		md = metadata.MD{}
	}
	md = md.Copy()
	md.Set(grpcmeta.ParticipantIDHeader, participantID)
	return metadata.NewIncomingContext(ctx, md)
}

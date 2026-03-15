package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const communicationGMHandoffGateType = "gm_handoff"

func (a communicationApplication) OpenCommunicationGate(
	ctx context.Context,
	campaignID string,
	in *campaignv1.OpenCommunicationGateRequest,
) (*campaignv1.CommunicationContext, error) {
	gateType, err := session.NormalizeGateType(in.GetGateType())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	controlMetadata := handler.StructToMap(in.GetMetadata())
	if err := handler.ValidateStructPayload(controlMetadata); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	controlMetadata, err = session.NormalizeGateWorkflowMetadata(gateType, controlMetadata)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	actor, activeSession, openGate, err := a.loadActiveCommunicationGateControlState(ctx, campaignID, domainauthz.CapabilityManageSessions, false)
	if err != nil {
		return nil, err
	}
	if openGate != nil {
		return nil, status.Error(codes.FailedPrecondition, "another session gate is already open")
	}
	gateID, err := nextCommunicationGateID(a.idGenerator)
	if err != nil {
		return nil, err
	}

	payload := session.GateOpenedPayload{
		GateID:   ids.GateID(gateID),
		GateType: gateType,
		Reason:   session.NormalizeGateReason(in.GetReason()),
		Metadata: controlMetadata,
	}
	return a.executeCommunicationGateOpen(withIncomingParticipantID(ctx, actor.ID), campaignID, activeSession.ID, payload, "communication.open_gate")
}

func (a communicationApplication) RequestGMHandoff(
	ctx context.Context,
	campaignID string,
	in *campaignv1.RequestGMHandoffRequest,
) (*campaignv1.CommunicationContext, error) {
	controlMetadata := handler.StructToMap(in.GetMetadata())
	if err := handler.ValidateStructPayload(controlMetadata); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	actor, activeSession, openGate, err := a.loadActiveCommunicationGateControlState(ctx, campaignID, domainauthz.CapabilityReadCampaign, true)
	if err != nil {
		return nil, err
	}
	commandCtx := withIncomingParticipantID(ctx, actor.ID)
	if openGate != nil {
		if openGate.GateType != communicationGMHandoffGateType {
			return nil, status.Error(codes.FailedPrecondition, "another session gate is already open")
		}
		return a.loadCommunicationContext(commandCtx, campaignID)
	}
	gateID, err := nextCommunicationGateID(a.idGenerator)
	if err != nil {
		return nil, err
	}

	payload := session.GateOpenedPayload{
		GateID:   ids.GateID(gateID),
		GateType: communicationGMHandoffGateType,
		Reason:   session.NormalizeGateReason(in.GetReason()),
		Metadata: controlMetadata,
	}
	return a.executeCommunicationGateOpen(commandCtx, campaignID, activeSession.ID, payload, "communication.request_gm_handoff")
}

func (a communicationApplication) ResolveCommunicationGate(
	ctx context.Context,
	campaignID string,
	in *campaignv1.ResolveCommunicationGateRequest,
) (*campaignv1.CommunicationContext, error) {
	resolution := handler.StructToMap(in.GetResolution())
	if err := handler.ValidateStructPayload(resolution); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	actor, activeSession, openGate, err := a.loadActiveCommunicationGateControlState(ctx, campaignID, domainauthz.CapabilityManageSessions, false)
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
	return a.executeCommunicationGateResolve(withIncomingParticipantID(ctx, actor.ID), campaignID, activeSession.ID, openGate.GateID, payload, "communication.resolve_gate")
}

func (a communicationApplication) RespondToCommunicationGate(
	ctx context.Context,
	campaignID string,
	in *campaignv1.RespondToCommunicationGateRequest,
) (*campaignv1.CommunicationContext, error) {
	responsePayload := handler.StructToMap(in.GetResponse())
	if err := handler.ValidateStructPayload(responsePayload); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	actor, activeSession, openGate, err := a.loadActiveCommunicationGateControlState(ctx, campaignID, domainauthz.CapabilityReadCampaign, true)
	if err != nil {
		return nil, err
	}
	if openGate == nil {
		return nil, status.Error(codes.FailedPrecondition, "campaign has no open communication gate")
	}

	decision, responsePayload, err := session.ValidateGateResponseMetadata(
		openGate.GateType,
		openGate.Metadata,
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
	return a.executeCommunicationGateResponse(withIncomingParticipantID(ctx, actor.ID), campaignID, activeSession.ID, openGate.GateID, payload, "communication.respond_gate")
}

func (a communicationApplication) ResolveGMHandoff(
	ctx context.Context,
	campaignID string,
	in *campaignv1.ResolveGMHandoffRequest,
) (*campaignv1.CommunicationContext, error) {
	resolution := handler.StructToMap(in.GetResolution())
	if err := handler.ValidateStructPayload(resolution); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	actor, activeSession, openGate, err := a.loadActiveCommunicationGateControlState(ctx, campaignID, domainauthz.CapabilityManageSessions, false)
	if err != nil {
		return nil, err
	}
	commandCtx := withIncomingParticipantID(ctx, actor.ID)
	if openGate == nil {
		return a.loadCommunicationContext(commandCtx, campaignID)
	}
	if openGate.GateType != communicationGMHandoffGateType {
		return nil, status.Error(codes.FailedPrecondition, "active session gate is not a gm handoff")
	}

	payload := session.GateResolvedPayload{
		GateID:     ids.GateID(openGate.GateID),
		Decision:   strings.TrimSpace(in.GetDecision()),
		Resolution: resolution,
	}
	return a.executeCommunicationGateResolve(commandCtx, campaignID, activeSession.ID, openGate.GateID, payload, "communication.resolve_gm_handoff")
}

func (a communicationApplication) AbandonCommunicationGate(
	ctx context.Context,
	campaignID string,
	in *campaignv1.AbandonCommunicationGateRequest,
) (*campaignv1.CommunicationContext, error) {
	actor, activeSession, openGate, err := a.loadActiveCommunicationGateControlState(ctx, campaignID, domainauthz.CapabilityManageSessions, false)
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
	return a.executeCommunicationGateAbandon(withIncomingParticipantID(ctx, actor.ID), campaignID, activeSession.ID, openGate.GateID, payload, "communication.abandon_gate")
}

func (a communicationApplication) AbandonGMHandoff(
	ctx context.Context,
	campaignID string,
	in *campaignv1.AbandonGMHandoffRequest,
) (*campaignv1.CommunicationContext, error) {
	actor, activeSession, openGate, err := a.loadActiveCommunicationGateControlState(ctx, campaignID, domainauthz.CapabilityManageSessions, false)
	if err != nil {
		return nil, err
	}
	commandCtx := withIncomingParticipantID(ctx, actor.ID)
	if openGate == nil {
		return a.loadCommunicationContext(commandCtx, campaignID)
	}
	if openGate.GateType != communicationGMHandoffGateType {
		return nil, status.Error(codes.FailedPrecondition, "active session gate is not a gm handoff")
	}

	payload := session.GateAbandonedPayload{
		GateID: ids.GateID(openGate.GateID),
		Reason: session.NormalizeGateReason(in.GetReason()),
	}
	return a.executeCommunicationGateAbandon(commandCtx, campaignID, activeSession.ID, openGate.GateID, payload, "communication.abandon_gm_handoff")
}

func (a communicationApplication) loadActiveCommunicationGateControlState(
	ctx context.Context,
	campaignID string,
	capability domainauthz.Capability,
	requireSessionAction bool,
) (storage.ParticipantRecord, *storage.SessionRecord, *storage.SessionGate, error) {
	controlState, err := a.sessions.LoadActiveSessionGateControlState(ctx, campaignID, capability, requireSessionAction)
	if err != nil {
		return storage.ParticipantRecord{}, nil, nil, err
	}
	return controlState.actor, controlState.session, controlState.gate, nil
}

func (a communicationApplication) executeCommunicationGateOpen(
	ctx context.Context,
	campaignID string,
	sessionID string,
	payload session.GateOpenedPayload,
	requireEventsLabel string,
) (*campaignv1.CommunicationContext, error) {
	if err := a.gateCommands.Execute(
		ctx,
		handler.CommandTypeSessionGateOpen,
		campaignID,
		sessionID,
		string(payload.GateID),
		payload,
		requireEventsLabel,
	); err != nil {
		return nil, err
	}
	return a.loadCommunicationContext(ctx, campaignID)
}

func (a communicationApplication) executeCommunicationGateResolve(
	ctx context.Context,
	campaignID string,
	sessionID string,
	gateID string,
	payload session.GateResolvedPayload,
	requireEventsLabel string,
) (*campaignv1.CommunicationContext, error) {
	if err := a.gateCommands.Execute(
		ctx,
		handler.CommandTypeSessionGateResolve,
		campaignID,
		sessionID,
		gateID,
		payload,
		requireEventsLabel,
	); err != nil {
		return nil, err
	}
	return a.loadCommunicationContext(ctx, campaignID)
}

func (a communicationApplication) executeCommunicationGateResponse(
	ctx context.Context,
	campaignID string,
	sessionID string,
	gateID string,
	payload session.GateResponseRecordedPayload,
	requireEventsLabel string,
) (*campaignv1.CommunicationContext, error) {
	if err := a.gateCommands.Execute(
		ctx,
		handler.CommandTypeSessionGateRespond,
		campaignID,
		sessionID,
		gateID,
		payload,
		requireEventsLabel,
	); err != nil {
		return nil, err
	}
	return a.loadCommunicationContext(ctx, campaignID)
}

func (a communicationApplication) executeCommunicationGateAbandon(
	ctx context.Context,
	campaignID string,
	sessionID string,
	gateID string,
	payload session.GateAbandonedPayload,
	requireEventsLabel string,
) (*campaignv1.CommunicationContext, error) {
	if err := a.gateCommands.Execute(
		ctx,
		handler.CommandTypeSessionGateAbandon,
		campaignID,
		sessionID,
		gateID,
		payload,
		requireEventsLabel,
	); err != nil {
		return nil, err
	}
	return a.loadCommunicationContext(ctx, campaignID)
}

func nextCommunicationGateID(idGenerator func() (string, error)) (string, error) {
	gateID, err := idGenerator()
	if err != nil {
		return "", status.Errorf(codes.Internal, "generate gate id: %v", err)
	}
	return gateID, nil
}

func (a communicationApplication) loadCommunicationContext(ctx context.Context, campaignID string) (*campaignv1.CommunicationContext, error) {
	return a.GetCommunicationContext(ctx, campaignID)
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

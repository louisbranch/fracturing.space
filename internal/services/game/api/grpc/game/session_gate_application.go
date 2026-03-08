package game

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a sessionApplication) OpenSessionGate(ctx context.Context, campaignID string, in *campaignv1.OpenSessionGateRequest) (storage.SessionGate, error) {
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return storage.SessionGate{}, err
	}
	gateType, err := session.NormalizeGateType(in.GetGateType())
	if err != nil {
		return storage.SessionGate{}, status.Error(codes.InvalidArgument, err.Error())
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.SessionGate{}, err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
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
	payload := session.GateOpenedPayload{
		GateID:   ids.GateID(gateID),
		GateType: gateType,
		Reason:   reason,
		Metadata: metadata,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionGate{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Write,
		a.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeSessionGateOpen,
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session_gate",
			EntityID:     gateID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("session.gate_open did not emit an event"),
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
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return storage.SessionGate{}, err
	}
	gateID, err := validate.RequiredID(in.GetGateId(), "gate id")
	if err != nil {
		return storage.SessionGate{}, err
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.SessionGate{}, err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
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
	payload := session.GateResolvedPayload{
		GateID:     ids.GateID(gateID),
		Decision:   strings.TrimSpace(in.GetDecision()),
		Resolution: resolution,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionGate{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Write,
		a.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeSessionGateResolve,
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session_gate",
			EntityID:     gateID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("session.gate_resolve did not emit an event"),
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
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return storage.SessionGate{}, err
	}
	gateID, err := validate.RequiredID(in.GetGateId(), "gate id")
	if err != nil {
		return storage.SessionGate{}, err
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.SessionGate{}, err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
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
	payload := session.GateAbandonedPayload{
		GateID: ids.GateID(gateID),
		Reason: session.NormalizeGateReason(in.GetReason()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionGate{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Write,
		a.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeSessionGateAbandon,
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session_gate",
			EntityID:     gateID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("session.gate_abandon did not emit an event"),
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

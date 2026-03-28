package sessiontransport

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"

	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type activeSessionGateControlState struct {
	actor   storage.ParticipantRecord
	session *storage.SessionRecord
	gate    *storage.SessionGate
}

func (a sessionApplication) LoadActiveSessionGateControlState(
	ctx context.Context,
	campaignID string,
	capability domainauthz.Capability,
	requireSessionAction bool,
) (activeSessionGateControlState, error) {
	if a.stores.Campaign == nil {
		return activeSessionGateControlState{}, status.Error(codes.Internal, "campaign store is not configured")
	}

	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return activeSessionGateControlState{}, err
	}
	actor, err := authz.RequirePolicyActor(ctx, a.auth, capability, campaignRecord)
	if err != nil {
		return activeSessionGateControlState{}, err
	}
	if requireSessionAction {
		if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpSessionAction); err != nil {
			return activeSessionGateControlState{}, err
		}
	}

	activeSessionState, err := a.GetActiveSessionContext(ctx, campaignID)
	if err != nil {
		return activeSessionGateControlState{}, err
	}
	if activeSessionState.session == nil {
		return activeSessionGateControlState{}, status.Error(codes.FailedPrecondition, "campaign has no active session")
	}

	return activeSessionGateControlState{
		actor:   actor,
		session: activeSessionState.session,
		gate:    activeSessionState.gate,
	}, nil
}

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
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions(), c); err != nil {
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
	} else if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "check session gate"); lookupErr != nil {
		return storage.SessionGate{}, lookupErr
	}

	gateID := strings.TrimSpace(in.GetGateId())
	if gateID == "" {
		gateID, err = a.idGenerator()
		if err != nil {
			return storage.SessionGate{}, grpcerror.Internal("generate gate id", err)
		}
	}
	reason := session.NormalizeGateReason(in.GetReason())
	metadata := handler.StructToMap(in.GetMetadata())
	if err := handler.ValidateStructPayload(metadata); err != nil {
		return storage.SessionGate{}, status.Error(codes.InvalidArgument, err.Error())
	}
	payload := session.GateOpenedPayload{
		GateID:   ids.GateID(gateID),
		GateType: gateType,
		Reason:   reason,
		Metadata: metadata,
	}
	if err := a.gateCommands.Execute(
		ctx,
		commandids.SessionGateOpen,
		campaignID,
		sessionID,
		gateID,
		payload,
		"session.gate_open",
	); err != nil {
		return storage.SessionGate{}, err
	}
	gate, err := a.stores.SessionGate.GetSessionGate(ctx, campaignID, sessionID, gateID)
	if err != nil {
		return storage.SessionGate{}, grpcerror.Internal("load session gate", err)
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
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions(), c); err != nil {
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

	resolution := handler.StructToMap(in.GetResolution())
	if err := handler.ValidateStructPayload(resolution); err != nil {
		return storage.SessionGate{}, status.Error(codes.InvalidArgument, err.Error())
	}
	payload := session.GateResolvedPayload{
		GateID:     ids.GateID(gateID),
		Decision:   strings.TrimSpace(in.GetDecision()),
		Resolution: resolution,
	}
	if err := a.gateCommands.Execute(
		ctx,
		commandids.SessionGateResolve,
		campaignID,
		sessionID,
		gateID,
		payload,
		"session.gate_resolve",
	); err != nil {
		return storage.SessionGate{}, err
	}
	updated, err := a.stores.SessionGate.GetSessionGate(ctx, campaignID, sessionID, gateID)
	if err != nil {
		return storage.SessionGate{}, grpcerror.Internal("load session gate", err)
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
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions(), c); err != nil {
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
	if err := a.gateCommands.Execute(
		ctx,
		commandids.SessionGateAbandon,
		campaignID,
		sessionID,
		gateID,
		payload,
		"session.gate_abandon",
	); err != nil {
		return storage.SessionGate{}, err
	}
	updated, err := a.stores.SessionGate.GetSessionGate(ctx, campaignID, sessionID, gateID)
	if err != nil {
		return storage.SessionGate{}, grpcerror.Internal("load session gate", err)
	}
	return updated, nil
}

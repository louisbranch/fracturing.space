package outcometransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// applyRollOutcomeRequest keeps the validated request state and derived roll
// semantics together so the execution helpers can work from one stable plan.
type applyRollOutcomeRequest struct {
	campaignID           string
	sessionID            string
	sceneID              string
	rollSeq              uint64
	rollRequestID        string
	invocationID         string
	targets              []string
	generateHopeFear     bool
	requiresComplication bool
	crit                 bool
	flavor               string
	gmFearDelta          int
}

// hasGMFearGain lets later stages reuse the transport's GM-fear decision
// without repeating the outcome-derived branching logic.
func (r *applyRollOutcomeRequest) hasGMFearGain() bool {
	return r.gmFearDelta > 0
}

// loadApplyRollOutcomeRequest validates the transport request and resolves the
// campaign/session/roll data needed by the durable apply phase.
func (h *Handler) loadApplyRollOutcomeRequest(
	ctx context.Context,
	in *pb.ApplyRollOutcomeRequest,
) (*applyRollOutcomeRequest, error) {
	campaignID, err := validateCampaignIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if in.GetRollSeq() == 0 {
		return nil, status.Error(codes.InvalidArgument, "roll seq is required")
	}

	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart outcomes"); err != nil {
		return nil, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	rollEvent, err := h.deps.Event.GetEventBySeq(ctx, campaignID, in.GetRollSeq())
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if rollEvent.Type != eventTypeActionRollResolved {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference action.roll_resolved")
	}
	if rollEvent.SessionID.String() != sessionID {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload action.RollResolvePayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return nil, grpcerror.Internal("decode roll payload", err)
	}
	rollMetadata, err := workflowtransport.DecodeRollSystemMetadata(rollPayload.SystemData)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid roll system_data: %v", err)
	}

	rollRequestID := strings.TrimSpace(rollPayload.RequestID)
	if rollRequestID == "" {
		rollRequestID = strings.TrimSpace(rollEvent.RequestID)
	}
	if rollRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "roll request id is required")
	}

	rollKind := rollMetadata.RollKindOrDefault()
	generateHopeFear := workflowtransport.BoolValue(rollMetadata.HopeFear, rollKind != pb.RollKind_ROLL_KIND_REACTION)
	triggerGMMove := workflowtransport.BoolValue(rollMetadata.GMMove, rollKind != pb.RollKind_ROLL_KIND_REACTION)
	rollOutcome := rollMetadata.OutcomeOrFallback(rollPayload.Outcome)
	if rollOutcome == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is required")
	}
	flavor := workflowtransport.OutcomeFlavorFromCode(rollOutcome)
	if flavor == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome flavor is required")
	}
	if !generateHopeFear {
		flavor = ""
	}
	crit := workflowtransport.BoolValue(
		rollMetadata.Crit,
		strings.TrimSpace(rollOutcome) == pb.Outcome_CRITICAL_SUCCESS.String(),
	)

	targets := workflowtransport.NormalizeTargets(in.GetTargets())
	if len(targets) == 0 {
		rollerID := strings.TrimSpace(rollMetadata.CharacterID)
		if rollerID == "" {
			return nil, status.Error(codes.InvalidArgument, "targets are required")
		}
		targets = []string{rollerID}
	}

	gmFearDelta := 0
	if triggerGMMove && flavor == outcomeFlavorFear && !crit {
		gmFearDelta = len(targets)
	}

	return &applyRollOutcomeRequest{
		campaignID:           campaignID,
		sessionID:            sessionID,
		sceneID:              strings.TrimSpace(in.GetSceneId()),
		rollSeq:              in.GetRollSeq(),
		rollRequestID:        rollRequestID,
		invocationID:         grpcmeta.InvocationIDFromContext(ctx),
		targets:              targets,
		generateHopeFear:     generateHopeFear,
		requiresComplication: flavor == outcomeFlavorFear && !crit && triggerGMMove,
		crit:                 crit,
		flavor:               flavor,
		gmFearDelta:          gmFearDelta,
	}, nil
}

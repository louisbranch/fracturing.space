package outcometransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/gmconsequence"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// sessionOutcomePrelude carries the validated state shared by the session-level
// outcome handlers.
type sessionOutcomePrelude struct {
	campaignID    string
	sessionID     string
	rollPayload   action.RollResolvePayload
	rollMetadata  workflowtransport.RollSystemMetadata
	rollRequestID string
}

// validateSessionOutcome loads and validates the resolved roll event shared by
// attack, adversary attack, and reaction outcome handlers.
func (h *Handler) validateSessionOutcome(
	ctx context.Context,
	sessionID string,
	rollSeq uint64,
) (sessionOutcomePrelude, error) {
	if err := h.requireSessionOutcomeDependencies(); err != nil {
		return sessionOutcomePrelude{}, err
	}

	campaignID := strings.TrimSpace(grpcmeta.CampaignIDFromContext(ctx))
	if campaignID == "" {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "session id is required")
	}
	if rollSeq == 0 {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "roll seq is required")
	}

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return sessionOutcomePrelude{}, handleDomainError(ctx, err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return sessionOutcomePrelude{}, handleDomainError(ctx, err)
	}
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart outcomes"); err != nil {
		return sessionOutcomePrelude{}, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return sessionOutcomePrelude{}, handleDomainError(ctx, err)
	}
	if sess.Status != session.StatusActive {
		return sessionOutcomePrelude{}, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := h.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return sessionOutcomePrelude{}, err
	}

	rollEvent, err := h.deps.Event.GetEventBySeq(ctx, campaignID, rollSeq)
	if err != nil {
		return sessionOutcomePrelude{}, handleDomainError(ctx, err)
	}
	if rollEvent.Type != eventTypeActionRollResolved {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "roll seq does not reference action.roll_resolved")
	}
	if rollEvent.SessionID.String() != sessionID {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload action.RollResolvePayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return sessionOutcomePrelude{}, grpcerror.Internal("decode roll payload", err)
	}
	rollMetadata, err := workflowtransport.DecodeRollSystemMetadata(rollPayload.SystemData)
	if err != nil {
		return sessionOutcomePrelude{}, status.Errorf(codes.InvalidArgument, "invalid roll system_data: %v", err)
	}

	rollRequestID := strings.TrimSpace(rollPayload.RequestID)
	if rollRequestID == "" {
		rollRequestID = strings.TrimSpace(rollEvent.RequestID)
	}
	if rollRequestID == "" {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "roll request id is required")
	}

	return sessionOutcomePrelude{
		campaignID:    campaignID,
		sessionID:     sessionID,
		rollPayload:   rollPayload,
		rollMetadata:  rollMetadata,
		rollRequestID: rollRequestID,
	}, nil
}

// outcomeAlreadyAppliedForSessionRequest checks whether an outcome event with
// the same request ID already exists after the roll event.
func (h *Handler) outcomeAlreadyAppliedForSessionRequest(ctx context.Context, campaignID, sessionID string, rollSeq uint64, requestID string) (bool, error) {
	return h.sessionRequestEventExists(
		ctx,
		campaignID,
		sessionID,
		rollSeq,
		requestID,
		eventTypeActionOutcomeApplied,
		requestID,
	)
}

// sessionRequestEventExists checks whether the session event stream already
// contains a matching post-roll event for the same request.
func (h *Handler) sessionRequestEventExists(
	ctx context.Context,
	campaignID string,
	sessionID string,
	rollSeq uint64,
	requestID string,
	eventType event.Type,
	entityID string,
) (bool, error) {
	requestID = strings.TrimSpace(requestID)
	entityID = strings.TrimSpace(entityID)
	if rollSeq == 0 || requestID == "" {
		return false, nil
	}

	result, err := h.deps.Event.ListEventsPage(ctx, storage.ListEventsPageRequest{
		CampaignID: campaignID,
		AfterSeq:   rollSeq - 1,
		PageSize:   1,
		Filter: storage.EventQueryFilter{
			SessionID: sessionID,
			RequestID: requestID,
			EventType: string(eventType),
			EntityID:  entityID,
		},
	})
	if err != nil {
		return false, err
	}

	return len(result.Events) > 0, nil
}

// buildApplyRollOutcomeIdempotentResponse reloads the current projection state
// so retries return the same shape as a fresh apply.
func (h *Handler) buildApplyRollOutcomeIdempotentResponse(
	ctx context.Context,
	campaignID string,
	rollSeq uint64,
	targets []string,
	requiresComplication bool,
	includeGMFear bool,
) (*pb.ApplyRollOutcomeResponse, error) {
	updatedStates := make([]*pb.OutcomeCharacterState, 0, len(targets))
	for _, target := range targets {
		state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, target)
		if err != nil {
			return nil, handleDomainError(ctx, err)
		}
		updatedStates = append(updatedStates, &pb.OutcomeCharacterState{
			CharacterId: target,
			Hope:        int32(state.Hope),
			Stress:      int32(state.Stress),
			Hp:          int32(state.Hp),
		})
	}

	response := &pb.ApplyRollOutcomeResponse{
		RollSeq:              rollSeq,
		RequiresComplication: requiresComplication,
		Updated: &pb.OutcomeUpdated{
			CharacterStates: updatedStates,
		},
	}
	if !includeGMFear {
		return response, nil
	}

	currentSnap, err := h.deps.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.Internal("load gm fear snapshot", err)
	}
	value := int32(currentSnap.GMFear)
	response.Updated.GmFear = &value
	return response, nil
}

// gmConsequenceResolution caches the follow-up gate and spotlight writes needed
// when a fear outcome requires a GM consequence.
type gmConsequenceResolution = gmconsequence.Resolution

// resolveGMConsequence computes the gate and spotlight repairs needed for a GM
// consequence without applying them yet.
func (h *Handler) resolveGMConsequence(
	ctx context.Context,
	campaignID, sessionID string,
	rollSeq uint64,
	rollRequestID string,
) (gmConsequenceResolution, error) {
	return gmconsequence.Resolve(ctx, h.gmConsequenceDependencies(), campaignID, sessionID, &rollSeq, rollRequestID)
}

// buildGMConsequenceOutcomeEffects reports the follow-up effects an outcome
// event should carry when a GM consequence is required.
func (h *Handler) buildGMConsequenceOutcomeEffects(
	ctx context.Context,
	campaignID string,
	sessionID string,
	rollSeq uint64,
	rollRequestID string,
) ([]action.OutcomeAppliedEffect, error) {
	res, err := h.resolveGMConsequence(ctx, campaignID, sessionID, rollSeq, rollRequestID)
	if err != nil {
		return nil, err
	}

	effects := make([]action.OutcomeAppliedEffect, 0, 2)
	if res.NeedsGate {
		effects = append(effects, action.OutcomeAppliedEffect{
			Type:        "session.gate_opened",
			EntityType:  "session_gate",
			EntityID:    res.GateID,
			PayloadJSON: res.GatePayloadJSON,
		})
	}
	if res.NeedsSpotlight {
		effects = append(effects, action.OutcomeAppliedEffect{
			Type:        "session.spotlight_set",
			EntityType:  "session_spotlight",
			EntityID:    sessionID,
			PayloadJSON: res.SpotlightPayloadJSON,
		})
	}
	return effects, nil
}

// openGMConsequenceGate repairs the session gate and spotlight immediately for
// idempotent retries that must still surface an open consequence.
func (h *Handler) openGMConsequenceGate(ctx context.Context, campaignID, sessionID, sceneID string, rollSeq uint64, rollRequestID string) error {
	return gmconsequence.Open(
		ctx,
		h.gmConsequenceDependencies(),
		campaignID,
		sessionID,
		sceneID,
		rollRequestID,
		grpcmeta.InvocationIDFromContext(ctx),
		&rollSeq,
	)
}

func (h *Handler) gmConsequenceDependencies() gmconsequence.Dependencies {
	return gmconsequence.Dependencies{
		SessionGate:      h.deps.SessionGate,
		SessionSpotlight: h.deps.SessionSpotlight,
		ExecuteCoreCommand: func(ctx context.Context, in gmconsequence.CoreCommandInput) error {
			return h.deps.ExecuteCoreCommand(ctx, CoreCommandInput{
				CampaignID:      in.CampaignID,
				CommandType:     in.CommandType,
				SessionID:       in.SessionID,
				SceneID:         in.SceneID,
				RequestID:       in.RequestID,
				InvocationID:    in.InvocationID,
				EntityType:      in.EntityType,
				EntityID:        in.EntityID,
				PayloadJSON:     in.PayloadJSON,
				MissingEventMsg: in.MissingEventMsg,
				ApplyErrMessage: in.ApplyErrMessage,
			})
		},
	}
}

// ensureNoOpenSessionGate blocks outcome writes while a session gate remains
// unresolved.
func (h *Handler) ensureNoOpenSessionGate(ctx context.Context, campaignID, sessionID string) error {
	if h.deps.SessionGate == nil {
		return status.Error(codes.Internal, "session gate store is not configured")
	}
	if strings.TrimSpace(campaignID) == "" || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	gate, err := h.deps.SessionGate.GetOpenSessionGate(ctx, campaignID, sessionID)
	if err == nil {
		return status.Errorf(codes.FailedPrecondition, "session gate is open: %s", gate.GateID)
	}
	return grpcerror.OptionalLookupErrorContext(ctx, err, "load session gate")
}

// requireSessionOutcomeDependencies checks the read-side dependencies shared by
// the session-level outcome handlers.
func (h *Handler) requireSessionOutcomeDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	default:
		return nil
	}
}

// requireRollOutcomeDependencies checks the broader read/write dependencies
// needed by ApplyRollOutcome.
func (h *Handler) requireRollOutcomeDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Content == nil:
		return status.Error(codes.Internal, "content store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.ExecuteSystemCommand == nil || h.deps.ExecuteCoreCommand == nil || h.deps.ApplyStressVulnerableCondition == nil:
		return status.Error(codes.Internal, "domain engine is not configured")
	default:
		return nil
	}
}

func (h *Handler) activeSubclassRuleSummary(ctx context.Context, profile projectionstore.DaggerheartCharacterProfile) (daggerheartstate.ActiveSubclassRuleSummary, error) {
	if len(profile.SubclassTracks) == 0 {
		return daggerheartstate.ActiveSubclassRuleSummary{}, nil
	}
	typed := daggerheartstate.CharacterProfileFromStorage(profile)
	featureSets, err := daggerheartstate.ActiveSubclassTrackFeaturesFromLoader(ctx, h.deps.Content.GetDaggerheartSubclass, typed.SubclassTracks)
	if err != nil {
		return daggerheartstate.ActiveSubclassRuleSummary{}, handleDomainError(ctx, err)
	}
	return daggerheartstate.SummarizeActiveSubclassRules(daggerheartstate.FlattenActiveSubclassFeatures(featureSets)), nil
}

// campaignSupportsDaggerheart reports whether the campaign belongs to the
// Daggerheart system.
func campaignSupportsDaggerheart(record storage.CampaignRecord) bool {
	systemID, ok := systembridge.NormalizeSystemID(record.System.String())
	return ok && systemID == systembridge.SystemIDDaggerheart
}

// requireDaggerheartSystem enforces that the transport only runs against
// Daggerheart campaigns.
func requireDaggerheartSystem(record storage.CampaignRecord, unsupportedMessage string) error {
	if campaignSupportsDaggerheart(record) {
		return nil
	}
	return status.Error(codes.FailedPrecondition, unsupportedMessage)
}

// handleDomainError preserves the existing status-code mapping used by the root
// Daggerheart service.
func handleDomainError(ctx context.Context, err error) error {
	return grpcerror.HandleDomainErrorContext(ctx, err)
}

// clamp keeps integer state transitions inside their legal domain bounds.
func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

package outcometransport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/gmconsequence"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	outcomeFlavorHope = "HOPE"
	outcomeFlavorFear = "FEAR"
)

const (
	commandTypeActionOutcomeApply             = commandids.ActionOutcomeApply
	commandTypeSessionGateOpen                = commandids.SessionGateOpen
	commandTypeSessionSpotlightSet            = commandids.SessionSpotlightSet
	commandTypeDaggerheartCharacterStatePatch = commandids.DaggerheartCharacterStatePatch
	commandTypeDaggerheartGMFearSet           = commandids.DaggerheartGMFearSet
)

const (
	eventTypeActionOutcomeApplied           = action.EventTypeOutcomeApplied
	eventTypeActionRollResolved             = action.EventTypeRollResolved
	eventTypeDaggerheartCharacterStatePatch = bridge.EventTypeCharacterStatePatched
	eventTypeDaggerheartGMFearChanged       = bridge.EventTypeGMFearChanged
)

// Handler owns the Daggerheart outcome transport surface behind an explicit
// dependency bundle so the root package can stay a thin facade.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a Daggerheart outcome transport handler.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

// ApplyRollOutcome applies the durable side effects of a previously resolved
// roll and returns the updated projection state.
func (h *Handler) ApplyRollOutcome(ctx context.Context, in *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply roll outcome request is required")
	}
	if err := h.requireRollOutcomeDependencies(); err != nil {
		return nil, err
	}

	campaignID, err := validate.RequiredID(grpcmeta.CampaignIDFromContext(ctx), "campaign id")
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
	sceneID := strings.TrimSpace(in.GetSceneId())
	if in.GetRollSeq() == 0 {
		return nil, status.Error(codes.InvalidArgument, "roll seq is required")
	}

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart outcomes"); err != nil {
		return nil, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	rollEvent, err := h.deps.Event.GetEventBySeq(ctx, campaignID, in.GetRollSeq())
	if err != nil {
		return nil, handleDomainError(err)
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
	invocationID := grpcmeta.InvocationIDFromContext(ctx)

	rollKind := rollMetadata.RollKindOrDefault()
	generateHopeFear := workflowtransport.BoolValue(rollMetadata.HopeFear, rollKind != pb.RollKind_ROLL_KIND_REACTION)
	triggerGMMove := workflowtransport.BoolValue(rollMetadata.GMMove, rollKind != pb.RollKind_ROLL_KIND_REACTION)
	rollOutcome := rollMetadata.OutcomeOrFallback(rollPayload.Outcome)
	if rollOutcome == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is required")
	}
	if in.GetSwapHopeFear() {
		rollOutcome = swapHopeFearOutcomeCode(rollOutcome)
	}
	flavor := workflowtransport.OutcomeFlavorFromCode(rollOutcome)
	if flavor == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome flavor is required")
	}
	if !generateHopeFear {
		flavor = ""
	}
	crit := workflowtransport.BoolValue(rollMetadata.Crit, strings.TrimSpace(rollOutcome) == pb.Outcome_CRITICAL_SUCCESS.String())

	targets := workflowtransport.NormalizeTargets(in.GetTargets())
	if len(targets) == 0 {
		rollerID := strings.TrimSpace(rollMetadata.CharacterID)
		if strings.TrimSpace(rollerID) == "" {
			return nil, status.Error(codes.InvalidArgument, "targets are required")
		}
		targets = []string{rollerID}
	}

	gmFearDelta := 0
	if triggerGMMove && flavor == outcomeFlavorFear && !crit {
		gmFearDelta = len(targets)
	}
	requiresComplication := flavor == outcomeFlavorFear && !crit && triggerGMMove

	alreadyApplied, err := h.outcomeAlreadyAppliedForSessionRequest(ctx, campaignID, sessionID, in.GetRollSeq(), rollRequestID)
	if err != nil {
		return nil, grpcerror.Internal("check outcome applied", err)
	}
	if alreadyApplied {
		if requiresComplication {
			if err := h.openGMConsequenceGate(ctx, campaignID, sessionID, sceneID, in.GetRollSeq(), rollRequestID); err != nil {
				return nil, err
			}
		}
		return h.buildApplyRollOutcomeIdempotentResponse(ctx, campaignID, in.GetRollSeq(), targets, requiresComplication, gmFearDelta > 0)
	}
	if err := h.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	changes := make([]action.OutcomeAppliedChange, 0)
	postEffects := make([]action.OutcomeAppliedEffect, 0)
	updatedStates := make([]*pb.OutcomeCharacterState, 0, len(targets))

	gmFearAlreadyApplied := false
	if gmFearDelta > 0 {
		gmFearAlreadyApplied, err = h.sessionRequestEventExists(
			ctx,
			campaignID,
			sessionID,
			in.GetRollSeq(),
			rollRequestID,
			eventTypeDaggerheartGMFearChanged,
			campaignID,
		)
		if err != nil {
			return nil, grpcerror.Internal("check gm fear applied", err)
		}
	}

	if gmFearDelta > 0 && !gmFearAlreadyApplied {
		currentSnap, err := h.deps.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return nil, grpcerror.Internal("load gm fear", err)
		}
		beforeFear := currentSnap.GMFear
		before, after, err := bridge.ApplyGMFearGain(beforeFear, gmFearDelta)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "gm fear update invalid: %v", err)
		}

		payload := bridge.GMFearSetPayload{After: &after}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return nil, grpcerror.Internal("encode gm fear payload", err)
		}

		if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandTypeDaggerheartGMFearSet,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       rollRequestID,
			InvocationID:    invocationID,
			CorrelationID:   rollRequestID,
			EntityType:      "campaign",
			EntityID:        campaignID,
			PayloadJSON:     payloadJSON,
			MissingEventMsg: "gm fear update did not emit an event",
			ApplyErrMessage: "apply gm fear event",
		}); err != nil {
			return nil, err
		}

		changes = append(changes, action.OutcomeAppliedChange{Field: action.OutcomeFieldGMFear, Before: before, After: after})
	}

	for _, target := range targets {
		profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, target)
		if err != nil {
			return nil, handleDomainError(err)
		}
		subclassRules, err := h.activeSubclassRuleSummary(ctx, profile)
		if err != nil {
			return nil, err
		}
		state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, target)
		if err != nil {
			return nil, handleDomainError(err)
		}

		hopeBefore := state.Hope
		stressBefore := state.Stress
		hopeMax := state.HopeMax
		if hopeMax == 0 {
			hopeMax = bridge.HopeMax
		}
		hopeAfter := hopeBefore
		stressAfter := stressBefore
		if generateHopeFear && flavor == outcomeFlavorHope {
			hopeAfter = clamp(hopeBefore+1, bridge.HopeMin, hopeMax)
		}
		if generateHopeFear && flavor == outcomeFlavorFear && subclassRules.GainHopeOnFailureWithFearAmount > 0 {
			if success, known := workflowtransport.OutcomeSuccessFromCode(rollOutcome); known && !success {
				hopeAfter = clamp(hopeAfter+subclassRules.GainHopeOnFailureWithFearAmount, bridge.HopeMin, hopeMax)
			}
		}
		if generateHopeFear && crit {
			stressAfter = clamp(stressBefore-1, bridge.StressMin, profile.StressMax)
		}

		if hopeAfter != hopeBefore || stressAfter != stressBefore {
			characterPatchAlreadyApplied, err := h.sessionRequestEventExists(
				ctx,
				campaignID,
				sessionID,
				in.GetRollSeq(),
				rollRequestID,
				eventTypeDaggerheartCharacterStatePatch,
				target,
			)
			if err != nil {
				return nil, grpcerror.Internal("check character state patch applied", err)
			}
			if !characterPatchAlreadyApplied {
				payload := bridge.CharacterStatePatchPayload{
					CharacterID:  ids.CharacterID(target),
					HopeBefore:   &hopeBefore,
					HopeAfter:    &hopeAfter,
					StressBefore: &stressBefore,
					StressAfter:  &stressAfter,
				}
				payloadJSON, err := json.Marshal(payload)
				if err != nil {
					return nil, grpcerror.Internal("encode character state payload", err)
				}
				if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
					CampaignID:      campaignID,
					CommandType:     commandTypeDaggerheartCharacterStatePatch,
					SessionID:       sessionID,
					SceneID:         sceneID,
					RequestID:       rollRequestID,
					InvocationID:    invocationID,
					CorrelationID:   rollRequestID,
					EntityType:      "character",
					EntityID:        target,
					PayloadJSON:     payloadJSON,
					MissingEventMsg: "character state update did not emit an event",
					ApplyErrMessage: "apply character state event",
				}); err != nil {
					return nil, err
				}
			}

			rollSeq := in.GetRollSeq()
			err = h.deps.ApplyStressVulnerableCondition(ctx, ApplyStressVulnerableConditionInput{
				CampaignID:    campaignID,
				SessionID:     sessionID,
				CharacterID:   target,
				Conditions:    state.Conditions,
				StressBefore:  stressBefore,
				StressAfter:   stressAfter,
				StressMax:     profile.StressMax,
				RollSeq:       &rollSeq,
				RequestID:     rollRequestID,
				CorrelationID: rollRequestID,
			})
			if err != nil {
				return nil, err
			}
		}

		if hopeAfter != hopeBefore {
			changes = append(changes, action.OutcomeAppliedChange{CharacterID: ids.CharacterID(target), Field: action.OutcomeFieldHope, Before: hopeBefore, After: hopeAfter})
		}
		if stressAfter != stressBefore {
			changes = append(changes, action.OutcomeAppliedChange{CharacterID: ids.CharacterID(target), Field: action.OutcomeFieldStress, Before: stressBefore, After: stressAfter})
		}
		updatedStates = append(updatedStates, &pb.OutcomeCharacterState{
			CharacterId: target,
			Hope:        int32(hopeAfter),
			Stress:      int32(stressAfter),
			Hp:          int32(state.Hp),
		})
	}

	if requiresComplication {
		postEffects, err = h.buildGMConsequenceOutcomeEffects(ctx, campaignID, sessionID, in.GetRollSeq(), rollRequestID)
		if err != nil {
			return nil, err
		}
	}

	payload := action.OutcomeApplyPayload{
		RequestID:            rollRequestID,
		RollSeq:              in.GetRollSeq(),
		Targets:              targets,
		RequiresComplication: requiresComplication,
		AppliedChanges:       changes,
		PostEffects:          postEffects,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, grpcerror.Internal("encode outcome payload", err)
	}

	if err := h.deps.ExecuteCoreCommand(ctx, CoreCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandTypeActionOutcomeApply,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       rollRequestID,
		InvocationID:    invocationID,
		CorrelationID:   rollRequestID,
		EntityType:      "outcome",
		EntityID:        rollRequestID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "outcome did not emit an event",
		ApplyErrMessage: "execute domain command",
	}); err != nil {
		return nil, err
	}

	response := &pb.ApplyRollOutcomeResponse{
		RollSeq:              in.GetRollSeq(),
		RequiresComplication: requiresComplication,
		Updated: &pb.OutcomeUpdated{
			CharacterStates: updatedStates,
		},
	}
	if gmFearDelta > 0 {
		currentSnap, err := h.deps.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil {
			return nil, grpcerror.Internal("load gm fear snapshot", err)
		}
		value := int32(currentSnap.GMFear)
		response.Updated.GmFear = &value
	}

	return response, nil
}

// ApplyAttackOutcome derives the transport-level attack result from a resolved
// non-reaction roll event.
func (h *Handler) ApplyAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAttackOutcomeRequest) (*pb.DaggerheartApplyAttackOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply attack outcome request is required")
	}
	if len(in.GetTargets()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}

	pre, err := h.validateSessionOutcome(ctx, in.GetSessionId(), in.GetRollSeq())
	if err != nil {
		return nil, err
	}

	rollKind := pre.rollMetadata.RollKindOrDefault()
	if rollKind == pb.RollKind_ROLL_KIND_REACTION {
		return nil, status.Error(codes.FailedPrecondition, "roll seq references a reaction roll")
	}
	rollOutcome := pre.rollMetadata.OutcomeOrFallback(pre.rollPayload.Outcome)
	if rollOutcome == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is required")
	}
	if in.GetSwapHopeFear() {
		rollOutcome = swapHopeFearOutcomeCode(rollOutcome)
	}
	crit := workflowtransport.BoolValue(pre.rollMetadata.Crit, strings.TrimSpace(rollOutcome) == pb.Outcome_CRITICAL_SUCCESS.String())
	flavor := workflowtransport.OutcomeFlavorFromCode(rollOutcome)
	if !workflowtransport.BoolValue(pre.rollMetadata.HopeFear, true) {
		flavor = ""
	}
	rollSuccess, ok := workflowtransport.OutcomeSuccessFromCode(rollOutcome)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is invalid")
	}
	attackerID := strings.TrimSpace(pre.rollMetadata.CharacterID)
	if attackerID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	targets := workflowtransport.NormalizeTargets(in.GetTargets())
	if len(targets) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}

	return &pb.DaggerheartApplyAttackOutcomeResponse{
		RollSeq:     in.GetRollSeq(),
		CharacterId: attackerID,
		Targets:     targets,
		Result: &pb.DaggerheartAttackOutcomeResult{
			Outcome: workflowtransport.OutcomeCodeToProto(rollOutcome),
			Success: rollSuccess,
			Crit:    crit,
			Flavor:  flavor,
		},
	}, nil
}

// ApplyAdversaryAttackOutcome derives the transport-level adversary result from
// a resolved adversary roll event.
func (h *Handler) ApplyAdversaryAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAdversaryAttackOutcomeRequest) (*pb.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply adversary attack outcome request is required")
	}
	if len(in.GetTargets()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}
	if in.GetDifficulty() < 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty must be non-negative")
	}

	pre, err := h.validateSessionOutcome(ctx, in.GetSessionId(), in.GetRollSeq())
	if err != nil {
		return nil, err
	}

	rollKind := pre.rollMetadata.RollKindCode()
	if rollKind != "adversary_roll" {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference an adversary roll")
	}
	adversaryID := strings.TrimSpace(pre.rollMetadata.CharacterID)
	if adversaryID == "" {
		adversaryID = strings.TrimSpace(pre.rollMetadata.AdversaryID)
	}
	if adversaryID == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}

	targets := workflowtransport.NormalizeTargets(in.GetTargets())
	if len(targets) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}

	roll, rollHasValue := workflowtransport.IntValue(pre.rollMetadata.Roll)
	if !rollHasValue {
		return nil, status.Error(codes.InvalidArgument, "roll payload missing roll")
	}
	_, hasModifier := workflowtransport.IntValue(pre.rollMetadata.Modifier)
	if !hasModifier {
		return nil, status.Error(codes.InvalidArgument, "roll payload missing modifier")
	}
	total, hasTotal := workflowtransport.IntValue(pre.rollMetadata.Total)
	if !hasTotal {
		return nil, status.Error(codes.InvalidArgument, "roll payload missing total")
	}
	difficulty := int(in.GetDifficulty())
	success := total >= difficulty
	crit := roll == 20

	return &pb.DaggerheartApplyAdversaryAttackOutcomeResponse{
		RollSeq:     in.GetRollSeq(),
		AdversaryId: adversaryID,
		Targets:     targets,
		Result: &pb.DaggerheartAdversaryAttackOutcomeResult{
			Success:    success,
			Crit:       crit,
			Roll:       int32(roll),
			Total:      int32(total),
			Difficulty: int32(difficulty),
		},
	}, nil
}

func swapHopeFearOutcomeCode(code string) string {
	switch strings.TrimSpace(code) {
	case pb.Outcome_SUCCESS_WITH_HOPE.String():
		return pb.Outcome_SUCCESS_WITH_FEAR.String()
	case pb.Outcome_SUCCESS_WITH_FEAR.String():
		return pb.Outcome_SUCCESS_WITH_HOPE.String()
	case pb.Outcome_FAILURE_WITH_HOPE.String():
		return pb.Outcome_FAILURE_WITH_FEAR.String()
	case pb.Outcome_FAILURE_WITH_FEAR.String():
		return pb.Outcome_FAILURE_WITH_HOPE.String()
	default:
		return code
	}
}

// ApplyReactionOutcome derives the transport-level reaction result from a
// resolved reaction roll event.
func (h *Handler) ApplyReactionOutcome(ctx context.Context, in *pb.DaggerheartApplyReactionOutcomeRequest) (*pb.DaggerheartApplyReactionOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply reaction outcome request is required")
	}
	pre, err := h.validateSessionOutcome(ctx, in.GetSessionId(), in.GetRollSeq())
	if err != nil {
		return nil, err
	}

	rollKind := pre.rollMetadata.RollKindOrDefault()
	if rollKind != pb.RollKind_ROLL_KIND_REACTION {
		return nil, status.Error(codes.FailedPrecondition, "roll seq does not reference a reaction roll")
	}
	rollOutcome := pre.rollMetadata.OutcomeOrFallback(pre.rollPayload.Outcome)
	if rollOutcome == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is required")
	}
	crit := workflowtransport.BoolValue(pre.rollMetadata.Crit, strings.TrimSpace(rollOutcome) == pb.Outcome_CRITICAL_SUCCESS.String())
	rollSuccess, ok := workflowtransport.OutcomeSuccessFromCode(rollOutcome)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is invalid")
	}
	critNegates := workflowtransport.BoolValue(pre.rollMetadata.CritNegates, crit)
	effectsNegated := crit && critNegates
	actorID := strings.TrimSpace(pre.rollMetadata.CharacterID)
	if actorID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	return &pb.DaggerheartApplyReactionOutcomeResponse{
		RollSeq:     in.GetRollSeq(),
		CharacterId: actorID,
		Result: &pb.DaggerheartReactionOutcomeResult{
			Outcome:            workflowtransport.OutcomeCodeToProto(rollOutcome),
			Success:            rollSuccess,
			Crit:               crit,
			CritNegatesEffects: critNegates,
			EffectsNegated:     effectsNegated,
		},
	}, nil
}

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
		return sessionOutcomePrelude{}, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return sessionOutcomePrelude{}, handleDomainError(err)
	}
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart outcomes"); err != nil {
		return sessionOutcomePrelude{}, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return sessionOutcomePrelude{}, handleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return sessionOutcomePrelude{}, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := h.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return sessionOutcomePrelude{}, err
	}

	rollEvent, err := h.deps.Event.GetEventBySeq(ctx, campaignID, rollSeq)
	if err != nil {
		return sessionOutcomePrelude{}, handleDomainError(err)
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
			return nil, handleDomainError(err)
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
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	return grpcerror.Internal("load session gate", err)
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

func (h *Handler) activeSubclassRuleSummary(ctx context.Context, profile projectionstore.DaggerheartCharacterProfile) (bridge.ActiveSubclassRuleSummary, error) {
	if len(profile.SubclassTracks) == 0 {
		return bridge.ActiveSubclassRuleSummary{}, nil
	}
	typed := bridge.CharacterProfileFromStorage(profile)
	featureSets, err := bridge.ActiveSubclassTrackFeaturesFromLoader(ctx, h.deps.Content.GetDaggerheartSubclass, typed.SubclassTracks)
	if err != nil {
		return bridge.ActiveSubclassRuleSummary{}, handleDomainError(err)
	}
	return bridge.SummarizeActiveSubclassRules(bridge.FlattenActiveSubclassFeatures(featureSets)), nil
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
func handleDomainError(err error) error {
	return grpcerror.HandleDomainError(err)
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

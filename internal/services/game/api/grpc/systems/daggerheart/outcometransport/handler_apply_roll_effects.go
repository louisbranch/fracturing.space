package outcometransport

import (
	"context"
	"encoding/json"
	"errors"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// applyRollOutcomeExecution accumulates the durable changes and response state
// produced while applying a resolved roll outcome.
type applyRollOutcomeExecution struct {
	request       *applyRollOutcomeRequest
	changes       []action.OutcomeAppliedChange
	postEffects   []action.OutcomeAppliedEffect
	updatedStates []*pb.OutcomeCharacterState
}

// newApplyRollOutcomeExecution initializes the mutable execution state sized to
// the resolved target list so later stages append without rederiving context.
func newApplyRollOutcomeExecution(request *applyRollOutcomeRequest) *applyRollOutcomeExecution {
	return &applyRollOutcomeExecution{
		request:       request,
		changes:       make([]action.OutcomeAppliedChange, 0),
		postEffects:   make([]action.OutcomeAppliedEffect, 0),
		updatedStates: make([]*pb.OutcomeCharacterState, 0, len(request.targets)),
	}
}

// applyRollOutcomeGMFear applies the campaign-level fear side effect once for
// the request if the resolved outcome semantics require it.
func (h *Handler) applyRollOutcomeGMFear(ctx context.Context, execution *applyRollOutcomeExecution) error {
	if !execution.request.hasGMFearGain() {
		return nil
	}

	alreadyApplied, err := h.sessionRequestEventExists(
		ctx,
		execution.request.campaignID,
		execution.request.sessionID,
		execution.request.rollSeq,
		execution.request.rollRequestID,
		eventTypeDaggerheartGMFearChanged,
		execution.request.campaignID,
	)
	if err != nil {
		return grpcerror.Internal("check gm fear applied", err)
	}
	if alreadyApplied {
		return nil
	}

	currentSnap, err := h.deps.Daggerheart.GetDaggerheartSnapshot(ctx, execution.request.campaignID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return grpcerror.Internal("load gm fear", err)
	}
	beforeFear := currentSnap.GMFear
	before, after, err := bridge.ApplyGMFearGain(beforeFear, execution.request.gmFearDelta)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "gm fear update invalid: %v", err)
	}

	payload := bridge.GMFearSetPayload{After: &after}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return grpcerror.Internal("encode gm fear payload", err)
	}

	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      execution.request.campaignID,
		CommandType:     commandTypeDaggerheartGMFearSet,
		SessionID:       execution.request.sessionID,
		SceneID:         execution.request.sceneID,
		RequestID:       execution.request.rollRequestID,
		InvocationID:    execution.request.invocationID,
		CorrelationID:   execution.request.rollRequestID,
		EntityType:      "campaign",
		EntityID:        execution.request.campaignID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "gm fear update did not emit an event",
		ApplyErrMessage: "apply gm fear event",
	}); err != nil {
		return err
	}

	execution.changes = append(
		execution.changes,
		action.OutcomeAppliedChange{Field: action.OutcomeFieldGMFear, Before: before, After: after},
	)
	return nil
}

// applyRollOutcomeCharacterStates applies each target's hope/stress effects and
// captures the resulting response state for the transport response.
func (h *Handler) applyRollOutcomeCharacterStates(ctx context.Context, execution *applyRollOutcomeExecution) error {
	for _, target := range execution.request.targets {
		if err := h.applyRollOutcomeCharacterState(ctx, execution, target); err != nil {
			return err
		}
	}
	return nil
}

// applyRollOutcomeCharacterState keeps the per-target state patch and stress
// side effect together so multi-target rolls reuse one consistent flow.
func (h *Handler) applyRollOutcomeCharacterState(
	ctx context.Context,
	execution *applyRollOutcomeExecution,
	target string,
) error {
	profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, execution.request.campaignID, target)
	if err != nil {
		return grpcerror.HandleDomainError(err)
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, execution.request.campaignID, target)
	if err != nil {
		return grpcerror.HandleDomainError(err)
	}

	hopeBefore := state.Hope
	stressBefore := state.Stress
	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = bridge.HopeMax
	}
	hopeAfter := hopeBefore
	stressAfter := stressBefore
	if execution.request.generateHopeFear && execution.request.flavor == outcomeFlavorHope {
		hopeAfter = clamp(hopeBefore+1, bridge.HopeMin, hopeMax)
	}
	if execution.request.generateHopeFear && execution.request.crit {
		stressAfter = clamp(stressBefore-1, bridge.StressMin, profile.StressMax)
	}

	if hopeAfter != hopeBefore || stressAfter != stressBefore {
		if err := h.applyOutcomeCharacterPatch(ctx, execution, target, profile.StressMax, state.Conditions, hopeBefore, hopeAfter, stressBefore, stressAfter); err != nil {
			return err
		}
	}

	if hopeAfter != hopeBefore {
		execution.changes = append(
			execution.changes,
			action.OutcomeAppliedChange{
				CharacterID: ids.CharacterID(target),
				Field:       action.OutcomeFieldHope,
				Before:      hopeBefore,
				After:       hopeAfter,
			},
		)
	}
	if stressAfter != stressBefore {
		execution.changes = append(
			execution.changes,
			action.OutcomeAppliedChange{
				CharacterID: ids.CharacterID(target),
				Field:       action.OutcomeFieldStress,
				Before:      stressBefore,
				After:       stressAfter,
			},
		)
	}
	execution.updatedStates = append(execution.updatedStates, &pb.OutcomeCharacterState{
		CharacterId: target,
		Hope:        int32(hopeAfter),
		Stress:      int32(stressAfter),
		Hp:          int32(state.Hp),
	})

	return nil
}

// applyOutcomeCharacterPatch replays the character patch idempotently, then
// repairs vulnerable-condition state from the stress transition.
func (h *Handler) applyOutcomeCharacterPatch(
	ctx context.Context,
	execution *applyRollOutcomeExecution,
	target string,
	stressMax int,
	conditions []string,
	hopeBefore int,
	hopeAfter int,
	stressBefore int,
	stressAfter int,
) error {
	alreadyApplied, err := h.sessionRequestEventExists(
		ctx,
		execution.request.campaignID,
		execution.request.sessionID,
		execution.request.rollSeq,
		execution.request.rollRequestID,
		eventTypeDaggerheartCharacterStatePatch,
		target,
	)
	if err != nil {
		return grpcerror.Internal("check character state patch applied", err)
	}
	if !alreadyApplied {
		payload := bridge.CharacterStatePatchPayload{
			CharacterID:  ids.CharacterID(target),
			HopeBefore:   &hopeBefore,
			HopeAfter:    &hopeAfter,
			StressBefore: &stressBefore,
			StressAfter:  &stressAfter,
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return grpcerror.Internal("encode character state payload", err)
		}
		if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
			CampaignID:      execution.request.campaignID,
			CommandType:     commandTypeDaggerheartCharacterStatePatch,
			SessionID:       execution.request.sessionID,
			SceneID:         execution.request.sceneID,
			RequestID:       execution.request.rollRequestID,
			InvocationID:    execution.request.invocationID,
			CorrelationID:   execution.request.rollRequestID,
			EntityType:      "character",
			EntityID:        target,
			PayloadJSON:     payloadJSON,
			MissingEventMsg: "character state update did not emit an event",
			ApplyErrMessage: "apply character state event",
		}); err != nil {
			return err
		}
	}

	rollSeq := execution.request.rollSeq
	return h.deps.ApplyStressVulnerableCondition(ctx, ApplyStressVulnerableConditionInput{
		CampaignID:    execution.request.campaignID,
		SessionID:     execution.request.sessionID,
		CharacterID:   target,
		Conditions:    conditions,
		StressBefore:  stressBefore,
		StressAfter:   stressAfter,
		StressMax:     stressMax,
		RollSeq:       &rollSeq,
		RequestID:     execution.request.rollRequestID,
		CorrelationID: execution.request.rollRequestID,
	})
}

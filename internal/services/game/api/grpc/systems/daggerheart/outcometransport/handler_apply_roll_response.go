package outcometransport

import (
	"context"
	"encoding/json"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
)

// applyRollOutcomePostEffects loads the follow-up effects only for rolls that
// must leave a GM consequence gate open after the durable apply step.
func (h *Handler) applyRollOutcomePostEffects(ctx context.Context, execution *applyRollOutcomeExecution) error {
	if !execution.request.requiresComplication {
		return nil
	}

	postEffects, err := h.buildGMConsequenceOutcomeEffects(
		ctx,
		execution.request.campaignID,
		execution.request.sessionID,
		execution.request.rollSeq,
		execution.request.rollRequestID,
	)
	if err != nil {
		return err
	}
	execution.postEffects = postEffects
	return nil
}

// persistApplyRollOutcome records the aggregate outcome event after all
// transport-owned side effects have been resolved.
func (h *Handler) persistApplyRollOutcome(ctx context.Context, execution *applyRollOutcomeExecution) error {
	payload := action.OutcomeApplyPayload{
		RequestID:            execution.request.rollRequestID,
		RollSeq:              execution.request.rollSeq,
		Targets:              execution.request.targets,
		RequiresComplication: execution.request.requiresComplication,
		AppliedChanges:       execution.changes,
		PostEffects:          execution.postEffects,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return grpcerror.Internal("encode outcome payload", err)
	}

	return h.deps.ExecuteCoreCommand(ctx, CoreCommandInput{
		CampaignID:      execution.request.campaignID,
		CommandType:     commandTypeActionOutcomeApply,
		SessionID:       execution.request.sessionID,
		SceneID:         execution.request.sceneID,
		RequestID:       execution.request.rollRequestID,
		InvocationID:    execution.request.invocationID,
		CorrelationID:   execution.request.rollRequestID,
		EntityType:      "outcome",
		EntityID:        execution.request.rollRequestID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "outcome did not emit an event",
		ApplyErrMessage: "execute domain command",
	})
}

// buildApplyRollOutcomeResponse reloads any campaign-level GM-fear projection
// state needed by the response after the outcome command has been persisted.
func (h *Handler) buildApplyRollOutcomeResponse(
	ctx context.Context,
	execution *applyRollOutcomeExecution,
) (*pb.ApplyRollOutcomeResponse, error) {
	response := &pb.ApplyRollOutcomeResponse{
		RollSeq:              execution.request.rollSeq,
		RequiresComplication: execution.request.requiresComplication,
		Updated: &pb.OutcomeUpdated{
			CharacterStates: execution.updatedStates,
		},
	}
	if execution.request.hasGMFearGain() {
		currentSnap, err := h.deps.Daggerheart.GetDaggerheartSnapshot(ctx, execution.request.campaignID)
		if err != nil {
			return nil, grpcerror.Internal("load gm fear snapshot", err)
		}
		value := int32(currentSnap.GMFear)
		response.Updated.GmFear = &value
	}

	return response, nil
}

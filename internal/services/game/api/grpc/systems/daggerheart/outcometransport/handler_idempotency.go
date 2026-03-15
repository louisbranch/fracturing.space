package outcometransport

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

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
			return nil, grpcerror.HandleDomainError(err)
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

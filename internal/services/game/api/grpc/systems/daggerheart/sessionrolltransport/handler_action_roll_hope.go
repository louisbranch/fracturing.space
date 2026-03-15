package sessionrolltransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) applyActionRollHopeSpends(
	ctx context.Context,
	actionRoll actionRollContext,
	rollSeq uint64,
	requestID string,
	invocationID string,
) error {
	if actionRoll.RollKind != pb.RollKind_ROLL_KIND_ACTION || actionRoll.SpendEventCount == 0 {
		return nil
	}
	if actionRoll.CharacterState.Hope < actionRoll.SpendTotal {
		return status.Error(codes.FailedPrecondition, "insufficient hope")
	}

	hopeAfter := actionRoll.CharacterState.Hope
	for _, spend := range actionRoll.HopeSpends {
		if spend.Amount <= 0 {
			continue
		}
		before := hopeAfter
		after := before - spend.Amount
		if err := h.deps.ExecuteHopeSpend(ctx, HopeSpendInput{
			CampaignID:   actionRoll.CampaignID,
			SessionID:    actionRoll.SessionID,
			SceneID:      actionRoll.SceneID,
			RequestID:    requestID,
			InvocationID: invocationID,
			CharacterID:  actionRoll.CharacterID,
			Source:       spend.Source,
			Amount:       spend.Amount,
			HopeBefore:   before,
			HopeAfter:    after,
			RollSeq:      rollSeq,
		}); err != nil {
			return err
		}
		hopeAfter = after
	}

	return nil
}

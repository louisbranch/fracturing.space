package outcometransport

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

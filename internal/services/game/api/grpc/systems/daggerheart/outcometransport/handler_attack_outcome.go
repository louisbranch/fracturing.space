package outcometransport

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

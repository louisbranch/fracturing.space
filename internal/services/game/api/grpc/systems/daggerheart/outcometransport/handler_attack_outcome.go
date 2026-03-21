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

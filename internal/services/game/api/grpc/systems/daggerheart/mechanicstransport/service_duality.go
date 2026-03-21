package mechanicstransport

import (
	"context"
	"errors"
	"fmt"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DualityOutcome evaluates a deterministic duality request without mutating
// state.
func (h *Handler) DualityOutcome(ctx context.Context, in *pb.DualityOutcomeRequest) (*pb.DualityOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "duality outcome request is required")
	}

	var difficulty *int
	if in.Difficulty != nil {
		value := int(*in.Difficulty)
		difficulty = &value
	}

	result, err := daggerheartdomain.EvaluateOutcome(daggerheartdomain.OutcomeRequest{
		Hope:       int(in.GetHope()),
		Fear:       int(in.GetFear()),
		Modifier:   int(in.GetModifier()),
		Difficulty: difficulty,
	})
	if err != nil {
		if errors.Is(err, daggerheartdomain.ErrInvalidDifficulty) || errors.Is(err, daggerheartdomain.ErrInvalidDualityDie) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, grpcerror.Internal("failed to evaluate outcome", err)
	}

	response := &pb.DualityOutcomeResponse{
		Hope:            int32(result.Hope),
		Fear:            int32(result.Fear),
		Modifier:        int32(result.Modifier),
		Total:           int32(result.Total),
		IsCrit:          result.IsCrit,
		MeetsDifficulty: result.MeetsDifficulty,
		Outcome:         outcomeToProto(result.Outcome),
	}
	if result.Difficulty != nil {
		value := int32(*result.Difficulty)
		response.Difficulty = &value
	}

	return response, nil
}

// DualityExplain provides the deterministic reasoning trace for a duality
// outcome request.
func (h *Handler) DualityExplain(ctx context.Context, in *pb.DualityExplainRequest) (*pb.DualityExplainResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "duality explain request is required")
	}

	var difficulty *int
	if in.Difficulty != nil {
		value := int(*in.Difficulty)
		difficulty = &value
	}

	result, err := daggerheartdomain.ExplainOutcome(daggerheartdomain.OutcomeRequest{
		Hope:       int(in.GetHope()),
		Fear:       int(in.GetFear()),
		Modifier:   int(in.GetModifier()),
		Difficulty: difficulty,
	})
	if err != nil {
		if errors.Is(err, daggerheartdomain.ErrInvalidDifficulty) || errors.Is(err, daggerheartdomain.ErrInvalidDualityDie) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, grpcerror.Internal("failed to explain outcome", err)
	}

	response := &pb.DualityExplainResponse{
		Hope:            int32(result.Hope),
		Fear:            int32(result.Fear),
		Modifier:        int32(result.Modifier),
		Total:           int32(result.Total),
		IsCrit:          result.IsCrit,
		MeetsDifficulty: result.MeetsDifficulty,
		Outcome:         outcomeToProto(result.Outcome),
		RulesVersion:    result.RulesVersion,
		Intermediates: &pb.Intermediates{
			BaseTotal:       int32(result.Intermediates.BaseTotal),
			Total:           int32(result.Intermediates.Total),
			IsCrit:          result.Intermediates.IsCrit,
			MeetsDifficulty: result.Intermediates.MeetsDifficulty,
			HopeGtFear:      result.Intermediates.HopeGtFear,
			FearGtHope:      result.Intermediates.FearGtHope,
		},
		Steps: make([]*pb.ExplainStep, 0, len(result.Steps)),
	}
	if result.Difficulty != nil {
		value := int32(*result.Difficulty)
		response.Difficulty = &value
	}

	for _, step := range result.Steps {
		data, err := stepDataToStruct(step.Data)
		if err != nil {
			return nil, grpcerror.Internal(fmt.Sprintf("failed to encode step %s", step.Code), err)
		}
		response.Steps = append(response.Steps, &pb.ExplainStep{
			Code:    step.Code,
			Message: step.Message,
			Data:    data,
		})
	}

	return response, nil
}

// DualityProbability computes aggregate outcome counts for duality dice.
func (h *Handler) DualityProbability(ctx context.Context, in *pb.DualityProbabilityRequest) (*pb.DualityProbabilityResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "duality probability request is required")
	}

	result, err := daggerheartdomain.DualityProbability(daggerheartdomain.ProbabilityRequest{
		Modifier:   int(in.GetModifier()),
		Difficulty: int(in.GetDifficulty()),
	})
	if err != nil {
		if errors.Is(err, daggerheartdomain.ErrInvalidDifficulty) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, grpcerror.Internal("failed to compute probability", err)
	}

	response := &pb.DualityProbabilityResponse{
		TotalOutcomes: int32(result.TotalOutcomes),
		CritCount:     int32(result.CritCount),
		SuccessCount:  int32(result.SuccessCount),
		FailureCount:  int32(result.FailureCount),
		OutcomeCounts: make([]*pb.OutcomeCount, 0, len(result.OutcomeCounts)),
	}
	for _, count := range result.OutcomeCounts {
		response.OutcomeCounts = append(response.OutcomeCounts, &pb.OutcomeCount{
			Outcome: outcomeToProto(count.Outcome),
			Count:   int32(count.Count),
		})
	}

	return response, nil
}

// RulesVersion returns static ruleset metadata used by duality evaluation and
// explanation responses.
func (h *Handler) RulesVersion(ctx context.Context, in *pb.RulesVersionRequest) (*pb.RulesVersionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "rules version request is required")
	}

	metadata := daggerheartdomain.RulesVersion()
	outcomes := make([]pb.Outcome, 0, len(metadata.Outcomes))
	for _, outcome := range metadata.Outcomes {
		outcomes = append(outcomes, outcomeToProto(outcome))
	}

	return &pb.RulesVersionResponse{
		System:         metadata.System,
		Module:         metadata.Module,
		RulesVersion:   metadata.RulesVersion,
		DiceModel:      metadata.DiceModel,
		TotalFormula:   metadata.TotalFormula,
		CritRule:       metadata.CritRule,
		DifficultyRule: metadata.DifficultyRule,
		Outcomes:       outcomes,
	}, nil
}

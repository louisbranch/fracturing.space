package grpc

import (
	"context"
	"errors"
	"fmt"

	pb "github.com/louisbranch/duality-protocol/api/gen/go/duality/v1"
	"github.com/louisbranch/duality-protocol/internal/duality"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// DualityService implements the Duality gRPC API.
type DualityService struct {
	pb.UnimplementedDualityServiceServer
	seedFunc func() (int64, error) // Generates per-request random seeds.
}

// NewDualityService creates a configured gRPC handler with a seed generator.
func NewDualityService(seedFunc func() (int64, error)) *DualityService {
	return &DualityService{seedFunc: seedFunc}
}

// ActionRoll handles action roll requests.
func (s *DualityService) ActionRoll(ctx context.Context, in *pb.ActionRollRequest) (*pb.ActionRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "action roll request is required")
	}
	if s.seedFunc == nil {
		return nil, status.Error(codes.Internal, "seed generator is not configured")
	}

	// TODO: Expose the seed in the gRPC request/response once the API supports it.
	seed, err := s.seedFunc()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate seed: %v", err)
	}

	var difficulty *int
	if in.Difficulty != nil {
		value := int(*in.Difficulty)
		difficulty = &value
	}

	result, err := duality.RollAction(duality.ActionRequest{
		Modifier:   int(in.GetModifier()),
		Difficulty: difficulty,
		Seed:       seed,
	})
	if err != nil {
		if errors.Is(err, duality.ErrInvalidDifficulty) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to roll action: %v", err)
	}

	response := &pb.ActionRollResponse{
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

// DualityOutcome evaluates a deterministic duality outcome request.
func (s *DualityService) DualityOutcome(ctx context.Context, in *pb.DualityOutcomeRequest) (*pb.DualityOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "duality outcome request is required")
	}

	var difficulty *int
	if in.Difficulty != nil {
		value := int(*in.Difficulty)
		difficulty = &value
	}

	result, err := duality.EvaluateOutcome(duality.OutcomeRequest{
		Hope:       int(in.GetHope()),
		Fear:       int(in.GetFear()),
		Modifier:   int(in.GetModifier()),
		Difficulty: difficulty,
	})
	if err != nil {
		if errors.Is(err, duality.ErrInvalidDifficulty) || errors.Is(err, duality.ErrInvalidDualityDie) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to evaluate outcome: %v", err)
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

// DualityExplain provides a deterministic explanation for a duality outcome.
func (s *DualityService) DualityExplain(ctx context.Context, in *pb.DualityExplainRequest) (*pb.DualityExplainResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "duality explain request is required")
	}

	var difficulty *int
	if in.Difficulty != nil {
		value := int(*in.Difficulty)
		difficulty = &value
	}

	result, err := duality.ExplainOutcome(duality.OutcomeRequest{
		Hope:       int(in.GetHope()),
		Fear:       int(in.GetFear()),
		Modifier:   int(in.GetModifier()),
		Difficulty: difficulty,
	})
	if err != nil {
		if errors.Is(err, duality.ErrInvalidDifficulty) || errors.Is(err, duality.ErrInvalidDualityDie) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to explain outcome: %v", err)
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
			return nil, status.Errorf(codes.Internal, "failed to encode step %s: %v", step.Code, err)
		}
		response.Steps = append(response.Steps, &pb.ExplainStep{
			Code:    step.Code,
			Message: step.Message,
			Data:    data,
		})
	}

	return response, nil
}

// DualityProbability computes outcome probabilities for duality dice.
func (s *DualityService) DualityProbability(ctx context.Context, in *pb.DualityProbabilityRequest) (*pb.DualityProbabilityResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "duality probability request is required")
	}

	result, err := duality.DualityProbability(duality.ProbabilityRequest{
		Modifier:   int(in.GetModifier()),
		Difficulty: int(in.GetDifficulty()),
	})
	if err != nil {
		if errors.Is(err, duality.ErrInvalidDifficulty) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to compute probability: %v", err)
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

// RulesVersion returns ruleset metadata for Duality roll interpretation.
func (s *DualityService) RulesVersion(ctx context.Context, in *pb.RulesVersionRequest) (*pb.RulesVersionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "rules version request is required")
	}

	metadata := duality.RulesVersion()
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

// RollDice handles generic dice roll requests.
func (s *DualityService) RollDice(ctx context.Context, in *pb.RollDiceRequest) (*pb.RollDiceResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "dice roll request is required")
	}
	if s.seedFunc == nil {
		return nil, status.Error(codes.Internal, "seed generator is not configured")
	}

	seed, err := s.seedFunc()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate seed: %v", err)
	}

	request := duality.RollRequest{
		Dice: make([]duality.DiceSpec, 0, len(in.GetDice())),
		Seed: seed,
	}
	for _, spec := range in.GetDice() {
		request.Dice = append(request.Dice, duality.DiceSpec{
			Sides: int(spec.GetSides()),
			Count: int(spec.GetCount()),
		})
	}

	result, err := duality.RollDice(request)
	if err != nil {
		if errors.Is(err, duality.ErrMissingDice) || errors.Is(err, duality.ErrInvalidDiceSpec) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to roll dice: %v", err)
	}

	response := &pb.RollDiceResponse{
		Rolls: make([]*pb.DiceRoll, 0, len(result.Rolls)),
		Total: int32(result.Total),
	}
	for _, roll := range result.Rolls {
		response.Rolls = append(response.Rolls, &pb.DiceRoll{
			Sides:   int32(roll.Sides),
			Results: int32Slice(roll.Results),
			Total:   int32(roll.Total),
		})
	}

	return response, nil
}

func stepDataToStruct(data map[string]any) (*structpb.Struct, error) {
	if data == nil {
		return &structpb.Struct{}, nil
	}

	converted := make(map[string]any, len(data))
	for key, value := range data {
		convertedValue, err := normalizeStructValue(value)
		if err != nil {
			return nil, fmt.Errorf("invalid value for %q: %w", key, err)
		}
		converted[key] = convertedValue
	}

	return structpb.NewStruct(converted)
}

func normalizeStructValue(value any) (any, error) {
	switch typed := value.(type) {
	case nil:
		return nil, errors.New("nil values are not supported")
	case int:
		return int64(typed), nil
	case int32:
		return int64(typed), nil
	case int64:
		return typed, nil
	case float64:
		return typed, nil
	case bool:
		return typed, nil
	case string:
		return typed, nil
	case map[string]any:
		converted := make(map[string]any, len(typed))
		for key, item := range typed {
			convertedItem, err := normalizeStructValue(item)
			if err != nil {
				return nil, fmt.Errorf("invalid nested value for %q: %w", key, err)
			}
			converted[key] = convertedItem
		}
		return converted, nil
	case []any:
		converted := make([]any, 0, len(typed))
		for _, item := range typed {
			convertedItem, err := normalizeStructValue(item)
			if err != nil {
				return nil, err
			}
			converted = append(converted, convertedItem)
		}
		return converted, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", value)
	}
}

// outcomeToProto maps dice outcomes to the protobuf outcome enum.
func outcomeToProto(outcome duality.Outcome) pb.Outcome {
	switch outcome {
	case duality.OutcomeRollWithHope:
		return pb.Outcome_ROLL_WITH_HOPE
	case duality.OutcomeRollWithFear:
		return pb.Outcome_ROLL_WITH_FEAR
	case duality.OutcomeSuccessWithHope:
		return pb.Outcome_SUCCESS_WITH_HOPE
	case duality.OutcomeSuccessWithFear:
		return pb.Outcome_SUCCESS_WITH_FEAR
	case duality.OutcomeFailureWithHope:
		return pb.Outcome_FAILURE_WITH_HOPE
	case duality.OutcomeFailureWithFear:
		return pb.Outcome_FAILURE_WITH_FEAR
	case duality.OutcomeCriticalSuccess:
		return pb.Outcome_CRITICAL_SUCCESS
	default:
		return pb.Outcome_OUTCOME_UNSPECIFIED
	}
}

// int32Slice converts a slice of ints to a slice of int32.
func int32Slice(values []int) []int32 {
	if len(values) == 0 {
		return nil
	}

	converted := make([]int32, len(values))
	for i, value := range values {
		converted[i] = int32(value)
	}
	return converted
}

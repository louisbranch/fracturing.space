package mechanicstransport

import (
	"errors"
	"fmt"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

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

func outcomeToProto(outcome daggerheartdomain.Outcome) pb.Outcome {
	switch outcome {
	case daggerheartdomain.OutcomeRollWithHope:
		return pb.Outcome_ROLL_WITH_HOPE
	case daggerheartdomain.OutcomeRollWithFear:
		return pb.Outcome_ROLL_WITH_FEAR
	case daggerheartdomain.OutcomeSuccessWithHope:
		return pb.Outcome_SUCCESS_WITH_HOPE
	case daggerheartdomain.OutcomeSuccessWithFear:
		return pb.Outcome_SUCCESS_WITH_FEAR
	case daggerheartdomain.OutcomeFailureWithHope:
		return pb.Outcome_FAILURE_WITH_HOPE
	case daggerheartdomain.OutcomeFailureWithFear:
		return pb.Outcome_FAILURE_WITH_FEAR
	case daggerheartdomain.OutcomeCriticalSuccess:
		return pb.Outcome_CRITICAL_SUCCESS
	default:
		return pb.Outcome_OUTCOME_UNSPECIFIED
	}
}

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
